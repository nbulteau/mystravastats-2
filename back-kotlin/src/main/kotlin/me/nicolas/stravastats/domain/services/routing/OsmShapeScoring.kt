package me.nicolas.stravastats.domain.services.routing

import me.nicolas.stravastats.domain.RuntimeConfig
import me.nicolas.stravastats.domain.business.ActivityShort
import me.nicolas.stravastats.domain.business.ActivityType
import me.nicolas.stravastats.domain.business.Coordinates
import me.nicolas.stravastats.domain.business.RouteGenerationDiagnostic
import me.nicolas.stravastats.domain.business.RouteRecommendation
import me.nicolas.stravastats.domain.business.RouteVariantType
import org.slf4j.LoggerFactory
import org.springframework.stereotype.Component
import tools.jackson.databind.json.JsonMapper
import tools.jackson.module.kotlin.KotlinModule
import tools.jackson.module.kotlin.readValue
import java.io.File
import java.net.URI
import java.net.http.HttpClient
import java.net.http.HttpRequest
import java.net.http.HttpResponse
import java.time.Duration
import java.time.Instant
import java.time.ZoneOffset
import java.time.format.DateTimeFormatter
import java.util.Locale
import kotlin.math.PI
import kotlin.math.abs
import kotlin.math.asin
import kotlin.math.atan2
import kotlin.math.ceil
import kotlin.math.cos
import kotlin.math.max
import kotlin.math.min
import kotlin.math.round
import kotlin.math.roundToInt
import kotlin.math.sin
import kotlin.math.sqrt

internal fun shapeModeScoringConfigFor(strategyCode: String): ShapeModeScoringConfig {
    return when (strategyCode.trim().lowercase(Locale.getDefault())) {
        SHAPE_MODE_STRATEGY_MAP_MATCH -> ShapeModeScoringConfig(
            baseMatchWeight = 0.10,
            shapeWeight = 0.90,
            lowSimilarityThreshold = 0.78,
            lowSimilarityPenaltyRate = 1.35,
        )

        SHAPE_MODE_STRATEGY_ROAD_FIRST -> ShapeModeScoringConfig(
            baseMatchWeight = 0.20,
            shapeWeight = 0.80,
            lowSimilarityThreshold = 0.76,
            lowSimilarityPenaltyRate = 1.35,
        )

        else -> ShapeModeScoringConfig(
            baseMatchWeight = 0.14,
            shapeWeight = 0.86,
            lowSimilarityThreshold = 0.72,
            lowSimilarityPenaltyRate = 1.10,
        )
    }
}

internal fun minShapeModeSimilarity(strategyCode: String): Double {
    return when (strategyCode.trim().lowercase(Locale.getDefault())) {
        SHAPE_MODE_STRATEGY_MAP_MATCH -> 0.76
        SHAPE_MODE_STRATEGY_ROAD_FIRST -> 0.72
        else -> 0.68
    }
}

internal fun shapeModeLowSimilarityPenalty(strategyCode: String, shapeScore: Double): Double {
    val config = shapeModeScoringConfigFor(strategyCode)
    val normalizedShapeScore = osmClampUnit(shapeScore)
    if (normalizedShapeScore >= config.lowSimilarityThreshold) {
        return 0.0
    }
    return (config.lowSimilarityThreshold - normalizedShapeScore) * 100.0 * config.lowSimilarityPenaltyRate
}

internal fun shapeModeMatchScore(
    baseMatchScore: Double,
    shapeScore: Double,
    backtrackingRatio: Double,
    corridorOverlap: Double,
    edgeReuseRatio: Double,
    maxAxisReuseRatio: Double,
    strategyCode: String,
): Pair<Double, Double> {
    val config = shapeModeScoringConfigFor(strategyCode)
    val normalizedShapeScore = osmClampUnit(shapeScore)
    val shapeDriftPenalty = shapeModeLowSimilarityPenalty(strategyCode, normalizedShapeScore)
    val score = baseMatchScore * config.baseMatchWeight +
        normalizedShapeScore * 100.0 * config.shapeWeight -
        backtrackingRatio * 28.0 -
        corridorOverlap * 35.0 -
        edgeReuseRatio * 40.0 -
        maxAxisReuseRatio * 48.0 -
        shapeDriftPenalty
    return clampScore(score) to shapeDriftPenalty
}

internal fun shapeSimilarityScore(routePoints: List<List<Double>>, shapePoints: List<List<Double>>): Double {
    return shapeSimilarityBreakdown(routePoints, shapePoints).score
}

