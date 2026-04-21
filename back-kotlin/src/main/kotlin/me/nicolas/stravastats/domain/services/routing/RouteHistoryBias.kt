package me.nicolas.stravastats.domain.services.routing

import me.nicolas.stravastats.domain.business.Coordinates
import java.util.Locale
import kotlin.math.atan2
import kotlin.math.cos
import kotlin.math.sin
import kotlin.math.sqrt

internal const val HISTORY_REUSE_BONUS_WEIGHT = 18.0
internal const val HISTORY_START_ZONE_BONUS_WEIGHT = 14.0
private const val HISTORY_AXIS_BIAS_WEIGHT = 0.75
private const val HISTORY_ZONE_BIAS_WEIGHT = 0.25
private const val HISTORY_AXIS_NODE_PRECISION = 4
private const val HISTORY_ZONE_PRECISION = 2
private const val MIN_HISTORY_SEGMENT_LENGTH_METERS = 25.0
private const val HISTORY_START_ZONE_METERS = 2_000.0

internal data class RoutingHistoryBiasContext(
    val enabled: Boolean = false,
    val normalizedRouteType: String = "",
    val axisScores: Map<String, Double> = emptyMap(),
    val zoneScores: Map<String, Double> = emptyMap(),
    val maxAxisScore: Double = 0.0,
    val maxZoneScore: Double = 0.0,
)

internal fun buildRoutingHistoryBiasContext(request: RoutingEngineRequest): RoutingHistoryBiasContext {
    if (!request.historyBiasEnabled) {
        return RoutingHistoryBiasContext()
    }
    val profile = request.historyProfile ?: return RoutingHistoryBiasContext()
    val normalizedRequestType = normalizeRoutingHistoryRouteType(request.routeType)
    val normalizedProfileType = normalizeRoutingHistoryRouteType(profile.routeType)
    if (normalizedRequestType != normalizedProfileType) {
        return RoutingHistoryBiasContext()
    }
    val maxAxisScore = maxPositiveScore(profile.axisScores)
    val maxZoneScore = maxPositiveScore(profile.zoneScores)
    if (maxAxisScore <= 0.0 && maxZoneScore <= 0.0) {
        return RoutingHistoryBiasContext()
    }
    return RoutingHistoryBiasContext(
        enabled = true,
        normalizedRouteType = normalizedRequestType,
        axisScores = profile.axisScores,
        zoneScores = profile.zoneScores,
        maxAxisScore = maxAxisScore,
        maxZoneScore = maxZoneScore,
    )
}

internal fun sortAnchorsByHistoryReuse(
    anchors: List<Coordinates>,
    start: Coordinates,
    context: RoutingHistoryBiasContext,
): List<Coordinates> {
    if (!context.enabled || anchors.size < 2 || context.maxZoneScore <= 0.0) {
        return anchors
    }
    return anchors
        .mapIndexed { index, anchor ->
            IndexedAnchor(
                anchor = anchor,
                score = historyAnchorReuseScore(anchor, start, context),
                index = index,
            )
        }
        .sortedWith(compareByDescending<IndexedAnchor> { it.score }.thenBy { it.index })
        .map { it.anchor }
}

internal fun computeHistoryReuseScore(points: List<List<Double>>, context: RoutingHistoryBiasContext): Double {
    if (!context.enabled || points.size < 2) {
        return 0.0
    }
    var totalLengthMeters = 0.0
    var axisWeighted = 0.0
    var zoneWeighted = 0.0

    for (index in 1 until points.size) {
        val from = points[index - 1]
        val to = points[index]
        if (from.size < 2 || to.size < 2) {
            continue
        }
        val segmentLengthMeters = haversineDistanceMeters(from[0], from[1], to[0], to[1])
        if (!segmentLengthMeters.isFinite() || segmentLengthMeters < MIN_HISTORY_SEGMENT_LENGTH_METERS) {
            continue
        }
        totalLengthMeters += segmentLengthMeters

        if (context.maxAxisScore > 0.0) {
            val axisId = historyAxisKey(from[0], from[1], to[0], to[1])
            axisWeighted += normalizedHistoryScore(context.axisScores[axisId] ?: 0.0, context.maxAxisScore) * segmentLengthMeters
        }
        if (context.maxZoneScore > 0.0) {
            val midLat = (from[0] + to[0]) / 2.0
            val midLng = (from[1] + to[1]) / 2.0
            val zoneId = historyZoneKey(midLat, midLng)
            zoneWeighted += normalizedHistoryScore(context.zoneScores[zoneId] ?: 0.0, context.maxZoneScore) * segmentLengthMeters
        }
    }

    if (totalLengthMeters <= 0.0) {
        return 0.0
    }
    return blendHistoryReuseRatios(axisWeighted, zoneWeighted, totalLengthMeters, context)
}

