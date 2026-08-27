package infrastructure

import (
	"encoding/json"
	"fmt"
	"math"
	routesDomain "mystravastats/internal/routes/domain"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

func parseShapePolylineCoordinates(raw string) []routesDomain.Coordinates {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return []routesDomain.Coordinates{}
	}

	var points [][]float64
	if err := json.Unmarshal([]byte(trimmed), &points); err != nil {
		var wrapped struct {
			Points      [][]float64 `json:"points"`
			Coordinates [][]float64 `json:"coordinates"`
			LatLng      [][]float64 `json:"latLng"`
		}
		if wrappedErr := json.Unmarshal([]byte(trimmed), &wrapped); wrappedErr == nil {
			switch {
			case len(wrapped.Points) > 0:
				points = wrapped.Points
			case len(wrapped.Coordinates) > 0:
				points = wrapped.Coordinates
			case len(wrapped.LatLng) > 0:
				points = wrapped.LatLng
			}
		}
	}

	if len(points) == 0 {
		if gpxPoints := parseShapeCoordinatesFromGPX(trimmed); len(gpxPoints) > 0 {
			return gpxPoints
		}
		encoded := trimmed
		var quoted string
		if err := json.Unmarshal([]byte(trimmed), &quoted); err == nil {
			encoded = strings.TrimSpace(quoted)
		}
		decoded := decodeEncodedPolylineCoordinates(encoded)
		if len(decoded) == 0 {
			return []routesDomain.Coordinates{}
		}
		return decoded
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

func parseShapeCoordinatesFromGPX(raw string) []routesDomain.Coordinates {
	pointTagPattern := regexp.MustCompile(`(?is)<(?:trkpt|rtept|wpt)\b([^>]*)>`)
	latAttrPattern := regexp.MustCompile(`(?i)\blat\s*=\s*["']([^"']+)["']`)
	lngAttrPattern := regexp.MustCompile(`(?i)\blon\s*=\s*["']([^"']+)["']`)

	matches := pointTagPattern.FindAllStringSubmatch(raw, -1)
	if len(matches) == 0 {
		return []routesDomain.Coordinates{}
	}

	points := make([]routesDomain.Coordinates, 0, len(matches))
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
		if lat < -90 || lat > 90 || lng < -180 || lng > 180 {
			continue
		}
		points = append(points, routesDomain.Coordinates{Lat: lat, Lng: lng})
	}
	return points
}

func decodeEncodedPolylineCoordinates(encoded string) []routesDomain.Coordinates {
	value := strings.TrimSpace(encoded)
	if value == "" {
		return []routesDomain.Coordinates{}
	}

	points := make([]routesDomain.Coordinates, 0, 64)
	index := 0
	lat := 0
	lng := 0
	for index < len(value) {
		latDelta, nextIndex, ok := decodePolylineDelta(value, index)
		if !ok {
			return []routesDomain.Coordinates{}
		}
		index = nextIndex

		lngDelta, nextIndex, ok := decodePolylineDelta(value, index)
		if !ok {
			return []routesDomain.Coordinates{}
		}
		index = nextIndex

		lat += latDelta
		lng += lngDelta
		point := routesDomain.Coordinates{
			Lat: float64(lat) / 1e5,
			Lng: float64(lng) / 1e5,
		}
		if point.Lat < -90 || point.Lat > 90 || point.Lng < -180 || point.Lng > 180 {
			continue
		}
		points = append(points, point)
	}

	return points
}

func decodePolylineDelta(encoded string, startIndex int) (int, int, bool) {
	result := 0
	shift := 0
	index := startIndex
	for index < len(encoded) {
		chunk := int(encoded[index]) - 63
		if chunk < 0 {
			return 0, index, false
		}
		result |= (chunk & 0x1F) << shift
		shift += 5
		index += 1
		if chunk < 0x20 {
			delta := result >> 1
			if result&1 == 1 {
				delta = ^delta
			}
			return delta, index, true
		}
	}
	return 0, index, false
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

	shapeCenter, shapeRadiusMeters := shapeCenterAndRadius(shape)
	scaleAnchor := shapeCenter
	projectedBase := cloneCoordinates(shape)

	if !preserveGeoreferencedShapePlacement(shape, start, shapeCenter, shapeRadiusMeters) {
		deltaLat := start.Lat - shapeCenter.Lat
		deltaLng := start.Lng - shapeCenter.Lng
		projectedBase = make([]routesDomain.Coordinates, 0, len(shape))
		for _, point := range shape {
			projectedBase = append(projectedBase, routesDomain.Coordinates{
				Lat: point.Lat + deltaLat,
				Lng: point.Lng + deltaLng,
			})
		}
		scaleAnchor = start
	}

	scale := 1.0
	shapeDistanceKm := polylineDistanceKmFromCoordinates(projectedBase)
	if targetDistanceKm > 0 && shapeDistanceKm > 0 {
		scale = targetDistanceKm / shapeDistanceKm
		if scale < 0.45 {
			scale = 0.45
		}
		if scale > 2.60 {
			scale = 2.60
		}
	}

	projected := make([]routesDomain.Coordinates, 0, len(projectedBase))
	for _, point := range projectedBase {
		projected = append(projected, routesDomain.Coordinates{
			Lat: scaleAnchor.Lat + (point.Lat-scaleAnchor.Lat)*scale,
			Lng: scaleAnchor.Lng + (point.Lng-scaleAnchor.Lng)*scale,
		})
	}
	return projected
}

func cloneCoordinates(points []routesDomain.Coordinates) []routesDomain.Coordinates {
	cloned := make([]routesDomain.Coordinates, len(points))
	copy(cloned, points)
	return cloned
}

func shapeCenterAndRadius(points []routesDomain.Coordinates) (routesDomain.Coordinates, float64) {
	if len(points) == 0 {
		return routesDomain.Coordinates{}, 0.0
	}
	minLat := points[0].Lat
	maxLat := points[0].Lat
	minLng := points[0].Lng
	maxLng := points[0].Lng
	for _, point := range points[1:] {
		if point.Lat < minLat {
			minLat = point.Lat
		}
		if point.Lat > maxLat {
			maxLat = point.Lat
		}
		if point.Lng < minLng {
			minLng = point.Lng
		}
		if point.Lng > maxLng {
			maxLng = point.Lng
		}
	}
	center := routesDomain.Coordinates{
		Lat: (minLat + maxLat) / 2.0,
		Lng: (minLng + maxLng) / 2.0,
	}
	radiusMeters := 0.0
	for _, point := range points {
		radiusMeters = math.Max(
			radiusMeters,
			haversineDistanceMeters(center.Lat, center.Lng, point.Lat, point.Lng),
		)
	}
	return center, radiusMeters
}

func preserveGeoreferencedShapePlacement(
	shape []routesDomain.Coordinates,
	start routesDomain.Coordinates,
	center routesDomain.Coordinates,
	radiusMeters float64,
) bool {
	centerDistanceMeters := haversineDistanceMeters(start.Lat, start.Lng, center.Lat, center.Lng)
	if centerDistanceMeters <= math.Max(900.0, radiusMeters*1.35) {
		return true
	}
	nearestPointMeters := math.MaxFloat64
	for _, point := range shape {
		nearestPointMeters = math.Min(
			nearestPointMeters,
			haversineDistanceMeters(start.Lat, start.Lng, point.Lat, point.Lng),
		)
	}
	return nearestPointMeters <= math.Max(500.0, radiusMeters*0.35)
}

func buildShapeRoutingVariants(
	shape []routesDomain.Coordinates,
	preferredStart routesDomain.Coordinates,
) []shapeRoutingVariant {
	if len(shape) < 2 {
		return []shapeRoutingVariant{}
	}
	primary := prepareShapeForRouting(shape, preferredStart)
	variants := make([]shapeRoutingVariant, 0, 18)
	seenShapes := make(map[string]struct{})

	addVariant := func(label string, points []routesDomain.Coordinates) {
		if len(points) < 2 {
			return
		}
		key := shapeVariantSignature(points)
		if key == "" {
			return
		}
		if _, exists := seenShapes[key]; exists {
			return
		}
		seenShapes[key] = struct{}{}
		variants = append(variants, shapeRoutingVariant{
			label: label,
			shape: cloneCoordinates(points),
		})
	}

	addVariant("", primary)
	for _, transform := range []struct {
		label           string
		scale           float64
		rotationDegrees float64
	}{
		{label: "scale 0.55x", scale: 0.55},
		{label: "scale 0.70x", scale: 0.70},
		{label: "rotate -12 deg", scale: 1.0, rotationDegrees: -12.0},
		{label: "rotate 12 deg", scale: 1.0, rotationDegrees: 12.0},
		{label: "scale 0.85x", scale: 0.85},
		{label: "scale 1.15x", scale: 1.15},
		{label: "rotate -24 deg", scale: 1.0, rotationDegrees: -24.0},
		{label: "rotate 24 deg", scale: 1.0, rotationDegrees: 24.0},
		{label: "scale 1.30x", scale: 1.30},
		{label: "rotate -36 deg", scale: 1.0, rotationDegrees: -36.0},
		{label: "rotate 36 deg", scale: 1.0, rotationDegrees: 36.0},
	} {
		addVariant(transform.label, transformShapePose(primary, transform.scale, transform.rotationDegrees, 0.0, 0.0))
	}
	_, radiusMeters := shapeCenterAndRadius(primary)
	shiftKm := math.Min(0.45, math.Max(0.18, radiusMeters/1000.0*0.18))
	addVariant(fmt.Sprintf("shift north %.2fkm", shiftKm), transformShapePose(primary, 1.0, 0.0, shiftKm, 0.0))
	addVariant(fmt.Sprintf("shift east %.2fkm", shiftKm), transformShapePose(primary, 1.0, 0.0, shiftKm, 90.0))
	addVariant(fmt.Sprintf("shift south %.2fkm", shiftKm), transformShapePose(primary, 1.0, 0.0, shiftKm, 180.0))
	addVariant(fmt.Sprintf("shift west %.2fkm", shiftKm), transformShapePose(primary, 1.0, 0.0, shiftKm, 270.0))
	return variants
}

func shapeVariantSignature(points []routesDomain.Coordinates) string {
	if len(points) < 2 {
		return ""
	}
	sampled := sampleCoordinates(points, 10)
	parts := make([]string, 0, len(sampled))
	for _, point := range sampled {
		parts = append(parts, fmt.Sprintf("%.5f,%.5f", point.Lat, point.Lng))
	}
	return strings.Join(parts, "|")
}

func transformShapePose(
	points []routesDomain.Coordinates,
	scale float64,
	rotationDegrees float64,
	shiftKm float64,
	shiftBearingDegrees float64,
) []routesDomain.Coordinates {
	if len(points) < 2 {
		return cloneCoordinates(points)
	}
	if scale <= 0 {
		scale = 1.0
	}
	center, _ := shapeCenterAndRadius(points)
	cosLat := math.Cos(degreesToRadians(center.Lat))
	if math.Abs(cosLat) < 0.000001 {
		return cloneCoordinates(points)
	}
	rotationRadians := degreesToRadians(rotationDegrees)
	cosRotation := math.Cos(rotationRadians)
	sinRotation := math.Sin(rotationRadians)
	shiftMeters := shiftKm * 1000.0
	shiftBearingRadians := degreesToRadians(shiftBearingDegrees)
	shiftX := math.Sin(shiftBearingRadians) * shiftMeters
	shiftY := math.Cos(shiftBearingRadians) * shiftMeters
	transformed := make([]routesDomain.Coordinates, 0, len(points))
	for _, point := range points {
		x := (point.Lng - center.Lng) * 111320.0 * cosLat
		y := (point.Lat - center.Lat) * 111320.0
		x *= scale
		y *= scale
		rotatedX := x*cosRotation - y*sinRotation + shiftX
		rotatedY := x*sinRotation + y*cosRotation + shiftY
		next := routesDomain.Coordinates{
			Lat: center.Lat + rotatedY/111320.0,
			Lng: center.Lng + rotatedX/(111320.0*cosLat),
		}
		if !isFiniteRouteCoordinate(next) {
			return cloneCoordinates(points)
		}
		transformed = append(transformed, next)
	}
	return transformed
}

func isFiniteRouteCoordinate(point routesDomain.Coordinates) bool {
	return !math.IsNaN(point.Lat) &&
		!math.IsNaN(point.Lng) &&
		!math.IsInf(point.Lat, 0) &&
		!math.IsInf(point.Lng, 0) &&
		point.Lat >= -90.0 &&
		point.Lat <= 90.0 &&
		point.Lng >= -180.0 &&
		point.Lng <= 180.0
}

func prepareShapeForRouting(
	shape []routesDomain.Coordinates,
	preferredStart routesDomain.Coordinates,
) []routesDomain.Coordinates {
	return cloneCoordinates(shape)
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

func sampleCoordinatesByDistance(
	points []routesDomain.Coordinates,
	maxPoints int,
	minSpacingMeters float64,
) []routesDomain.Coordinates {
	if len(points) <= 2 || maxPoints <= 0 {
		return cloneCoordinates(points)
	}
	if minSpacingMeters < 1.0 {
		minSpacingMeters = 1.0
	}

	totalMeters := 0.0
	segmentLengths := make([]float64, len(points)-1)
	for index := 0; index < len(points)-1; index++ {
		length := haversineDistanceMeters(points[index].Lat, points[index].Lng, points[index+1].Lat, points[index+1].Lng)
		segmentLengths[index] = length
		totalMeters += length
	}
	if totalMeters <= 0.0 {
		return cloneCoordinates(points[:1])
	}

	targetCount := int(math.Floor(totalMeters/minSpacingMeters)) + 1
	if targetCount < 2 {
		targetCount = 2
	}
	if targetCount > maxPoints {
		targetCount = maxPoints
	}
	if targetCount >= len(points) {
		return cloneCoordinates(points)
	}

	intervalMeters := totalMeters / float64(targetCount-1)
	sampled := make([]routesDomain.Coordinates, 0, targetCount)
	sampled = append(sampled, points[0])
	nextDistance := intervalMeters
	traversed := 0.0
	segmentIndex := 0

	for len(sampled) < targetCount-1 && segmentIndex < len(segmentLengths) {
		segmentLength := segmentLengths[segmentIndex]
		if segmentLength <= 0.0 {
			segmentIndex++
			continue
		}
		if traversed+segmentLength < nextDistance {
			traversed += segmentLength
			segmentIndex++
			continue
		}
		t := (nextDistance - traversed) / segmentLength
		t = clampUnit(t)
		start := points[segmentIndex]
		end := points[segmentIndex+1]
		sampled = append(sampled, routesDomain.Coordinates{
			Lat: start.Lat + (end.Lat-start.Lat)*t,
			Lng: start.Lng + (end.Lng-start.Lng)*t,
		})
		nextDistance += intervalMeters
	}

	last := points[len(points)-1]
	if len(sampled) == 0 ||
		haversineDistanceMeters(sampled[len(sampled)-1].Lat, sampled[len(sampled)-1].Lng, last.Lat, last.Lng) > 1.0 {
		sampled = append(sampled, last)
	}
	return sampled
}

func buildShapeLoopWaypoints(
	start routesDomain.Coordinates,
	shape []routesDomain.Coordinates,
) []routesDomain.Coordinates {
	sampled := sampleCoordinates(shape, 18)
	waypoints := make([]routesDomain.Coordinates, 0, len(sampled)+2)
	waypoints = append(waypoints, start)
	previous := start
	for _, point := range sampled {
		if haversineDistanceMeters(previous.Lat, previous.Lng, point.Lat, point.Lng) < 80.0 {
			continue
		}
		waypoints = append(waypoints, point)
		previous = point
	}
	return appendShapeEndWaypoint(waypoints, shape)
}

func buildShapeDenseWaypoints(
	start routesDomain.Coordinates,
	shape []routesDomain.Coordinates,
) []routesDomain.Coordinates {
	if len(shape) < 2 {
		return []routesDomain.Coordinates{}
	}
	sampled := sampleCoordinates(shape, 28)
	waypoints := make([]routesDomain.Coordinates, 0, len(sampled)+2)
	waypoints = append(waypoints, start)
	previous := start
	for _, point := range sampled {
		if haversineDistanceMeters(previous.Lat, previous.Lng, point.Lat, point.Lng) < 60.0 {
			continue
		}
		waypoints = append(waypoints, point)
		previous = point
	}
	if len(waypoints) < 3 {
		return buildShapeLoopWaypoints(start, shape)
	}
	return appendShapeEndWaypoint(waypoints, shape)
}

func buildShapeFidelityStitchedWaypoints(
	start routesDomain.Coordinates,
	shape []routesDomain.Coordinates,
) []routesDomain.Coordinates {
	if len(shape) < 2 {
		return []routesDomain.Coordinates{}
	}
	sampled := sampleCoordinatesByDistance(shape, 26, 90.0)
	waypoints := make([]routesDomain.Coordinates, 0, len(sampled)+2)
	waypoints = append(waypoints, start)
	previous := start
	for _, point := range sampled {
		if haversineDistanceMeters(previous.Lat, previous.Lng, point.Lat, point.Lng) < 55.0 {
			continue
		}
		waypoints = append(waypoints, point)
		previous = point
	}
	if len(waypoints) < 3 {
		return buildShapeStitchedWaypoints(start, shape)
	}
	return appendShapeEndWaypoint(waypoints, shape)
}

func buildShapeStitchedWaypoints(
	start routesDomain.Coordinates,
	shape []routesDomain.Coordinates,
) []routesDomain.Coordinates {
	if len(shape) < 2 {
		return []routesDomain.Coordinates{}
	}
	sampled := sampleCoordinates(shape, 14)
	waypoints := make([]routesDomain.Coordinates, 0, len(sampled)+2)
	waypoints = append(waypoints, start)
	previous := start
	for _, point := range sampled {
		if haversineDistanceMeters(previous.Lat, previous.Lng, point.Lat, point.Lng) < 120.0 {
			continue
		}
		waypoints = append(waypoints, point)
		previous = point
	}
	if len(waypoints) < 3 {
		return buildShapeSimplifiedWaypoints(start, shape)
	}
	return appendShapeEndWaypoint(waypoints, shape)
}

func buildShapeSimplifiedWaypoints(
	start routesDomain.Coordinates,
	shape []routesDomain.Coordinates,
) []routesDomain.Coordinates {
	if len(shape) < 2 {
		return []routesDomain.Coordinates{}
	}
	sampled := sampleCoordinates(shape, 12)
	waypoints := make([]routesDomain.Coordinates, 0, len(sampled)+2)
	waypoints = append(waypoints, start)
	previous := start
	for _, point := range sampled {
		if haversineDistanceMeters(previous.Lat, previous.Lng, point.Lat, point.Lng) < 160.0 {
			continue
		}
		waypoints = append(waypoints, point)
		previous = point
	}
	if len(waypoints) < 3 {
		return buildShapeLoopWaypoints(start, shape)
	}
	return appendShapeEndWaypoint(waypoints, shape)
}

func buildShapeRoadFirstWaypoints(
	start routesDomain.Coordinates,
	shape []routesDomain.Coordinates,
) []routesDomain.Coordinates {
	if len(shape) < 2 {
		return []routesDomain.Coordinates{}
	}

	sampled := sampleCoordinates(shape, 20)
	if len(sampled) < 2 {
		return []routesDomain.Coordinates{}
	}

	type indexedPoint struct {
		index    int
		point    routesDomain.Coordinates
		distance float64
	}
	scored := make([]indexedPoint, 0, len(sampled))
	for index := 0; index < len(sampled); index++ {
		point := sampled[index]
		distance := haversineDistanceMeters(start.Lat, start.Lng, point.Lat, point.Lng)
		if distance < 280.0 {
			continue
		}
		scored = append(scored, indexedPoint{
			index:    index,
			point:    point,
			distance: distance,
		})
	}
	if len(scored) == 0 {
		return buildShapeLoopWaypoints(start, shape)
	}

	sort.Slice(scored, func(i, j int) bool {
		if scored[i].distance == scored[j].distance {
			return scored[i].index < scored[j].index
		}
		return scored[i].distance > scored[j].distance
	})
	if len(scored) > 8 {
		scored = scored[:8]
	}
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].index < scored[j].index
	})

	waypoints := make([]routesDomain.Coordinates, 0, len(scored)+2)
	waypoints = append(waypoints, start)
	previous := start
	for _, entry := range scored {
		point := entry.point
		if haversineDistanceMeters(previous.Lat, previous.Lng, point.Lat, point.Lng) < 180.0 {
			continue
		}
		waypoints = append(waypoints, point)
		previous = point
	}
	if len(waypoints) < 3 {
		return buildShapeLoopWaypoints(start, shape)
	}
	return appendShapeEndWaypoint(waypoints, shape)
}

