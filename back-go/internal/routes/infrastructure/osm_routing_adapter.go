package infrastructure

import (
	"fmt"
	"hash/fnv"
	"log"
	"math"
	"mystravastats/internal/platform/runtimeconfig"
	"mystravastats/internal/routes/application"
	routesDomain "mystravastats/internal/routes/domain"
	"mystravastats/internal/shared/domain/business"
	"sort"
	"strings"
	"time"
)

func NewOSMRoutingAdapter() *OSMRoutingAdapter {
	enabled := runtimeconfig.BoolValue("OSM_ROUTING_ENABLED", true)
	baseURL := strings.TrimRight(strings.TrimSpace(runtimeconfig.StringValue("OSM_ROUTING_BASE_URL", defaultOSMRoutingBaseURL)), "/")
	timeoutMs := runtimeconfig.OSMRoutingTimeoutMs()
	profileOverride := strings.TrimSpace(runtimeconfig.StringValue("OSM_ROUTING_PROFILE", ""))
	extractProfileEnv := strings.TrimSpace(runtimeconfig.StringValue("OSM_ROUTING_EXTRACT_PROFILE", ""))
	extractProfileCfgFile := strings.TrimSpace(runtimeconfig.StringValue("OSM_ROUTING_EXTRACT_PROFILE_FILE", defaultOSRMProfileFilePath))

	transport := newOSRMClient(baseURL, time.Duration(timeoutMs)*time.Millisecond)
	return &OSMRoutingAdapter{
		enabled:               enabled,
		v3Enabled:             runtimeconfig.BoolValue("OSM_ROUTING_V3_ENABLED", defaultOSMRoutingV3Enabled),
		debug:                 runtimeconfig.BoolValue("OSM_ROUTING_DEBUG", false),
		baseURL:               baseURL,
		timeout:               time.Duration(timeoutMs) * time.Millisecond,
		osrmClient:            transport,
		client:                transport.client,
		profileOverride:       profileOverride,
		extractProfileEnv:     extractProfileEnv,
		extractProfileCfgFile: extractProfileCfgFile,
	}
}

func (adapter *OSMRoutingAdapter) transport() *osrmClient {
	if adapter.osrmClient != nil {
		return adapter.osrmClient
	}
	return &osrmClient{baseURL: adapter.baseURL, client: adapter.client}
}

func (adapter *OSMRoutingAdapter) HealthDetails() map[string]any {
	extractProfile := adapter.detectExtractProfile()
	effectiveProfile := adapter.effectiveRoutingProfile(extractProfile)
	details := map[string]any{
		"engine":              "osrm",
		"enabled":             adapter.enabled,
		"v3Enabled":           adapter.v3Enabled,
		"debug":               adapter.debug,
		"baseUrl":             adapter.baseURL,
		"profile":             strings.TrimSpace(adapter.profileOverride),
		"extractProfile":      extractProfile,
		"effectiveProfile":    effectiveProfile,
		"supportedRouteTypes": supportedRouteTypesByProfile(extractProfile, effectiveProfile),
	}
	if !adapter.enabled {
		details["status"] = "disabled"
		details["reachable"] = false
		return details
	}
	if adapter.baseURL == "" {
		details["status"] = "misconfigured"
		details["reachable"] = false
		details["error"] = "OSM_ROUTING_BASE_URL is empty"
		return details
	}

	statusCode, err := adapter.transport().healthStatus()
	if err != nil {
		details["status"] = "down"
		details["reachable"] = false
		details["error"] = err.Error()
		return details
	}
	details["statusCode"] = statusCode
	if statusCode >= 500 {
		details["status"] = "down"
		details["reachable"] = false
		return details
	}

	details["status"] = "up"
	details["reachable"] = true
	return details
}

func (adapter *OSMRoutingAdapter) GenerateTargetLoops(
	request application.RoutingEngineRequest,
) ([]routesDomain.RouteRecommendation, error) {
	if !adapter.enabled || adapter.baseURL == "" {
		return []routesDomain.RouteRecommendation{}, nil
	}
	if request.DistanceTargetKm <= 0 || request.Limit <= 0 {
		return []routesDomain.RouteRecommendation{}, nil
	}

	profile := adapter.profileForRouteType(request.RouteType)
	usedLegacyFallback := false
	if isCustomTargetMode(request) {
		return adapter.generateCustomWaypointLoops(request, profile), nil
	}
	if adapter.v3Enabled {
		if disjointRecommendations, ok := adapter.generateTargetLoopsDisjoint(request, profile); ok {
			return disjointRecommendations, nil
		}
		usedLegacyFallback = true
		if adapter.debug {
			log.Printf("OSRM target generation v3 produced no valid route, falling back to legacy generator")
		}
	}

	baseBearing := startDirectionToBearing(request.StartDirection)
	hasDirection := strings.TrimSpace(request.StartDirection) != ""
	directionStrict := hasDirection && request.DirectionStrict
	radiusBaseKm := math.Max(1.0, request.DistanceTargetKm/(2.0*math.Pi))
	radiusMultipliers := []float64{1.00, 0.92, 1.08, 0.84, 1.16, 1.24, 0.76, 1.32, 0.68, 1.40, 1.48, 0.60}
	rotations := []float64{0, 22, -22, 45, -45, 68, -68, 95, -95, 125, -125, 155, -155}
	if hasDirection {
		// When a direction is requested in automatic mode, rotations stay tight around
		// the requested bearing to preserve a clear global orientation.
		rotations = []float64{0, 8, -8, 15, -15, 24, -24, 32, -32}
		if directionStrict {
			// Strict mode keeps the directional cone narrower.
			rotations = []float64{0, 5, -5, 10, -10, 16, -16}
		}
	}
	// Keep a high candidate pool even when request.Limit is small, otherwise
	// strict anti-backtracking filters would only have near-identical routes to choose from.
	// We intentionally explore the full candidate budget so we can keep
	// anti-overlap constraints strict while still finding a route.
	maxCalls := maxOSRMRoutingCalls

	// Pipeline:
	// 1) generate multiple OSRM candidates around the start point
	// 2) convert each route to scored candidate metrics
	// 3) deduplicate by geometry signature
	// 4) pick top routes with progressive constraint relaxation
	candidates := make([]osrmRouteCandidate, 0, request.Limit*4)
	seenSignatures := make(map[string]struct{}, request.Limit*6)
	rejectCounts := make(map[string]int)
	fetchedRouteCount := 0
	fetchErrors := 0
	generatedCount := 0

	for callIndex := 0; callIndex < maxCalls; callIndex++ {
		radiusKm := radiusBaseKm * radiusMultipliers[callIndex%len(radiusMultipliers)]
		rotation := rotations[callIndex%len(rotations)]
		waypoints := adapter.syntheticLoopWaypoints(
			request.StartPoint,
			radiusKm,
			baseBearing+rotation,
			request.StartDirection,
			request.RouteType,
			callIndex,
		)
		routes, err := adapter.fetchOSRMRoutes(profile, waypoints)
		if err != nil {
			fetchErrors++
			incrementRejectCount(rejectCounts, "OSRM_CALL_FAILED")
			if adapter.debug {
				log.Printf(
					"OSRM target generation call failed: call=%d profile=%s radiusKm=%.2f rotation=%.1f err=%v",
					callIndex+1, profile, radiusKm, rotation, err,
				)
			}
			// Do not fail the whole request: caller will fallback to in-cache generation.
			continue
		}
		fetchedRouteCount += len(routes)
		for routeIndex, osrmRoute := range routes {
			candidate, ok := adapter.toRouteCandidate(request, osrmRoute, generatedCount+routeIndex, rejectCounts)
			if !ok {
				continue
			}
			signature := routeGeometrySignature(candidate.recommendation.PreviewLatLng)
			if signature == "" {
				incrementRejectCount(rejectCounts, "EMPTY_GEOMETRY_SIGNATURE")
				continue
			}
			if _, exists := seenSignatures[signature]; exists {
				incrementRejectCount(rejectCounts, "DUPLICATE_GEOMETRY")
				continue
			}
			seenSignatures[signature] = struct{}{}
			candidates = append(candidates, candidate)
		}
		generatedCount += len(routes)
	}
	recommendations := selectCandidatesWithRelaxation(request, candidates, rejectCounts)
	if len(recommendations) > request.Limit {
		recommendations = recommendations[:request.Limit]
	}
	if len(recommendations) == 0 && strings.TrimSpace(request.StartDirection) != "" {
		// Last-resort fallback: if direction-constrained generation yields no route,
		// retry once without direction so the user still gets a practical loop.
		relaxedRequest := request
		relaxedRequest.StartDirection = ""
		relaxedRequest.DirectionStrict = false
		fallbackRecommendations, fallbackErr := adapter.GenerateTargetLoops(relaxedRequest)
		if fallbackErr == nil && len(fallbackRecommendations) > 0 {
			for index := range fallbackRecommendations {
				fallbackRecommendations[index].Reasons = append(
					fallbackRecommendations[index].Reasons,
					"Direction relaxed: no route found with requested heading",
				)
			}
			return fallbackRecommendations, nil
		}
	}
	if len(recommendations) == 0 && request.StrictBacktracking {
		// Secondary fallback: strict anti-backtracking can be too restrictive in dense
		// urban/off-road graphs. Retry once with relaxed anti-backtracking instead
		// of returning no route at all.
		relaxedRequest := request
		relaxedRequest.StrictBacktracking = false
		relaxedRequest.DirectionStrict = false
		fallbackRecommendations, fallbackErr := adapter.GenerateTargetLoops(relaxedRequest)
		if fallbackErr == nil && len(fallbackRecommendations) > 0 {
			for index := range fallbackRecommendations {
				fallbackRecommendations[index].Reasons = append(
					fallbackRecommendations[index].Reasons,
					"Anti-backtracking relaxed: strict mode found no valid loop",
				)
			}
			return fallbackRecommendations, nil
		}
	}
	if len(recommendations) == 0 {
		// Absolute fallback: snap start to nearest routable node and retry once.
		if snappedStart, snapDistanceM, snapped := adapter.snapToNearestRoutablePoint(profile, request.StartPoint); snapped {
			snapOffset := haversineDistanceMeters(request.StartPoint.Lat, request.StartPoint.Lng, snappedStart.Lat, snappedStart.Lng)
			if snapOffset > 3.0 {
				snappedRequest := request
				snappedRequest.StartPoint = snappedStart
				snappedRequest.StrictBacktracking = false
				snappedRequest.DirectionStrict = false
				snappedRequest.StartDirection = ""
				fallbackRecommendations, fallbackErr := adapter.GenerateTargetLoops(snappedRequest)
				if fallbackErr == nil && len(fallbackRecommendations) > 0 {
					for index := range fallbackRecommendations {
						fallbackRecommendations[index].Reasons = append(
							fallbackRecommendations[index].Reasons,
							fmt.Sprintf(
								"Start snapped to nearest routable point (+%.0fm from request, OSRM nearest %.0fm)",
								snapOffset,
								snapDistanceM,
							),
						)
					}
					return fallbackRecommendations, nil
				}
			}
		}
	}
	if len(recommendations) == 0 {
		// Route-type fallback chain:
		// MTB -> Gravel -> Ride
		// Gravel -> Ride
		for _, fallbackType := range fallbackRouteTypes(request.RouteType) {
			fallbackRequest := request
			fallbackRequest.RouteType = fallbackType
			fallbackRequest.StartDirection = ""
			fallbackRequest.DirectionStrict = false
			fallbackRequest.StrictBacktracking = false
			fallbackRecommendations, fallbackErr := adapter.GenerateTargetLoops(fallbackRequest)
			if fallbackErr == nil && len(fallbackRecommendations) > 0 {
				for index := range fallbackRecommendations {
					fallbackRecommendations[index].Reasons = append(
						fallbackRecommendations[index].Reasons,
						fmt.Sprintf(
							"Route type fallback: %s -> %s",
							strings.ToUpper(strings.TrimSpace(request.RouteType)),
							fallbackType,
						),
					)
				}
				return fallbackRecommendations, nil
			}
		}
	}
	if usedLegacyFallback {
		for index := range recommendations {
			recommendations[index].Reasons = append(
				recommendations[index].Reasons,
				"Generation engine fallback: legacy synthetic waypoints",
			)
		}
	}
	if adapter.debug || len(recommendations) == 0 {
		targetElevation := "n/a"
		if request.ElevationTargetM != nil {
			targetElevation = fmt.Sprintf("%.0fm", *request.ElevationTargetM)
		}
		log.Printf(
			"OSRM target generation summary: routeType=%s direction=%s target=%.1fkm/%s calls=%d fetched=%d accepted=%d fetchErrors=%d rejects=%s",
			strings.ToUpper(strings.TrimSpace(request.RouteType)),
			strings.ToUpper(strings.TrimSpace(request.StartDirection)),
			request.DistanceTargetKm,
			targetElevation,
			maxCalls,
			fetchedRouteCount,
			len(recommendations),
			fetchErrors,
			formatRejectCounts(rejectCounts),
		)
	}

	return recommendations, nil
}

