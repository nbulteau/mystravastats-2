package api

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"math"
	"mystravastats/api/dto"
	routesDomain "mystravastats/internal/routes/domain"
	"mystravastats/internal/shared/domain/business"
	"regexp"
	"strconv"
	"strings"
)

// Route payload types

type routeStartPointPayload struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

type generateShapeRoutesPayload struct {
	ShapeInputType string                  `json:"shapeInputType"`
	ShapeData      string                  `json:"shapeData"`
	StartPoint     *routeStartPointPayload `json:"startPoint,omitempty"`
	RouteType      string                  `json:"routeType"`
	VariantCount   *int                    `json:"variantCount,omitempty"`
}

// Payload validation

func validateGenerateShapeRoutesPayload(payload generateShapeRoutesPayload) error {
	inputType := strings.ToLower(strings.TrimSpace(payload.ShapeInputType))
	if inputType == "" {
		return fmt.Errorf("shapeInputType is required")
	}
	switch inputType {
	case "draw", "gpx", "svg", "polyline":
	default:
		return fmt.Errorf("shapeInputType must be one of draw/gpx/svg/polyline")
	}
	if strings.TrimSpace(payload.ShapeData) == "" {
		return fmt.Errorf("shapeData is required")
	}
	if payload.StartPoint != nil && !isValidLatLng(payload.StartPoint.Lat, payload.StartPoint.Lng) {
		return fmt.Errorf("startPoint has invalid coordinates")
	}
	if payload.VariantCount != nil && (*payload.VariantCount < 1 || *payload.VariantCount > maxGeneratedVariantCount) {
		return fmt.Errorf("variantCount must be between 1 and %d", maxGeneratedVariantCount)
	}
	return nil
}

// Normalize helpers

func normalizeGenerateRouteType(value string) string {
	normalized := strings.ToUpper(strings.TrimSpace(value))
	switch normalized {
	case "RIDE", "MTB", "GRAVEL", "RUN", "TRAIL", "HIKE":
		return normalized
	default:
		return "RIDE"
	}
}

func normalizeGenerateVariantCount(value *int) int {
	if value == nil {
		return defaultRoutesVariantCount
	}
	if *value < 1 {
		return 1
	}
	if *value > maxGeneratedVariantCount {
		return maxGeneratedVariantCount
	}
	return *value
}

// Activity type defaults for route generation

func defaultRouteGenerationActivityTypes() []business.ActivityType {
	return []business.ActivityType{
		business.Ride,
		business.GravelRide,
		business.MountainBikeRide,
		business.Commute,
		business.VirtualRide,
		business.Run,
		business.TrailRun,
		business.Hike,
		business.Walk,
	}
}

// Route response builders

func buildShapeGeneratedRoutesResponse(
	result routesDomain.RouteExplorerResult,
	routeType string,
	shapeInputType string,
	shapeFilter string,
	requestID string,
	limit int,
) dto.GenerateRoutesResponseDto {
	routes := make([]dto.GeneratedRouteDto, 0, limit)
	seen := make(map[string]struct{}, limit)

	appendRoute := func(recommendation routesDomain.RouteRecommendation) {
		if len(routes) >= limit {
			return
		}
		if _, exists := seen[recommendation.RouteID]; exists {
			return
		}
		score := buildGeneratedRouteScore(recommendation)
		converted := dto.ToGeneratedRouteDto(recommendation, score, routeType)
		routes = append(routes, converted)
		seen[recommendation.RouteID] = struct{}{}
	}

	for _, recommendation := range result.ShapeMatches {
		if !isShapeGeneratedRouteCandidate(recommendation) {
			continue
		}
		appendRoute(recommendation)
	}
	for _, recommendation := range result.RoadGraphLoops {
		if !isShapeGeneratedRouteCandidate(recommendation) {
			continue
		}
		appendRoute(recommendation)
	}

	diagnostics := buildShapeGenerationDiagnostics(
		routes,
		routeType,
		shapeInputType,
		shapeFilter,
		requestID,
		countIgnoredShapeGenerationCandidates(result),
	)
	return dto.GenerateRoutesResponseDto{
		Routes:      routes,
		Diagnostics: diagnostics,
	}
}