func buildShapeBestEffortRoutingStrategies(
	start routesDomain.Coordinates,
	shape []routesDomain.Coordinates,
) []shapeRoutingStrategy {
	strategies := make([]shapeRoutingStrategy, 0, 2)
	if waypoints := buildShapeBestEffortWaypoints(start, shape); len(waypoints) >= 3 {
		strategies = append(strategies, shapeRoutingStrategy{
			code:       shapeModeStrategyBestEffort,
			label:      "simplified sketch fallback",
			waypoints:  waypoints,
			bestEffort: true,
		})
	}
	if waypoints := buildShapeEnvelopeWaypoints(start, shape); len(waypoints) >= 3 {
		strategies = append(strategies, shapeRoutingStrategy{
			code:       shapeModeStrategyBestEffort,
			label:      "shape envelope fallback",
			waypoints:  waypoints,
			bestEffort: true,
		})
	}
	return strategies
}

func buildShapeBestEffortWaypoints(
	start routesDomain.Coordinates,
	shape []routesDomain.Coordinates,
) []routesDomain.Coordinates {
	sampled := sampleCoordinates(shape, 8)
	waypoints := make([]routesDomain.Coordinates, 0, len(sampled)+2)
	waypoints = append(waypoints, start)
	previous := start
	for _, point := range sampled {
		if haversineDistanceMeters(previous.Lat, previous.Lng, point.Lat, point.Lng) < 220.0 {
			continue
		}
		waypoints = append(waypoints, point)
		previous = point
	}
	return appendShapeEndWaypoint(waypoints, shape)
}