internal fun shapeSimilarityBreakdown(routePoints: List<List<Double>>, shapePoints: List<List<Double>>): ShapeSimilarityBreakdown {
    val sampledRoute = samplePolylinePoints(routePoints, 90)
    val sampledShape = samplePolylinePoints(shapePoints, 90)
    val normalizedRoute = normalizeShapePolyline(sampledRoute)
    val normalizedShape = normalizeShapePolyline(sampledShape)
    if (normalizedRoute.size < 2 || normalizedShape.size < 2) {
        return ShapeSimilarityBreakdown(0.0, 0.0, 0.0, 0.0, 0.0)
    }
    val meanForward = meanNearestShapeDistance(normalizedShape, normalizedRoute)
    val meanBackward = meanNearestShapeDistance(normalizedRoute, normalizedShape)
    val contourDistance = (meanForward + meanBackward) / 2.0
    val contourScore = osmClampUnit(1.0 - (contourDistance / 1.35))

    val shapeCenter = latLngCentroid(sampledShape)
        ?: return ShapeSimilarityBreakdown(contourScore, contourScore, 0.0, 0.0, 0.0)
    val shapeRadius = max(1.0, maxLatLngRadiusMeters(sampledShape, shapeCenter.first, shapeCenter.second))
    val anchoredShape = projectLatLngToShapeSpace(sampledShape, shapeCenter.first, shapeCenter.second, shapeRadius)
    val anchoredRoute = projectLatLngToShapeSpace(sampledRoute, shapeCenter.first, shapeCenter.second, shapeRadius)
    if (anchoredShape.size < 2 || anchoredRoute.size < 2) {
        return ShapeSimilarityBreakdown(contourScore, contourScore, 0.0, 0.0, 0.0)
    }

    val anchoredForward = meanNearestShapeDistance(anchoredShape, anchoredRoute)
    val anchoredBackward = meanNearestShapeDistance(anchoredRoute, anchoredShape)
    val anchoredDistance = (anchoredForward + anchoredBackward) / 2.0
    val anchoredScore = osmClampUnit(1.0 - (anchoredDistance / 0.82))
    val orderedDistance = meanIndexedShapeDistance(anchoredShape, anchoredRoute)
    val orderedScore = osmClampUnit(1.0 - (orderedDistance / 0.92))

    val centroidScore = latLngCentroid(sampledRoute)?.let { routeCenter ->
        val centroidDrift = osmHaversineDistanceMeters(
            shapeCenter.first,
            shapeCenter.second,
            routeCenter.first,
            routeCenter.second,
        ) / shapeRadius
        osmClampUnit(1.0 - (centroidDrift / 0.80))
    } ?: 0.0

    val corridorScore = shapeCorridorFitScore(
        routePoints = sampledRoute,
        shapePoints = sampledShape,
        centerLat = shapeCenter.first,
        centerLng = shapeCenter.second,
        shapeRadiusMeters = shapeRadius,
    )
    val shapeLengthKm = polylineDistanceKmFromLatLng(sampledShape)
    val routeLengthKm = polylineDistanceKmFromLatLng(sampledRoute)
    val lengthScore = shapeLengthSimilarityScore(routeLengthKm, shapeLengthKm)

    val rawScore = contourScore * 0.05 +
        anchoredScore * 0.30 +
        orderedScore * 0.28 +
        centroidScore * 0.10 +
        corridorScore * 0.22 +
        lengthScore * 0.05
    val score = min(rawScore, 0.54 + centroidScore * 0.28 + corridorScore * 0.18)
    return ShapeSimilarityBreakdown(
        score = osmClampUnit(score),
        contourScore = contourScore,
        anchoredScore = anchoredScore,
        orderedScore = orderedScore,
        centroidScore = centroidScore,
        corridorScore = corridorScore,
        lengthScore = lengthScore,
    )
}