func isShapeGeneratedRouteCandidate(recommendation routesDomain.RouteRecommendation) bool {
	switch recommendation.VariantType {
	case routesDomain.RouteVariantShape, routesDomain.RouteVariantRoadGraph:
	default:
		return false
	}
	for _, reason := range recommendation.Reasons {
		if strings.HasPrefix(strings.TrimSpace(reason), "Shape mode:") {
			return true
		}
	}
	return false
}

func countIgnoredShapeGenerationCandidates(result routesDomain.RouteExplorerResult) int {
	ignored := len(result.ClosestLoops) + len(result.ShapeRemixes)
	for _, recommendation := range result.ShapeMatches {
		if !isShapeGeneratedRouteCandidate(recommendation) {
			ignored++
		}
	}
	for _, recommendation := range result.RoadGraphLoops {
		if !isShapeGeneratedRouteCandidate(recommendation) {
			ignored++
		}
	}
	return ignored
}

func buildShapeGenerationDiagnostics(
	routes []dto.GeneratedRouteDto,
	routeType string,
	shapeInputType string,
	shapeFilter string,
	requestID string,
	ignoredCandidateCount int,
) []dto.RouteGenerationDiagnosticDto {
	if len(routes) > 0 {
		return buildSuccessfulGenerationDiagnostics(routes)
	}

	diagnostics := []dto.RouteGenerationDiagnosticDto{
		{
			Code:    "NO_CANDIDATE",
			Message: "No route candidate matched the provided shape.",
		},
	}
	if ignoredCandidateCount > 0 {
		diagnostics = append(diagnostics, dto.RouteGenerationDiagnosticDto{
			Code:    "NON_SHAPE_CANDIDATES_IGNORED",
			Message: "Historical or non-shape route candidates were ignored because Strava Art only returns OSRM routes generated from the drawing.",
		})
	}

	shapeLabel := strings.TrimSpace(shapeFilter)
	if shapeLabel == "" {
		shapeLabel = "UNKNOWN"
	}
	targetParts := []string{
		fmt.Sprintf("%s shape=%s", normalizeGenerateRouteType(routeType), shapeLabel),
	}
	if input := strings.TrimSpace(shapeInputType); input != "" {
		targetParts = append(targetParts, fmt.Sprintf("input=%s", strings.ToLower(input)))
	}
	diagnostics = append(diagnostics, dto.RouteGenerationDiagnosticDto{
		Code: "FAILURE_SUMMARY",
		Message: fmt.Sprintf(
			"No route generated (%s). Try simplifying the shape or moving the start point. requestId=%s",
			strings.Join(targetParts, ", "),
			requestID,
		),
	})

	return diagnostics
}

func buildSuccessfulGenerationDiagnostics(routes []dto.GeneratedRouteDto) []dto.RouteGenerationDiagnosticDto {
	diagnostics := []dto.RouteGenerationDiagnosticDto{}
	seenCodes := map[string]struct{}{}
	appendOnce := func(code string, message string) {
		if _, exists := seenCodes[code]; exists {
			return
		}
		seenCodes[code] = struct{}{}
		diagnostics = append(diagnostics, dto.RouteGenerationDiagnosticDto{
			Code:    code,
			Message: message,
		})
	}

	for _, route := range routes {
		for _, reason := range route.Reasons {
			normalized := strings.TrimSpace(reason)
			switch {
			case strings.HasPrefix(normalized, "Direction relaxed:"):
				appendOnce("DIRECTION_RELAXED", "Direction constraint was relaxed to return a valid route.")
			case strings.HasPrefix(normalized, "Anti-backtracking relaxed:"):
				appendOnce("BACKTRACKING_RELAXED", "Anti-backtracking constraints were relaxed to return a valid route.")
			case strings.HasPrefix(normalized, "Route type fallback:"):
				appendOnce("ROUTE_TYPE_FALLBACK", normalized)
			case strings.HasPrefix(normalized, "Start snapped to nearest routable point"):
				appendOnce("START_POINT_SNAPPED", normalized)
			case normalized == "Generation engine fallback: legacy synthetic waypoints":
				appendOnce("ENGINE_FALLBACK_LEGACY", "Legacy waypoint generator was used as fallback.")
			case strings.HasPrefix(normalized, "Selection profile: best-effort-soft"):
				appendOnce("SELECTION_RELAXED", "Selection constraints were softened to preserve route availability.")
			case strings.HasPrefix(normalized, "Selection profile: directional-best-effort"):
				appendOnce("DIRECTION_BEST_EFFORT", "Directional constraints were softened to preserve route availability.")
			case strings.Contains(normalized, "Selection profile: emergency-fallback"):
				appendOnce("EMERGENCY_FALLBACK", "Emergency fallback selected the best available generated route.")
			case normalized == "Generation fallback: historical route cache":
				appendOnce("ENGINE_CACHE_FALLBACK", "Road-graph generation was unavailable, historical cache routes were returned.")
			}
		}
	}

	return diagnostics
}

