package infrastructure

import (
	"math"
	routesDomain "mystravastats/internal/routes/domain"
	"sort"
	"strings"
)

type normalizedShapePoint struct {
	x float64
	y float64
}

type shapeSimilarityBreakdown struct {
	score         float64
	contourScore  float64
	anchoredScore float64
	orderedScore  float64
	centroidScore float64
	corridorScore float64
	lengthScore   float64
}

type shapeModeScoringConfig struct {
	baseMatchWeight          float64
	shapeWeight              float64
	lowSimilarityThreshold   float64
	lowSimilarityPenaltyRate float64
}

func shapeModeScoringConfigFor(strategyCode string) shapeModeScoringConfig {
	switch strings.ToLower(strings.TrimSpace(strategyCode)) {
	case shapeModeStrategyMapMatch:
		return shapeModeScoringConfig{
			baseMatchWeight:          0.10,
			shapeWeight:              0.90,
			lowSimilarityThreshold:   0.78,
			lowSimilarityPenaltyRate: 1.35,
		}
	case shapeModeStrategyRoadFirst:
		return shapeModeScoringConfig{
			baseMatchWeight:          0.20,
			shapeWeight:              0.80,
			lowSimilarityThreshold:   0.76,
			lowSimilarityPenaltyRate: 1.35,
		}
	default:
		return shapeModeScoringConfig{
			baseMatchWeight:          0.14,
			shapeWeight:              0.86,
			lowSimilarityThreshold:   0.72,
			lowSimilarityPenaltyRate: 1.10,
		}
	}
}

func minShapeModeSimilarity(strategyCode string) float64 {
	switch strings.ToLower(strings.TrimSpace(strategyCode)) {
	case shapeModeStrategyMapMatch:
		return 0.76
	case shapeModeStrategyRoadFirst:
		return 0.72
	default:
		return 0.68
	}
}

func shapeModeLowSimilarityPenalty(strategyCode string, shapeScore float64) float64 {
	config := shapeModeScoringConfigFor(strategyCode)
	normalizedShapeScore := clampUnit(shapeScore)
	if normalizedShapeScore >= config.lowSimilarityThreshold {
		return 0.0
	}
	return (config.lowSimilarityThreshold - normalizedShapeScore) * 100.0 * config.lowSimilarityPenaltyRate
}

func shapeModeMatchScore(
	baseMatchScore float64,
	shapeScore float64,
	backtrackingRatio float64,
	corridorOverlap float64,
	edgeReuseRatio float64,
	maxAxisReuseRatio float64,
	strategyCode string,
) (float64, float64) {
	config := shapeModeScoringConfigFor(strategyCode)
	normalizedShapeScore := clampUnit(shapeScore)
	shapeDriftPenalty := shapeModeLowSimilarityPenalty(strategyCode, normalizedShapeScore)
	score := baseMatchScore*config.baseMatchWeight +
		normalizedShapeScore*100.0*config.shapeWeight -
		backtrackingRatio*28.0 -
		corridorOverlap*35.0 -
		edgeReuseRatio*40.0 -
		maxAxisReuseRatio*48.0 -
		shapeDriftPenalty
	return clampOSMScore(score), shapeDriftPenalty
}

func shapeSimilarityScore(routePoints [][]float64, shapePoints [][]float64) float64 {
	return shapeSimilarityBreakdownFor(routePoints, shapePoints).score
}