internal fun computeHistoryStartZoneReuseScore(
    points: List<List<Double>>,
    start: Coordinates,
    context: RoutingHistoryBiasContext,
): Double {
    if (!context.enabled || points.size < 2) {
        return 0.0
    }
    var totalLengthMeters = 0.0
    var axisWeighted = 0.0
    var zoneWeighted = 0.0

    for (index in 1 until points.size) {
        val from = points[index - 1]
        val to = points[index]
        if (from.size < 2 || to.size < 2) {
            continue
        }
        val segmentLengthMeters = haversineDistanceMeters(from[0], from[1], to[0], to[1])
        if (!segmentLengthMeters.isFinite() || segmentLengthMeters < MIN_HISTORY_SEGMENT_LENGTH_METERS) {
            continue
        }
        val midLat = (from[0] + to[0]) / 2.0
        val midLng = (from[1] + to[1]) / 2.0
        if (haversineDistanceMeters(midLat, midLng, start.lat, start.lng) > HISTORY_START_ZONE_METERS) {
            continue
        }
        totalLengthMeters += segmentLengthMeters

        if (context.maxAxisScore > 0.0) {
            val axisId = historyAxisKey(from[0], from[1], to[0], to[1])
            axisWeighted += normalizedHistoryScore(context.axisScores[axisId] ?: 0.0, context.maxAxisScore) * segmentLengthMeters
        }
        if (context.maxZoneScore > 0.0) {
            val zoneId = historyZoneKey(midLat, midLng)
            zoneWeighted += normalizedHistoryScore(context.zoneScores[zoneId] ?: 0.0, context.maxZoneScore) * segmentLengthMeters
        }
    }

    if (totalLengthMeters <= 0.0) {
        return 0.0
    }
    return blendHistoryReuseRatios(axisWeighted, zoneWeighted, totalLengthMeters, context)
}

private fun blendHistoryReuseRatios(
    axisWeighted: Double,
    zoneWeighted: Double,
    totalLengthMeters: Double,
    context: RoutingHistoryBiasContext,
): Double {
    val hasAxisScores = context.maxAxisScore > 0.0 && context.axisScores.isNotEmpty()
    val hasZoneScores = context.maxZoneScore > 0.0 && context.zoneScores.isNotEmpty()
    val axisRatio = if (hasAxisScores) axisWeighted / totalLengthMeters else 0.0
    val zoneRatio = if (hasZoneScores) zoneWeighted / totalLengthMeters else 0.0
    return when {
        hasAxisScores && hasZoneScores -> clampUnit(axisRatio * HISTORY_AXIS_BIAS_WEIGHT + zoneRatio * HISTORY_ZONE_BIAS_WEIGHT)
        hasAxisScores -> clampUnit(axisRatio)
        hasZoneScores -> clampUnit(zoneRatio)
        else -> 0.0
    }
}

private data class IndexedAnchor(
    val anchor: Coordinates,
    val score: Double,
    val index: Int,
)

private fun normalizeRoutingHistoryRouteType(routeType: String?): String {
    return when (routeType.orEmpty().trim().uppercase(Locale.US)) {
        "RIDE", "MTB", "GRAVEL", "RUN", "TRAIL", "HIKE" -> routeType.orEmpty().trim().uppercase(Locale.US)
        else -> "RIDE"
    }
}

private fun maxPositiveScore(scores: Map<String, Double>): Double {
    var max = 0.0
    for (value in scores.values) {
        if (value.isFinite() && value > max) {
            max = value
        }
    }
    return max
}

private fun historyAnchorReuseScore(anchor: Coordinates, start: Coordinates, context: RoutingHistoryBiasContext): Double {
    val anchorZoneScore = normalizedHistoryZoneScore(anchor.lat, anchor.lng, context)
    val midLat = (anchor.lat + start.lat) / 2.0
    val midLng = (anchor.lng + start.lng) / 2.0
    val midZoneScore = normalizedHistoryZoneScore(midLat, midLng, context)
    return clampUnit(anchorZoneScore * 0.65 + midZoneScore * 0.35)
}

private fun normalizedHistoryZoneScore(lat: Double, lng: Double, context: RoutingHistoryBiasContext): Double {
    if (!context.enabled || context.maxZoneScore <= 0.0) {
        return 0.0
    }
    val zoneId = historyZoneKey(lat, lng)
    return normalizedHistoryScore(context.zoneScores[zoneId] ?: 0.0, context.maxZoneScore)
}

private fun normalizedHistoryScore(score: Double, maxScore: Double): Double {
    if (!score.isFinite() || !maxScore.isFinite() || score <= 0.0 || maxScore <= 0.0) {
        return 0.0
    }
    return clampUnit(score / maxScore)
}

private fun historyAxisKey(lat1: Double, lng1: Double, lat2: Double, lng2: Double): String {
    val from = historyNodeKey(lat1, lng1, HISTORY_AXIS_NODE_PRECISION)
    val to = historyNodeKey(lat2, lng2, HISTORY_AXIS_NODE_PRECISION)
    return if (from <= to) "$from|$to" else "$to|$from"
}

private fun historyZoneKey(lat: Double, lng: Double): String {
    return historyNodeKey(lat, lng, HISTORY_ZONE_PRECISION)
}

private fun historyNodeKey(lat: Double, lng: Double, precision: Int): String {
    return "%.${precision}f:%.${precision}f".format(Locale.US, lat, lng)
}

private fun clampUnit(value: Double): Double {
    return value.coerceIn(0.0, 1.0)
}

private fun haversineDistanceMeters(lat1: Double, lng1: Double, lat2: Double, lng2: Double): Double {
    val earthRadiusMeters = 6_371_000.0
    val dLat = Math.toRadians(lat2 - lat1)
    val dLng = Math.toRadians(lng2 - lng1)
    val sinLat = sin(dLat / 2.0)
    val sinLng = sin(dLng / 2.0)
    val a = sinLat * sinLat + cos(Math.toRadians(lat1)) * cos(Math.toRadians(lat2)) * sinLng * sinLng
    val c = 2.0 * atan2(sqrt(a), sqrt(1.0 - a))
    return earthRadiusMeters * c
}