func (adapter *OSMRoutingAdapter) GenerateShapeLoops(
	request application.RoutingEngineRequest,
) ([]routesDomain.RouteRecommendation, error) {
	if !adapter.enabled || adapter.baseURL == "" {
		return []routesDomain.RouteRecommendation{}, nil
	}
	if request.Limit <= 0 {
		return []routesDomain.RouteRecommendation{}, nil
	}

	shapePolyline := strings.TrimSpace(request.ShapePolyline)
	if shapePolyline == "" {
		return []routesDomain.RouteRecommendation{}, nil
	}
	rawShape := parseShapePolylineCoordinates(shapePolyline)
	if len(rawShape) < 2 {
		return []routesDomain.RouteRecommendation{}, nil
	}

	targetDistanceKm := request.DistanceTargetKm
	if targetDistanceKm <= 0 {
		targetDistanceKm = polylineDistanceKmFromCoordinates(rawShape)
	}
	if targetDistanceKm <= 0 {
		targetDistanceKm = 20.0
	}

	projectedShape := projectShapePolylineToStart(rawShape, request.StartPoint, targetDistanceKm)
	profile := adapter.profileForRouteType(request.RouteType)
	if err := adapter.validateShapeWithinOSRMCoverage(profile, request.StartPoint, projectedShape); err != nil {
		return []routesDomain.RouteRecommendation{}, err
	}
	routingVariants := buildShapeRoutingVariants(projectedShape, request.StartPoint)
	if len(routingVariants) == 0 {
		return []routesDomain.RouteRecommendation{}, nil
	}
	shapePreview := coordinatesToLatLngPoints(projectedShape)
	if len(shapePreview) < 2 {
		shapePreview = coordinatesToLatLngPoints(routingVariants[0].shape)
	}
	if len(shapePreview) < 2 {
		return []routesDomain.RouteRecommendation{}, nil
	}

	rejectCounts := make(map[string]int)
	candidates := make([]osrmRouteCandidate, 0, request.Limit*12)
	seenSignatures := make(map[string]struct{}, request.Limit*10)
	fetchedRouteCount := 0
	usedStrategies := 0

	appendShapeCandidate := func(
		shapeRequest application.RoutingEngineRequest,
		variant shapeRoutingVariant,
		strategy shapeRoutingStrategy,
		osrmRoute osrmRoute,
		routeIndex int,
	) {
		var candidate osrmRouteCandidate
		var ok bool
		if strategy.bestEffort {
			candidate, ok = adapter.toRouteCandidateBestEffort(shapeRequest, osrmRoute, routeIndex, rejectCounts)
		} else {
			candidate, ok = adapter.toRouteCandidate(shapeRequest, osrmRoute, routeIndex, rejectCounts)
		}
		if !ok {
			return
		}
		signature := routeGeometrySignature(candidate.recommendation.PreviewLatLng)
		if signature == "" {
			incrementRejectCount(rejectCounts, "EMPTY_GEOMETRY_SIGNATURE")
			return
		}
		if _, exists := seenSignatures[signature]; exists {
			incrementRejectCount(rejectCounts, "DUPLICATE_GEOMETRY")
			return
		}

		shapeScore := shapeSimilarityScore(candidate.recommendation.PreviewLatLng, shapePreview)
		shapeName := "CUSTOM_SHAPE"
		recommendation := candidate.recommendation
		recommendation.VariantType = routesDomain.RouteVariantShape
		recommendation.Shape = &shapeName
		recommendation.ShapeScore = &shapeScore
		matchScore, shapeDriftPenalty := shapeModeMatchScore(
			recommendation.MatchScore,
			shapeScore,
			candidate.backtrackingRatio,
			candidate.corridorOverlap,
			candidate.edgeReuseRatio,
			candidate.maxAxisReuseRatio,
			strategy.code,
		)
		recommendation.MatchScore = matchScore
		recommendation.Reasons = append(
			recommendation.Reasons,
			fmt.Sprintf("Shape similarity: %.0f%%", shapeScore*100.0),
			fmt.Sprintf("Shape mode: %s", strategy.label),
		)
		if variant.label != "" {
			recommendation.Reasons = append(recommendation.Reasons, fmt.Sprintf("Shape transform: %s", variant.label))
		}
		if strategy.code == shapeModeStrategyRoadSnap {
			recommendation.Reasons = append(
				recommendation.Reasons,
				"Shape trace snap: nearest routable anchors routed segment-by-segment",
			)
		}
		if strategy.code == shapeModeStrategyMapMatch {
			recommendation.Reasons = append(
				recommendation.Reasons,
				"Shape trace match: OSRM map-matched the drawn trace before routing",
			)
		}
		idealShapeScore := minShapeModeSimilarity(strategy.code)
		if shapeScore < idealShapeScore {
			recommendation.Reasons = append(
				recommendation.Reasons,
				fmt.Sprintf("Shape similarity below ideal: %.0f%% (target %.0f%%)", shapeScore*100.0, idealShapeScore*100.0),
			)
		}
		if strategy.bestEffort {
			recommendation.Reasons = append(
				recommendation.Reasons,
				"Shape best effort: returned despite weak matching to avoid blocking generation",
			)
		}
		if shapeDriftPenalty > 0.05 {
			recommendation.Reasons = append(
				recommendation.Reasons,
				fmt.Sprintf("Shape drift penalty: -%.1f", shapeDriftPenalty),
			)
		}

		candidate.recommendation = recommendation
		candidate.effectiveMatchScore = clampOSMScore(
			recommendation.MatchScore -
				candidate.backtrackingRatio*95.0 -
				candidate.corridorOverlap*125.0 -
				candidate.edgeReuseRatio*140.0 -
				candidate.maxAxisReuseRatio*170.0,
		)
		candidates = append(candidates, candidate)
		seenSignatures[signature] = struct{}{}
	}

	for variantIndex, variant := range routingVariants {
		if len(variant.shape) < 2 {
			continue
		}
		routeAnchor := variant.shape[0]
		shapeRequest := request
		shapeRequest.StartPoint = routeAnchor
		shapeRequest.DistanceTargetKm = targetDistanceKm
		shapeRequest.StartDirection = ""
		shapeRequest.DirectionStrict = false
		routeIndexOffset := variantIndex * 100
		strategies := []shapeRoutingStrategy{
			{
				code:      shapeModeStrategyShapeFirst,
				label:     "dense sketch anchors",
				waypoints: buildShapeDenseWaypoints(routeAnchor, variant.shape),
			},
			{
				code:      shapeModeStrategyShapeFirst,
				label:     "map sketch waypoints",
				waypoints: buildShapeLoopWaypoints(routeAnchor, variant.shape),
			},
			{
				code:      shapeModeStrategySimplified,
				label:     "simplified sketch anchors",
				waypoints: buildShapeSimplifiedWaypoints(routeAnchor, variant.shape),
			},
			{
				code:      shapeModeStrategyRoadFirst,
				label:     "road-first anchors",
				waypoints: buildShapeRoadFirstWaypoints(routeAnchor, variant.shape),
			},
		}

		if variantIndex == 0 || (variantIndex < maxShapeTraceVariants && len(candidates) < request.Limit) {
			if matchedRoutes, err := adapter.fetchOSRMShapeMapMatchedRoutes(profile, variant.shape); err == nil && len(matchedRoutes) > 0 {
				strategy := shapeRoutingStrategy{
					code:      shapeModeStrategyMapMatch,
					label:     "map-matched trace",
					waypoints: variant.shape,
				}
				usedStrategies++
				fetchedRouteCount += len(matchedRoutes)
				for routeIndex, osrmRoute := range matchedRoutes {
					appendShapeCandidate(shapeRequest, variant, strategy, osrmRoute, 4000+routeIndexOffset+routeIndex)
				}
			} else {
				incrementRejectCount(rejectCounts, "OSRM_TRACE_MATCH_FAILED")
				if adapter.debug && err != nil {
					log.Printf(
						"OSRM shape map-match generation call failed: profile=%s points=%d err=%v",
						profile,
						len(variant.shape),
						err,
					)
				}
			}

			if snappedRoute, ok := adapter.fetchOSRMNearestRoadTraceRoute(profile, variant.shape); ok {
				strategy := shapeRoutingStrategy{
					code:      shapeModeStrategyRoadSnap,
					label:     "nearest-road trace",
					waypoints: variant.shape,
				}
				usedStrategies++
				fetchedRouteCount++
				appendShapeCandidate(shapeRequest, variant, strategy, snappedRoute, 3000+routeIndexOffset)
			} else {
				incrementRejectCount(rejectCounts, "OSRM_TRACE_SNAP_FAILED")
			}

			if fidelityWaypoints := buildShapeFidelityStitchedWaypoints(routeAnchor, variant.shape); variantIndex == 0 && len(fidelityWaypoints) >= 3 {
				routes, err := adapter.fetchOSRMShapeSegmentStitchedRoutes(profile, fidelityWaypoints)
				if err != nil {
					incrementRejectCount(rejectCounts, "OSRM_FIDELITY_STITCHED_CALL_FAILED")
					if adapter.debug {
						log.Printf(
							"OSRM shape high-fidelity stitched generation call failed: profile=%s waypoints=%d err=%v",
							profile,
							len(fidelityWaypoints),
							err,
						)
					}
				} else {
					usedStrategies++
					fetchedRouteCount += len(routes)
					strategy := shapeRoutingStrategy{
						code:      shapeModeStrategyStitched,
						label:     "high-fidelity stitched trace",
						waypoints: fidelityWaypoints,
					}
					for routeIndex, osrmRoute := range routes {
						appendShapeCandidate(shapeRequest, variant, strategy, osrmRoute, 2500+routeIndexOffset+routeIndex)
					}
				}
			}

			if stitchedWaypoints := buildShapeStitchedWaypoints(routeAnchor, variant.shape); len(stitchedWaypoints) >= 3 {
				routes, err := adapter.fetchOSRMShapeSegmentStitchedRoutes(profile, stitchedWaypoints)
				if err != nil {
					incrementRejectCount(rejectCounts, "OSRM_STITCHED_CALL_FAILED")
					if adapter.debug {
						log.Printf(
							"OSRM shape stitched generation call failed: profile=%s waypoints=%d err=%v",
							profile,
							len(stitchedWaypoints),
							err,
						)
					}
				} else {
					usedStrategies++
					fetchedRouteCount += len(routes)
					strategy := shapeRoutingStrategy{
						code:      shapeModeStrategyStitched,
						label:     "segment stitched alternatives",
						waypoints: stitchedWaypoints,
					}
					for routeIndex, osrmRoute := range routes {
						appendShapeCandidate(shapeRequest, variant, strategy, osrmRoute, 2000+routeIndexOffset+routeIndex)
					}
				}
			}
		}

		for _, strategy := range strategies {
			if len(strategy.waypoints) < 3 {
				incrementRejectCount(rejectCounts, "SHAPE_WAYPOINTS_TOO_FEW")
				continue
			}
			routes, err := adapter.fetchOSRMRoutesForShape(profile, strategy.waypoints)
			if err != nil {
				incrementRejectCount(rejectCounts, "OSRM_CALL_FAILED")
				if adapter.debug {
					log.Printf(
						"OSRM shape generation call failed: mode=%s profile=%s waypoints=%d err=%v",
						strategy.code,
						profile,
						len(strategy.waypoints),
						err,
					)
				}
				continue
			}
			usedStrategies++
			fetchedRouteCount += len(routes)

			for routeIndex, osrmRoute := range routes {
				appendShapeCandidate(shapeRequest, variant, strategy, osrmRoute, routeIndexOffset+routeIndex)
			}
		}
	}

	if len(candidates) == 0 {
		maxFallbackVariants := int(math.Min(float64(maxShapeBestEffortVariants), float64(len(routingVariants))))
		for variantIndex := 0; variantIndex < maxFallbackVariants; variantIndex++ {
			variant := routingVariants[variantIndex]
			if len(variant.shape) < 2 {
				continue
			}
			routeAnchor := variant.shape[0]
			shapeRequest := request
			shapeRequest.StartPoint = routeAnchor
			shapeRequest.DistanceTargetKm = targetDistanceKm
			shapeRequest.StartDirection = ""
			shapeRequest.DirectionStrict = false
			routeIndexOffset := variantIndex * 100
			for strategyIndex, strategy := range buildShapeBestEffortRoutingStrategies(routeAnchor, variant.shape) {
				if len(strategy.waypoints) < 3 {
					incrementRejectCount(rejectCounts, "SHAPE_BEST_EFFORT_WAYPOINTS_TOO_FEW")
					continue
				}
				routes, err := adapter.fetchOSRMRoutesForShape(profile, strategy.waypoints)
				if err != nil {
					incrementRejectCount(rejectCounts, "OSRM_BEST_EFFORT_CALL_FAILED")
					if adapter.debug {
						log.Printf(
							"OSRM shape best-effort call failed: mode=%s profile=%s waypoints=%d err=%v",
							strategy.code,
							profile,
							len(strategy.waypoints),
							err,
						)
					}
					continue
				}
				usedStrategies++
				fetchedRouteCount += len(routes)
				for routeIndex, osrmRoute := range routes {
					appendShapeCandidate(shapeRequest, variant, strategy, osrmRoute, 1000+routeIndexOffset+(strategyIndex*20)+routeIndex)
				}
			}
		}
	}

	selectionRequest := request
	selectionRequest.ShapePolyline = shapePolyline
	selectionRequest.DistanceTargetKm = targetDistanceKm
	recommendations := selectCandidatesWithRelaxation(selectionRequest, candidates, rejectCounts)
	if len(recommendations) > request.Limit {
		recommendations = recommendations[:request.Limit]
	}

	if adapter.debug || len(recommendations) == 0 {
		log.Printf(
			"OSRM shape generation summary: routeType=%s shapePoints=%d strategies=%d fetched=%d accepted=%d rejects=%s",
			strings.ToUpper(strings.TrimSpace(request.RouteType)),
			len(rawShape),
			usedStrategies,
			fetchedRouteCount,
			len(recommendations),
			formatRejectCounts(rejectCounts),
		)
	}
	return recommendations, nil
}

