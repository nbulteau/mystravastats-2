package infrastructure

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"log"
	"math"
	"mystravastats/internal/routes/application"
	routesDomain "mystravastats/internal/routes/domain"
	"mystravastats/internal/shared/domain/business"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	defaultOSMRoutingBaseURL    = "http://localhost:5000"
	defaultOSMRoutingTimeoutMs  = 3000
	defaultOSMRoutingV3Enabled  = true
	maxOSRMRoutingCalls         = 24
	startSnapToleranceMeters    = 900.0
	fallbackStartSnapTolerance  = 4000.0
	directionToleranceMeters    = 120.0
	backtrackingStartZoneM      = 2000.0
	minAxisSegmentLengthM       = 25.0
	minOppositeReuseMeters      = 120.0
	historyReuseBonusWeight     = 18.0
	historyStartZoneBonusWeight = 14.0
	historyAxisBiasWeight       = 0.75
	historyZoneBiasWeight       = 0.25
	defaultOSRMProfileFilePath  = "./osm/region.osrm.profile"
	fallbackOSRMProfilePath     = "../osm/region.osrm.profile"
)

type osrmRouteCandidate struct {
	recommendation      routesDomain.RouteRecommendation
	directionPenalty    float64
	backtrackingRatio   float64
	corridorOverlap     float64
	edgeReuseRatio      float64
	maxAxisReuseCount   int
	maxAxisReuseRatio   float64
	segmentDiversity    float64
	distanceDeltaRatio  float64
	pathRatio           float64
	historyReuseScore   float64
	effectiveMatchScore float64
}

type routingHistoryBiasContext struct {
	enabled             bool
	normalizedRouteType string
	axisScores          map[string]float64
	zoneScores          map[string]float64
	maxAxisScore        float64
	maxZoneScore        float64
}

type routeRelaxationLevel struct {
	name                  string
	maxDirectionPenalty   float64
	maxBacktrackingRatio  float64
	maxCorridorOverlap    float64
	maxEdgeReuseRatio     float64
	maxAxisReuseCount     int
	minSegmentDiversity   float64
	maxDistanceDeltaRatio float64
}

type routeSurfaceBreakdown struct {
	pavedM   float64
	gravelM  float64
	trailM   float64
	unknownM float64
}

type osrmRouteResponse struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Routes  []osrmRoute `json:"routes"`
}

type osrmNearestResponse struct {
	Code      string             `json:"code"`
	Message   string             `json:"message"`
	Waypoints []osrmNearestPoint `json:"waypoints"`
}

type osrmNearestPoint struct {
	Distance float64   `json:"distance"`
	Location []float64 `json:"location"`
}

type osrmRoute struct {
	Distance float64      `json:"distance"`
	Duration float64      `json:"duration"`
	Geometry osrmGeometry `json:"geometry"`
	Legs     []osrmLeg    `json:"legs"`
}

type osrmGeometry struct {
	Type        string      `json:"type"`
	Coordinates [][]float64 `json:"coordinates"`
}

type osrmLeg struct {
	Steps []osrmStep `json:"steps"`
}

type osrmStep struct {
	Distance float64  `json:"distance"`
	Mode     string   `json:"mode"`
	Classes  []string `json:"classes"`
}

// OSMRoutingAdapter integrates a local OSRM endpoint as a routing engine.
type OSMRoutingAdapter struct {
	enabled               bool
	v3Enabled             bool
	debug                 bool
	baseURL               string
	timeout               time.Duration
	client                *http.Client
	profileOverride       string
	extractProfileEnv     string
	extractProfileCfgFile string
}

func NewOSMRoutingAdapter() *OSMRoutingAdapter {
	enabled := readBoolEnv("OSM_ROUTING_ENABLED", true)
	baseURL := strings.TrimRight(strings.TrimSpace(readStringEnv("OSM_ROUTING_BASE_URL", defaultOSMRoutingBaseURL)), "/")
	timeoutMs := readIntEnv("OSM_ROUTING_TIMEOUT_MS", defaultOSMRoutingTimeoutMs)
	if timeoutMs < 200 {
		timeoutMs = defaultOSMRoutingTimeoutMs
	}
	profileOverride := strings.TrimSpace(readStringEnv("OSM_ROUTING_PROFILE", ""))
	extractProfileEnv := strings.TrimSpace(readStringEnv("OSM_ROUTING_EXTRACT_PROFILE", ""))
	extractProfileCfgFile := strings.TrimSpace(readStringEnv("OSM_ROUTING_EXTRACT_PROFILE_FILE", defaultOSRMProfileFilePath))

	return &OSMRoutingAdapter{
		enabled:               enabled,
		v3Enabled:             readBoolEnv("OSM_ROUTING_V3_ENABLED", defaultOSMRoutingV3Enabled),
		debug:                 readBoolEnv("OSM_ROUTING_DEBUG", false),
		baseURL:               baseURL,
		timeout:               time.Duration(timeoutMs) * time.Millisecond,
		client:                &http.Client{Timeout: time.Duration(timeoutMs) * time.Millisecond},
		profileOverride:       profileOverride,
		extractProfileEnv:     extractProfileEnv,
		extractProfileCfgFile: extractProfileCfgFile,
	}
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

	request, err := http.NewRequest(http.MethodGet, adapter.baseURL+"/", nil)
	if err != nil {
		details["status"] = "down"
		details["reachable"] = false
		details["error"] = err.Error()
		return details
	}
	response, err := adapter.client.Do(request)
	if err != nil {
		details["status"] = "down"
		details["reachable"] = false
		details["error"] = err.Error()
		return details
	}
	defer func() { _ = response.Body.Close() }()

	details["statusCode"] = response.StatusCode
	if response.StatusCode >= 500 {
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
	waypoints := buildShapeLoopWaypoints(request.StartPoint, projectedShape)
	if len(waypoints) < 3 {
		return []routesDomain.RouteRecommendation{}, nil
	}

	profile := adapter.profileForRouteType(request.RouteType)
	routes, err := adapter.fetchOSRMRoutes(profile, waypoints)
	if err != nil {
		return []routesDomain.RouteRecommendation{}, err
	}

	shapeRequest := request
	shapeRequest.DistanceTargetKm = targetDistanceKm
	shapeRequest.StartDirection = ""
	shapeRequest.DirectionStrict = false
	shapePreview := coordinatesToLatLngPoints(projectedShape)
	rejectCounts := make(map[string]int)
	candidates := make([]osrmRouteCandidate, 0, len(routes))
	seenSignatures := make(map[string]struct{}, len(routes))

	for routeIndex, osrmRoute := range routes {
		candidate, ok := adapter.toRouteCandidate(shapeRequest, osrmRoute, routeIndex, rejectCounts)
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

		shapeScore := shapeSimilarityScore(candidate.recommendation.PreviewLatLng, shapePreview)
		shapeName := "CUSTOM_SHAPE"
		recommendation := candidate.recommendation
		recommendation.VariantType = routesDomain.RouteVariantShape
		recommendation.Shape = &shapeName
		recommendation.ShapeScore = &shapeScore
		recommendation.MatchScore = clampOSMScore(
			recommendation.MatchScore*0.35 + shapeScore*100.0*0.65 -
				candidate.backtrackingRatio*28.0 -
				candidate.corridorOverlap*35.0 -
				candidate.edgeReuseRatio*40.0 -
				candidate.maxAxisReuseRatio*48.0,
		)
		recommendation.Reasons = append(
			recommendation.Reasons,
			fmt.Sprintf("Shape similarity: %.0f%%", shapeScore*100.0),
			"Shape mode: projected waypoints",
		)

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

	recommendations := selectCandidatesWithRelaxation(shapeRequest, candidates, rejectCounts)
	if len(recommendations) > request.Limit {
		recommendations = recommendations[:request.Limit]
	}

	if adapter.debug || len(recommendations) == 0 {
		log.Printf(
			"OSRM shape generation summary: routeType=%s shapePoints=%d waypoints=%d fetched=%d accepted=%d rejects=%s",
			strings.ToUpper(strings.TrimSpace(request.RouteType)),
			len(rawShape),
			len(waypoints),
			len(routes),
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
	if len(waypoints) < 2 {
		return nil, fmt.Errorf("at least 2 waypoints are required")
	}

	coordinates := make([]string, 0, len(waypoints))
	for _, point := range waypoints {
		coordinates = append(coordinates, fmt.Sprintf("%.6f,%.6f", point.Lng, point.Lat))
	}
	url := fmt.Sprintf(
		"%s/route/v1/%s/%s?alternatives=true&steps=true&overview=full&geometries=geojson&continue_straight=true",
		adapter.baseURL,
		profile,
		strings.Join(coordinates, ";"),
	)

	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	response, err := adapter.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("osrm route API returned status %d", response.StatusCode)
	}

	var payload osrmRouteResponse
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return nil, err
	}
	if strings.ToLower(payload.Code) != "ok" {
		if payload.Message == "" {
			return nil, fmt.Errorf("osrm route API returned code %s", payload.Code)
		}
		return nil, fmt.Errorf("osrm route API returned code %s: %s", payload.Code, payload.Message)
	}
	return payload.Routes, nil
}

func (adapter *OSMRoutingAdapter) snapToNearestRoutablePoint(
	profile string,
	point routesDomain.Coordinates,
) (routesDomain.Coordinates, float64, bool) {
	url := fmt.Sprintf(
		"%s/nearest/v1/%s/%.6f,%.6f?number=1",
		adapter.baseURL,
		profile,
		point.Lng,
		point.Lat,
	)

	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return routesDomain.Coordinates{}, 0.0, false
	}
	response, err := adapter.client.Do(request)
	if err != nil {
		return routesDomain.Coordinates{}, 0.0, false
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return routesDomain.Coordinates{}, 0.0, false
	}

	var payload osrmNearestResponse
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return routesDomain.Coordinates{}, 0.0, false
	}
	if strings.ToLower(strings.TrimSpace(payload.Code)) != "ok" || len(payload.Waypoints) == 0 {
		return routesDomain.Coordinates{}, 0.0, false
	}
	location := payload.Waypoints[0].Location
	if len(location) < 2 {
		return routesDomain.Coordinates{}, 0.0, false
	}
	snapped := routesDomain.Coordinates{
		Lat: location[1],
		Lng: location[0],
	}
	return snapped, math.Max(0.0, payload.Waypoints[0].Distance), true
}

