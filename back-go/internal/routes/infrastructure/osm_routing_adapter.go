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
	maxOSRMRoutingCalls         = 16
	startSnapToleranceMeters    = 900.0
	directionToleranceMeters    = 120.0
	backtrackingProfileBalanced = "BALANCED"
	backtrackingProfileStrict   = "STRICT"
	backtrackingProfileUltra    = "ULTRA"
)

type osrmRouteCandidate struct {
	recommendation      routesDomain.RouteRecommendation
	directionPenalty    float64
	backtrackingRatio   float64
	corridorOverlap     float64
	edgeReuseRatio      float64
	segmentDiversity    float64
	distanceDeltaRatio  float64
	effectiveMatchScore float64
}

type routeRelaxationLevel struct {
	name                  string
	maxDirectionPenalty   float64
	maxBacktrackingRatio  float64
	maxCorridorOverlap    float64
	maxEdgeReuseRatio     float64
	minSegmentDiversity   float64
	maxDistanceDeltaRatio float64
}

type osrmRouteResponse struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Routes  []osrmRoute `json:"routes"`
}

type osrmRoute struct {
	Distance float64      `json:"distance"`
	Duration float64      `json:"duration"`
	Geometry osrmGeometry `json:"geometry"`
}

type osrmGeometry struct {
	Type        string      `json:"type"`
	Coordinates [][]float64 `json:"coordinates"`
}

// OSMRoutingAdapter integrates a local OSRM endpoint as a routing engine.
type OSMRoutingAdapter struct {
	enabled         bool
	debug           bool
	baseURL         string
	timeout         time.Duration
	client          *http.Client
	profileOverride string
}

func NewOSMRoutingAdapter() *OSMRoutingAdapter {
	enabled := readBoolEnv("OSM_ROUTING_ENABLED", true)
	baseURL := strings.TrimRight(strings.TrimSpace(readStringEnv("OSM_ROUTING_BASE_URL", defaultOSMRoutingBaseURL)), "/")
	timeoutMs := readIntEnv("OSM_ROUTING_TIMEOUT_MS", defaultOSMRoutingTimeoutMs)
	if timeoutMs < 200 {
		timeoutMs = defaultOSMRoutingTimeoutMs
	}
	profileOverride := strings.TrimSpace(readStringEnv("OSM_ROUTING_PROFILE", ""))

	return &OSMRoutingAdapter{
		enabled:         enabled,
		debug:           readBoolEnv("OSM_ROUTING_DEBUG", false),
		baseURL:         baseURL,
		timeout:         time.Duration(timeoutMs) * time.Millisecond,
		client:          &http.Client{Timeout: time.Duration(timeoutMs) * time.Millisecond},
		profileOverride: profileOverride,
	}
}