internal fun shapeCorridorFitScore(
    routePoints: List<List<Double>>,
    shapePoints: List<List<Double>>,
    centerLat: Double,
    centerLng: Double,
    shapeRadiusMeters: Double,
): Double {
    val routeMeters = projectLatLngToMeters(routePoints, centerLat, centerLng)
    val shapeMeters = projectLatLngToMeters(shapePoints, centerLat, centerLng)
    if (routeMeters.size < 2 || shapeMeters.size < 2) {
        return 0.0
    }
    val safeShapeRadius = shapeRadiusMeters.coerceAtLeast(1.0)
    val routeMean = meanNearestPolylineDistanceMeters(routeMeters, shapeMeters)
    val shapeMean = meanNearestPolylineDistanceMeters(shapeMeters, routeMeters)
    val routeTail = percentileNearestPolylineDistanceMeters(routeMeters, shapeMeters, 0.90)
    val shapeTail = percentileNearestPolylineDistanceMeters(shapeMeters, routeMeters, 0.90)
    val meanDistance = (routeMean + shapeMean) / 2.0
    val tailDistance = max(routeTail, shapeTail)

    val meanTolerance = min(420.0, max(120.0, safeShapeRadius * 0.24))
    val tailTolerance = min(760.0, max(240.0, safeShapeRadius * 0.48))
    val meanScore = osmClampUnit(1.0 - meanDistance / meanTolerance)
    val tailScore = osmClampUnit(1.0 - tailDistance / tailTolerance)
    return osmClampUnit(meanScore * 0.72 + tailScore * 0.28)
}

internal fun projectLatLngToMeters(
    points: List<List<Double>>,
    centerLat: Double,
    centerLng: Double,
): List<NormalizedShapePoint> {
    val cosLat = cos(Math.toRadians(centerLat))
    return points.mapNotNull { point ->
        if (point.size < 2) {
            null
        } else {
            NormalizedShapePoint(
                x = (point[1] - centerLng) * 111320.0 * cosLat,
                y = (point[0] - centerLat) * 111320.0,
            )
        }
    }
}

internal fun meanNearestPolylineDistanceMeters(
    from: List<NormalizedShapePoint>,
    to: List<NormalizedShapePoint>,
): Double {
    if (from.isEmpty() || to.size < 2) return Double.MAX_VALUE
    return from.sumOf { point -> nearestPolylineDistanceMeters(point, to) } / from.size.toDouble()
}

internal fun percentileNearestPolylineDistanceMeters(
    from: List<NormalizedShapePoint>,
    to: List<NormalizedShapePoint>,
    percentile: Double,
): Double {
    if (from.isEmpty() || to.size < 2) return Double.MAX_VALUE
    val distances = from.map { point -> nearestPolylineDistanceMeters(point, to) }.sorted()
    val index = (ceil(osmClampUnit(percentile) * distances.size.toDouble()).toInt() - 1)
        .coerceIn(0, distances.lastIndex)
    return distances[index]
}

internal fun nearestPolylineDistanceMeters(
    point: NormalizedShapePoint,
    polyline: List<NormalizedShapePoint>,
): Double {
    if (polyline.isEmpty()) return Double.MAX_VALUE
    if (polyline.size == 1) return euclideanDistanceMeters(point, polyline.first())
    var minDistance = Double.MAX_VALUE
    for (index in 0 until polyline.lastIndex) {
        val distance = pointToSegmentDistanceMeters(point, polyline[index], polyline[index + 1])
        if (distance < minDistance) {
            minDistance = distance
        }
    }
    return minDistance
}

internal fun pointToSegmentDistanceMeters(
    point: NormalizedShapePoint,
    start: NormalizedShapePoint,
    end: NormalizedShapePoint,
): Double {
    val dx = end.x - start.x
    val dy = end.y - start.y
    val lengthSquared = dx * dx + dy * dy
    if (lengthSquared <= 0.0) {
        return euclideanDistanceMeters(point, start)
    }
    val t = (((point.x - start.x) * dx + (point.y - start.y) * dy) / lengthSquared).coerceIn(0.0, 1.0)
    val projection = NormalizedShapePoint(
        x = start.x + dx * t,
        y = start.y + dy * t,
    )
    return euclideanDistanceMeters(point, projection)
}

internal fun euclideanDistanceMeters(left: NormalizedShapePoint, right: NormalizedShapePoint): Double {
    val dx = left.x - right.x
    val dy = left.y - right.y
    return sqrt(dx * dx + dy * dy)
}

internal fun normalizeShapePolyline(points: List<List<Double>>): List<NormalizedShapePoint> {
    if (points.isEmpty()) return emptyList()
    var sumLat = 0.0
    var sumLng = 0.0
    var count = 0
    points.forEach { point ->
        if (point.size < 2) return@forEach
        sumLat += point[0]
        sumLng += point[1]
        count++
    }
    if (count == 0) return emptyList()
    val centerLat = sumLat / count.toDouble()
    val centerLng = sumLng / count.toDouble()
    val cosLat = cos(Math.toRadians(centerLat))
    var maxRadius = 0.0
    val normalized = mutableListOf<NormalizedShapePoint>()
    points.forEach { point ->
        if (point.size < 2) return@forEach
        val x = (point[1] - centerLng) * 111320.0 * cosLat
        val y = (point[0] - centerLat) * 111320.0
        val radius = sqrt(x * x + y * y)
        if (radius > maxRadius) {
            maxRadius = radius
        }
        normalized += NormalizedShapePoint(x = x, y = y)
    }
    if (maxRadius < 1.0) {
        maxRadius = 1.0
    }
    normalized.forEach { point ->
        point.x /= maxRadius
        point.y /= maxRadius
    }
    return normalized
}