func (adapter *OSMRoutingAdapter) toRouteCandidate(
	request application.RoutingEngineRequest,
	route osrmRoute,
	index int,
	rejectCounts map[string]int,
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
	return adapter.toRouteCandidateFromPreview(
		request,
		points,
		computeSurfaceBreakdown(route),
		distanceKm,
		durationSec,
		index,
		rejectCounts,
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
	if len(points) < 2 {
		incrementRejectCount(rejectCounts, "INVALID_COORDINATES")
		return osrmRouteCandidate{}, false
	}
	startOffsetMeters := haversineDistanceMeters(points[0][0], points[0][1], request.StartPoint.Lat, request.StartPoint.Lng)
	if !startsNearRequestedStart(points, request.StartPoint, startSnapToleranceMeters) {
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
	if hasOppositeOutsideStart {
		if request.StrictBacktracking {
			incrementRejectCount(rejectCounts, "STRICT_BACKTRACKING_OUTSIDE_START")
		} else {
			incrementRejectCount(rejectCounts, "BACKTRACKING_FILTERED")
		}
		return osrmRouteCandidate{}, false
	}
	if maxAxisReuseOutsideStart > maxAxisReuseOutsideStartLimit {
		incrementRejectCount(rejectCounts, "AXIS_REUSE_OUTSIDE_START")
		return osrmRouteCandidate{}, false
	}
	if !meetsMinimumDistance(distanceKm, request.DistanceTargetKm) {
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
	if backtrackingRatio > maxBacktrackingReject ||
		corridorOverlap > maxCorridorReject ||
		edgeReuse > maxEdgeReuseReject ||
		maxAxisReuseCount > maxAxisReuseReject {
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
	if request.ElevationTargetM != nil {
		reasons = append(reasons, fmt.Sprintf("Elevation estimate: %s", formatElevationDelta(elevationGainM-*request.ElevationTargetM)))
	}
	if request.StartDirection != "" {
		reasons = append(reasons, fmt.Sprintf("Direction: %s", startDirectionLabel(request.StartDirection)))
	}
	if !request.StrictBacktracking && startOffsetMeters > startSnapToleranceMeters {
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
	if normalizedRouteType == "GRAVEL" && pathRatio < requiredPathRatio {
		incrementRejectCount(rejectCounts, "GRAVEL_MIN_PATH_RATIO")
		return osrmRouteCandidate{}, false
	}
	reasons = append(
		reasons,
		fmt.Sprintf("Surface mix: %s", formatSurfaceBreakdown(surfaceBreakdown)),
		fmt.Sprintf("Path ratio: %.0f%%", pathRatio*100.0),
		fmt.Sprintf("Surface fitness: %.0f%%", surfaceScore),
		"Surface source: OSRM step classes and mode",
	)
	if request.StrictBacktracking {
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
	// effectiveScore is an internal ranking score (not API score):
	// it aggressively penalizes backtracking and bad directional fit to keep
	// generated loops practical even in relaxed levels.

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

	sortedCandidates := make([]osrmRouteCandidate, len(candidates))
	copy(sortedCandidates, candidates)
	sort.SliceStable(sortedCandidates, func(i, j int) bool {
		left := sortedCandidates[i]
		right := sortedCandidates[j]
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

	// Levels are evaluated in order: strict -> balanced -> relaxed -> fallback.
	// We fill results incrementally: if strict cannot fill the target limit,
	// next levels progressively loosen constraints while keeping quality.
	levels := buildRouteRelaxationLevels(
		request.RouteType,
		hasDirection,
		request.DirectionStrict,
		request.DistanceTargetKm,
	)
	selected := make([]routesDomain.RouteRecommendation, 0, limit)
	selectedIDs := make(map[string]struct{}, limit)

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
			if candidate.directionPenalty > level.maxDirectionPenalty {
				incrementRejectCount(rejectCounts, "DIRECTION_CONSTRAINT")
				continue
			}
			if candidate.backtrackingRatio > level.maxBacktrackingRatio {
				incrementRejectCount(rejectCounts, "OPPOSITE_EDGE_TRAVERSAL")
				continue
			}
			if candidate.corridorOverlap > level.maxCorridorOverlap {
				incrementRejectCount(rejectCounts, "CORRIDOR_OVERLAP")
				continue
			}
			if candidate.edgeReuseRatio > level.maxEdgeReuseRatio {
				incrementRejectCount(rejectCounts, "EDGE_REUSE")
				continue
			}
			if candidate.maxAxisReuseCount > level.maxAxisReuseCount {
				incrementRejectCount(rejectCounts, "MAX_AXIS_REUSE")
				continue
			}
			if candidate.segmentDiversity < level.minSegmentDiversity {
				incrementRejectCount(rejectCounts, "LOW_SEGMENT_DIVERSITY")
				continue
			}
			if candidate.distanceDeltaRatio > level.maxDistanceDeltaRatio {
				incrementRejectCount(rejectCounts, "DISTANCE_CONSTRAINT")
				continue
			}

			recommendation := candidate.recommendation
			recommendation.Reasons = append(recommendation.Reasons, fmt.Sprintf("Selection profile: %s", level.name))
			selected = append(selected, recommendation)
			selectedIDs[routeID] = struct{}{}
		}
	}

	// Safety net: if all configured levels reject candidates, return the best
	// ranked loops with softer anti-overlap limits instead of returning zero.
	softAxisCap, directionalAxisCap := bestEffortAxisReuseCaps(request.DistanceTargetKm, hasDirection, request.DirectionStrict)
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

func computeSurfaceBreakdown(route osrmRoute) routeSurfaceBreakdown {
	breakdown := routeSurfaceBreakdown{}
	for _, leg := range route.Legs {
		for _, step := range leg.Steps {
			distance := math.Max(0.0, step.Distance)
			if distance <= 0 {
				continue
			}
			switch classifySurfaceBucket(step) {
			case "paved":
				breakdown.pavedM += distance
			case "gravel":
				breakdown.gravelM += distance
			case "trail":
				breakdown.trailM += distance
			default:
				breakdown.unknownM += distance
			}
		}
	}
	// If no step-level data is available, keep an explicit "unknown" fallback.
	if breakdown.totalDistanceM() <= 0 && route.Distance > 0 {
		breakdown.unknownM = route.Distance
	}
	return breakdown
}

func mergeSurfaceBreakdowns(left routeSurfaceBreakdown, right routeSurfaceBreakdown) routeSurfaceBreakdown {
	return routeSurfaceBreakdown{
		pavedM:   left.pavedM + right.pavedM,
		gravelM:  left.gravelM + right.gravelM,
		trailM:   left.trailM + right.trailM,
		unknownM: left.unknownM + right.unknownM,
	}
}

func classifySurfaceBucket(step osrmStep) string {
	mode := strings.ToLower(strings.TrimSpace(step.Mode))
	if strings.Contains(mode, "pushing") || mode == "foot" || mode == "walking" {
		return "trail"
	}
	classes := make(map[string]struct{}, len(step.Classes))
	for _, rawClass := range step.Classes {
		normalized := strings.ToLower(strings.TrimSpace(rawClass))
		if normalized == "" {
			continue
		}
		classes[normalized] = struct{}{}
	}
	if _, hasFerry := classes["ferry"]; hasFerry {
		return "unknown"
	}
	if hasAnyClass(classes, "path", "track", "steps", "bridleway", "cycleway_unpaved") {
		return "trail"
	}
	if hasAnyClass(classes, "unpaved", "gravel", "dirt", "ground", "earth", "compacted", "fine_gravel", "sand", "mud") {
		return "gravel"
	}
	if mode == "cycling" || mode == "driving" || mode == "running" {
		return "paved"
	}
	return "unknown"
}

func hasAnyClass(classes map[string]struct{}, keys ...string) bool {
	for _, key := range keys {
		if _, exists := classes[key]; exists {
			return true
		}
	}
	return false
}

func (breakdown routeSurfaceBreakdown) totalDistanceM() float64 {
	return breakdown.pavedM + breakdown.gravelM + breakdown.trailM + breakdown.unknownM
}

func (breakdown routeSurfaceBreakdown) normalizedRatios() (float64, float64, float64, float64) {
	total := breakdown.totalDistanceM()
	if total <= 0 {
		return 0, 0, 0, 1
	}
	return breakdown.pavedM / total, breakdown.gravelM / total, breakdown.trailM / total, breakdown.unknownM / total
}

func (breakdown routeSurfaceBreakdown) pathRatio() float64 {
	_, gravel, trail, _ := breakdown.normalizedRatios()
	return clampUnit(gravel + trail)
}

func formatSurfaceBreakdown(breakdown routeSurfaceBreakdown) string {
	paved, gravel, trail, unknown := breakdown.normalizedRatios()
	return fmt.Sprintf(
		"paved %.0f%%, gravel %.0f%%, trail %.0f%%, unknown %.0f%%",
		paved*100.0,
		gravel*100.0,
		trail*100.0,
		unknown*100.0,
	)
}

func surfaceMatchScore(routeType string, breakdown routeSurfaceBreakdown) float64 {
	paved, gravel, trail, unknown := breakdown.normalizedRatios()
	pathRatio := clampUnit(gravel + trail)

	targetPaved := 0.60
	targetGravel := 0.25
	targetTrail := 0.15
	switch strings.ToUpper(strings.TrimSpace(routeType)) {
	case "RIDE":
		targetPaved, targetGravel, targetTrail = 0.92, 0.06, 0.02
	case "GRAVEL":
		// Gravel contract:
		// - minimum 25% paths (gravel + trail)
		// - no hard upper bound once this minimum is reached
		shortfall := math.Max(0.0, 0.25-pathRatio)
		pavedExcess := math.Max(0.0, paved-0.75)
		penalty := shortfall*220.0 + pavedExcess*36.0 + unknown*22.0
		return clampOSMScore(100.0 - penalty)
	case "MTB":
		// MTB should prefer paths as much as possible.
		pavedExcess := math.Max(0.0, paved-0.20)
		score := 28.0 + pathRatio*74.0 - unknown*24.0 - pavedExcess*48.0
		return clampOSMScore(score)
	case "RUN":
		targetPaved, targetGravel, targetTrail = 0.50, 0.25, 0.25
	case "TRAIL", "HIKE":
		targetPaved, targetGravel, targetTrail = 0.12, 0.28, 0.60
	}

	penalty := math.Abs(paved-targetPaved)*85.0 +
		math.Abs(gravel-targetGravel)*78.0 +
		math.Abs(trail-targetTrail)*92.0 +
		unknown*35.0
	return clampOSMScore(100.0 - penalty)
}

func surfaceScoreWeight(routeType string) float64 {
	switch strings.ToUpper(strings.TrimSpace(routeType)) {
	case "RIDE":
		return 1.10
	case "GRAVEL":
		return 1.25
	case "MTB":
		return 1.70
	case "TRAIL", "HIKE":
		return 1.40
	default:
		return 0.45
	}
}

func pathPreferenceBonus(routeType string, pathRatio float64) float64 {
	normalizedType := strings.ToUpper(strings.TrimSpace(routeType))
	switch normalizedType {
	case "RIDE":
		// Road rides should avoid off-road sections as much as possible.
		return (0.10 - pathRatio) * 35.0
	case "MTB":
		// Strongly reward path-heavy candidates for MTB.
		return (pathRatio - 0.50) * 60.0
	case "GRAVEL":
		// Encourage higher path ratio once the 25% minimum is reached.
		return (pathRatio - 0.25) * 30.0
	default:
		return 0.0
	}
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

func parseShapePolylineCoordinates(raw string) []routesDomain.Coordinates {
	trimmed := strings.TrimSpace(raw)
	if !strings.HasPrefix(trimmed, "[") {
		return []routesDomain.Coordinates{}
	}
	var points [][]float64
	if err := json.Unmarshal([]byte(trimmed), &points); err != nil {
		return []routesDomain.Coordinates{}
	}
	result := make([]routesDomain.Coordinates, 0, len(points))
	for _, point := range points {
		if len(point) < 2 {
			continue
		}
		lat := point[0]
		lng := point[1]
		if lat < -90 || lat > 90 || lng < -180 || lng > 180 {
			continue
		}
		result = append(result, routesDomain.Coordinates{Lat: lat, Lng: lng})
	}
	return result
}

func polylineDistanceKmFromCoordinates(points []routesDomain.Coordinates) float64 {
	if len(points) < 2 {
		return 0.0
	}
	totalMeters := 0.0
	for index := 0; index < len(points)-1; index++ {
		left := points[index]
		right := points[index+1]
		totalMeters += haversineDistanceMeters(left.Lat, left.Lng, right.Lat, right.Lng)
	}
	return totalMeters / 1000.0
}

func projectShapePolylineToStart(
	shape []routesDomain.Coordinates,
	start routesDomain.Coordinates,
	targetDistanceKm float64,
) []routesDomain.Coordinates {
	if len(shape) == 0 {
		return []routesDomain.Coordinates{}
	}
	translated := make([]routesDomain.Coordinates, 0, len(shape))
	deltaLat := start.Lat - shape[0].Lat
	deltaLng := start.Lng - shape[0].Lng
	for _, point := range shape {
		translated = append(translated, routesDomain.Coordinates{
			Lat: point.Lat + deltaLat,
			Lng: point.Lng + deltaLng,
		})
	}

	scale := 1.0
	shapeDistanceKm := polylineDistanceKmFromCoordinates(translated)
	if targetDistanceKm > 0 && shapeDistanceKm > 0 {
		scale = targetDistanceKm / shapeDistanceKm
		if scale < 0.45 {
			scale = 0.45
		}
		if scale > 2.60 {
			scale = 2.60
		}
	}

	projected := make([]routesDomain.Coordinates, 0, len(translated))
	projected = append(projected, start)
	for index := 1; index < len(translated); index++ {
		point := translated[index]
		projected = append(projected, routesDomain.Coordinates{
			Lat: start.Lat + (point.Lat-start.Lat)*scale,
			Lng: start.Lng + (point.Lng-start.Lng)*scale,
		})
	}
	return projected
}

func sampleCoordinates(points []routesDomain.Coordinates, maxPoints int) []routesDomain.Coordinates {
	if len(points) <= maxPoints || maxPoints <= 0 {
		return points
	}
	step := int(math.Ceil(float64(len(points)) / float64(maxPoints)))
	if step < 1 {
		step = 1
	}
	sampled := make([]routesDomain.Coordinates, 0, maxPoints+1)
	lastIndex := len(points) - 1
	for index := 0; index < len(points); index += step {
		sampled = append(sampled, points[index])
	}
	lastSample := sampled[len(sampled)-1]
	lastPoint := points[lastIndex]
	if lastSample.Lat != lastPoint.Lat || lastSample.Lng != lastPoint.Lng {
		sampled = append(sampled, lastPoint)
	}
	return sampled
}

func buildShapeLoopWaypoints(
	start routesDomain.Coordinates,
	shape []routesDomain.Coordinates,
) []routesDomain.Coordinates {
	sampled := sampleCoordinates(shape, 10)
	waypoints := make([]routesDomain.Coordinates, 0, len(sampled)+2)
	waypoints = append(waypoints, start)
	previous := start
	for index := 1; index < len(sampled); index++ {
		point := sampled[index]
		if haversineDistanceMeters(previous.Lat, previous.Lng, point.Lat, point.Lng) < 120.0 {
			continue
		}
		waypoints = append(waypoints, point)
		previous = point
	}
	waypoints = append(waypoints, start)
	return waypoints
}

func coordinatesToLatLngPoints(points []routesDomain.Coordinates) [][]float64 {
	result := make([][]float64, 0, len(points))
	for _, point := range points {
		result = append(result, []float64{point.Lat, point.Lng})
	}
	return result
}

type normalizedShapePoint struct {
	x float64
	y float64
}

func shapeSimilarityScore(routePoints [][]float64, shapePoints [][]float64) float64 {
	normalizedRoute := normalizeShapePolyline(samplePolylinePoints(routePoints, 90))
	normalizedShape := normalizeShapePolyline(samplePolylinePoints(shapePoints, 90))
	if len(normalizedRoute) < 2 || len(normalizedShape) < 2 {
		return 0.0
	}
	meanForward := meanNearestShapeDistance(normalizedShape, normalizedRoute)
	meanBackward := meanNearestShapeDistance(normalizedRoute, normalizedShape)
	distance := (meanForward + meanBackward) / 2.0
	score := 1.0 - (distance / 1.35)
	return clampUnit(score)
}

func normalizeShapePolyline(points [][]float64) []normalizedShapePoint {
	if len(points) == 0 {
		return []normalizedShapePoint{}
	}
	sumLat := 0.0
	sumLng := 0.0
	count := 0
	for _, point := range points {
		if len(point) < 2 {
			continue
		}
		sumLat += point[0]
		sumLng += point[1]
		count++
	}
	if count == 0 {
		return []normalizedShapePoint{}
	}
	centerLat := sumLat / float64(count)
	centerLng := sumLng / float64(count)
	cosLat := math.Cos(degreesToRadians(centerLat))
	maxRadius := 0.0
	normalized := make([]normalizedShapePoint, 0, count)
	for _, point := range points {
		if len(point) < 2 {
			continue
		}
		x := (point[1] - centerLng) * 111320.0 * cosLat
		y := (point[0] - centerLat) * 111320.0
		radius := math.Sqrt(x*x + y*y)
		if radius > maxRadius {
			maxRadius = radius
		}
		normalized = append(normalized, normalizedShapePoint{x: x, y: y})
	}
	if maxRadius < 1.0 {
		maxRadius = 1.0
	}
	for index := range normalized {
		normalized[index].x = normalized[index].x / maxRadius
		normalized[index].y = normalized[index].y / maxRadius
	}
	return normalized
}

func meanNearestShapeDistance(from []normalizedShapePoint, to []normalizedShapePoint) float64 {
	if len(from) == 0 || len(to) == 0 {
		return 1.0
	}
	total := 0.0
	for _, left := range from {
		minDistance := math.MaxFloat64
		for _, right := range to {
			dx := left.x - right.x
			dy := left.y - right.y
			distance := math.Sqrt(dx*dx + dy*dy)
			if distance < minDistance {
				minDistance = distance
			}
		}
		total += minDistance
	}
	return total / float64(len(from))
}

func degreesToRadians(value float64) float64 {
	return value * math.Pi / 180.0
}

func radiansToDegrees(value float64) float64 {
	return value * 180.0 / math.Pi
}

func normalizeLongitude(value float64) float64 {
	for value < -180.0 {
		value += 360.0
	}
	for value > 180.0 {
		value -= 360.0
	}
	return value
}

func clampOSMScore(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return math.Round(value*10.0) / 10.0
}

func startsNearRequestedStart(points [][]float64, start routesDomain.Coordinates, toleranceMeters float64) bool {
	if len(points) == 0 {
		return false
	}
	first := points[0]
	if len(first) < 2 {
		return false
	}
	return haversineDistanceMeters(first[0], first[1], start.Lat, start.Lng) <= toleranceMeters
}

func respectsHalfPlaneDirection(
	points [][]float64,
	start routesDomain.Coordinates,
	direction string,
	toleranceMeters float64,
) bool {
	normalized := strings.ToUpper(strings.TrimSpace(direction))
	if normalized == "" || len(points) == 0 {
		return true
	}

	latTolerance := toleranceMeters / 111320.0
	lngTolerance := toleranceMeters / math.Max(1000.0, 111320.0*math.Cos(degreesToRadians(start.Lat)))

	switch normalized {
	case "N":
		limit := start.Lat - latTolerance
		for _, point := range points {
			if len(point) < 2 {
				continue
			}
			if point[0] < limit {
				return false
			}
		}
	case "S":
		limit := start.Lat + latTolerance
		for _, point := range points {
			if len(point) < 2 {
				continue
			}
			if point[0] > limit {
				return false
			}
		}
	case "E":
		limit := start.Lng - lngTolerance
		for _, point := range points {
			if len(point) < 2 {
				continue
			}
			if point[1] < limit {
				return false
			}
		}
	case "W":
		limit := start.Lng + lngTolerance
		for _, point := range points {
			if len(point) < 2 {
				continue
			}
			if point[1] > limit {
				return false
			}
		}
	}
	return true
}

func combinedDirectionPenalty(
	points [][]float64,
	start routesDomain.Coordinates,
	direction string,
	toleranceMeters float64,
) float64 {
	if strings.TrimSpace(direction) == "" {
		return 0.0
	}
	// We combine three direction signals:
	// - initial heading alignment (bearing-based)
	// - half-plane violations (did the route go too much in the opposite side)
	// - global lobe dominance (does the whole loop stay mostly in requested direction)
	// The max keeps enforcement robust in dense urban grids.
	// Bearing is intentionally softened because local street orientation near the
	// start can temporarily oppose the desired global direction.
	bearingPenalty := directionPenaltyFromPreview(points, direction)
	halfPlanePenalty := halfPlaneViolationRatio(points, start, direction, toleranceMeters)
	lobePenalty := directionalLobePenalty(points, start, direction)
	farOppositePenalty := farOppositeViolationRatio(points, start, direction, toleranceMeters)
	return math.Max(
		math.Max(bearingPenalty*0.65, halfPlanePenalty),
		math.Max(lobePenalty, farOppositePenalty),
	)
}

func halfPlaneViolationRatio(
	points [][]float64,
	start routesDomain.Coordinates,
	direction string,
	toleranceMeters float64,
) float64 {
	normalized := strings.ToUpper(strings.TrimSpace(direction))
	if normalized == "" || len(points) == 0 {
		return 0.0
	}
	latTolerance := toleranceMeters / 111320.0
	lngTolerance := toleranceMeters / math.Max(1000.0, 111320.0*math.Cos(degreesToRadians(start.Lat)))

	total := 0
	violations := 0
	for _, point := range points {
		if len(point) < 2 {
			continue
		}
		total++
		switch normalized {
		case "N":
			if point[0] < start.Lat-latTolerance {
				violations++
			}
		case "S":
			if point[0] > start.Lat+latTolerance {
				violations++
			}
		case "E":
			if point[1] < start.Lng-lngTolerance {
				violations++
			}
		case "W":
			if point[1] > start.Lng+lngTolerance {
				violations++
			}
		}
	}
	if total == 0 {
		return 0.0
	}
	return float64(violations) / float64(total)
}

func directionalLobePenalty(
	points [][]float64,
	start routesDomain.Coordinates,
	direction string,
) float64 {
	normalized := strings.ToUpper(strings.TrimSpace(direction))
	if normalized == "" || len(points) == 0 {
		return 0.0
	}

	desiredExtent := 0.0
	oppositeExtent := 0.0
	sumProjection := 0.0
	projectionCount := 0

	for _, point := range points {
		if len(point) < 2 {
			continue
		}
		projection, ok := directionProjectionMeters(point[0], point[1], start, normalized)
		if !ok {
			continue
		}
		if projection > desiredExtent {
			desiredExtent = projection
		}
		if projection < 0 && -projection > oppositeExtent {
			oppositeExtent = -projection
		}
		sumProjection += projection
		projectionCount++
	}

	if projectionCount == 0 {
		return 0.0
	}

	// Dominance asks: "how much of the route envelope is on requested side?"
	// 1.0 means full dominance on requested side, 0.5 is symmetric, 0 is opposite.
	dominancePenalty := 0.0
	totalExtent := desiredExtent + oppositeExtent
	if totalExtent > 1.0 {
		dominanceRatio := desiredExtent / totalExtent
		// Keep a clearer direction dominance in dense grids.
		dominancePenalty = clampUnit((0.68 - dominanceRatio) / 0.68)
	}

	// Average projection guard: route center of mass should not drift opposite.
	avgPenalty := 0.0
	if desiredExtent > 1.0 {
		avgProjection := sumProjection / float64(projectionCount)
		avgPenalty = clampUnit((-avgProjection) / math.Max(desiredExtent*0.25, 1.0))
	}

	return math.Max(dominancePenalty, avgPenalty)
}

func farOppositeViolationRatio(
	points [][]float64,
	start routesDomain.Coordinates,
	direction string,
	toleranceMeters float64,
) float64 {
	normalized := strings.ToUpper(strings.TrimSpace(direction))
	if normalized == "" || len(points) == 0 {
		return 0.0
	}

	guardBand := math.Max(toleranceMeters*1.8, 220.0)
	total := 0
	violations := 0

	for _, point := range points {
		if len(point) < 2 {
			continue
		}
		projection, ok := directionProjectionMeters(point[0], point[1], start, normalized)
		if !ok {
			continue
		}
		if math.Abs(projection) < guardBand {
			// Ignore local oscillations around start/return hub.
			continue
		}
		total++
		if projection < -guardBand {
			violations++
		}
	}
	if total == 0 {
		return 0.0
	}
	return float64(violations) / float64(total)
}

func directionProjectionMeters(
	lat float64,
	lng float64,
	start routesDomain.Coordinates,
	normalizedDirection string,
) (float64, bool) {
	latMeters := (lat - start.Lat) * 111320.0
	lngMeters := (lng - start.Lng) * 111320.0 * math.Cos(degreesToRadians(start.Lat))
	switch normalizedDirection {
	case "N":
		return latMeters, true
	case "S":
		return -latMeters, true
	case "E":
		return lngMeters, true
	case "W":
		return -lngMeters, true
	default:
		return 0.0, false
	}
}

func clampUnit(value float64) float64 {
	if value <= 0 {
		return 0
	}
	if value >= 1 {
		return 1
	}
	return value
}

type pathSegment struct {
	startLat float64
	startLng float64
	endLat   float64
	endLng   float64
	midLat   float64
	midLng   float64
	lengthM  float64
	bearing  float64
}

func corridorOverlapRatio(points [][]float64) float64 {
	if len(points) < 4 {
		return 0.0
	}
	sampled := samplePolylinePoints(points, 260)
	segments := buildPathSegments(sampled)
	if len(segments) < 2 {
		return 0.0
	}

	flagged := make([]bool, len(segments))
	for i := 0; i < len(segments); i++ {
		// Skip only immediate neighbors to avoid counting normal local curvature as overlap.
		for j := 0; j < i-1; j++ {
			if segmentsLikelySameCorridor(segments[i], segments[j]) {
				flagged[i] = true
				flagged[j] = true
			}
		}
	}
	overlapped := 0
	for _, value := range flagged {
		if value {
			overlapped++
		}
	}
	return float64(overlapped) / float64(len(segments))
}

func samplePolylinePoints(points [][]float64, maxPoints int) [][]float64 {
	if len(points) <= maxPoints || maxPoints <= 0 {
		return points
	}
	step := int(math.Ceil(float64(len(points)) / float64(maxPoints)))
	if step < 1 {
		step = 1
	}
	sampled := make([][]float64, 0, maxPoints+1)
	lastIndex := len(points) - 1
	for index := 0; index < len(points); index += step {
		sampled = append(sampled, points[index])
	}
	lastSample := sampled[len(sampled)-1]
	lastPoint := points[lastIndex]
	if len(lastSample) < 2 || len(lastPoint) < 2 || lastSample[0] != lastPoint[0] || lastSample[1] != lastPoint[1] {
		sampled = append(sampled, lastPoint)
	}
	return sampled
}

func buildPathSegments(points [][]float64) []pathSegment {
	segments := make([]pathSegment, 0, len(points))
	for index := 0; index < len(points)-1; index++ {
		left := points[index]
		right := points[index+1]
		if len(left) < 2 || len(right) < 2 {
			continue
		}
		lengthM := haversineDistanceMeters(left[0], left[1], right[0], right[1])
		if lengthM < 12.0 {
			continue
		}
		segments = append(segments, pathSegment{
			startLat: left[0],
			startLng: left[1],
			endLat:   right[0],
			endLng:   right[1],
			midLat:   (left[0] + right[0]) / 2.0,
			midLng:   (left[1] + right[1]) / 2.0,
			lengthM:  lengthM,
			bearing:  osrmBearingDegrees(left[0], left[1], right[0], right[1]),
		})
	}
	return segments
}

func segmentsLikelySameCorridor(left pathSegment, right pathSegment) bool {
	const midpointToleranceMeters = 50.0
	const endpointToleranceMeters = 80.0

	midpointDistance := haversineDistanceMeters(left.midLat, left.midLng, right.midLat, right.midLng)
	if midpointDistance > midpointToleranceMeters {
		return false
	}
	leftToRightStart := haversineDistanceMeters(left.startLat, left.startLng, right.startLat, right.startLng)
	leftToRightEnd := haversineDistanceMeters(left.startLat, left.startLng, right.endLat, right.endLng)
	rightToLeftStart := haversineDistanceMeters(left.endLat, left.endLng, right.startLat, right.startLng)
	rightToLeftEnd := haversineDistanceMeters(left.endLat, left.endLng, right.endLat, right.endLng)
	if math.Min(leftToRightStart, leftToRightEnd) > endpointToleranceMeters ||
		math.Min(rightToLeftStart, rightToLeftEnd) > endpointToleranceMeters {
		return false
	}
	bearingDiff := math.Abs(left.bearing - right.bearing)
	if bearingDiff > 180.0 {
		bearingDiff = 360.0 - bearingDiff
	}
	if bearingDiff > 22.0 && bearingDiff < 158.0 {
		return false
	}
	maxLength := math.Max(left.lengthM, right.lengthM)
	minLength := math.Min(left.lengthM, right.lengthM)
	if minLength <= 0 || maxLength/minLength > 6.0 {
		return false
	}
	return true
}

type axisTraversal struct {
	axisID    string
	isForward bool
}

type axisUsageSummary struct {
	totalTraversals      int
	uniqueAxisCount      int
	conflictingAxisCount int
	reusedTraversals     int
	maxAxisReuseCount    int
}

func evaluateAxisUsage(points [][]float64) axisUsageSummary {
	traversals := extractAxisTraversals(points)
	if len(traversals) == 0 {
		return axisUsageSummary{}
	}

	axisCounts := make(map[string]int, len(traversals))
	axisDirections := make(map[string]uint8, len(traversals))
	maxReuse := 0

	for _, traversal := range traversals {
		axisCounts[traversal.axisID]++
		if axisCounts[traversal.axisID] > maxReuse {
			maxReuse = axisCounts[traversal.axisID]
		}
		mask := axisDirections[traversal.axisID]
		if traversal.isForward {
			mask |= 0b01
		} else {
			mask |= 0b10
		}
		axisDirections[traversal.axisID] = mask
	}

	conflicting := 0
	reused := 0
	for axisID, count := range axisCounts {
		if axisDirections[axisID] == 0b11 {
			conflicting++
		}
		if count > 1 {
			reused += count - 1
		}
	}

	return axisUsageSummary{
		totalTraversals:      len(traversals),
		uniqueAxisCount:      len(axisCounts),
		conflictingAxisCount: conflicting,
		reusedTraversals:     reused,
		maxAxisReuseCount:    maxReuse,
	}
}

func extractAxisTraversals(points [][]float64) []axisTraversal {
	if len(points) < 3 {
		return []axisTraversal{}
	}

	traversals := make([]axisTraversal, 0, len(points)-1)
	for index := 0; index < len(points)-1; index++ {
		left := points[index]
		right := points[index+1]
		if len(left) < 2 || len(right) < 2 {
			continue
		}
		fromID := quantizedPointKey(left[0], left[1])
		toID := quantizedPointKey(right[0], right[1])
		if fromID == "" || toID == "" || fromID == toID {
			continue
		}
		traversals = append(traversals, axisTraversal{
			axisID:    canonicalEdgeKey(fromID, toID),
			isForward: fromID < toID,
		})
	}
	return traversals
}

func evaluateAxisReuseOutsideStartZone(
	points [][]float64,
	start routesDomain.Coordinates,
	startZoneMeters float64,
	minOppositeMeters float64,
) (bool, int, float64) {
	if len(points) < 2 {
		return false, 0, 0.0
	}

	type localAxisUsage struct {
		count         int
		directionMask uint8
		forwardMeters float64
		reverseMeters float64
	}

	axisUsage := make(map[string]localAxisUsage, len(points))
	maxReuseOutsideStart := 0
	outsideTotalMeters := 0.0

	for index := 0; index < len(points)-1; index++ {
		left := points[index]
		right := points[index+1]
		if len(left) < 2 || len(right) < 2 {
			continue
		}

		midLat := (left[0] + right[0]) / 2.0
		midLng := (left[1] + right[1]) / 2.0
		midDistance := haversineDistanceMeters(midLat, midLng, start.Lat, start.Lng)
		if midDistance <= startZoneMeters {
			// Reuse around start/finish hub is allowed.
			// Midpoint classification avoids exempting long segments that
			// cross the hub boundary and then retrace outside it.
			continue
		}

		fromID := quantizedPointKey(left[0], left[1])
		toID := quantizedPointKey(right[0], right[1])
		if fromID == "" || toID == "" || fromID == toID {
			continue
		}

		axisID := canonicalEdgeKey(fromID, toID)
		segmentMeters := haversineDistanceMeters(left[0], left[1], right[0], right[1])
		if segmentMeters < minAxisSegmentLengthM {
			continue
		}
		current := axisUsage[axisID]
		current.count++
		if fromID < toID {
			current.directionMask |= 0b01
			current.forwardMeters += segmentMeters
		} else {
			current.directionMask |= 0b10
			current.reverseMeters += segmentMeters
		}
		axisUsage[axisID] = current
		outsideTotalMeters += segmentMeters
		if current.count > maxReuseOutsideStart {
			maxReuseOutsideStart = current.count
		}
	}

	oppositeMeters := 0.0
	for _, usage := range axisUsage {
		if usage.directionMask == 0b11 {
			oppositeMeters += math.Min(usage.forwardMeters, usage.reverseMeters)
		}
	}
	if outsideTotalMeters <= 0 {
		return false, maxReuseOutsideStart, 0.0
	}
	oppositeRatio := oppositeMeters / outsideTotalMeters
	// Ignore tiny opposite-direction artifacts caused by local snap/geometry noise.
	minimum := math.Max(minOppositeReuseMeters, minOppositeMeters)
	return oppositeMeters >= minimum, maxReuseOutsideStart, clampUnit(oppositeRatio)
}

func (summary axisUsageSummary) oppositeTraversalRatio() float64 {
	if summary.totalTraversals == 0 {
		return 0.0
	}
	return float64(summary.conflictingAxisCount) / float64(summary.totalTraversals)
}

func (summary axisUsageSummary) reuseRatio() float64 {
	if summary.totalTraversals == 0 {
		return 0.0
	}
	return float64(summary.reusedTraversals) / float64(summary.totalTraversals)
}

func (summary axisUsageSummary) segmentDiversityRatio() float64 {
	if summary.totalTraversals == 0 {
		return 0.0
	}
	return float64(summary.uniqueAxisCount) / float64(summary.totalTraversals)
}

func (summary axisUsageSummary) maxAxisReuseRatio() float64 {
	if summary.totalTraversals == 0 {
		return 0.0
	}
	return float64(summary.maxAxisReuseCount) / float64(summary.totalTraversals)
}

func hasOppositeEdgeTraversal(points [][]float64) bool {
	return evaluateAxisUsage(points).conflictingAxisCount > 0
}

func oppositeEdgeTraversalRatio(points [][]float64) float64 {
	return evaluateAxisUsage(points).oppositeTraversalRatio()
}

func edgeReuseRatio(points [][]float64) float64 {
	return evaluateAxisUsage(points).reuseRatio()
}

func hasMinimumSegmentDiversity(points [][]float64, routeType string) bool {
	axisStats := evaluateAxisUsage(points)
	if axisStats.totalTraversals == 0 {
		return false
	}
	// Allow local loops, but reject routes that hammer the exact same axis too often.
	if axisStats.maxAxisReuseCount > 3 {
		return false
	}
	return axisStats.segmentDiversityRatio() >= minSegmentDiversityRatio(routeType)
}

func minSegmentDiversityRatio(routeType string) float64 {
	switch strings.ToUpper(strings.TrimSpace(routeType)) {
	case "MTB":
		return 0.55
	case "GRAVEL":
		return 0.54
	case "RUN":
		return 0.35
	case "TRAIL":
		return 0.46
	case "HIKE":
		return 0.40
	case "WALK":
		return 0.42
	default:
		return 0.32
	}
}

func segmentDiversityRatio(points [][]float64) float64 {
	return evaluateAxisUsage(points).segmentDiversityRatio()
}

func distanceShortfallRatio(distanceKm float64, targetKm float64) float64 {
	if targetKm <= 0 {
		return 0
	}
	shortfall := targetKm - distanceKm
	if shortfall <= 0 {
		return 0
	}
	return shortfall / math.Max(targetKm, 1.0)
}

func distanceOvershootRatio(distanceKm float64, targetKm float64) float64 {
	if targetKm <= 0 {
		return 0
	}
	overshoot := distanceKm - targetKm
	if overshoot <= 0 {
		return 0
	}
	return overshoot / math.Max(targetKm, 1.0)
}

func outsideStartAxisReuseLimit(routeType string, strict bool) int {
	_ = strict
	_ = routeType
	// P0-02 policy: outside start/finish zone, an axis cannot be reused.
	return 1
}

func allowedOppositeOutsideStartRatio(routeType string, strict bool) float64 {
	_ = strict
	_ = routeType
	// P0-02 policy: opposite-direction overlap is forbidden outside start zone.
	return 0.0
}

func minimumOppositeReuseMetersForRequest(routeType string, strict bool, distanceTargetKm float64) float64 {
	_ = strict
	base := math.Max(minOppositeReuseMeters, distanceTargetKm*6.0)
	switch strings.ToUpper(strings.TrimSpace(routeType)) {
	case "MTB", "TRAIL", "HIKE":
		return math.Max(base, 320.0)
	case "GRAVEL":
		return math.Max(base, 280.0)
	default:
		return math.Max(base, 240.0)
	}
}

func requiredPathRatioForRequest(routeType string, strict bool) float64 {
	normalized := strings.ToUpper(strings.TrimSpace(routeType))
	_ = strict
	if normalized != "GRAVEL" {
		return 0.0
	}
	// Gravel contract: keep a 25% path target; fallback to Ride handles impossible cases.
	return 0.25
}

func meetsMinimumDistance(distanceKm float64, targetKm float64) bool {
	if targetKm <= 0.0 {
		return true
	}
	// Keep a small tolerance for geometry simplification / snapping noise.
	toleranceKm := math.Max(0.25, targetKm*0.02)
	return distanceKm+toleranceKm >= targetKm
}

func fallbackRouteTypes(routeType string) []string {
	switch strings.ToUpper(strings.TrimSpace(routeType)) {
	case "MTB":
		return []string{"GRAVEL", "RIDE"}
	case "GRAVEL":
		return []string{"RIDE"}
	case "RIDE":
		return nil
	default:
		// Conservative default for unsupported types.
		return []string{"RIDE"}
	}
}

type osrmScoringProfile struct {
	distanceWeight  float64
	elevationWeight float64
	directionWeight float64
	diversityWeight float64
}

func osrmMatchScore(
	request application.RoutingEngineRequest,
	distanceKm float64,
	elevationGainM float64,
	points [][]float64,
) float64 {
	hasElevationTarget := request.ElevationTargetM != nil && *request.ElevationTargetM > 0
	hasDirection := strings.TrimSpace(request.StartDirection) != ""
	profile := buildOSRMScoringProfile(request.RouteType, hasElevationTarget, hasDirection)

	distanceComponent := distanceShortfallRatio(distanceKm, request.DistanceTargetKm) +
		distanceOvershootRatio(distanceKm, request.DistanceTargetKm)*0.15
	elevationComponent := 0.0
	if hasElevationTarget {
		elevationComponent = math.Abs(elevationGainM-*request.ElevationTargetM) / math.Max(*request.ElevationTargetM, 150.0)
	}
	directionComponent := 0.0
	if hasDirection {
		directionComponent = directionPenaltyFromPreview(points, request.StartDirection)
	}
	diversityComponent := 1.0 - segmentDiversityRatio(points)

	weighted := distanceComponent*profile.distanceWeight +
		elevationComponent*profile.elevationWeight +
		directionComponent*profile.directionWeight +
		diversityComponent*profile.diversityWeight

	return clampOSMScore(100.0 - weighted*100.0)
}

func buildOSRMScoringProfile(routeType string, hasElevationTarget bool, hasDirection bool) osrmScoringProfile {
	profile := osrmScoringProfile{
		distanceWeight:  0.70,
		elevationWeight: 0.22,
		directionWeight: 0.06,
		diversityWeight: 0.02,
	}

	switch strings.ToUpper(strings.TrimSpace(routeType)) {
	case "MTB":
		profile = osrmScoringProfile{distanceWeight: 0.36, elevationWeight: 0.29, directionWeight: 0.07, diversityWeight: 0.28}
	case "GRAVEL":
		profile = osrmScoringProfile{distanceWeight: 0.44, elevationWeight: 0.26, directionWeight: 0.06, diversityWeight: 0.24}
	case "RUN":
		profile = osrmScoringProfile{distanceWeight: 0.56, elevationWeight: 0.17, directionWeight: 0.13, diversityWeight: 0.14}
	case "TRAIL":
		profile = osrmScoringProfile{distanceWeight: 0.34, elevationWeight: 0.28, directionWeight: 0.10, diversityWeight: 0.28}
	case "HIKE":
		profile = osrmScoringProfile{distanceWeight: 0.30, elevationWeight: 0.35, directionWeight: 0.09, diversityWeight: 0.26}
	case "WALK":
		profile = osrmScoringProfile{distanceWeight: 0.33, elevationWeight: 0.28, directionWeight: 0.10, diversityWeight: 0.29}
	}

	if !hasElevationTarget {
		profile.distanceWeight += profile.elevationWeight * 0.70
		profile.diversityWeight += profile.elevationWeight * 0.30
		profile.elevationWeight = 0.0
	}
	if !hasDirection {
		profile.distanceWeight += profile.directionWeight * 0.60
		profile.diversityWeight += profile.directionWeight * 0.40
		profile.directionWeight = 0.0
	}

	return normalizeOSRMScoringProfile(profile)
}

func normalizeOSRMScoringProfile(profile osrmScoringProfile) osrmScoringProfile {
	total := profile.distanceWeight + profile.elevationWeight + profile.directionWeight + profile.diversityWeight
	if total <= 0 {
		return osrmScoringProfile{
			distanceWeight:  0.72,
			elevationWeight: 0.20,
			directionWeight: 0.04,
			diversityWeight: 0.04,
		}
	}
	return osrmScoringProfile{
		distanceWeight:  profile.distanceWeight / total,
		elevationWeight: profile.elevationWeight / total,
		directionWeight: profile.directionWeight / total,
		diversityWeight: profile.diversityWeight / total,
	}
}

func directionPenaltyFromPreview(points [][]float64, startDirection string) float64 {
	initialBearing, ok := initialBearingFromPreview(points)
	if !ok {
		return 1.0
	}
	targetBearing, ok := targetBearingFromDirection(startDirection)
	if !ok {
		return 0.0
	}
	diff := math.Abs(initialBearing - targetBearing)
	if diff > 180 {
		diff = 360 - diff
	}
	return diff / 180.0
}

func initialBearingFromPreview(points [][]float64) (float64, bool) {
	if len(points) < 2 {
		return 0, false
	}
	start := points[0]
	if len(start) < 2 {
		return 0, false
	}
	for index := 1; index < len(points); index++ {
		next := points[index]
		if len(next) < 2 {
			continue
		}
		if haversineDistanceMeters(start[0], start[1], next[0], next[1]) < 35.0 {
			continue
		}
		return osrmBearingDegrees(start[0], start[1], next[0], next[1]), true
	}
	last := points[len(points)-1]
	if len(last) < 2 {
		return 0, false
	}
	return osrmBearingDegrees(start[0], start[1], last[0], last[1]), true
}

func targetBearingFromDirection(direction string) (float64, bool) {
	switch strings.ToUpper(strings.TrimSpace(direction)) {
	case "N":
		return 0, true
	case "E":
		return 90, true
	case "S":
		return 180, true
	case "W":
		return 270, true
	default:
		return 0, false
	}
}

func osrmBearingDegrees(lat1, lng1, lat2, lng2 float64) float64 {
	lat1r := degreesToRadians(lat1)
	lat2r := degreesToRadians(lat2)
	deltaLng := degreesToRadians(lng2 - lng1)
	y := math.Sin(deltaLng) * math.Cos(lat2r)
	x := math.Cos(lat1r)*math.Sin(lat2r) - math.Sin(lat1r)*math.Cos(lat2r)*math.Cos(deltaLng)
	bearing := math.Atan2(y, x) * 180.0 / math.Pi
	if bearing < 0 {
		bearing += 360
	}
	return bearing
}

func buildRoutingHistoryBiasContext(request application.RoutingEngineRequest) routingHistoryBiasContext {
	profile := request.HistoryProfile
	if !request.HistoryBiasEnabled || profile == nil {
		return routingHistoryBiasContext{}
	}
	normalizedRouteType := normalizeHistoryRouteType(request.RouteType)
	profileRouteType := normalizeHistoryRouteType(profile.RouteType)
	if profileRouteType != normalizedRouteType {
		return routingHistoryBiasContext{}
	}
	maxAxisScore := maxPositiveMapScore(profile.AxisScores)
	maxZoneScore := maxPositiveMapScore(profile.ZoneScores)
	if maxAxisScore <= 0 && maxZoneScore <= 0 {
		return routingHistoryBiasContext{}
	}
	return routingHistoryBiasContext{
		enabled:             true,
		normalizedRouteType: normalizedRouteType,
		axisScores:          profile.AxisScores,
		zoneScores:          profile.ZoneScores,
		maxAxisScore:        maxAxisScore,
		maxZoneScore:        maxZoneScore,
	}
}

func sortAnchorsByHistoryReuse(
	anchors []routesDomain.Coordinates,
	start routesDomain.Coordinates,
	context routingHistoryBiasContext,
) []routesDomain.Coordinates {
	if !context.enabled || len(anchors) < 2 || context.maxZoneScore <= 0 {
		return anchors
	}
	type scoredAnchor struct {
		anchor routesDomain.Coordinates
		score  float64
		index  int
	}
	scoredAnchors := make([]scoredAnchor, 0, len(anchors))
	for index, anchor := range anchors {
		scoredAnchors = append(scoredAnchors, scoredAnchor{
			anchor: anchor,
			score:  historyAnchorReuseScore(anchor, start, context),
			index:  index,
		})
	}
	sort.SliceStable(scoredAnchors, func(i, j int) bool {
		left := scoredAnchors[i]
		right := scoredAnchors[j]
		if left.score != right.score {
			return left.score > right.score
		}
		return left.index < right.index
	})
	sortedAnchors := make([]routesDomain.Coordinates, 0, len(scoredAnchors))
	for _, entry := range scoredAnchors {
		sortedAnchors = append(sortedAnchors, entry.anchor)
	}
	return sortedAnchors
}

func historyAnchorReuseScore(
	anchor routesDomain.Coordinates,
	start routesDomain.Coordinates,
	context routingHistoryBiasContext,
) float64 {
	anchorZoneScore := normalizedHistoryZoneScore(anchor.Lat, anchor.Lng, context)
	midLat := (anchor.Lat + start.Lat) / 2.0
	midLng := (anchor.Lng + start.Lng) / 2.0
	midZoneScore := normalizedHistoryZoneScore(midLat, midLng, context)
	return clampUnit(anchorZoneScore*0.65 + midZoneScore*0.35)
}

func applyHistoryBiasToCandidate(
	candidate osrmRouteCandidate,
	start routesDomain.Coordinates,
	context routingHistoryBiasContext,
) osrmRouteCandidate {
	if !context.enabled {
		return candidate
	}
	corridorReuseScore := computeHistoryReuseScore(candidate.recommendation.PreviewLatLng, context)
	startZoneReuseScore := computeHistoryStartZoneReuseScore(candidate.recommendation.PreviewLatLng, start, context)
	reuseScore := clampUnit(corridorReuseScore*0.55 + startZoneReuseScore*0.45)
	candidate.historyReuseScore = reuseScore
	candidate.effectiveMatchScore = clampOSMScore(
		candidate.effectiveMatchScore +
			corridorReuseScore*historyReuseBonusWeight +
			startZoneReuseScore*historyStartZoneBonusWeight,
	)
	candidate.recommendation.Reasons = append(
		candidate.recommendation.Reasons,
		fmt.Sprintf(
			"History guidance (%s): %.0f%% corridor reuse / %.0f%% start-return reuse",
			context.normalizedRouteType,
			corridorReuseScore*100.0,
			startZoneReuseScore*100.0,
		),
	)
	return candidate
}

func computeHistoryReuseScore(points [][]float64, context routingHistoryBiasContext) float64 {
	if !context.enabled || len(points) < 2 {
		return 0.0
	}
	totalLengthM := 0.0
	axisWeighted := 0.0
	zoneWeighted := 0.0
	for index := 1; index < len(points); index++ {
		from := points[index-1]
		to := points[index]
		if len(from) < 2 || len(to) < 2 {
			continue
		}
		segmentLengthM := haversineDistanceMeters(from[0], from[1], to[0], to[1])
		if !isFinitePositive(segmentLengthM) || segmentLengthM < minHistorySegmentLengthM {
			continue
		}
		totalLengthM += segmentLengthM
		if context.maxAxisScore > 0 {
			axisID := historyAxisKey(from[0], from[1], to[0], to[1])
			axisWeighted += normalizedHistoryScore(context.axisScores[axisID], context.maxAxisScore) * segmentLengthM
		}
		if context.maxZoneScore > 0 {
			midLat := (from[0] + to[0]) / 2.0
			midLng := (from[1] + to[1]) / 2.0
			zoneID := historyZoneKey(midLat, midLng)
			zoneWeighted += normalizedHistoryScore(context.zoneScores[zoneID], context.maxZoneScore) * segmentLengthM
		}
	}
	if totalLengthM <= 0 {
		return 0.0
	}
	return blendHistoryReuseRatios(axisWeighted, zoneWeighted, totalLengthM, context)
}

func computeHistoryStartZoneReuseScore(
	points [][]float64,
	start routesDomain.Coordinates,
	context routingHistoryBiasContext,
) float64 {
	if !context.enabled || len(points) < 2 {
		return 0.0
	}
	totalLengthM := 0.0
	axisWeighted := 0.0
	zoneWeighted := 0.0
	for index := 1; index < len(points); index++ {
		from := points[index-1]
		to := points[index]
		if len(from) < 2 || len(to) < 2 {
			continue
		}
		segmentLengthM := haversineDistanceMeters(from[0], from[1], to[0], to[1])
		if !isFinitePositive(segmentLengthM) || segmentLengthM < minHistorySegmentLengthM {
			continue
		}
		midLat := (from[0] + to[0]) / 2.0
		midLng := (from[1] + to[1]) / 2.0
		if haversineDistanceMeters(midLat, midLng, start.Lat, start.Lng) > backtrackingStartZoneM {
			continue
		}
		totalLengthM += segmentLengthM
		if context.maxAxisScore > 0 {
			axisID := historyAxisKey(from[0], from[1], to[0], to[1])
			axisWeighted += normalizedHistoryScore(context.axisScores[axisID], context.maxAxisScore) * segmentLengthM
		}
		if context.maxZoneScore > 0 {
			zoneID := historyZoneKey(midLat, midLng)
			zoneWeighted += normalizedHistoryScore(context.zoneScores[zoneID], context.maxZoneScore) * segmentLengthM
		}
	}
	if totalLengthM <= 0 {
		return 0.0
	}
	return blendHistoryReuseRatios(axisWeighted, zoneWeighted, totalLengthM, context)
}

func blendHistoryReuseRatios(
	axisWeighted float64,
	zoneWeighted float64,
	totalLengthM float64,
	context routingHistoryBiasContext,
) float64 {
	hasAxisScores := context.maxAxisScore > 0 && len(context.axisScores) > 0
	hasZoneScores := context.maxZoneScore > 0 && len(context.zoneScores) > 0
	axisReuseRatio := 0.0
	zoneReuseRatio := 0.0
	if hasAxisScores {
		axisReuseRatio = axisWeighted / totalLengthM
	}
	if hasZoneScores {
		zoneReuseRatio = zoneWeighted / totalLengthM
	}
	switch {
	case hasAxisScores && hasZoneScores:
		return clampUnit(axisReuseRatio*historyAxisBiasWeight + zoneReuseRatio*historyZoneBiasWeight)
	case hasAxisScores:
		return clampUnit(axisReuseRatio)
	case hasZoneScores:
		return clampUnit(zoneReuseRatio)
	default:
		return 0.0
	}
}

func normalizedHistoryZoneScore(lat float64, lng float64, context routingHistoryBiasContext) float64 {
	if !context.enabled || context.maxZoneScore <= 0 {
		return 0.0
	}
	zoneID := historyZoneKey(lat, lng)
	return normalizedHistoryScore(context.zoneScores[zoneID], context.maxZoneScore)
}

func normalizedHistoryScore(score float64, maxScore float64) float64 {
	if !isFinitePositive(score) || !isFinitePositive(maxScore) {
		return 0.0
	}
	return clampUnit(score / maxScore)
}

func maxPositiveMapScore(scores map[string]float64) float64 {
	maxScore := 0.0
	for _, score := range scores {
		if isFinitePositive(score) && score > maxScore {
			maxScore = score
		}
	}
	return maxScore
}

func quantizedPointKey(lat float64, lng float64) string {
	return fmt.Sprintf("%.5f:%.5f", lat, lng)
}

func canonicalEdgeKey(a string, b string) string {
	if a < b {
		return a + "|" + b
	}
	return b + "|" + a
}

func haversineDistanceMeters(lat1 float64, lng1 float64, lat2 float64, lng2 float64) float64 {
	const earthRadiusMeters = 6371000.0
	dLat := degreesToRadians(lat2 - lat1)
	dLng := degreesToRadians(lng2 - lng1)
	sinLat := math.Sin(dLat / 2.0)
	sinLng := math.Sin(dLng / 2.0)
	a := sinLat*sinLat + math.Cos(degreesToRadians(lat1))*math.Cos(degreesToRadians(lat2))*sinLng*sinLng
	c := 2.0 * math.Atan2(math.Sqrt(a), math.Sqrt(1.0-a))
	return earthRadiusMeters * c
}

func readStringEnv(key string, fallback string) string {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	return raw
}

func readBoolEnv(key string, fallback bool) bool {
	raw := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if raw == "" {
		return fallback
	}
	switch raw {
	case "1", "true", "yes", "y", "on":
		return true
	case "0", "false", "no", "n", "off":
		return false
	default:
		return fallback
	}
}

func readIntEnv(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}

func (adapter *OSMRoutingAdapter) detectExtractProfile() string {
	if normalized := normalizeOSRMProfile(strings.TrimSpace(adapter.extractProfileEnv)); normalized != "" {
		return normalized
	}
	for _, candidatePath := range adapter.profileMarkerCandidatePaths() {
		if normalized := normalizeOSRMProfile(readFirstLine(candidatePath)); normalized != "" {
			return normalized
		}
	}
	if normalized := normalizeOSRMProfile(strings.TrimSpace(adapter.profileOverride)); normalized != "" {
		return normalized
	}
	return "unknown"
}

func (adapter *OSMRoutingAdapter) profileMarkerCandidatePaths() []string {
	rawCandidates := []string{
		strings.TrimSpace(adapter.extractProfileCfgFile),
		defaultOSRMProfileFilePath,
		fallbackOSRMProfilePath,
	}
	seen := map[string]struct{}{}
	candidates := make([]string, 0, len(rawCandidates))
	for _, rawPath := range rawCandidates {
		cleanPath := strings.TrimSpace(rawPath)
		if cleanPath == "" {
			continue
		}
		if _, alreadyExists := seen[cleanPath]; alreadyExists {
			continue
		}
		seen[cleanPath] = struct{}{}
		candidates = append(candidates, cleanPath)
	}
	return candidates
}

func (adapter *OSMRoutingAdapter) effectiveRoutingProfile(extractProfile string) string {
	if normalized := normalizeOSRMProfile(strings.TrimSpace(adapter.profileOverride)); normalized != "" && normalized != "unknown" {
		switch normalized {
		case "/opt/bicycle.lua":
			return "cycling"
		case "/opt/foot.lua":
			return "walking"
		case "/opt/car.lua":
			return "driving"
		}
	}
	switch extractProfile {
	case "/opt/bicycle.lua":
		return "cycling"
	case "/opt/foot.lua":
		return "walking"
	case "/opt/car.lua":
		return "driving"
	default:
		// Conservative default for this product: cycling is the primary OSRM mode.
		return "cycling"
	}
}

func supportedRouteTypesByProfile(extractProfile string, effectiveProfile string) []string {
	switch strings.TrimSpace(strings.ToLower(effectiveProfile)) {
	case "cycling":
		return []string{"RIDE", "MTB", "GRAVEL"}
	case "walking":
		return []string{"RUN", "TRAIL", "HIKE"}
	case "driving":
		return []string{"RIDE"}
	default:
		return supportedRouteTypesByExtractProfile(extractProfile)
	}
}

func normalizeOSRMProfile(raw string) string {
	normalized := strings.TrimSpace(strings.ToLower(raw))
	switch {
	case normalized == "":
		return ""
	case strings.Contains(normalized, "bicycle.lua"), normalized == "cycling":
		return "/opt/bicycle.lua"
	case strings.Contains(normalized, "foot.lua"), normalized == "walking":
		return "/opt/foot.lua"
	case strings.Contains(normalized, "car.lua"), normalized == "driving":
		return "/opt/car.lua"
	default:
		return "unknown"
	}
}

func supportedRouteTypesByExtractProfile(extractProfile string) []string {
	switch extractProfile {
	case "/opt/bicycle.lua":
		return []string{"RIDE", "MTB", "GRAVEL"}
	case "/opt/foot.lua":
		return []string{"RUN", "TRAIL", "HIKE"}
	case "/opt/car.lua":
		return []string{"RIDE"}
	default:
		return []string{"RIDE", "MTB", "GRAVEL", "RUN", "TRAIL", "HIKE"}
	}
}

func readFirstLine(path string) string {
	cleanPath := strings.TrimSpace(path)
	if cleanPath == "" {
		return ""
	}
	data, err := os.ReadFile(cleanPath)
	if err != nil {
		return ""
	}
	lines := strings.Split(string(data), "\n")
	if len(lines) == 0 {
		return ""
	}
	return strings.TrimSpace(lines[0])
}