func (adapter *OSMRoutingAdapter) generateCustomWaypointLoops(
	request application.RoutingEngineRequest,
	profile string,
) []routesDomain.RouteRecommendation {
	rejectCounts := make(map[string]int)
	waypoints := buildCustomLoopWaypoints(request.StartPoint, request.Waypoints)
	if len(waypoints) < 3 {
		incrementRejectCount(rejectCounts, "CUSTOM_WAYPOINTS_TOO_FEW")
		return []routesDomain.RouteRecommendation{}
	}

	routes, err := adapter.fetchOSRMRoutes(profile, waypoints)
	if err != nil {
		incrementRejectCount(rejectCounts, "OSRM_CALL_FAILED")
		if adapter.debug {
			log.Printf(
				"OSRM custom target generation call failed: profile=%s waypoints=%d err=%v",
				profile, len(waypoints), err,
			)
		}
		return []routesDomain.RouteRecommendation{}
	}

	candidates := make([]osrmRouteCandidate, 0, len(routes))
	seenSignatures := make(map[string]struct{}, len(routes))
	for routeIndex, osrmRoute := range routes {
		candidate, ok := adapter.toRouteCandidate(request, osrmRoute, routeIndex, rejectCounts)
		if !ok {
			continue
		}
		signature := routeGeometrySignature(candidate.recommendation.PreviewLatLng)
		if signature == "" {
			incrementRejectCount(rejectCounts, "EMPTY_GEOMETRY_SIGNATURE")
			continue
		}
		if _, exists := seenSignatures[signature]; exists {
			incrementRejectCount(rejectCounts, "DUPLICATE_GEOMETRY")
			continue
		}
		seenSignatures[signature] = struct{}{}
		candidates = append(candidates, candidate)
	}

	recommendations := selectCandidatesWithRelaxation(request, candidates, rejectCounts)
	if len(recommendations) > request.Limit {
		recommendations = recommendations[:request.Limit]
	}
	for index := range recommendations {
		recommendations[index].Reasons = append(recommendations[index].Reasons, "Target mode: custom waypoints")
	}
	if adapter.debug || len(recommendations) == 0 {
		targetElevation := "n/a"
		if request.ElevationTargetM != nil {
			targetElevation = fmt.Sprintf("%.0fm", *request.ElevationTargetM)
		}
		log.Printf(
			"OSRM custom target generation summary: routeType=%s target=%.1fkm/%s customWaypoints=%d fetched=%d accepted=%d rejects=%s",
			strings.ToUpper(strings.TrimSpace(request.RouteType)),
			request.DistanceTargetKm,
			targetElevation,
			len(request.Waypoints),
			len(routes),
			len(recommendations),
			formatRejectCounts(rejectCounts),
		)
	}
	return recommendations
}

func (adapter *OSMRoutingAdapter) generateTargetLoopsDisjoint(
	request application.RoutingEngineRequest,
	profile string,
) ([]routesDomain.RouteRecommendation, bool) {
	anchors := adapter.sampleTargetAnchors(request)
	if len(anchors) == 0 {
		return []routesDomain.RouteRecommendation{}, false
	}
	historyBias := buildRoutingHistoryBiasContext(request)
	if historyBias.enabled {
		anchors = sortAnchorsByHistoryReuse(anchors, request.StartPoint, historyBias)
	}
	hardAxisReuseCap := disjointHardAxisReuseCap(request)

	rejectCounts := make(map[string]int)
	candidates := make([]osrmRouteCandidate, 0, request.Limit*6)
	seenSignatures := make(map[string]struct{}, request.Limit*8)
	maxCandidates := int(math.Max(24.0, float64(request.Limit*12)))
	candidateIndex := 0
	fetchedRouteCount := 0
	fetchErrors := 0

outerAnchors:
	for anchorIndex, anchor := range anchors {
		outboundRoutes, err := adapter.fetchOSRMRoutes(profile, []routesDomain.Coordinates{request.StartPoint, anchor})
		if err != nil {
			fetchErrors++
			incrementRejectCount(rejectCounts, "OSRM_CALL_FAILED")
			continue
		}
		fetchedRouteCount += len(outboundRoutes)
		if len(outboundRoutes) == 0 {
			incrementRejectCount(rejectCounts, "NO_OUTBOUND_ROUTE")
			continue
		}

		maxOutbound := int(math.Min(3.0, float64(len(outboundRoutes))))
		for outboundIndex := 0; outboundIndex < maxOutbound; outboundIndex++ {
			outboundRoute := outboundRoutes[outboundIndex]
			outboundPreview, ok := osrmRouteToPreviewPoints(outboundRoute)
			if !ok {
				incrementRejectCount(rejectCounts, "INVALID_OUTBOUND_GEOMETRY")
				continue
			}

			returnVariants := adapter.buildReturnWaypointVariants(
				anchor,
				request.StartPoint,
				request.StartDirection,
				request.RouteType,
				anchorIndex+outboundIndex,
			)
			maxVariants := int(math.Min(4.0, float64(len(returnVariants))))
			for variantIndex := 0; variantIndex < maxVariants; variantIndex++ {
				inboundRoutes, err := adapter.fetchOSRMRoutes(profile, returnVariants[variantIndex])
				if err != nil {
					fetchErrors++
					incrementRejectCount(rejectCounts, "OSRM_CALL_FAILED")
					continue
				}
				fetchedRouteCount += len(inboundRoutes)
				if len(inboundRoutes) == 0 {
					incrementRejectCount(rejectCounts, "NO_INBOUND_ROUTE")
					continue
				}

				maxInbound := int(math.Min(2.0, float64(len(inboundRoutes))))
				for inboundIndex := 0; inboundIndex < maxInbound; inboundIndex++ {
					inboundRoute := inboundRoutes[inboundIndex]
					inboundPreview, ok := osrmRouteToPreviewPoints(inboundRoute)
					if !ok {
						incrementRejectCount(rejectCounts, "INVALID_INBOUND_GEOMETRY")
						continue
					}
					combinedPreview := mergeRoutePreviews(outboundPreview, inboundPreview)
					if len(combinedPreview) < 2 {
						incrementRejectCount(rejectCounts, "INVALID_COMBINED_GEOMETRY")
						continue
					}

					axisStats := evaluateAxisUsage(combinedPreview)
					minOppositeReuseMetersForRequest := minimumOppositeReuseMetersForRequest(
						request.RouteType,
						request.StrictBacktracking,
						request.DistanceTargetKm,
					)
					hasOppositeOutsideStart, maxAxisReuseOutsideStart, oppositeOutsideStartRatio := evaluateAxisReuseOutsideStartZone(
						combinedPreview,
						request.StartPoint,
						backtrackingStartZoneM,
						minOppositeReuseMetersForRequest,
					)
					maxAxisReuseOutsideStartLimit := outsideStartAxisReuseLimit(
						request.RouteType,
						request.StrictBacktracking,
					)
					oppositeOutsideStartLimit := allowedOppositeOutsideStartRatio(
						request.RouteType,
						request.StrictBacktracking,
					)
					// Construction-phase hard rules for v3:
					// 1) never accept opposite traversal on same axis outside start/finish zone
					// 2) cap repeated traversal of a single axis outside start/finish zone
					if request.StrictBacktracking && hasOppositeOutsideStart {
						incrementRejectCount(rejectCounts, "NO_DISJOINT_LOOP")
						continue
					}
					if !request.StrictBacktracking && oppositeOutsideStartRatio > oppositeOutsideStartLimit {
						incrementRejectCount(rejectCounts, "NO_DISJOINT_LOOP")
						continue
					}
					if maxAxisReuseOutsideStart > maxAxisReuseOutsideStartLimit {
						incrementRejectCount(rejectCounts, "AXIS_REUSE_OUTSIDE_START")
						continue
					}
					if axisStats.maxAxisReuseCount > hardAxisReuseCap {
						incrementRejectCount(rejectCounts, "AXIS_REUSE_HARD_REJECT")
						continue
					}

					totalDistanceKm := (outboundRoute.Distance + inboundRoute.Distance) / 1000.0
					totalDurationSec := int(math.Round(outboundRoute.Duration + inboundRoute.Duration))
					combinedSurface := mergeSurfaceBreakdowns(
						computeSurfaceBreakdown(outboundRoute),
						computeSurfaceBreakdown(inboundRoute),
					)
					candidate, ok := adapter.toRouteCandidateFromPreview(
						request,
						combinedPreview,
						combinedSurface,
						totalDistanceKm,
						totalDurationSec,
						candidateIndex,
						rejectCounts,
					)
					candidateIndex++
					if !ok {
						continue
					}
					if historyBias.enabled {
						candidate = applyHistoryBiasToCandidate(candidate, request.StartPoint, historyBias)
					}
					signature := routeGeometrySignature(candidate.recommendation.PreviewLatLng)
					if signature == "" {
						incrementRejectCount(rejectCounts, "EMPTY_GEOMETRY_SIGNATURE")
						continue
					}
					if _, exists := seenSignatures[signature]; exists {
						incrementRejectCount(rejectCounts, "DUPLICATE_GEOMETRY")
						continue
					}
					seenSignatures[signature] = struct{}{}
					candidate.recommendation.Reasons = append(
						candidate.recommendation.Reasons,
						"Generation engine: disjoint anchors (v3)",
					)
					candidates = append(candidates, candidate)
					if len(candidates) >= maxCandidates {
						break outerAnchors
					}
				}
			}
		}
	}

	recommendations := selectCandidatesWithRelaxation(request, candidates, rejectCounts)
	if len(recommendations) > request.Limit {
		recommendations = recommendations[:request.Limit]
	}

	if adapter.debug || len(recommendations) == 0 {
		targetElevation := "n/a"
		if request.ElevationTargetM != nil {
			targetElevation = fmt.Sprintf("%.0fm", *request.ElevationTargetM)
		}
		log.Printf(
			"OSRM target generation v3 summary: routeType=%s direction=%s target=%.1fkm/%s anchors=%d fetched=%d accepted=%d fetchErrors=%d rejects=%s",
			strings.ToUpper(strings.TrimSpace(request.RouteType)),
			strings.ToUpper(strings.TrimSpace(request.StartDirection)),
			request.DistanceTargetKm,
			targetElevation,
			len(anchors),
			fetchedRouteCount,
			len(recommendations),
			fetchErrors,
			formatRejectCounts(rejectCounts),
		)
	}

	if len(recommendations) == 0 {
		return []routesDomain.RouteRecommendation{}, false
	}
	return recommendations, true
}