func shapeSimilarityBreakdownFor(routePoints [][]float64, shapePoints [][]float64) shapeSimilarityBreakdown {
	sampledRoute := samplePolylinePoints(routePoints, 90)
	sampledShape := samplePolylinePoints(shapePoints, 90)
	normalizedRoute := normalizeShapePolyline(sampledRoute)
	normalizedShape := normalizeShapePolyline(sampledShape)
	if len(normalizedRoute) < 2 || len(normalizedShape) < 2 {
		return shapeSimilarityBreakdown{}
	}
	meanForward := meanNearestShapeDistance(normalizedShape, normalizedRoute)
	meanBackward := meanNearestShapeDistance(normalizedRoute, normalizedShape)
	contourDistance := (meanForward + meanBackward) / 2.0
	contourScore := clampUnit(1.0 - (contourDistance / 1.35))

	shapeCenterLat, shapeCenterLng, ok := latLngCentroid(sampledShape)
	if !ok {
		return shapeSimilarityBreakdown{score: contourScore, contourScore: contourScore}
	}
	shapeRadius := maxLatLngRadiusMeters(sampledShape, shapeCenterLat, shapeCenterLng)
	if shapeRadius < 1.0 {
		shapeRadius = 1.0
	}
	anchoredShape := projectLatLngToShapeSpace(sampledShape, shapeCenterLat, shapeCenterLng, shapeRadius)
	anchoredRoute := projectLatLngToShapeSpace(sampledRoute, shapeCenterLat, shapeCenterLng, shapeRadius)
	if len(anchoredShape) < 2 || len(anchoredRoute) < 2 {
		return shapeSimilarityBreakdown{score: contourScore, contourScore: contourScore}
	}

	anchoredForward := meanNearestShapeDistance(anchoredShape, anchoredRoute)
	anchoredBackward := meanNearestShapeDistance(anchoredRoute, anchoredShape)
	anchoredDistance := (anchoredForward + anchoredBackward) / 2.0
	anchoredScore := clampUnit(1.0 - (anchoredDistance / 0.82))
	orderedDistance := meanIndexedShapeDistance(anchoredShape, anchoredRoute)
	orderedScore := clampUnit(1.0 - (orderedDistance / 0.92))

	centroidScore := 0.0
	if routeCenterLat, routeCenterLng, ok := latLngCentroid(sampledRoute); ok {
		centroidDrift := haversineDistanceMeters(shapeCenterLat, shapeCenterLng, routeCenterLat, routeCenterLng) / shapeRadius
		centroidScore = clampUnit(1.0 - (centroidDrift / 0.80))
	}

	corridorScore := shapeCorridorFitScore(sampledRoute, sampledShape, shapeCenterLat, shapeCenterLng, shapeRadius)
	shapeLengthKm := polylineDistanceKmFromLatLng(sampledShape)
	routeLengthKm := polylineDistanceKmFromLatLng(sampledRoute)
	lengthScore := shapeLengthSimilarityScore(routeLengthKm, shapeLengthKm)

	score := contourScore*0.05 +
		anchoredScore*0.30 +
		orderedScore*0.28 +
		centroidScore*0.10 +
		corridorScore*0.22 +
		lengthScore*0.05
	score = math.Min(score, 0.54+centroidScore*0.28+corridorScore*0.18)
	return shapeSimilarityBreakdown{
		score:         clampUnit(score),
		contourScore:  contourScore,
		anchoredScore: anchoredScore,
		orderedScore:  orderedScore,
		centroidScore: centroidScore,
		corridorScore: corridorScore,
		lengthScore:   lengthScore,
	}
}

func shapeCorridorFitScore(
	routePoints [][]float64,
	shapePoints [][]float64,
	centerLat float64,
	centerLng float64,
	shapeRadiusMeters float64,
) float64 {
	routeMeters := projectLatLngToMeters(routePoints, centerLat, centerLng)
	shapeMeters := projectLatLngToMeters(shapePoints, centerLat, centerLng)
	if len(routeMeters) < 2 || len(shapeMeters) < 2 {
		return 0.0
	}
	if shapeRadiusMeters < 1.0 {
		shapeRadiusMeters = 1.0
	}

	routeMean := meanNearestPolylineDistanceMeters(routeMeters, shapeMeters)
	shapeMean := meanNearestPolylineDistanceMeters(shapeMeters, routeMeters)
	routeTail := percentileNearestPolylineDistanceMeters(routeMeters, shapeMeters, 0.90)
	shapeTail := percentileNearestPolylineDistanceMeters(shapeMeters, routeMeters, 0.90)
	meanDistance := (routeMean + shapeMean) / 2.0
	tailDistance := math.Max(routeTail, shapeTail)

	meanTolerance := math.Min(420.0, math.Max(120.0, shapeRadiusMeters*0.24))
	tailTolerance := math.Min(760.0, math.Max(240.0, shapeRadiusMeters*0.48))
	meanScore := clampUnit(1.0 - meanDistance/meanTolerance)
	tailScore := clampUnit(1.0 - tailDistance/tailTolerance)
	return clampUnit(meanScore*0.72 + tailScore*0.28)
}

func projectLatLngToMeters(points [][]float64, centerLat float64, centerLng float64) []normalizedShapePoint {
	cosLat := math.Cos(degreesToRadians(centerLat))
	projected := make([]normalizedShapePoint, 0, len(points))
	for _, point := range points {
		if len(point) < 2 {
			continue
		}
		projected = append(projected, normalizedShapePoint{
			x: (point[1] - centerLng) * 111320.0 * cosLat,
			y: (point[0] - centerLat) * 111320.0,
		})
	}
	return projected
}

func meanNearestPolylineDistanceMeters(from []normalizedShapePoint, to []normalizedShapePoint) float64 {
	if len(from) == 0 || len(to) < 2 {
		return math.MaxFloat64
	}
	total := 0.0
	for _, point := range from {
		total += nearestPolylineDistanceMeters(point, to)
	}
	return total / float64(len(from))
}