func (adapter *OSMRoutingAdapter) HealthDetails() map[string]any {
	details := map[string]any{
		"engine":  "osrm",
		"enabled": adapter.enabled,
		"debug":   adapter.debug,
		"baseUrl": adapter.baseURL,
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
	details["profile"] = adapter.profileOverride
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
	if isCustomTargetMode(request) {
		return adapter.generateCustomWaypointLoops(request, profile), nil
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
				candidate.edgeReuseRatio*40.0,
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
				candidate.edgeReuseRatio*140.0,
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

func normalizeBacktrackingProfile(profile string, strictBacktracking bool) string {
	normalized := strings.ToUpper(strings.TrimSpace(profile))
	switch normalized {
	case backtrackingProfileBalanced, backtrackingProfileStrict, backtrackingProfileUltra:
		return normalized
	}
	if strictBacktracking {
		return backtrackingProfileStrict
	}
	return backtrackingProfileBalanced
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
			bearingOffsets: []float64{0, 28, -28, 56, -56},
			radiusScales:   []float64{1.18, 1.06, 1.06, 0.90, 0.90},
		},
		{
			bearingOffsets: []float64{12, -12, 40, -40, 70, -70},
			radiusScales:   []float64{1.20, 1.20, 1.00, 1.00, 0.82, 0.82},
		},
		{
			bearingOffsets: []float64{0, 22, -22, 48, -48, 78, -78},
			radiusScales:   []float64{1.14, 1.12, 1.12, 0.98, 0.98, 0.78, 0.78},
		},
		{
			bearingOffsets: []float64{6, -6, 34, -34, 62, -62},
			radiusScales:   []float64{1.24, 1.24, 1.05, 1.05, 0.86, 0.86},
		},
	}
	hasDirection := strings.TrimSpace(startDirection) != ""
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
		"%s/route/v1/%s/%s?alternatives=true&steps=false&overview=full&geometries=geojson&continue_straight=true",
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
	if len(points) < 2 {
		incrementRejectCount(rejectCounts, "INVALID_COORDINATES")
		return osrmRouteCandidate{}, false
	}
	if !startsNearRequestedStart(points, request.StartPoint, startSnapToleranceMeters) {
		incrementRejectCount(rejectCounts, "START_TOO_FAR")
		return osrmRouteCandidate{}, false
	}

	start := &routesDomain.Coordinates{Lat: points[0][0], Lng: points[0][1]}
	end := &routesDomain.Coordinates{Lat: points[len(points)-1][0], Lng: points[len(points)-1][1]}

	distanceKm := route.Distance / 1000.0
	durationSec := int(math.Round(route.Duration))
	if durationSec <= 0 {
		durationSec = int(math.Round(distanceKm * 180.0))
	}

	directionPenalty := combinedDirectionPenalty(points, request.StartPoint, request.StartDirection, directionToleranceMeters)
	backtrackingRatio := oppositeEdgeTraversalRatio(points)
	corridorOverlap := corridorOverlapRatio(points)
	edgeReuse := edgeReuseRatio(points)
	diversityRatio := segmentDiversityRatio(points)
	distanceDeltaRatio := math.Abs(distanceKm-request.DistanceTargetKm) / math.Max(1.0, request.DistanceTargetKm)

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
		fmt.Sprintf("Distance delta: %s", formatDistanceDelta(distanceKm-request.DistanceTargetKm)),
		fmt.Sprintf("Segment diversity: %.0f%% unique edges", diversityRatio*100.0),
		fmt.Sprintf("Directional alignment: %.0f%%", (1.0-directionPenalty)*100.0),
		fmt.Sprintf("Backtracking: %.0f%%", backtrackingRatio*100.0),
		fmt.Sprintf("Corridor overlap: %.0f%%", corridorOverlap*100.0),
		fmt.Sprintf("Axis retrace: %.0f%%", edgeReuse*100.0),
	}
	if request.ElevationTargetM != nil {
		reasons = append(reasons, fmt.Sprintf("Elevation estimate: %s", formatElevationDelta(elevationGainM-*request.ElevationTargetM)))
	}
	if request.StartDirection != "" {
		reasons = append(reasons, fmt.Sprintf("Direction: %s", startDirectionLabel(request.StartDirection)))
	}
	backtrackingProfile := normalizeBacktrackingProfile(request.BacktrackingProfile, request.StrictBacktracking)
	if backtrackingProfile == backtrackingProfileStrict {
		reasons = append(reasons, "Anti-backtracking: strict")
	}
	if backtrackingProfile == backtrackingProfileUltra {
		reasons = append(reasons, "Anti-backtracking: ultra")
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
		directionPenalty*22.0 -
		backtrackingRatio*70.0 -
		corridorOverlap*110.0 -
		edgeReuse*120.0 -
		math.Max(0.0, minSegmentDiversityRatio(request.RouteType)-diversityRatio)*35.0 -
		math.Max(0.0, distanceDeltaRatio-0.15)*45.0)
	// effectiveScore is an internal ranking score (not API score):
	// it aggressively penalizes backtracking and bad directional fit to keep
	// generated loops practical even in relaxed levels.

	return osrmRouteCandidate{
		recommendation:      recommendation,
		directionPenalty:    directionPenalty,
		backtrackingRatio:   backtrackingRatio,
		corridorOverlap:     corridorOverlap,
		edgeReuseRatio:      edgeReuse,
		segmentDiversity:    diversityRatio,
		distanceDeltaRatio:  distanceDeltaRatio,
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
		if left.effectiveMatchScore != right.effectiveMatchScore {
			return left.effectiveMatchScore > right.effectiveMatchScore
		}
		if left.directionPenalty != right.directionPenalty {
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
	backtrackingProfile := normalizeBacktrackingProfile(request.BacktrackingProfile, request.StrictBacktracking)
	levels := buildRouteRelaxationLevels(
		request.RouteType,
		strings.TrimSpace(request.StartDirection) != "",
		request.DirectionStrict,
		backtrackingProfile,
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
	if len(selected) < limit {
		softMaxBacktracking := 0.32
		softMaxCorridor := 0.40
		softMaxEdgeReuse := 0.24
		if backtrackingProfile == backtrackingProfileStrict {
			softMaxBacktracking = 0.20
			softMaxCorridor = 0.12
			softMaxEdgeReuse = 0.12
		}
		if backtrackingProfile == backtrackingProfileUltra {
			softMaxBacktracking = 0.12
			softMaxCorridor = 0.08
			softMaxEdgeReuse = 0.08
		}
		selected = appendBestEffortCandidates(
			sortedCandidates,
			selected,
			selectedIDs,
			limit,
			softMaxBacktracking,
			softMaxCorridor,
			softMaxEdgeReuse,
			"best-effort-soft",
		)
	}
	if len(selected) < limit && backtrackingProfile == backtrackingProfileBalanced {
		selected = appendBestEffortCandidates(
			sortedCandidates,
			selected,
			selectedIDs,
			limit,
			1.0,
			1.0,
			1.0,
			"best-effort-hard",
		)
	}

	return selected
}

func appendBestEffortCandidates(
	sortedCandidates []osrmRouteCandidate,
	selected []routesDomain.RouteRecommendation,
	selectedIDs map[string]struct{},
	limit int,
	maxBacktrackingRatio float64,
	maxCorridorOverlap float64,
	maxEdgeReuseRatio float64,
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
		if candidate.backtrackingRatio > maxBacktrackingRatio {
			continue
		}
		if candidate.corridorOverlap > maxCorridorOverlap {
			continue
		}
		if candidate.edgeReuseRatio > maxEdgeReuseRatio {
			continue
		}
		recommendation := candidate.recommendation
		recommendation.Reasons = append(recommendation.Reasons, fmt.Sprintf("Selection profile: %s", profileName))
		selected = append(selected, recommendation)
		selectedIDs[routeID] = struct{}{}
	}
	return selected
}

func buildRouteRelaxationLevels(routeType string, hasDirection bool, directionStrict bool, backtrackingProfile string) []routeRelaxationLevel {
	baseMinDiversity := minSegmentDiversityRatio(routeType)
	strictDirection := 1.0
	balancedDirection := 1.0
	relaxedDirection := 1.0
	fallbackDirection := 1.0
	if hasDirection {
		strictDirection = 0.18
		balancedDirection = 0.28
		relaxedDirection = 0.40
		fallbackDirection = 0.52
		if directionStrict {
			strictDirection = 0.10
			balancedDirection = 0.16
			relaxedDirection = 0.22
			fallbackDirection = 0.30
		}
	}
	profile := normalizeBacktrackingProfile(backtrackingProfile, false)
	strictBacktrackingRatio := 0.0025
	balancedBacktrackingRatio := 0.010
	relaxedBacktrackingRatio := 0.022
	fallbackBacktrackingRatio := 0.055
	strictCorridorOverlap := 0.007
	balancedCorridorOverlap := 0.014
	relaxedCorridorOverlap := 0.022
	fallbackCorridorOverlap := 0.030
	strictEdgeReuseRatio := 0.018
	balancedEdgeReuseRatio := 0.060
	relaxedEdgeReuseRatio := 0.120
	fallbackEdgeReuseRatio := 0.180
	if profile == backtrackingProfileStrict {
		baseMinDiversity = math.Min(0.92, baseMinDiversity+0.08)
		strictBacktrackingRatio = 0.0015
		balancedBacktrackingRatio = 0.006
		relaxedBacktrackingRatio = 0.015
		fallbackBacktrackingRatio = 0.035
		strictCorridorOverlap = 0.006
		balancedCorridorOverlap = 0.012
		relaxedCorridorOverlap = 0.020
		fallbackCorridorOverlap = 0.028
		strictEdgeReuseRatio = 0.015
		balancedEdgeReuseRatio = 0.040
		relaxedEdgeReuseRatio = 0.080
		fallbackEdgeReuseRatio = 0.120
	}
	if profile == backtrackingProfileUltra {
		baseMinDiversity = math.Min(0.95, baseMinDiversity+0.12)
		strictBacktrackingRatio = 0.0012
		balancedBacktrackingRatio = 0.0045
		relaxedBacktrackingRatio = 0.010
		fallbackBacktrackingRatio = 0.024
		strictCorridorOverlap = 0.005
		balancedCorridorOverlap = 0.010
		relaxedCorridorOverlap = 0.017
		fallbackCorridorOverlap = 0.024
		strictEdgeReuseRatio = 0.012
		balancedEdgeReuseRatio = 0.030
		relaxedEdgeReuseRatio = 0.060
		fallbackEdgeReuseRatio = 0.090
	}

	return []routeRelaxationLevel{
		{
			name:                  "strict",
			maxDirectionPenalty:   strictDirection,
			maxBacktrackingRatio:  strictBacktrackingRatio,
			maxCorridorOverlap:    strictCorridorOverlap,
			maxEdgeReuseRatio:     strictEdgeReuseRatio,
			minSegmentDiversity:   baseMinDiversity,
			maxDistanceDeltaRatio: 0.35,
		},
		{
			name:                  "balanced",
			maxDirectionPenalty:   balancedDirection,
			maxBacktrackingRatio:  balancedBacktrackingRatio,
			maxCorridorOverlap:    balancedCorridorOverlap,
			maxEdgeReuseRatio:     balancedEdgeReuseRatio,
			minSegmentDiversity:   math.Max(0.22, baseMinDiversity-0.08),
			maxDistanceDeltaRatio: 0.60,
		},
		{
			name:                  "relaxed",
			maxDirectionPenalty:   relaxedDirection,
			maxBacktrackingRatio:  relaxedBacktrackingRatio,
			maxCorridorOverlap:    relaxedCorridorOverlap,
			maxEdgeReuseRatio:     relaxedEdgeReuseRatio,
			minSegmentDiversity:   math.Max(0.12, baseMinDiversity-0.18),
			maxDistanceDeltaRatio: 1.00,
		},
		{
			name:                  "fallback",
			maxDirectionPenalty:   fallbackDirection,
			maxBacktrackingRatio:  fallbackBacktrackingRatio,
			maxCorridorOverlap:    fallbackCorridorOverlap,
			maxEdgeReuseRatio:     fallbackEdgeReuseRatio,
			minSegmentDiversity:   0.08,
			maxDistanceDeltaRatio: 2.20,
		},
	}
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
	return math.Max(math.Max(bearingPenalty*0.65, halfPlanePenalty), lobePenalty)
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
		dominancePenalty = clampUnit((0.63 - dominanceRatio) / 0.63)
	}

	// Average projection guard: route center of mass should not drift opposite.
	avgPenalty := 0.0
	if desiredExtent > 1.0 {
		avgProjection := sumProjection / float64(projectionCount)
		avgPenalty = clampUnit((-avgProjection) / math.Max(desiredExtent*0.25, 1.0))
	}

	return math.Max(dominancePenalty, avgPenalty)
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

func hasOppositeEdgeTraversal(points [][]float64) bool {
	return oppositeEdgeTraversalRatio(points) > 0.0
}

func oppositeEdgeTraversalRatio(points [][]float64) float64 {
	if len(points) < 3 {
		return 0.0
	}

	type edgeDirection struct {
		hasForward bool
		hasReverse bool
	}
	seen := make(map[string]edgeDirection, len(points))
	totalEdges := 0

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
		totalEdges++
		edgeKey := canonicalEdgeKey(fromID, toID)
		entry := seen[edgeKey]
		if fromID < toID {
			entry.hasForward = true
		} else {
			entry.hasReverse = true
		}
		seen[edgeKey] = entry
	}
	if totalEdges == 0 {
		return 0.0
	}
	conflictingEdges := 0
	for _, entry := range seen {
		if entry.hasForward && entry.hasReverse {
			conflictingEdges++
		}
	}

	return float64(conflictingEdges) / float64(totalEdges)
}

func edgeReuseRatio(points [][]float64) float64 {
	if len(points) < 3 {
		return 0.0
	}
	usage := make(map[string]int, len(points))
	totalEdges := 0
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
		totalEdges++
		usage[canonicalEdgeKey(fromID, toID)]++
	}
	if totalEdges == 0 {
		return 0.0
	}
	reusedEdges := 0
	for _, count := range usage {
		if count > 1 {
			reusedEdges += count - 1
		}
	}
	return float64(reusedEdges) / float64(totalEdges)
}