func (adapter *OSMRoutingAdapter) sampleTargetAnchors(
	request application.RoutingEngineRequest,
) []routesDomain.Coordinates {
	baseBearing := startDirectionToBearing(request.StartDirection)
	hasDirection := strings.TrimSpace(request.StartDirection) != ""
	directionStrict := hasDirection && request.DirectionStrict
	normalizedRouteType := strings.ToUpper(strings.TrimSpace(request.RouteType))
	radiusBaseKm := math.Max(1.0, request.DistanceTargetKm/(2.0*math.Pi))
	radiusMultipliers := []float64{1.00, 0.92, 1.08, 0.84, 1.16, 1.24, 0.76, 1.32, 0.68, 1.40, 1.48, 0.60}
	rotations := []float64{0, 22, -22, 45, -45, 68, -68, 95, -95, 125, -125, 155, -155}
	switch normalizedRouteType {
	case "GRAVEL":
		radiusMultipliers = []float64{1.00, 0.86, 1.14, 0.74, 1.26, 0.66, 1.34, 1.44, 0.58, 1.52}
		rotations = []float64{0, 30, -30, 62, -62, 95, -95, 128, -128, 158, -158}
	case "MTB", "TRAIL", "HIKE":
		radiusMultipliers = []float64{0.90, 1.00, 0.82, 1.10, 0.72, 1.22, 0.64, 1.32, 1.42}
		rotations = []float64{0, 34, -34, 70, -70, 108, -108, 145, -145}
	}
	if hasDirection {
		rotations = []float64{0, 8, -8, 15, -15, 24, -24, 32, -32}
		if directionStrict {
			rotations = []float64{0, 5, -5, 10, -10, 16, -16}
		}
		switch normalizedRouteType {
		case "GRAVEL":
			rotations = []float64{0, 10, -10, 20, -20, 32, -32, 44, -44}
			if directionStrict {
				rotations = []float64{0, 6, -6, 12, -12, 18, -18, 26, -26}
			}
		case "MTB", "TRAIL", "HIKE":
			rotations = []float64{0, 12, -12, 24, -24, 38, -38, 52, -52}
			if directionStrict {
				rotations = []float64{0, 8, -8, 16, -16, 24, -24, 34, -34}
			}
		}
	}

	anchors := make([]routesDomain.Coordinates, 0, maxOSRMRoutingCalls)
	seen := make(map[string]struct{}, maxOSRMRoutingCalls)
	for callIndex := 0; callIndex < maxOSRMRoutingCalls; callIndex++ {
		radiusKm := radiusBaseKm * radiusMultipliers[callIndex%len(radiusMultipliers)]
		rotation := rotations[callIndex%len(rotations)]
		anchor := destinationFromBearing(
			request.StartPoint,
			radiusKm,
			normalizeBearing(baseBearing+rotation),
		)
		key := quantizedPointKey(anchor.Lat, anchor.Lng)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		anchors = append(anchors, anchor)
	}
	return anchors
}

func (adapter *OSMRoutingAdapter) buildReturnWaypointVariants(
	anchor routesDomain.Coordinates,
	start routesDomain.Coordinates,
	startDirection string,
	routeType string,
	seed int,
) [][]routesDomain.Coordinates {
	distanceKm := math.Max(1.0, haversineDistanceMeters(anchor.Lat, anchor.Lng, start.Lat, start.Lng)/1000.0)
	directBearing := osrmBearingDegrees(anchor.Lat, anchor.Lng, start.Lat, start.Lng)
	offsets := []float64{58, -58, 92, -92, 125, -125, 155, -155}
	scales := []float64{0.48, 0.48, 0.56, 0.56, 0.68, 0.68, 0.80, 0.80}
	directionBlend := 0.28
	switch strings.ToUpper(strings.TrimSpace(routeType)) {
	case "GRAVEL":
		offsets = []float64{72, -72, 108, -108, 140, -140, 168, -168}
		scales = []float64{0.56, 0.56, 0.66, 0.66, 0.78, 0.78, 0.90, 0.90}
		directionBlend = 0.20
	case "MTB", "TRAIL", "HIKE":
		offsets = []float64{78, -78, 116, -116, 148, -148, 174, -174}
		scales = []float64{0.60, 0.60, 0.72, 0.72, 0.84, 0.84, 0.96, 0.96}
		directionBlend = 0.16
	case "RIDE":
		offsets = []float64{52, -52, 84, -84, 118, -118, 150, -150}
		scales = []float64{0.42, 0.42, 0.50, 0.50, 0.62, 0.62, 0.74, 0.74}
		directionBlend = 0.34
	}
	variants := make([][]routesDomain.Coordinates, 0, len(offsets)+1)
	// Keep direct route as first fallback.
	variants = append(variants, []routesDomain.Coordinates{anchor, start})

	shift := 0
	if len(offsets) > 0 {
		shift = seed % len(offsets)
	}
	for i := 0; i < len(offsets); i++ {
		idx := (shift + i) % len(offsets)
		offset := offsets[idx]
		scale := scales[idx]
		pivotBearing := normalizeBearing(directBearing + offset)
		// With global direction set, nudge the pivot so the return remains globally
		// aligned with requested direction while still avoiding the outbound corridor.
		if strings.TrimSpace(startDirection) != "" {
			dirBearing := startDirectionToBearing(startDirection)
			pivotBearing = normalizeBearing(pivotBearing*(1.0-directionBlend) + dirBearing*directionBlend)
		}
		pivot := destinationFromBearing(anchor, distanceKm*scale, pivotBearing)
		variants = append(variants, []routesDomain.Coordinates{anchor, pivot, start})
	}
	return variants
}

func osrmRouteToPreviewPoints(route osrmRoute) ([][]float64, bool) {
	if len(route.Geometry.Coordinates) == 0 {
		return [][]float64{}, false
	}
	points := make([][]float64, 0, len(route.Geometry.Coordinates))
	for _, coordinate := range route.Geometry.Coordinates {
		if len(coordinate) < 2 {
			continue
		}
		lng := coordinate[0]
		lat := coordinate[1]
		if lat < -90 || lat > 90 || lng < -180 || lng > 180 {
			continue
		}
		points = append(points, []float64{lat, lng})
	}
	return points, len(points) >= 2
}

func mergeRoutePreviews(outbound [][]float64, inbound [][]float64) [][]float64 {
	if len(outbound) == 0 {
		return inbound
	}
	if len(inbound) == 0 {
		return outbound
	}
	merged := make([][]float64, 0, len(outbound)+len(inbound))
	merged = append(merged, outbound...)
	inboundStart := inbound[0]
	outboundEnd := outbound[len(outbound)-1]
	startIndex := 0
	if len(inboundStart) >= 2 &&
		len(outboundEnd) >= 2 &&
		haversineDistanceMeters(inboundStart[0], inboundStart[1], outboundEnd[0], outboundEnd[1]) <= 20.0 {
		startIndex = 1
	}
	for i := startIndex; i < len(inbound); i++ {
		merged = append(merged, inbound[i])
	}
	return merged
}

func (adapter *OSMRoutingAdapter) profileForRouteType(routeType string) string {
	override := strings.TrimSpace(strings.ToLower(adapter.profileOverride))
	if override != "" {
		return override
	}

	switch strings.ToUpper(strings.TrimSpace(routeType)) {
	case "RUN", "TRAIL", "HIKE":
		return "walking"
	default:
		return "cycling"
	}
}

func isCustomTargetMode(request application.RoutingEngineRequest) bool {
	if strings.EqualFold(strings.TrimSpace(request.TargetMode), "CUSTOM") {
		return true
	}
	return len(request.Waypoints) > 0
}

func buildCustomLoopWaypoints(
	start routesDomain.Coordinates,
	customWaypoints []routesDomain.Coordinates,
) []routesDomain.Coordinates {
	waypoints := make([]routesDomain.Coordinates, 0, len(customWaypoints)+2)
	waypoints = append(waypoints, start)
	for _, point := range customWaypoints {
		if point.Lat < -90 || point.Lat > 90 || point.Lng < -180 || point.Lng > 180 {
			continue
		}
		waypoints = append(waypoints, point)
	}
	waypoints = append(waypoints, start)
	return waypoints
}

func (adapter *OSMRoutingAdapter) syntheticLoopWaypoints(
	start routesDomain.Coordinates,
	radiusKm float64,
	initialBearing float64,
	startDirection string,
	routeType string,
	callIndex int,
) []routesDomain.Coordinates {
	// We rotate through multiple waypoint "shapes" so OSRM gets distinct
	// loop intents and does not keep returning the same corridor.
	circularPatterns := []struct {
		bearingOffsets []float64
		radiusScales   []float64
	}{
		{
			bearingOffsets: []float64{0, 120, 240},
			radiusScales:   []float64{1.00, 1.05, 0.95},
		},
		{
			bearingOffsets: []float64{0, 85, 170, 255},
			radiusScales:   []float64{1.10, 0.92, 1.08, 0.88},
		},
		{
			bearingOffsets: []float64{0, 70, 155, 230, 300},
			radiusScales:   []float64{1.00, 1.20, 0.85, 1.10, 0.90},
		},
		{
			bearingOffsets: []float64{0, 60, 135, 210, 285},
			radiusScales:   []float64{1.15, 0.90, 1.18, 0.86, 1.00},
		},
	}
	// Directional patterns keep waypoints in the forward half of the compass
	// (relative to requested direction). This guides the loop's global heading.
	directionalPatterns := []struct {
		bearingOffsets []float64
		radiusScales   []float64
	}{
		{
			bearingOffsets: []float64{0, 28, 56, -28, -56},
			radiusScales:   []float64{1.18, 1.06, 1.06, 0.90, 0.90},
		},
		{
			bearingOffsets: []float64{12, 40, 70, -12, -40, -70},
			radiusScales:   []float64{1.20, 1.20, 1.00, 1.00, 0.82, 0.82},
		},
		{
			bearingOffsets: []float64{0, 22, 48, 78, -22, -48, -78},
			radiusScales:   []float64{1.14, 1.12, 1.12, 0.98, 0.98, 0.78, 0.78},
		},
		{
			bearingOffsets: []float64{6, 34, 62, -6, -34, -62},
			radiusScales:   []float64{1.24, 1.24, 1.05, 1.05, 0.86, 0.86},
		},
	}
	hasDirection := strings.TrimSpace(startDirection) != ""
	switch strings.ToUpper(strings.TrimSpace(routeType)) {
	case "GRAVEL":
		circularPatterns = []struct {
			bearingOffsets []float64
			radiusScales   []float64
		}{
			{
				bearingOffsets: []float64{0, 78, 146, 214, 292},
				radiusScales:   []float64{1.00, 1.18, 0.88, 1.14, 0.82},
			},
			{
				bearingOffsets: []float64{0, 62, 124, 186, 248, 310},
				radiusScales:   []float64{1.06, 0.94, 1.22, 0.86, 1.14, 0.80},
			},
		}
		directionalPatterns = []struct {
			bearingOffsets []float64
			radiusScales   []float64
		}{
			{
				bearingOffsets: []float64{0, 24, 46, 68, 92, -22, -44, -66},
				radiusScales:   []float64{1.20, 1.12, 1.00, 0.92, 0.84, 1.04, 0.92, 0.80},
			},
			{
				bearingOffsets: []float64{8, 30, 52, 76, 98, -18, -40, -62, -84},
				radiusScales:   []float64{1.24, 1.16, 1.04, 0.94, 0.86, 1.08, 0.96, 0.86, 0.78},
			},
		}
	case "MTB", "TRAIL", "HIKE":
		circularPatterns = []struct {
			bearingOffsets []float64
			radiusScales   []float64
		}{
			{
				bearingOffsets: []float64{0, 66, 132, 198, 264, 330},
				radiusScales:   []float64{1.00, 1.20, 0.90, 1.16, 0.84, 1.08},
			},
		}
		directionalPatterns = []struct {
			bearingOffsets []float64
			radiusScales   []float64
		}{
			{
				bearingOffsets: []float64{0, 26, 50, 74, 98, -24, -48, -72},
				radiusScales:   []float64{1.22, 1.14, 1.02, 0.92, 0.84, 1.06, 0.94, 0.82},
			},
		}
	case "RIDE":
		circularPatterns = []struct {
			bearingOffsets []float64
			radiusScales   []float64
		}{
			{
				bearingOffsets: []float64{0, 110, 220, 300},
				radiusScales:   []float64{1.00, 1.04, 0.96, 1.00},
			},
			{
				bearingOffsets: []float64{0, 95, 190, 285},
				radiusScales:   []float64{1.08, 0.98, 1.02, 0.92},
			},
		}
		directionalPatterns = []struct {
			bearingOffsets []float64
			radiusScales   []float64
		}{
			{
				bearingOffsets: []float64{0, 20, 40, -20, -40},
				radiusScales:   []float64{1.14, 1.04, 0.94, 1.00, 0.88},
			},
			{
				bearingOffsets: []float64{6, 26, 46, -14, -34, -54},
				radiusScales:   []float64{1.18, 1.08, 0.96, 1.02, 0.90, 0.82},
			},
		}
	}
	pattern := circularPatterns[callIndex%len(circularPatterns)]
	if hasDirection {
		pattern = directionalPatterns[callIndex%len(directionalPatterns)]
	}
	waypoints := make([]routesDomain.Coordinates, 0, len(pattern.bearingOffsets)+2)
	waypoints = append(waypoints, start)
	for idx, bearingOffset := range pattern.bearingOffsets {
		scale := 1.0
		if idx < len(pattern.radiusScales) && pattern.radiusScales[idx] > 0 {
			scale = pattern.radiusScales[idx]
		}
		waypoints = append(
			waypoints,
			destinationFromBearing(start, radiusKm*scale, normalizeBearing(initialBearing+bearingOffset)),
		)
	}
	waypoints = append(waypoints, start)
	return waypoints
}