func percentileNearestPolylineDistanceMeters(
	from []normalizedShapePoint,
	to []normalizedShapePoint,
	percentile float64,
) float64 {
	if len(from) == 0 || len(to) < 2 {
		return math.MaxFloat64
	}
	distances := make([]float64, 0, len(from))
	for _, point := range from {
		distances = append(distances, nearestPolylineDistanceMeters(point, to))
	}
	sort.Float64s(distances)
	index := int(math.Ceil(clampUnit(percentile)*float64(len(distances)))) - 1
	if index < 0 {
		index = 0
	}
	if index >= len(distances) {
		index = len(distances) - 1
	}
	return distances[index]
}

func nearestPolylineDistanceMeters(point normalizedShapePoint, polyline []normalizedShapePoint) float64 {
	if len(polyline) == 0 {
		return math.MaxFloat64
	}
	if len(polyline) == 1 {
		return euclideanDistanceMeters(point, polyline[0])
	}
	minDistance := math.MaxFloat64
	for index := 0; index < len(polyline)-1; index++ {
		distance := pointToSegmentDistanceMeters(point, polyline[index], polyline[index+1])
		if distance < minDistance {
			minDistance = distance
		}
	}
	return minDistance
}

func pointToSegmentDistanceMeters(
	point normalizedShapePoint,
	start normalizedShapePoint,
	end normalizedShapePoint,
) float64 {
	dx := end.x - start.x
	dy := end.y - start.y
	lengthSquared := dx*dx + dy*dy
	if lengthSquared <= 0.0 {
		return euclideanDistanceMeters(point, start)
	}
	t := ((point.x-start.x)*dx + (point.y-start.y)*dy) / lengthSquared
	t = clampUnit(t)
	projection := normalizedShapePoint{
		x: start.x + dx*t,
		y: start.y + dy*t,
	}
	return euclideanDistanceMeters(point, projection)
}

func euclideanDistanceMeters(left normalizedShapePoint, right normalizedShapePoint) float64 {
	dx := left.x - right.x
	dy := left.y - right.y
	return math.Sqrt(dx*dx + dy*dy)
}

func polylineDistanceKmFromLatLng(points [][]float64) float64 {
	if len(points) < 2 {
		return 0.0
	}
	totalMeters := 0.0
	for index := 0; index < len(points)-1; index++ {
		left := points[index]
		right := points[index+1]
		if len(left) < 2 || len(right) < 2 {
			continue
		}
		totalMeters += haversineDistanceMeters(left[0], left[1], right[0], right[1])
	}
	return totalMeters / 1000.0
}

func shapeLengthSimilarityScore(routeLengthKm float64, shapeLengthKm float64) float64 {
	if routeLengthKm <= 0 || shapeLengthKm <= 0 {
		return 0.0
	}
	deltaRatio := math.Abs(routeLengthKm-shapeLengthKm) / math.Max(routeLengthKm, shapeLengthKm)
	return clampUnit(1.0 - deltaRatio*1.35)
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

func latLngCentroid(points [][]float64) (float64, float64, bool) {
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
		return 0, 0, false
	}
	return sumLat / float64(count), sumLng / float64(count), true
}

func maxLatLngRadiusMeters(points [][]float64, centerLat float64, centerLng float64) float64 {
	maxRadius := 0.0
	for _, point := range points {
		if len(point) < 2 {
			continue
		}
		radius := haversineDistanceMeters(centerLat, centerLng, point[0], point[1])
		if radius > maxRadius {
			maxRadius = radius
		}
	}
	return maxRadius
}

func projectLatLngToShapeSpace(points [][]float64, centerLat float64, centerLng float64, scaleMeters float64) []normalizedShapePoint {
	if scaleMeters < 1.0 {
		scaleMeters = 1.0
	}
	cosLat := math.Cos(degreesToRadians(centerLat))
	projected := make([]normalizedShapePoint, 0, len(points))
	for _, point := range points {
		if len(point) < 2 {
			continue
		}
		x := (point[1] - centerLng) * 111320.0 * cosLat / scaleMeters
		y := (point[0] - centerLat) * 111320.0 / scaleMeters
		projected = append(projected, normalizedShapePoint{x: x, y: y})
	}
	return projected
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

func meanIndexedShapeDistance(left []normalizedShapePoint, right []normalizedShapePoint) float64 {
	count := minInt(len(left), len(right))
	if count == 0 {
		return 1.0
	}
	total := 0.0
	for index := 0; index < count; index++ {
		dx := left[index].x - right[index].x
		dy := left[index].y - right[index].y
		total += math.Sqrt(dx*dx + dy*dy)
	}
	return total / float64(count)
}

func routeShapeScore(recommendation routesDomain.RouteRecommendation) float64 {
	if recommendation.ShapeScore == nil {
		return 0.0
	}
	return clampUnit(*recommendation.ShapeScore)
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