internal fun latLngCentroid(points: List<List<Double>>): Pair<Double, Double>? {
    var sumLat = 0.0
    var sumLng = 0.0
    var count = 0
    points.forEach { point ->
        if (point.size < 2) return@forEach
        sumLat += point[0]
        sumLng += point[1]
        count++
    }
    if (count == 0) {
        return null
    }
    return (sumLat / count.toDouble()) to (sumLng / count.toDouble())
}

internal fun maxLatLngRadiusMeters(points: List<List<Double>>, centerLat: Double, centerLng: Double): Double {
    var maxRadius = 0.0
    points.forEach { point ->
        if (point.size < 2) return@forEach
        val radius = osmHaversineDistanceMeters(centerLat, centerLng, point[0], point[1])
        if (radius > maxRadius) {
            maxRadius = radius
        }
    }
    return maxRadius
}

internal fun projectLatLngToShapeSpace(
    points: List<List<Double>>,
    centerLat: Double,
    centerLng: Double,
    scaleMeters: Double,
): List<NormalizedShapePoint> {
    val scale = max(1.0, scaleMeters)
    val cosLat = cos(Math.toRadians(centerLat))
    return points.mapNotNull { point ->
        if (point.size < 2) {
            null
        } else {
            NormalizedShapePoint(
                x = (point[1] - centerLng) * 111320.0 * cosLat / scale,
                y = (point[0] - centerLat) * 111320.0 / scale,
            )
        }
    }
}

internal fun meanNearestShapeDistance(from: List<NormalizedShapePoint>, to: List<NormalizedShapePoint>): Double {
    if (from.isEmpty() || to.isEmpty()) return 1.0
    var total = 0.0
    from.forEach { left ->
        var minDistance = Double.MAX_VALUE
        to.forEach { right ->
            val dx = left.x - right.x
            val dy = left.y - right.y
            val distance = sqrt(dx * dx + dy * dy)
            if (distance < minDistance) {
                minDistance = distance
            }
        }
        total += minDistance
    }
    return total / from.size.toDouble()
}

internal fun meanIndexedShapeDistance(left: List<NormalizedShapePoint>, right: List<NormalizedShapePoint>): Double {
    val count = min(left.size, right.size)
    if (count == 0) {
        return 1.0
    }
    var total = 0.0
    for (index in 0 until count) {
        val dx = left[index].x - right[index].x
        val dy = left[index].y - right[index].y
        total += sqrt(dx * dx + dy * dy)
    }
    return total / count.toDouble()
}

internal fun routeShapeScore(recommendation: RouteRecommendation): Double {
    return osmClampUnit(recommendation.shapeScore ?: 0.0)
}

internal fun geometrySignature(points: List<List<Double>>): String {
    if (points.isEmpty()) return ""
    val step = if (points.size > 60) max(1, points.size / 60) else 1
    return buildString {
        points.indices.step(step).forEach { idx ->
            val point = points[idx]
            if (point.size >= 2) {
                append("%.5f,%.5f|".format(Locale.US, point[0], point[1]))
            }
        }
    }
}

internal fun formatDistanceDelta(deltaKm: Double): String {
    val absolute = abs(deltaKm)
    return if (absolute < 1.0) {
        "${round(absolute * 1000.0).toInt()} m"
    } else {
        "${"%.2f".format(Locale.US, absolute)} km"
    }
}

internal fun formatElevationDelta(deltaM: Double): String {
    return "${round(abs(deltaM)).toInt()} m"
}

internal fun clampScore(value: Double): Double {
    val normalized = min(100.0, max(0.0, value))
    return round(normalized * 10.0) / 10.0
}

internal fun seasonFromDate(date: Instant): String {
    return when (date.atZone(ZoneOffset.UTC).monthValue) {
        12, 1, 2 -> "WINTER"
        3, 4, 5 -> "SPRING"
        6, 7, 8 -> "SUMMER"
        else -> "AUTUMN"
    }
}