func (adapter *OSMRoutingAdapter) fetchOSRMRoutes(
	profile string,
	waypoints []routesDomain.Coordinates,
) ([]osrmRoute, error) {
	return adapter.fetchOSRMRoutesWithContinueStraight(profile, waypoints, true)
}

func (adapter *OSMRoutingAdapter) fetchOSRMRoutesForShape(
	profile string,
	waypoints []routesDomain.Coordinates,
) ([]osrmRoute, error) {
	return adapter.fetchOSRMRoutesWithContinueStraight(profile, waypoints, false)
}

func (adapter *OSMRoutingAdapter) fetchOSRMRoutesWithContinueStraight(
	profile string,
	waypoints []routesDomain.Coordinates,
	continueStraight bool,
) ([]osrmRoute, error) {
	return adapter.transport().routes(profile, waypoints, continueStraight)
}

func (adapter *OSMRoutingAdapter) fetchOSRMShapeMapMatchedRoutes(
	profile string,
	shape []routesDomain.Coordinates,
) ([]osrmRoute, error) {
	return adapter.transport().match(profile, shape)
}

func (adapter *OSMRoutingAdapter) fetchOSRMShapeSegmentStitchedRoutes(
	profile string,
	waypoints []routesDomain.Coordinates,
) ([]osrmRoute, error) {
	if len(waypoints) < 3 {
		return nil, fmt.Errorf("at least 3 waypoints are required for stitched shape routing")
	}

	segments := make([]osrmRoute, 0, len(waypoints)-1)
	for index := 0; index < len(waypoints)-1; index++ {
		segmentWaypoints := []routesDomain.Coordinates{waypoints[index], waypoints[index+1]}
		routes, err := adapter.fetchOSRMRoutesForShape(profile, segmentWaypoints)
		if err != nil {
			return nil, err
		}
		targetSegment := coordinatesToLatLngPoints(segmentWaypoints)
		route, ok := chooseBestShapeSegmentRoute(routes, targetSegment)
		if !ok {
			return nil, fmt.Errorf("no valid OSRM segment route at index %d", index)
		}
		segments = append(segments, route)
	}

	stitched, ok := stitchOSRMRoutes(segments)
	if !ok {
		return nil, fmt.Errorf("stitched OSRM route has invalid geometry")
	}
	return []osrmRoute{stitched}, nil
}

func (adapter *OSMRoutingAdapter) fetchOSRMNearestRoadTraceRoute(
	profile string,
	shape []routesDomain.Coordinates,
) (osrmRoute, bool) {
	if len(shape) < 2 {
		return osrmRoute{}, false
	}
	sampled := sampleCoordinates(shape, 20)
	snapped := make([]routesDomain.Coordinates, 0, len(sampled)+1)
	maxSnapDistanceMeters := 0.0
	totalSnapDistanceMeters := 0.0
	for _, point := range sampled {
		snappedPoint, snapDistanceMeters, ok := adapter.snapToNearestRoutablePoint(profile, point)
		if !ok {
			return osrmRoute{}, false
		}
		if snapDistanceMeters > 650.0 {
			return osrmRoute{}, false
		}
		if len(snapped) > 0 {
			previous := snapped[len(snapped)-1]
			if haversineDistanceMeters(previous.Lat, previous.Lng, snappedPoint.Lat, snappedPoint.Lng) < 35.0 {
				continue
			}
		}
		snapped = append(snapped, snappedPoint)
		totalSnapDistanceMeters += snapDistanceMeters
		maxSnapDistanceMeters = math.Max(maxSnapDistanceMeters, snapDistanceMeters)
	}
	if len(snapped) < 3 {
		return osrmRoute{}, false
	}
	averageSnapDistanceMeters := totalSnapDistanceMeters / float64(len(snapped))
	if averageSnapDistanceMeters > 260.0 || maxSnapDistanceMeters > 650.0 {
		return osrmRoute{}, false
	}
	routes, err := adapter.fetchOSRMShapeSegmentStitchedRoutes(profile, snapped)
	if err != nil || len(routes) == 0 {
		return osrmRoute{}, false
	}
	return routes[0], true
}

func (adapter *OSMRoutingAdapter) EditRoute(
	request application.RoutingEngineEditRequest,
) (application.RoutingEngineEditResult, error) {
	result := application.RoutingEngineEditResult{
		Diagnostics: []routesDomain.RouteGenerationDiagnostic{},
	}
	if !adapter.enabled || adapter.baseURL == "" {
		result.Diagnostics = append(result.Diagnostics, routesDomain.RouteGenerationDiagnostic{
			Code:    "EDIT_ENGINE_UNAVAILABLE",
			Message: "OSRM route editing is disabled or misconfigured.",
		})
		return result, nil
	}
	if len(request.ControlPoints) < 2 {
		result.Diagnostics = append(result.Diagnostics, routesDomain.RouteGenerationDiagnostic{
			Code:    "EDIT_CONTROL_POINTS_TOO_FEW",
			Message: "At least two control points are required to edit a Strava Art route.",
		})
		return result, nil
	}

	profile := adapter.profileForRouteType(request.RouteType)
	snappedControlPoints := make([]routesDomain.Coordinates, 0, len(request.ControlPoints))
	maxSnapDistanceMeters := 0.0
	for index, point := range request.ControlPoints {
		snappedPoint, snapDistanceMeters, ok := adapter.snapToNearestRoutablePoint(profile, point)
		if !ok {
			result.Diagnostics = append(result.Diagnostics, routesDomain.RouteGenerationDiagnostic{
				Code:    "EDIT_POINT_NOT_ROUTABLE",
				Message: fmt.Sprintf("Control point %d could not be matched to a routable OSRM road.", index+1),
			})
			return result, nil
		}
		if snapDistanceMeters > editControlSnapMaxMeters {
			result.Diagnostics = append(result.Diagnostics, routesDomain.RouteGenerationDiagnostic{
				Code: "EDIT_POINT_OUT_OF_COVERAGE",
				Message: fmt.Sprintf(
					"Control point %d is %.0fm from the nearest routable OSRM road, above the %.0fm edit limit.",
					index+1,
					snapDistanceMeters,
					editControlSnapMaxMeters,
				),
			})
			return result, nil
		}
		if len(snappedControlPoints) > 0 {
			previous := snappedControlPoints[len(snappedControlPoints)-1]
			if haversineDistanceMeters(previous.Lat, previous.Lng, snappedPoint.Lat, snappedPoint.Lng) < editMinControlSpacingMeters {
				continue
			}
		}
		snappedControlPoints = append(snappedControlPoints, snappedPoint)
		maxSnapDistanceMeters = math.Max(maxSnapDistanceMeters, snapDistanceMeters)
	}
	result.ControlPoints = snappedControlPoints
	if len(snappedControlPoints) < 2 {
		result.Diagnostics = append(result.Diagnostics, routesDomain.RouteGenerationDiagnostic{
			Code:    "EDIT_CONTROL_POINTS_TOO_FEW",
			Message: "Control points collapsed to fewer than two routable OSRM points after snapping.",
		})
		return result, nil
	}

	segments := make([]osrmRoute, 0, len(snappedControlPoints)-1)
	for index := 0; index < len(snappedControlPoints)-1; index++ {
		segmentWaypoints := []routesDomain.Coordinates{snappedControlPoints[index], snappedControlPoints[index+1]}
		routes, err := adapter.fetchOSRMRoutesForShape(profile, segmentWaypoints)
		if err != nil || len(routes) == 0 {
			message := fmt.Sprintf("Edited segment %d could not be routed by OSRM.", index+1)
			if err != nil {
				message = fmt.Sprintf("%s %s", message, err.Error())
			}
			result.Diagnostics = append(result.Diagnostics, routesDomain.RouteGenerationDiagnostic{
				Code:    "EDIT_SEGMENT_NO_ROUTE",
				Message: message,
			})
			return result, nil
		}
		segment, ok := chooseBestShapeSegmentRoute(routes, coordinatesToLatLngPoints(segmentWaypoints))
		if !ok {
			result.Diagnostics = append(result.Diagnostics, routesDomain.RouteGenerationDiagnostic{
				Code:    "EDIT_SEGMENT_NO_ROUTE",
				Message: fmt.Sprintf("Edited segment %d returned no valid OSRM geometry.", index+1),
			})
			return result, nil
		}
		segments = append(segments, segment)
	}

	stitched, ok := stitchOSRMRoutes(segments)
	if !ok {
		result.Diagnostics = append(result.Diagnostics, routesDomain.RouteGenerationDiagnostic{
			Code:    "EDIT_SEGMENT_NO_ROUTE",
			Message: "Edited OSRM segments could not be stitched into a valid point-to-point route.",
		})
		return result, nil
	}
	points, ok := osrmRouteToPreviewPoints(stitched)
	if !ok {
		result.Diagnostics = append(result.Diagnostics, routesDomain.RouteGenerationDiagnostic{
			Code:    "EDIT_SEGMENT_NO_ROUTE",
			Message: "Edited OSRM route returned invalid geometry.",
		})
		return result, nil
	}

	distanceKm := stitched.Distance / 1000.0
	durationSec := int(math.Round(stitched.Duration))
	if durationSec <= 0 {
		durationSec = int(math.Round(distanceKm * 180.0))
	}
	start := &routesDomain.Coordinates{Lat: points[0][0], Lng: points[0][1]}
	end := &routesDomain.Coordinates{Lat: points[len(points)-1][0], Lng: points[len(points)-1][1]}
	surfaceBreakdown := computeSurfaceBreakdown(stitched)
	surfaceScore := surfaceMatchScore(request.RouteType, surfaceBreakdown)
	shapeScore := 1.0
	matchScore := clampOSMScore(92.0 - math.Min(maxSnapDistanceMeters/30.0, 18.0) + (surfaceScore-70.0)*0.08)
	routeID := generatedEditedOSMRouteID(points, request.RouteID)
	reasons := []string{
		"Generated with OSM road graph (OSRM)",
		"Shape mode: edited OSRM control route",
		"Edit mode: magnetized control points",
		fmt.Sprintf("Control points: %d", len(snappedControlPoints)),
		fmt.Sprintf("Max control snap: %.0fm", maxSnapDistanceMeters),
		fmt.Sprintf("Surface mix: %s", formatSurfaceBreakdown(surfaceBreakdown)),
		fmt.Sprintf("Surface fitness: %.0f%%", surfaceScore),
		"Retrace policy: art-fit first (diagnostic only)",
	}
	if strings.TrimSpace(request.RouteID) != "" {
		reasons = append(reasons, fmt.Sprintf("Edited from route: %s", strings.TrimSpace(request.RouteID)))
	}

	result.Recommendation = routesDomain.RouteRecommendation{
		RouteID: routeID,
		Activity: business.ActivityShort{
			Id:   0,
			Name: "Edited Strava Art route",
			Type: activityTypeFromRouteType(request.RouteType),
		},
		ActivityDate:   time.Now().UTC().Format(time.RFC3339),
		DistanceKm:     distanceKm,
		ElevationGainM: math.Max(0.0, distanceKm*8.0),
		DurationSec:    durationSec,
		IsLoop:         false,
		Start:          start,
		End:            end,
		StartArea:      formatStartArea(start),
		Season:         seasonFromDate(time.Now().UTC()),
		VariantType:    routesDomain.RouteVariantShape,
		MatchScore:     matchScore,
		Reasons:        reasons,
		PreviewLatLng:  points,
		Shape:          nil,
		ShapeScore:     &shapeScore,
		Experimental:   false,
	}
	result.Diagnostics = append(result.Diagnostics, routesDomain.RouteGenerationDiagnostic{
		Code:    "EDIT_ROUTE_UPDATED",
		Message: "Edited route was snapped and rerouted on the OSRM road graph.",
	})
	return result, nil
}