// Scoring

func buildGeneratedRouteScore(
	recommendation routesDomain.RouteRecommendation,
) dto.RouteGenerationScoreDto {
	global := clampScore(recommendation.MatchScore)
	shape := 50.0
	if recommendation.ShapeScore != nil {
		shape = clampScore(*recommendation.ShapeScore * 100.0)
	}

	roadFitness := 70.0
	if parsedRoadFitness, ok := parseSurfaceFitnessReason(recommendation.Reasons); ok {
		roadFitness = parsedRoadFitness
	} else if recommendation.VariantType == routesDomain.RouteVariantRoadGraph {
		roadFitness = 100.0
	} else if recommendation.IsLoop {
		roadFitness = 82.0
	}

	return dto.RouteGenerationScoreDto{
		Global:      global,
		Distance:    global,
		Elevation:   global,
		Duration:    global,
		Direction:   global,
		Shape:       shape,
		RoadFitness: roadFitness,
	}
}

func parseSurfaceFitnessReason(reasons []string) (float64, bool) {
	for _, reason := range reasons {
		normalized := strings.TrimSpace(reason)
		if !strings.HasPrefix(normalized, "Surface fitness:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(normalized, "Surface fitness:"))
		payload = strings.TrimSuffix(payload, "%")
		value, err := strconv.ParseFloat(strings.TrimSpace(payload), 64)
		if err != nil {
			continue
		}
		return clampScore(value), true
	}
	return 0, false
}

func clampScore(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return math.Round(value*10.0) / 10.0
}

// GPX building

func buildRouteGPX(name string, latLng [][]float64) (string, error) {
	validPoints := make([][]float64, 0, len(latLng))
	for _, point := range latLng {
		if len(point) < 2 {
			continue
		}
		lat := point[0]
		lng := point[1]
		if lat < -90 || lat > 90 || lng < -180 || lng > 180 {
			continue
		}
		validPoints = append(validPoints, []float64{lat, lng})
	}

	if len(validPoints) < 2 {
		return "", errors.New("at least 2 valid points are required to export GPX")
	}

	safeName := strings.TrimSpace(name)
	if safeName == "" {
		safeName = "MyStravaStats route"
	}
	var escapedNameBuffer bytes.Buffer
	if err := xml.EscapeText(&escapedNameBuffer, []byte(safeName)); err != nil {
		return "", err
	}
	escapedName := escapedNameBuffer.String()

	var builder strings.Builder
	builder.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	builder.WriteString(`<gpx version="1.1" creator="MyStravaStats" xmlns="http://www.topografix.com/GPX/1/1">` + "\n")
	builder.WriteString("  <trk>\n")
	builder.WriteString("    <name>" + escapedName + "</name>\n")
	builder.WriteString("    <trkseg>\n")
	for _, point := range validPoints {
		builder.WriteString(fmt.Sprintf("      <trkpt lat=\"%.7f\" lon=\"%.7f\"></trkpt>\n", point[0], point[1]))
	}
	builder.WriteString("    </trkseg>\n")
	builder.WriteString("  </trk>\n")
	builder.WriteString("</gpx>\n")
	return builder.String(), nil
}

func sanitizeRouteFileName(input string) string {
	value := strings.ToLower(strings.TrimSpace(input))
	if value == "" {
		return ""
	}
	replacer := strings.NewReplacer(
		" ", "-", "/", "-", "\\", "-", ":", "-", ";", "-",
		",", "-", "\"", "", "'", "", "(", "", ")", "", "[", "", "]", "",
	)
	value = replacer.Replace(value)
	value = strings.Trim(value, "-._")
	return value
}

// Shape inference

func inferShapeFilter(shapeInputType string, shapeData string) string {
	switch strings.ToLower(strings.TrimSpace(shapeInputType)) {
	case "draw", "polyline", "gpx":
		points, err := parseShapePolylineCoordinates(shapeData)
		if err != nil || len(points) < 2 {
			return ""
		}
		return inferShapeFromCoordinates(points)
	default:
		return ""
	}
}

func parseShapePolylineCoordinates(raw string) ([][]float64, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, errors.New("shape data is empty")
	}
	if gpxPoints := parseShapeCoordinatesFromGPX(trimmed); len(gpxPoints) > 0 {
		return sanitizePolylineCoordinates(gpxPoints), nil
	}

	var points [][]float64
	if err := json.Unmarshal([]byte(trimmed), &points); err == nil {
		return sanitizePolylineCoordinates(points), nil
	}

	var wrapped struct {
		Points      [][]float64 `json:"points"`
		Coordinates [][]float64 `json:"coordinates"`
		LatLng      [][]float64 `json:"latLng"`
	}
	if err := json.Unmarshal([]byte(trimmed), &wrapped); err != nil {
		encoded := trimmed
		var quotedEncoded string
		if decodeErr := json.Unmarshal([]byte(trimmed), &quotedEncoded); decodeErr == nil {
			encoded = strings.TrimSpace(quotedEncoded)
		}
		decodedPoints, decodeErr := decodeEncodedPolylineCoordinates(encoded)
		if decodeErr != nil {
			return nil, errors.New("shapeData must be a JSON array of [lat,lng] coordinates or an encoded polyline string")
		}
		return sanitizePolylineCoordinates(decodedPoints), nil
	}
	switch {
	case len(wrapped.Points) > 0:
		return sanitizePolylineCoordinates(wrapped.Points), nil
	case len(wrapped.Coordinates) > 0:
		return sanitizePolylineCoordinates(wrapped.Coordinates), nil
	case len(wrapped.LatLng) > 0:
		return sanitizePolylineCoordinates(wrapped.LatLng), nil
	default:
		return nil, errors.New("shapeData does not contain coordinates")
	}
}