func hasMinimumSegmentDiversity(points [][]float64, routeType string) bool {
	if len(points) < 3 {
		return false
	}

	maxEdgeReuse := 3
	segmentUsage := make(map[string]int, len(points))
	totalEdges := 0
	uniqueEdges := 0

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

		totalEdges++
		segmentKey := canonicalEdgeKey(fromID, toID)
		segmentUsage[segmentKey]++
		if segmentUsage[segmentKey] == 1 {
			uniqueEdges++
		}
		if segmentUsage[segmentKey] > maxEdgeReuse {
			return false
		}
	}

	if totalEdges == 0 {
		return false
	}
	return float64(uniqueEdges)/float64(totalEdges) >= minSegmentDiversityRatio(routeType)
}

func minSegmentDiversityRatio(routeType string) float64 {
	switch strings.ToUpper(strings.TrimSpace(routeType)) {
	case "MTB":
		return 0.40
	case "GRAVEL":
		return 0.42
	case "RUN":
		return 0.35
	case "TRAIL":
		return 0.30
	case "HIKE":
		return 0.28
	default:
		return 0.45
	}
}

func segmentDiversityRatio(points [][]float64) float64 {
	if len(points) < 2 {
		return 0
	}
	totalEdges := 0
	uniqueEdges := make(map[string]struct{}, len(points))
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
		totalEdges++
		uniqueEdges[canonicalEdgeKey(fromID, toID)] = struct{}{}
	}
	if totalEdges == 0 {
		return 0
	}
	return float64(len(uniqueEdges)) / float64(totalEdges)
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

	distanceComponent := math.Abs(distanceKm-request.DistanceTargetKm) / math.Max(request.DistanceTargetKm, 1.0)
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
		distanceWeight:  0.58,
		elevationWeight: 0.30,
		directionWeight: 0.08,
		diversityWeight: 0.04,
	}

	switch strings.ToUpper(strings.TrimSpace(routeType)) {
	case "MTB":
		profile = osrmScoringProfile{distanceWeight: 0.48, elevationWeight: 0.38, directionWeight: 0.09, diversityWeight: 0.05}
	case "GRAVEL":
		profile = osrmScoringProfile{distanceWeight: 0.54, elevationWeight: 0.33, directionWeight: 0.08, diversityWeight: 0.05}
	case "RUN":
		profile = osrmScoringProfile{distanceWeight: 0.55, elevationWeight: 0.20, directionWeight: 0.15, diversityWeight: 0.10}
	case "TRAIL":
		profile = osrmScoringProfile{distanceWeight: 0.42, elevationWeight: 0.33, directionWeight: 0.15, diversityWeight: 0.10}
	case "HIKE":
		profile = osrmScoringProfile{distanceWeight: 0.34, elevationWeight: 0.41, directionWeight: 0.15, diversityWeight: 0.10}
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