func (adapter *OSMRoutingAdapter) validateShapeWithinOSRMCoverage(
	profile string,
	start routesDomain.Coordinates,
	shape []routesDomain.Coordinates,
) error {
	if len(shape) < 2 {
		return nil
	}
	points := make([]routesDomain.Coordinates, 0, shapeCoverageSamplePoints+1)
	points = append(points, start)
	points = append(points, sampleCoordinates(shape, shapeCoverageSamplePoints)...)

	var worstPoint routesDomain.Coordinates
	var worstSnapped routesDomain.Coordinates
	worstDistanceMeters := -1.0
	checked := 0
	seen := map[string]struct{}{}
	for _, point := range points {
		key := fmt.Sprintf("%.5f,%.5f", point.Lat, point.Lng)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		snapped, distanceMeters, ok := adapter.snapToNearestRoutablePoint(profile, point)
		if !ok {
			return &routingDiagnosticError{diagnostic: routesDomain.RouteGenerationDiagnostic{
				Code:    "OSRM_COVERAGE_UNAVAILABLE",
				Message: "OSRM nearest-road lookup failed while checking artwork coverage. Check that the local OSRM service is running with the expected profile.",
			}}
		}
		checked++
		if distanceMeters > worstDistanceMeters {
			worstDistanceMeters = distanceMeters
			worstPoint = point
			worstSnapped = snapped
		}
	}
	if checked == 0 || worstDistanceMeters <= shapeCoverageMaxSnapMeters {
		return nil
	}

	return &routingDiagnosticError{diagnostic: routesDomain.RouteGenerationDiagnostic{
		Code: "OSRM_COVERAGE_MISMATCH",
		Message: fmt.Sprintf(
			"The current OSRM extract does not cover this artwork: the nearest routable point is %.1f km from the sketch/start (%.5f, %.5f), snapping to %.5f, %.5f. Use an OSRM extract covering this map area or move the artwork inside the covered region.",
			worstDistanceMeters/1000.0,
			worstPoint.Lat,
			worstPoint.Lng,
			worstSnapped.Lat,
			worstSnapped.Lng,
		),
	}}
}

func chooseBestShapeSegmentRoute(routes []osrmRoute, targetSegment [][]float64) (osrmRoute, bool) {
	var bestRoute osrmRoute
	bestScore := math.Inf(-1)
	bestDistance := math.Inf(1)
	found := false
	for _, route := range routes {
		points, ok := osrmRouteToPreviewPoints(route)
		if !ok || len(points) < 2 || route.Distance <= 0 {
			continue
		}
		score := shapeSimilarityScore(points, targetSegment)
		if !found || score > bestScore+0.0001 || (math.Abs(score-bestScore) <= 0.0001 && route.Distance < bestDistance) {
			bestRoute = route
			bestScore = score
			bestDistance = route.Distance
			found = true
		}
	}
	return bestRoute, found
}

func stitchOSRMRoutes(routes []osrmRoute) (osrmRoute, bool) {
	if len(routes) == 0 {
		return osrmRoute{}, false
	}
	stitched := osrmRoute{
		Geometry: osrmGeometry{Type: "LineString"},
		Legs:     make([]osrmLeg, 0, len(routes)),
	}
	for routeIndex, route := range routes {
		if route.Distance <= 0 || len(route.Geometry.Coordinates) < 2 {
			return osrmRoute{}, false
		}
		stitched.Distance += route.Distance
		stitched.Duration += route.Duration
		stitched.Legs = append(stitched.Legs, route.Legs...)
		for coordinateIndex, coordinate := range route.Geometry.Coordinates {
			if len(coordinate) < 2 {
				continue
			}
			if routeIndex > 0 && coordinateIndex == 0 && coordinatesEqualOSRM(
				stitched.Geometry.Coordinates[len(stitched.Geometry.Coordinates)-1],
				coordinate,
			) {
				continue
			}
			stitched.Geometry.Coordinates = append(stitched.Geometry.Coordinates, []float64{coordinate[0], coordinate[1]})
		}
	}
	return stitched, len(stitched.Geometry.Coordinates) >= 2
}

func coordinatesEqualOSRM(left []float64, right []float64) bool {
	if len(left) < 2 || len(right) < 2 {
		return false
	}
	return math.Abs(left[0]-right[0]) < 0.000001 && math.Abs(left[1]-right[1]) < 0.000001
}

func (adapter *OSMRoutingAdapter) snapToNearestRoutablePoint(
	profile string,
	point routesDomain.Coordinates,
) (routesDomain.Coordinates, float64, bool) {
	return adapter.transport().nearest(profile, point)
}

func (adapter *OSMRoutingAdapter) toRouteCandidate(
	request application.RoutingEngineRequest,
	route osrmRoute,
	index int,
	rejectCounts map[string]int,
) (osrmRouteCandidate, bool) {
	return adapter.toRouteCandidateWithMode(request, route, index, rejectCounts, false)
}

func (adapter *OSMRoutingAdapter) toRouteCandidateBestEffort(
	request application.RoutingEngineRequest,
	route osrmRoute,
	index int,
	rejectCounts map[string]int,
) (osrmRouteCandidate, bool) {
	return adapter.toRouteCandidateWithMode(request, route, index, rejectCounts, true)
}

func (adapter *OSMRoutingAdapter) toRouteCandidateWithMode(
	request application.RoutingEngineRequest,
	route osrmRoute,
	index int,
	rejectCounts map[string]int,
	bestEffort bool,
) (osrmRouteCandidate, bool) {
	if route.Distance <= 0 || len(route.Geometry.Coordinates) < 2 {
		incrementRejectCount(rejectCounts, "INVALID_ROUTE_GEOMETRY")
		return osrmRouteCandidate{}, false
	}
	points, ok := osrmRouteToPreviewPoints(route)
	if !ok {
		incrementRejectCount(rejectCounts, "INVALID_COORDINATES")
		return osrmRouteCandidate{}, false
	}
	distanceKm := route.Distance / 1000.0
	durationSec := int(math.Round(route.Duration))
	if durationSec <= 0 {
		durationSec = int(math.Round(distanceKm * 180.0))
	}
	return adapter.toRouteCandidateFromPreviewWithMode(
		request,
		points,
		computeSurfaceBreakdown(route),
		distanceKm,
		durationSec,
		index,
		rejectCounts,
		bestEffort,
	)
}

func (adapter *OSMRoutingAdapter) toRouteCandidateFromPreview(
	request application.RoutingEngineRequest,
	points [][]float64,
	surfaceBreakdown routeSurfaceBreakdown,
	distanceKm float64,
	durationSec int,
	index int,
	rejectCounts map[string]int,
) (osrmRouteCandidate, bool) {
	return adapter.toRouteCandidateFromPreviewWithMode(
		request,
		points,
		surfaceBreakdown,
		distanceKm,
		durationSec,
		index,
		rejectCounts,
		false,
	)
}

func (adapter *OSMRoutingAdapter) toRouteCandidateFromPreviewWithMode(
	request application.RoutingEngineRequest,
	points [][]float64,
	surfaceBreakdown routeSurfaceBreakdown,
	distanceKm float64,
	durationSec int,
	index int,
	rejectCounts map[string]int,
	bestEffort bool,
) (osrmRouteCandidate, bool) {
	if len(points) < 2 {
		incrementRejectCount(rejectCounts, "INVALID_COORDINATES")
		return osrmRouteCandidate{}, false
	}
	startOffsetMeters := haversineDistanceMeters(points[0][0], points[0][1], request.StartPoint.Lat, request.StartPoint.Lng)
	if !bestEffort && !startsNearRequestedStart(points, request.StartPoint, startSnapToleranceMeters) {
		// In fallback mode, allow larger snap distance to avoid returning no route.
		if request.StrictBacktracking || !startsNearRequestedStart(points, request.StartPoint, fallbackStartSnapTolerance) {
			incrementRejectCount(rejectCounts, "START_TOO_FAR")
			return osrmRouteCandidate{}, false
		}
	}
	start := &routesDomain.Coordinates{Lat: points[0][0], Lng: points[0][1]}
	end := &routesDomain.Coordinates{Lat: points[len(points)-1][0], Lng: points[len(points)-1][1]}
	if durationSec <= 0 {
		durationSec = int(math.Round(distanceKm * 180.0))
	}
	directionPenalty := combinedDirectionPenalty(points, request.StartPoint, request.StartDirection, directionToleranceMeters)
	axisStats := evaluateAxisUsage(points)
	backtrackingRatio := axisStats.oppositeTraversalRatio()
	corridorOverlap := corridorOverlapRatio(points)
	edgeReuse := axisStats.reuseRatio()
	maxAxisReuseCount := axisStats.maxAxisReuseCount
	maxAxisReuseRatio := axisStats.maxAxisReuseRatio()
	diversityRatio := axisStats.segmentDiversityRatio()
	distanceDeltaRatio := distanceShortfallRatio(distanceKm, request.DistanceTargetKm)
	distanceOvershootRatioValue := distanceOvershootRatio(distanceKm, request.DistanceTargetKm)
	shapeMode := strings.TrimSpace(request.ShapePolyline) != ""
	minOppositeReuseMetersForRequest := minimumOppositeReuseMetersForRequest(
		request.RouteType,
		request.StrictBacktracking,
		request.DistanceTargetKm,
	)
	hasOppositeOutsideStart, maxAxisReuseOutsideStart, oppositeOutsideStartRatio := evaluateAxisReuseOutsideStartZone(
		points,
		request.StartPoint,
		backtrackingStartZoneM,
		minOppositeReuseMetersForRequest,
	)
	maxAxisReuseOutsideStartLimit := outsideStartAxisReuseLimit(
		request.RouteType,
		request.StrictBacktracking,
	)
	if !bestEffort && !shapeMode && hasOppositeOutsideStart {
		if request.StrictBacktracking {
			incrementRejectCount(rejectCounts, "STRICT_BACKTRACKING_OUTSIDE_START")
		} else {
			incrementRejectCount(rejectCounts, "BACKTRACKING_FILTERED")
		}
		return osrmRouteCandidate{}, false
	}
	if !bestEffort && !shapeMode && maxAxisReuseOutsideStart > maxAxisReuseOutsideStartLimit {
		incrementRejectCount(rejectCounts, "AXIS_REUSE_OUTSIDE_START")
		return osrmRouteCandidate{}, false
	}
	if !bestEffort && !shapeMode && !meetsMinimumDistance(distanceKm, request.DistanceTargetKm) {
		incrementRejectCount(rejectCounts, "DISTANCE_BELOW_MINIMUM")
		return osrmRouteCandidate{}, false
	}
	maxBacktrackingReject := 0.32
	maxCorridorReject := 0.30
	maxEdgeReuseReject := 0.28
	maxAxisReuseReject := 8
	if !request.StrictBacktracking {
		// Fallback pass: keep anti-retrace guardrails, but avoid returning 0 route.
		maxBacktrackingReject = 0.60
		maxCorridorReject = 0.55
		maxEdgeReuseReject = 0.55
		maxAxisReuseReject = 14
	}
	if !bestEffort && !shapeMode && (backtrackingRatio > maxBacktrackingReject ||
		corridorOverlap > maxCorridorReject ||
		edgeReuse > maxEdgeReuseReject ||
		maxAxisReuseCount > maxAxisReuseReject) {
		incrementRejectCount(rejectCounts, "EXCESSIVE_RETRACE")
		return osrmRouteCandidate{}, false
	}

	var elevationGainM float64
	if request.ElevationTargetM != nil && *request.ElevationTargetM > 0 {
		deltaRatio := distanceDeltaRatio
		elevationGainM = math.Max(0.0, *request.ElevationTargetM*(1.0-deltaRatio*0.5))
	} else {
		elevationGainM = math.Max(0.0, distanceKm*8.0)
	}

	matchScore := osrmMatchScore(request, distanceKm, elevationGainM, points)
	routeID := generatedOSMRouteID(points, request.StartPoint, index)
	activityType := activityTypeFromRouteType(request.RouteType)
	title := fmt.Sprintf("Generated loop near %.4f, %.4f", request.StartPoint.Lat, request.StartPoint.Lng)
	if index > 0 {
		title = fmt.Sprintf("%s #%d", title, index+1)
	}

	reasons := []string{
		"Generated with OSM road graph (OSRM)",
		fmt.Sprintf("Distance vs minimum target: %s", formatDistanceDelta(distanceKm-request.DistanceTargetKm)),
		fmt.Sprintf("Segment diversity: %.0f%% unique edges", diversityRatio*100.0),
		fmt.Sprintf("Directional alignment: %.0f%%", (1.0-directionPenalty)*100.0),
		fmt.Sprintf("Backtracking: %.0f%%", backtrackingRatio*100.0),
		fmt.Sprintf("Corridor overlap: %.0f%%", corridorOverlap*100.0),
		fmt.Sprintf("Axis retrace: %.0f%%", edgeReuse*100.0),
		fmt.Sprintf("Max axis reuse: %dx", maxAxisReuseCount),
		fmt.Sprintf("Max axis reuse outside start zone: %dx (limit %dx)", maxAxisReuseOutsideStart, maxAxisReuseOutsideStartLimit),
		fmt.Sprintf(
			"Opposite-axis overlap outside start zone: %.0f%% (limit %.0f%%)",
			oppositeOutsideStartRatio*100.0,
			allowedOppositeOutsideStartRatio(request.RouteType, request.StrictBacktracking)*100.0,
		),
	}
	if bestEffort {
		reasons = append(reasons, "Generation engine fallback: shape best effort")
	}
	if request.ElevationTargetM != nil {
		reasons = append(reasons, fmt.Sprintf("Elevation estimate: %s", formatElevationDelta(elevationGainM-*request.ElevationTargetM)))
	}
	if request.StartDirection != "" {
		reasons = append(reasons, fmt.Sprintf("Direction: %s", startDirectionLabel(request.StartDirection)))
	}
	if (bestEffort || !request.StrictBacktracking) && startOffsetMeters > startSnapToleranceMeters {
		reasons = append(
			reasons,
			fmt.Sprintf(
				"Start offset accepted in fallback mode: %.0fm (normal limit %.0fm)",
				startOffsetMeters,
				startSnapToleranceMeters,
			),
		)
	}
	surfaceScore := surfaceMatchScore(request.RouteType, surfaceBreakdown)
	pathRatio := surfaceBreakdown.pathRatio()
	requiredPathRatio := requiredPathRatioForRequest(request.RouteType, request.StrictBacktracking)
	normalizedRouteType := strings.ToUpper(strings.TrimSpace(request.RouteType))
	if !bestEffort && normalizedRouteType == "GRAVEL" && pathRatio < requiredPathRatio {
		incrementRejectCount(rejectCounts, "GRAVEL_MIN_PATH_RATIO")
		return osrmRouteCandidate{}, false
	}
	reasons = append(
		reasons,
		fmt.Sprintf("Surface mix: %s", formatSurfaceBreakdown(surfaceBreakdown)),
		fmt.Sprintf("Path ratio: %.0f%%", pathRatio*100.0),
		fmt.Sprintf("Surface fitness: %.0f%%", surfaceScore),
		"Surface source: OSRM step classes, mode, and surface/tracktype tags when available",
	)
	if shapeMode {
		reasons = append(reasons, "Retrace policy: art-fit first (diagnostic only)")
	} else if bestEffort {
		reasons = append(reasons, "Anti-backtracking: best-effort fallback")
	} else if request.StrictBacktracking {
		reasons = append(reasons, "Anti-backtracking: native ultra")
	} else {
		reasons = append(reasons, "Anti-backtracking: relaxed fallback")
	}

	recommendation := routesDomain.RouteRecommendation{
		RouteID: routeID,
		Activity: business.ActivityShort{
			Id:   0,
			Name: title,
			Type: activityType,
		},
		ActivityDate:   time.Now().UTC().Format(time.RFC3339),
		DistanceKm:     distanceKm,
		ElevationGainM: elevationGainM,
		DurationSec:    durationSec,
		IsLoop:         true,
		Start:          start,
		End:            end,
		StartArea:      formatStartArea(start),
		Season:         seasonFromDate(time.Now().UTC()),
		VariantType:    routesDomain.RouteVariantRoadGraph,
		MatchScore:     matchScore,
		Reasons:        reasons,
		PreviewLatLng:  points,
		Shape:          nil,
		ShapeScore:     nil,
		Experimental:   false,
	}
	effectiveScore := clampOSMScore(matchScore -
		directionPenalty*34.0 -
		backtrackingRatio*90.0 -
		corridorOverlap*170.0 -
		edgeReuse*180.0 -
		maxAxisReuseRatio*180.0 -
		math.Max(0.0, minSegmentDiversityRatio(request.RouteType)-diversityRatio)*35.0 -
		math.Max(0.0, distanceDeltaRatio-0.15)*45.0 +
		// Overshoot is penalized softly: lower impact than shortfall.
		-math.Max(0.0, distanceOvershootRatioValue-0.25)*12.0 +
		(surfaceScore-70.0)*surfaceScoreWeight(request.RouteType) +
		pathPreferenceBonus(request.RouteType, pathRatio))
	if bestEffort {
		effectiveScore = clampOSMScore(effectiveScore - 22.0)
	}
	// effectiveScore is an internal ranking score (not API score). For classic
	// loops it penalizes retrace heavily; Strava Art selection still sorts by
	// shape score first and keeps retrace as a rideability signal.

	return osrmRouteCandidate{
		recommendation:      recommendation,
		directionPenalty:    directionPenalty,
		backtrackingRatio:   backtrackingRatio,
		corridorOverlap:     corridorOverlap,
		edgeReuseRatio:      edgeReuse,
		maxAxisReuseCount:   maxAxisReuseCount,
		maxAxisReuseRatio:   maxAxisReuseRatio,
		segmentDiversity:    diversityRatio,
		distanceDeltaRatio:  distanceDeltaRatio,
		pathRatio:           pathRatio,
		effectiveMatchScore: effectiveScore,
	}, true
}