func parseShapeCoordinatesFromGPX(raw string) [][]float64 {
	pointTagPattern := regexp.MustCompile(`(?is)<(?:trkpt|rtept|wpt)\b([^>]*)>`)
	latAttrPattern := regexp.MustCompile(`(?i)\blat\s*=\s*["']([^"']+)["']`)
	lngAttrPattern := regexp.MustCompile(`(?i)\blon\s*=\s*["']([^"']+)["']`)

	matches := pointTagPattern.FindAllStringSubmatch(raw, -1)
	if len(matches) == 0 {
		return nil
	}

	points := make([][]float64, 0, len(matches))
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		attributes := match[1]
		latMatch := latAttrPattern.FindStringSubmatch(attributes)
		lngMatch := lngAttrPattern.FindStringSubmatch(attributes)
		if len(latMatch) < 2 || len(lngMatch) < 2 {
			continue
		}
		lat, latErr := strconv.ParseFloat(strings.TrimSpace(latMatch[1]), 64)
		lng, lngErr := strconv.ParseFloat(strings.TrimSpace(lngMatch[1]), 64)
		if latErr != nil || lngErr != nil {
			continue
		}
		if !isValidLatLng(lat, lng) {
			continue
		}
		points = append(points, []float64{lat, lng})
	}
	return points
}

func decodeEncodedPolylineCoordinates(encoded string) ([][]float64, error) {
	value := strings.TrimSpace(encoded)
	if value == "" {
		return nil, errors.New("encoded polyline is empty")
	}

	points := make([][]float64, 0, 32)
	index := 0
	lat := 0
	lng := 0
	for index < len(value) {
		latDelta, nextIndex, err := decodePolylineDelta(value, index)
		if err != nil {
			return nil, err
		}
		index = nextIndex

		lngDelta, nextIndex, err := decodePolylineDelta(value, index)
		if err != nil {
			return nil, err
		}
		index = nextIndex

		lat += latDelta
		lng += lngDelta
		points = append(points, []float64{float64(lat) / 1e5, float64(lng) / 1e5})
	}

	if len(points) == 0 {
		return nil, errors.New("encoded polyline contains no coordinates")
	}
	return points, nil
}

func decodePolylineDelta(encoded string, startIndex int) (int, int, error) {
	result := 0
	shift := 0
	index := startIndex
	for index < len(encoded) {
		chunk := int(encoded[index]) - 63
		if chunk < 0 {
			return 0, index, errors.New("encoded polyline contains invalid characters")
		}
		result |= (chunk & 0x1F) << shift
		shift += 5
		index += 1
		if chunk < 0x20 {
			delta := result >> 1
			if result&1 == 1 {
				delta = ^delta
			}
			return delta, index, nil
		}
	}
	return 0, index, errors.New("encoded polyline is truncated")
}