func buildShapeEnvelopeWaypoints(
	start routesDomain.Coordinates,
	shape []routesDomain.Coordinates,
) []routesDomain.Coordinates {
	if len(shape) < 2 {
		return []routesDomain.Coordinates{}
	}
	center, radiusMeters := shapeCenterAndRadius(shape)
	radiusKm := radiusMeters / 1000.0
	if radiusKm < 0.55 {
		radiusKm = 0.55
	}
	if radiusKm > 5.0 {
		radiusKm = 5.0
	}
	bearings := []float64{0.0, 90.0, 180.0, 270.0}
	waypoints := make([]routesDomain.Coordinates, 0, len(bearings)+2)
	waypoints = append(waypoints, start)
	previous := start
	for _, bearing := range bearings {
		point := destinationFromBearing(center, radiusKm, bearing)
		if haversineDistanceMeters(previous.Lat, previous.Lng, point.Lat, point.Lng) < 220.0 {
			continue
		}
		waypoints = append(waypoints, point)
		previous = point
	}
	return appendShapeEndWaypoint(waypoints, shape)
}

func coordinatesToLatLngPoints(points []routesDomain.Coordinates) [][]float64 {
	result := make([][]float64, 0, len(points))
	for _, point := range points {
		result = append(result, []float64{point.Lat, point.Lng})
	}
	return result
}

func appendShapeEndWaypoint(
	waypoints []routesDomain.Coordinates,
	shape []routesDomain.Coordinates,
) []routesDomain.Coordinates {
	if len(waypoints) == 0 {
		return cloneCoordinates(shape)
	}
	if len(shape) == 0 {
		return waypoints
	}
	end := shape[len(shape)-1]
	last := waypoints[len(waypoints)-1]
	if haversineDistanceMeters(last.Lat, last.Lng, end.Lat, end.Lng) > 80.0 {
		waypoints = append(waypoints, end)
	}
	return waypoints
}