func selectCandidatesWithRelaxation(
	request application.RoutingEngineRequest,
	candidates []osrmRouteCandidate,
	rejectCounts map[string]int,
) []routesDomain.RouteRecommendation {
	if len(candidates) == 0 {
		return []routesDomain.RouteRecommendation{}
	}
	limit := request.Limit
	if limit <= 0 {
		limit = 1
	}
	hasDirection := strings.TrimSpace(request.StartDirection) != ""
	shapeMode := strings.TrimSpace(request.ShapePolyline) != ""

	sortedCandidates := make([]osrmRouteCandidate, len(candidates))
	copy(sortedCandidates, candidates)
	sort.SliceStable(sortedCandidates, func(i, j int) bool {
		left := sortedCandidates[i]
		right := sortedCandidates[j]
		if shapeMode {
			leftShapeScore := routeShapeScore(left.recommendation)
			rightShapeScore := routeShapeScore(right.recommendation)
			if leftShapeScore != rightShapeScore {
				return leftShapeScore > rightShapeScore
			}
			if left.effectiveMatchScore != right.effectiveMatchScore {
				return left.effectiveMatchScore > right.effectiveMatchScore
			}
			if left.recommendation.MatchScore != right.recommendation.MatchScore {
				return left.recommendation.MatchScore > right.recommendation.MatchScore
			}
			if left.corridorOverlap != right.corridorOverlap {
				return left.corridorOverlap < right.corridorOverlap
			}
			if left.backtrackingRatio != right.backtrackingRatio {
				return left.backtrackingRatio < right.backtrackingRatio
			}
			if left.edgeReuseRatio != right.edgeReuseRatio {
				return left.edgeReuseRatio < right.edgeReuseRatio
			}
			if left.maxAxisReuseCount != right.maxAxisReuseCount {
				return left.maxAxisReuseCount < right.maxAxisReuseCount
			}
			if left.distanceDeltaRatio != right.distanceDeltaRatio {
				return left.distanceDeltaRatio < right.distanceDeltaRatio
			}
			return left.recommendation.RouteID < right.recommendation.RouteID
		}
		if left.corridorOverlap != right.corridorOverlap {
			return left.corridorOverlap < right.corridorOverlap
		}
		if left.backtrackingRatio != right.backtrackingRatio {
			return left.backtrackingRatio < right.backtrackingRatio
		}
		if left.edgeReuseRatio != right.edgeReuseRatio {
			return left.edgeReuseRatio < right.edgeReuseRatio
		}
		if left.maxAxisReuseCount != right.maxAxisReuseCount {
			return left.maxAxisReuseCount < right.maxAxisReuseCount
		}
		if hasDirection && left.directionPenalty != right.directionPenalty {
			return left.directionPenalty < right.directionPenalty
		}
		if left.historyReuseScore != right.historyReuseScore {
			return left.historyReuseScore > right.historyReuseScore
		}
		normalizedRouteType := strings.ToUpper(strings.TrimSpace(request.RouteType))
		if (normalizedRouteType == "MTB" || normalizedRouteType == "GRAVEL") && left.pathRatio != right.pathRatio {
			return left.pathRatio > right.pathRatio
		}
		if left.effectiveMatchScore != right.effectiveMatchScore {
			return left.effectiveMatchScore > right.effectiveMatchScore
		}
		if !hasDirection && left.directionPenalty != right.directionPenalty {
			return left.directionPenalty < right.directionPenalty
		}
		if left.recommendation.MatchScore != right.recommendation.MatchScore {
			return left.recommendation.MatchScore > right.recommendation.MatchScore
		}
		if left.distanceDeltaRatio != right.distanceDeltaRatio {
			return left.distanceDeltaRatio < right.distanceDeltaRatio
		}
		return left.recommendation.RouteID < right.recommendation.RouteID
	})

	levels := buildRouteRelaxationLevels(
		request.RouteType,
		hasDirection,
		request.DirectionStrict,
		request.DistanceTargetKm,
	)
	selected := make([]routesDomain.RouteRecommendation, 0, limit)
	selectedIDs := make(map[string]struct{}, limit)

	if shapeMode {
		// Strava Art is judged first by the drawing. Retrace can be necessary to
		// preserve the model, so route-loop relaxation levels are diagnostics here,
		// not hard selection gates.
		for _, candidate := range sortedCandidates {
			if len(selected) >= limit {
				break
			}
			routeID := candidate.recommendation.RouteID
			if _, exists := selectedIDs[routeID]; exists {
				continue
			}
			recommendation := candidate.recommendation
			recommendation.Reasons = append(
				recommendation.Reasons,
				"Selection priority: art-fit first",
				"Selection profile: art-fit-diagnostic (retrace allowed)",
			)
			selected = append(selected, recommendation)
			selectedIDs[routeID] = struct{}{}
		}
	} else {
		// Levels are evaluated in order: strict -> balanced -> relaxed -> fallback.
		// We fill results incrementally: if strict cannot fill the target limit,
		// next levels progressively loosen constraints while keeping quality.
		for _, level := range levels {
			if len(selected) >= limit {
				break
			}
			for _, candidate := range sortedCandidates {
				if len(selected) >= limit {
					break
				}
				routeID := candidate.recommendation.RouteID
				if _, exists := selectedIDs[routeID]; exists {
					continue
				}
				if !candidatePassesRelaxation(candidate, level, shapeMode, rejectCounts) {
					continue
				}

				recommendation := candidate.recommendation
				recommendation.Reasons = append(recommendation.Reasons, fmt.Sprintf("Selection profile: %s", level.name))
				selected = append(selected, recommendation)
				selectedIDs[routeID] = struct{}{}
			}
		}
	}

	// Safety net: if all configured levels reject candidates, return the best
	// ranked loops with softer anti-overlap limits instead of returning zero.
	softAxisCap, directionalAxisCap := bestEffortAxisReuseCaps(request.DistanceTargetKm, hasDirection, request.DirectionStrict)
	if shapeMode && len(selected) < limit {
		selected = appendBestEffortCandidates(
			sortedCandidates,
			selected,
			selectedIDs,
			limit,
			1.0,
			0.24,
			0.60,
			0.26,
			4,
			0.80,
			"best-effort-soft (art-fit first)",
			"Selection priority: art-fit first",
		)
	}
	if len(selected) < limit {
		softMaxBacktracking := 0.16
		softMaxCorridor := 0.12
		softMaxEdgeReuse := 0.12
		softMaxDirection := 1.0
		// Directional generation naturally creates more corridor pressure.
		// We relax slightly, but stay far from permissive settings.
		if hasDirection {
			softMaxBacktracking = 0.20
			softMaxCorridor = 0.16
			softMaxEdgeReuse = 0.14
			softMaxDirection = 0.40
		}
		selected = appendBestEffortCandidates(
			sortedCandidates,
			selected,
			selectedIDs,
			limit,
			softMaxDirection,
			softMaxBacktracking,
			softMaxCorridor,
			softMaxEdgeReuse,
			softAxisCap,
			0.20,
			"best-effort-soft",
			"",
		)
	}
	if len(selected) < limit && hasDirection {
		// Last safety net in directional mode: keep anti-retrace filters, but relax them
		// just enough to avoid returning zero route too often.
		selected = appendBestEffortCandidates(
			sortedCandidates,
			selected,
			selectedIDs,
			limit,
			0.46,
			0.18,
			0.14,
			0.13,
			directionalAxisCap,
			0.25,
			"directional-best-effort",
			"",
		)
	}
	if len(selected) == 0 {
		// Absolute last resort: return best-ranked generated candidates rather than none.
		// This keeps UX responsive while preserving all generation diagnostics in reasons.
		for _, candidate := range sortedCandidates {
			if len(selected) >= limit {
				break
			}
			recommendation := candidate.recommendation
			recommendation.Reasons = append(
				recommendation.Reasons,
				"Selection profile: emergency-fallback (constraints fully relaxed)",
			)
			selected = append(selected, recommendation)
		}
	}

	return selected
}