func sanitizePolylineCoordinates(points [][]float64) [][]float64 {
	result := make([][]float64, 0, len(points))
	for _, point := range points {
		if len(point) < 2 {
			continue
		}
		if !isValidLatLng(point[0], point[1]) {
			continue
		}
		result = append(result, []float64{point[0], point[1]})
	}
	return result
}

func inferShapeFromCoordinates(points [][]float64) string {
	if len(points) < 2 {
		return ""
	}
	start := routesDomain.Coordinates{Lat: points[0][0], Lng: points[0][1]}
	end := routesDomain.Coordinates{Lat: points[len(points)-1][0], Lng: points[len(points)-1][1]}
	startEndDistance := haversineDistanceMeters(start, end)
	pathDistance := 0.0
	maxFromStart := 0.0
	for index := 1; index < len(points); index++ {
		prev := routesDomain.Coordinates{Lat: points[index-1][0], Lng: points[index-1][1]}
		next := routesDomain.Coordinates{Lat: points[index][0], Lng: points[index][1]}
		segment := haversineDistanceMeters(prev, next)
		pathDistance += segment
		startDistance := haversineDistanceMeters(start, next)
		if startDistance > maxFromStart {
			maxFromStart = startDistance
		}
	}

	loopThreshold := math.Max(350.0, pathDistance*0.08)
	if startEndDistance <= loopThreshold {
		return "LOOP"
	}
	if maxFromStart > 0 && startEndDistance <= math.Max(220.0, maxFromStart*0.18) {
		return "OUT_AND_BACK"
	}
	return "POINT_TO_POINT"
}

func haversineDistanceMeters(left routesDomain.Coordinates, right routesDomain.Coordinates) float64 {
	const earthRadiusM = 6371000.0
	lat1 := left.Lat * (math.Pi / 180.0)
	lat2 := right.Lat * (math.Pi / 180.0)
	dLat := (right.Lat - left.Lat) * (math.Pi / 180.0)
	dLng := (right.Lng - left.Lng) * (math.Pi / 180.0)

	a := math.Sin(dLat/2.0)*math.Sin(dLat/2.0) +
		math.Cos(lat1)*math.Cos(lat2)*math.Sin(dLng/2.0)*math.Sin(dLng/2.0)
	c := 2.0 * math.Atan2(math.Sqrt(a), math.Sqrt(1.0-a))
	return earthRadiusM * c
}

func isValidLatLng(lat float64, lng float64) bool {
	return lat >= -90.0 && lat <= 90.0 && lng >= -180.0 && lng <= 180.0
}

func optionalNonEmptyString(value string) *string {
	normalized := strings.TrimSpace(value)
	if normalized == "" {
		return nil
	}
	return &normalized
}

// findRouteForGPXExport searches a RouteExplorerResult for a matching routeID.
func findRouteForGPXExport(result routesDomain.RouteExplorerResult, routeID string) (string, [][]float64, bool) {
	recommendations := make([]routesDomain.RouteRecommendation, 0,
		len(result.ClosestLoops)+len(result.Variants)+len(result.Seasonal)+len(result.RoadGraphLoops)+len(result.ShapeMatches))
	recommendations = append(recommendations, result.ClosestLoops...)
	recommendations = append(recommendations, result.Variants...)
	recommendations = append(recommendations, result.Seasonal...)
	recommendations = append(recommendations, result.RoadGraphLoops...)
	recommendations = append(recommendations, result.ShapeMatches...)

	for _, recommendation := range recommendations {
		if recommendation.RouteID == routeID {
			name := recommendation.Activity.Name
			if name == "" {
				name = recommendation.RouteID
			}
			return name, recommendation.PreviewLatLng, true
		}
	}

	for _, remix := range result.ShapeRemixes {
		if remix.ID == routeID {
			name := remix.ID
			if len(remix.Components) > 0 && remix.Components[0].Name != "" {
				name = fmt.Sprintf("Remix - %s", remix.Components[0].Name)
			}
			return name, remix.PreviewLatLng, true
		}
	}

	return "", nil, false
}
