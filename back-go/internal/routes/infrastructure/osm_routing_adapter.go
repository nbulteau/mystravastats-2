package infrastructure

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"math"
	"mystravastats/internal/routes/application"
	routesDomain "mystravastats/internal/routes/domain"
	"mystravastats/internal/shared/domain/business"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultOSMRoutingBaseURL   = "http://localhost:5000"
	defaultOSMRoutingTimeoutMs = 3000
	maxOSRMRoutingCalls        = 16
	startSnapToleranceMeters   = 500.0
	directionToleranceMeters   = 120.0
)

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
	baseBearing := startDirectionToBearing(request.StartDirection)
	radiusBaseKm := math.Max(1.0, request.DistanceTargetKm/(2.0*math.Pi))
	radiusMultipliers := []float64{1.00, 0.90, 1.10, 0.80, 1.20, 1.30, 0.70, 1.40}
	rotations := []float64{0, 18, -18, 35, -35, 52, -52, 70, -70}
	maxCalls := minInt(maxOSRMRoutingCalls, request.Limit*2+2)

	recommendations := make([]routesDomain.RouteRecommendation, 0, request.Limit)
	seenSignatures := make(map[string]struct{}, request.Limit*2)
	generatedCount := 0

	for callIndex := 0; callIndex < maxCalls && len(recommendations) < request.Limit; callIndex++ {
		radiusKm := radiusBaseKm * radiusMultipliers[callIndex%len(radiusMultipliers)]
		rotation := rotations[callIndex%len(rotations)]
		waypoints := adapter.syntheticLoopWaypoints(request.StartPoint, radiusKm, baseBearing+rotation)
		routes, err := adapter.fetchOSRMRoutes(profile, waypoints)
		if err != nil {
			// Do not fail the whole request: caller will fallback to in-cache generation.
			continue
		}
		for routeIndex, osrmRoute := range routes {
			recommendation, ok := adapter.toRouteRecommendation(request, osrmRoute, generatedCount+routeIndex)
			if !ok {
				continue
			}
			signature := routeGeometrySignature(recommendation.PreviewLatLng)
			if signature == "" {
				continue
			}
			if _, exists := seenSignatures[signature]; exists {
				continue
			}
			seenSignatures[signature] = struct{}{}
			recommendations = append(recommendations, recommendation)
			if len(recommendations) >= request.Limit {
				break
			}
		}
		generatedCount += len(routes)
	}

	return recommendations, nil
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

func (adapter *OSMRoutingAdapter) syntheticLoopWaypoints(
	start routesDomain.Coordinates,
	radiusKm float64,
	initialBearing float64,
) []routesDomain.Coordinates {
	bearing1 := normalizeBearing(initialBearing)
	bearing2 := normalizeBearing(initialBearing + 120.0)
	bearing3 := normalizeBearing(initialBearing + 240.0)

	point1 := destinationFromBearing(start, radiusKm, bearing1)
	point2 := destinationFromBearing(start, radiusKm*1.05, bearing2)
	point3 := destinationFromBearing(start, radiusKm*0.95, bearing3)

	return []routesDomain.Coordinates{
		start,
		point1,
		point2,
		point3,
		start,
	}
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
		"%s/route/v1/%s/%s?alternatives=true&steps=false&overview=full&geometries=geojson&continue_straight=false",
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

func (adapter *OSMRoutingAdapter) toRouteRecommendation(
	request application.RoutingEngineRequest,
	route osrmRoute,
	index int,
) (routesDomain.RouteRecommendation, bool) {
	if route.Distance <= 0 || len(route.Geometry.Coordinates) < 2 {
		return routesDomain.RouteRecommendation{}, false
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
		return routesDomain.RouteRecommendation{}, false
	}
	if !startsNearRequestedStart(points, request.StartPoint, startSnapToleranceMeters) {
		return routesDomain.RouteRecommendation{}, false
	}
	if !respectsHalfPlaneDirection(points, request.StartPoint, request.StartDirection, directionToleranceMeters) {
		return routesDomain.RouteRecommendation{}, false
	}
	if hasOppositeEdgeTraversal(points) {
		return routesDomain.RouteRecommendation{}, false
	}

	start := &routesDomain.Coordinates{Lat: points[0][0], Lng: points[0][1]}
	end := &routesDomain.Coordinates{Lat: points[len(points)-1][0], Lng: points[len(points)-1][1]}

	distanceKm := route.Distance / 1000.0
	durationSec := int(math.Round(route.Duration))
	if durationSec <= 0 {
		durationSec = int(math.Round(distanceKm * 180.0))
	}

	var elevationGainM float64
	if request.ElevationTargetM != nil && *request.ElevationTargetM > 0 {
		deltaRatio := math.Abs(distanceKm-request.DistanceTargetKm) / math.Max(1.0, request.DistanceTargetKm)
		elevationGainM = math.Max(0.0, *request.ElevationTargetM*(1.0-deltaRatio*0.5))
	} else {
		elevationGainM = math.Max(0.0, distanceKm*8.0)
	}

	matchScore := clampOSMScore(100.0 - (math.Abs(distanceKm-request.DistanceTargetKm)/math.Max(1.0, request.DistanceTargetKm))*100.0)
	routeID := generatedOSMRouteID(points, request.StartPoint, index)
	activityType := activityTypeFromRouteType(request.RouteType)
	title := fmt.Sprintf("Generated loop near %.4f, %.4f", request.StartPoint.Lat, request.StartPoint.Lng)
	if index > 0 {
		title = fmt.Sprintf("%s #%d", title, index+1)
	}

	reasons := []string{
		"Generated with OSM road graph (OSRM)",
		fmt.Sprintf("Distance delta: %s", formatDistanceDelta(distanceKm-request.DistanceTargetKm)),
	}
	if request.ElevationTargetM != nil {
		reasons = append(reasons, fmt.Sprintf("Elevation estimate: %s", formatElevationDelta(elevationGainM-*request.ElevationTargetM)))
	}
	if request.StartDirection != "" {
		reasons = append(reasons, fmt.Sprintf("Departure direction: %s", startDirectionLabel(request.StartDirection)))
	}

	return routesDomain.RouteRecommendation{
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
	}, true
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

func hasOppositeEdgeTraversal(points [][]float64) bool {
	if len(points) < 3 {
		return false
	}

	type edgeDirection struct {
		hasForward bool
		hasReverse bool
	}
	seen := make(map[string]edgeDirection, len(points))

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
		edgeKey := canonicalEdgeKey(fromID, toID)
		entry := seen[edgeKey]
		if fromID < toID {
			if entry.hasReverse {
				return true
			}
			entry.hasForward = true
		} else {
			if entry.hasForward {
				return true
			}
			entry.hasReverse = true
		}
		seen[edgeKey] = entry
	}

	return false
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