func candidatePassesRelaxation(
	candidate osrmRouteCandidate,
	level routeRelaxationLevel,
	shapeMode bool,
	rejectCounts map[string]int,
) bool {
	if candidate.directionPenalty > level.maxDirectionPenalty {
		incrementRejectCount(rejectCounts, "DIRECTION_CONSTRAINT")
		return false
	}
	if candidate.backtrackingRatio > level.maxBacktrackingRatio {
		incrementRejectCount(rejectCounts, "OPPOSITE_EDGE_TRAVERSAL")
		return false
	}
	if candidate.corridorOverlap > level.maxCorridorOverlap {
		incrementRejectCount(rejectCounts, "CORRIDOR_OVERLAP")
		return false
	}
	if candidate.edgeReuseRatio > level.maxEdgeReuseRatio {
		incrementRejectCount(rejectCounts, "EDGE_REUSE")
		return false
	}
	if candidate.maxAxisReuseCount > level.maxAxisReuseCount {
		incrementRejectCount(rejectCounts, "MAX_AXIS_REUSE")
		return false
	}
	if candidate.segmentDiversity < level.minSegmentDiversity {
		incrementRejectCount(rejectCounts, "LOW_SEGMENT_DIVERSITY")
		return false
	}
	if !shapeMode && candidate.distanceDeltaRatio > level.maxDistanceDeltaRatio {
		incrementRejectCount(rejectCounts, "DISTANCE_CONSTRAINT")
		return false
	}
	return true
}

func appendBestEffortCandidates(
	sortedCandidates []osrmRouteCandidate,
	selected []routesDomain.RouteRecommendation,
	selectedIDs map[string]struct{},
	limit int,
	maxDirectionPenalty float64,
	maxBacktrackingRatio float64,
	maxCorridorOverlap float64,
	maxEdgeReuseRatio float64,
	maxAxisReuseCount int,
	maxDistanceShortfallRatio float64,
	profileName string,
	priorityReason string,
) []routesDomain.RouteRecommendation {
	for _, candidate := range sortedCandidates {
		if len(selected) >= limit {
			break
		}
		routeID := candidate.recommendation.RouteID
		if _, exists := selectedIDs[routeID]; exists {
			continue
		}
		if candidate.directionPenalty > maxDirectionPenalty {
			continue
		}
		if candidate.backtrackingRatio > maxBacktrackingRatio {
			continue
		}
		if candidate.corridorOverlap > maxCorridorOverlap {
			continue
		}
		if candidate.edgeReuseRatio > maxEdgeReuseRatio {
			continue
		}
		if candidate.maxAxisReuseCount > maxAxisReuseCount {
			continue
		}
		if candidate.distanceDeltaRatio > maxDistanceShortfallRatio {
			continue
		}
		recommendation := candidate.recommendation
		if strings.TrimSpace(priorityReason) != "" {
			recommendation.Reasons = append(recommendation.Reasons, priorityReason)
		}
		recommendation.Reasons = append(recommendation.Reasons, fmt.Sprintf("Selection profile: %s", profileName))
		selected = append(selected, recommendation)
		selectedIDs[routeID] = struct{}{}
	}
	return selected
}

func buildRouteRelaxationLevels(routeType string, hasDirection bool, directionStrict bool, distanceTargetKm float64) []routeRelaxationLevel {
	baseMinDiversity := minSegmentDiversityRatio(routeType)
	strictDirection := 1.0
	balancedDirection := 1.0
	relaxedDirection := 1.0
	fallbackDirection := 1.0
	if hasDirection {
		// Keep global direction more stable across selection levels.
		strictDirection = 0.14
		balancedDirection = 0.22
		relaxedDirection = 0.32
		fallbackDirection = 0.42
		if directionStrict {
			strictDirection = 0.08
			balancedDirection = 0.12
			relaxedDirection = 0.18
			fallbackDirection = 0.24
		}
	}
	// Native ultra anti-backtracking policy (always-on).
	baseMinDiversity = math.Min(0.95, baseMinDiversity+0.06)
	strictBacktrackingRatio := 0.0010
	balancedBacktrackingRatio := 0.0030
	relaxedBacktrackingRatio := 0.0070
	fallbackBacktrackingRatio := 0.015
	strictCorridorOverlap := 0.003
	balancedCorridorOverlap := 0.007
	relaxedCorridorOverlap := 0.012
	fallbackCorridorOverlap := 0.018
	strictEdgeReuseRatio := 0.008
	balancedEdgeReuseRatio := 0.020
	relaxedEdgeReuseRatio := 0.040
	fallbackEdgeReuseRatio := 0.065
	strictAxisCap, balancedAxisCap, relaxedAxisCap, fallbackAxisCap := adaptiveAxisReuseThresholds(distanceTargetKm, hasDirection, directionStrict)

	return []routeRelaxationLevel{
		{
			name:                  "strict",
			maxDirectionPenalty:   strictDirection,
			maxBacktrackingRatio:  strictBacktrackingRatio,
			maxCorridorOverlap:    strictCorridorOverlap,
			maxEdgeReuseRatio:     strictEdgeReuseRatio,
			maxAxisReuseCount:     strictAxisCap,
			minSegmentDiversity:   baseMinDiversity,
			maxDistanceDeltaRatio: 0.04,
		},
		{
			name:                  "balanced",
			maxDirectionPenalty:   balancedDirection,
			maxBacktrackingRatio:  balancedBacktrackingRatio,
			maxCorridorOverlap:    balancedCorridorOverlap,
			maxEdgeReuseRatio:     balancedEdgeReuseRatio,
			maxAxisReuseCount:     balancedAxisCap,
			minSegmentDiversity:   math.Max(0.22, baseMinDiversity-0.08),
			maxDistanceDeltaRatio: 0.08,
		},
		{
			name:                  "relaxed",
			maxDirectionPenalty:   relaxedDirection,
			maxBacktrackingRatio:  relaxedBacktrackingRatio,
			maxCorridorOverlap:    relaxedCorridorOverlap,
			maxEdgeReuseRatio:     relaxedEdgeReuseRatio,
			maxAxisReuseCount:     relaxedAxisCap,
			minSegmentDiversity:   math.Max(0.12, baseMinDiversity-0.18),
			maxDistanceDeltaRatio: 0.14,
		},
		{
			name:                  "fallback",
			maxDirectionPenalty:   fallbackDirection,
			maxBacktrackingRatio:  fallbackBacktrackingRatio,
			maxCorridorOverlap:    fallbackCorridorOverlap,
			maxEdgeReuseRatio:     fallbackEdgeReuseRatio,
			maxAxisReuseCount:     fallbackAxisCap,
			minSegmentDiversity:   0.08,
			maxDistanceDeltaRatio: 0.20,
		},
	}
}

func adaptiveAxisReuseThresholds(distanceTargetKm float64, hasDirection bool, directionStrict bool) (int, int, int, int) {
	strictCap := 2
	balancedCap := 3
	relaxedCap := 4
	fallbackCap := 5

	switch {
	case distanceTargetKm >= 130:
		strictCap, balancedCap, relaxedCap, fallbackCap = 4, 5, 6, 8
	case distanceTargetKm >= 90:
		strictCap, balancedCap, relaxedCap, fallbackCap = 3, 4, 6, 7
	case distanceTargetKm >= 60:
		strictCap, balancedCap, relaxedCap, fallbackCap = 3, 4, 5, 6
	case distanceTargetKm >= 30:
		strictCap, balancedCap, relaxedCap, fallbackCap = 2, 3, 5, 6
	}

	if hasDirection {
		strictCap++
		balancedCap++
		relaxedCap++
		fallbackCap++
	}
	if directionStrict {
		strictCap++
		balancedCap++
	}

	return clampInt(strictCap, 2, 6), clampInt(balancedCap, 3, 7), clampInt(relaxedCap, 4, 8), clampInt(fallbackCap, 5, 9)
}

func bestEffortAxisReuseCaps(distanceTargetKm float64, hasDirection bool, directionStrict bool) (int, int) {
	_, _, _, fallbackCap := adaptiveAxisReuseThresholds(distanceTargetKm, hasDirection, directionStrict)
	softCap := clampInt(fallbackCap+1, 6, 10)
	directionalCap := clampInt(fallbackCap+2, 7, 11)
	return softCap, directionalCap
}

func disjointHardAxisReuseCap(request application.RoutingEngineRequest) int {
	_, _, relaxedCap, fallbackCap := adaptiveAxisReuseThresholds(
		request.DistanceTargetKm,
		strings.TrimSpace(request.StartDirection) != "",
		request.DirectionStrict,
	)
	// Construction phase should stay tighter than post-selection fallback.
	return clampInt(maxInt(relaxedCap, fallbackCap-1), 4, 8)
}

func clampInt(value int, minValue int, maxValue int) int {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func incrementRejectCount(rejectCounts map[string]int, reason string) {
	if rejectCounts == nil {
		return
	}
	normalizedReason := strings.TrimSpace(reason)
	if normalizedReason == "" {
		return
	}
	rejectCounts[normalizedReason] = rejectCounts[normalizedReason] + 1
}

func formatRejectCounts(rejectCounts map[string]int) string {
	if len(rejectCounts) == 0 {
		return "none"
	}
	keys := make([]string, 0, len(rejectCounts))
	for key := range rejectCounts {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		left := keys[i]
		right := keys[j]
		leftCount := rejectCounts[left]
		rightCount := rejectCounts[right]
		if leftCount == rightCount {
			return left < right
		}
		return leftCount > rightCount
	})

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%d", key, rejectCounts[key]))
	}
	return strings.Join(parts, ", ")
}

func activityTypeFromRouteType(routeType string) business.ActivityType {
	switch strings.ToUpper(strings.TrimSpace(routeType)) {
	case "RUN":
		return business.Run
	case "TRAIL":
		return business.TrailRun
	case "HIKE":
		return business.Hike
	case "MTB":
		return business.MountainBikeRide
	case "GRAVEL":
		return business.GravelRide
	default:
		return business.Ride
	}
}

func destinationFromBearing(
	start routesDomain.Coordinates,
	distanceKm float64,
	bearingDegrees float64,
) routesDomain.Coordinates {
	lat1 := degreesToRadians(start.Lat)
	lon1 := degreesToRadians(start.Lng)
	bearing := degreesToRadians(bearingDegrees)
	angularDistance := distanceKm / 6371.0

	lat2 := math.Asin(math.Sin(lat1)*math.Cos(angularDistance) + math.Cos(lat1)*math.Sin(angularDistance)*math.Cos(bearing))
	lon2 := lon1 + math.Atan2(
		math.Sin(bearing)*math.Sin(angularDistance)*math.Cos(lat1),
		math.Cos(angularDistance)-math.Sin(lat1)*math.Sin(lat2),
	)

	return routesDomain.Coordinates{
		Lat: radiansToDegrees(lat2),
		Lng: normalizeLongitude(radiansToDegrees(lon2)),
	}
}

func normalizeBearing(value float64) float64 {
	normalized := math.Mod(value, 360.0)
	if normalized < 0 {
		return normalized + 360.0
	}
	return normalized
}

func startDirectionToBearing(direction string) float64 {
	switch strings.ToUpper(strings.TrimSpace(direction)) {
	case "N":
		return 0
	case "E":
		return 90
	case "S":
		return 180
	case "W":
		return 270
	default:
		return 0
	}
}

func generatedOSMRouteID(points [][]float64, start routesDomain.Coordinates, index int) string {
	hasher := fnv.New64a()
	_, _ = hasher.Write([]byte(fmt.Sprintf("%.5f|%.5f|%d|", start.Lat, start.Lng, index)))
	step := 1
	if len(points) > 40 {
		step = int(math.Ceil(float64(len(points)) / 40.0))
	}
	for i := 0; i < len(points); i += step {
		point := points[i]
		_, _ = hasher.Write([]byte(fmt.Sprintf("%.5f,%.5f|", point[0], point[1])))
	}
	return fmt.Sprintf("generated-osm-%x", hasher.Sum64())
}

func generatedEditedOSMRouteID(points [][]float64, sourceRouteID string) string {
	hasher := fnv.New64a()
	_, _ = hasher.Write([]byte("edit|"))
	_, _ = hasher.Write([]byte(strings.TrimSpace(sourceRouteID)))
	_, _ = hasher.Write([]byte("|"))
	step := 1
	if len(points) > 40 {
		step = int(math.Ceil(float64(len(points)) / 40.0))
	}
	for i := 0; i < len(points); i += step {
		point := points[i]
		_, _ = hasher.Write([]byte(fmt.Sprintf("%.5f,%.5f|", point[0], point[1])))
	}
	return fmt.Sprintf("edited-osm-%x", hasher.Sum64())
}
