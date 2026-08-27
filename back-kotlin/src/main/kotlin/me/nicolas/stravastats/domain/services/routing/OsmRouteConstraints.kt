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

internal fun startsNearRequestedStart(
    points: List<List<Double>>,
    start: Coordinates,
    toleranceMeters: Double,
): Boolean {
    if (points.isEmpty()) return false
    val first = points.first()
    if (first.size < 2) return false
    return osmHaversineDistanceMeters(first[0], first[1], start.lat, start.lng) <= toleranceMeters
}

internal fun combinedDirectionPenalty(
    points: List<List<Double>>,
    start: Coordinates,
    direction: String?,
    toleranceMeters: Double,
): Double {
    if (direction.isNullOrBlank()) {
        return 0.0
    }
    // We combine three direction signals:
    // - initial heading alignment (bearing-based)
    // - half-plane violations (did the route go too much in the opposite side)
    // - global lobe dominance (does the whole loop stay mostly in requested direction)
    // Taking the max keeps direction enforcement robust in dense urban grids.
    // Bearing is intentionally softened because local street orientation near
    // the start can be briefly opposite to the desired global direction.
    val bearingPenalty = directionPenaltyFromPreview(points, direction)
    val halfPlanePenalty = halfPlaneViolationRatio(points, start, direction, toleranceMeters)
    val lobePenalty = directionalLobePenalty(points, start, direction)
    val farOppositePenalty = farOppositeViolationRatio(points, start, direction, toleranceMeters)
    val quadrantPenalty = directionalQuadrantPenalty(points, start, direction, toleranceMeters)
    return max(
        max(
            max(bearingPenalty * 0.65, halfPlanePenalty),
            max(lobePenalty, farOppositePenalty),
        ),
        quadrantPenalty,
    )
}

internal fun halfPlaneViolationRatio(
    points: List<List<Double>>,
    start: Coordinates,
    direction: String?,
    toleranceMeters: Double,
): Double {
    val normalized = direction.orEmpty().trim().uppercase(Locale.getDefault())
    if (normalized.isBlank() || points.isEmpty()) return 0.0

    val latTolerance = toleranceMeters / 111320.0
    val lngTolerance = toleranceMeters / max(1000.0, 111320.0 * cos(Math.toRadians(start.lat)))
    var total = 0
    var violations = 0

    for (point in points) {
        if (point.size < 2) continue
        total++
        when (normalized) {
            "N" -> if (point[0] < start.lat - latTolerance) violations++
            "S" -> if (point[0] > start.lat + latTolerance) violations++
            "E" -> if (point[1] < start.lng - lngTolerance) violations++
            "W" -> if (point[1] > start.lng + lngTolerance) violations++
        }
    }
    if (total == 0) return 0.0
    return violations.toDouble() / total.toDouble()
}

internal fun directionalLobePenalty(
    points: List<List<Double>>,
    start: Coordinates,
    direction: String?,
): Double {
    val normalized = direction.orEmpty().trim().uppercase(Locale.getDefault())
    if (normalized.isBlank() || points.isEmpty()) return 0.0

    var desiredExtent = 0.0
    var oppositeExtent = 0.0
    var sumProjection = 0.0
    var projectionCount = 0

    for (point in points) {
        if (point.size < 2) continue
        val projection = directionProjectionMeters(point[0], point[1], start, normalized) ?: continue
        if (projection > desiredExtent) {
            desiredExtent = projection
        }
        if (projection < 0 && -projection > oppositeExtent) {
            oppositeExtent = -projection
        }
        sumProjection += projection
        projectionCount++
    }

    if (projectionCount == 0) return 0.0

    // Dominance asks: "how much of the route envelope is on requested side?"
    // 1.0 means full dominance on requested side, 0.5 is symmetric, 0 is opposite.
    var dominancePenalty = 0.0
    val totalExtent = desiredExtent + oppositeExtent
    if (totalExtent > 1.0) {
        val dominanceRatio = desiredExtent / totalExtent
        // Keep a clearer direction dominance in dense grids.
        dominancePenalty = osmClampUnit((0.68 - dominanceRatio) / 0.68)
    }

    // Average projection guard: route center of mass should not drift opposite.
    var avgPenalty = 0.0
    if (desiredExtent > 1.0) {
        val avgProjection = sumProjection / projectionCount.toDouble()
        avgPenalty = osmClampUnit((-avgProjection) / max(desiredExtent * 0.25, 1.0))
    }

    return max(dominancePenalty, avgPenalty)
}

internal fun farOppositeViolationRatio(
    points: List<List<Double>>,
    start: Coordinates,
    direction: String?,
    toleranceMeters: Double,
): Double {
    val normalized = direction.orEmpty().trim().uppercase(Locale.getDefault())
    if (normalized.isBlank() || points.isEmpty()) return 0.0

    val guardBand = max(toleranceMeters * 1.8, 220.0)
    var total = 0
    var violations = 0

    for (point in points) {
        if (point.size < 2) continue
        val projection = directionProjectionMeters(point[0], point[1], start, normalized) ?: continue
        if (abs(projection) < guardBand) {
            // Ignore local oscillations around start/return hub.
            continue
        }
        total++
        if (projection < -guardBand) {
            violations++
        }
    }
    if (total == 0) return 0.0
    return violations.toDouble() / total.toDouble()
}

internal fun directionalQuadrantPenalty(
    points: List<List<Double>>,
    start: Coordinates,
    direction: String?,
    toleranceMeters: Double,
): Double {
    val normalized = direction.orEmpty().trim().uppercase(Locale.getDefault())
    if (normalized.isBlank() || points.size < 2) return 0.0

    // Ignore local oscillations around start and focus on dominant travel zones.
    val guardBand = max(toleranceMeters * 1.2, 160.0)
    var desiredMeters = 0.0
    var oppositeMeters = 0.0

    for (index in 0 until points.size - 1) {
        val from = points[index]
        val to = points[index + 1]
        if (from.size < 2 || to.size < 2) continue
        val segmentMeters = osmHaversineDistanceMeters(from[0], from[1], to[0], to[1])
        if (segmentMeters < 12.0) continue

        val midLat = (from[0] + to[0]) / 2.0
        val midLng = (from[1] + to[1]) / 2.0
        val projection = directionProjectionMeters(midLat, midLng, start, normalized) ?: continue
        if (abs(projection) < guardBand) continue

        if (projection >= 0.0) {
            desiredMeters += segmentMeters
        } else {
            oppositeMeters += segmentMeters
        }
    }

    val totalMeters = desiredMeters + oppositeMeters
    if (totalMeters <= 0.0) return 0.0
    val desiredRatio = desiredMeters / totalMeters
    // Keep at least ~62% of routed distance in requested quadrant.
    return osmClampUnit((0.62 - desiredRatio) / 0.62)
}

internal fun directionProjectionMeters(
    lat: Double,
    lng: Double,
    start: Coordinates,
    normalizedDirection: String,
): Double? {
    val latMeters = (lat - start.lat) * 111320.0
    val lngMeters = (lng - start.lng) * 111320.0 * cos(Math.toRadians(start.lat))
    return when (normalizedDirection) {
        "N" -> latMeters
        "S" -> -latMeters
        "E" -> lngMeters
        "W" -> -lngMeters
        else -> null
    }
}

internal fun osmClampUnit(value: Double): Double {
    return when {
        value <= 0.0 -> 0.0
        value >= 1.0 -> 1.0
        else -> value
    }
}

internal fun corridorOverlapRatio(points: List<List<Double>>): Double {
    if (points.size < 4) return 0.0
    val sampled = samplePolylinePoints(points, 260)
    val segments = buildPathSegments(sampled)
    if (segments.size < 2) return 0.0

    val flagged = BooleanArray(segments.size)
    for (i in segments.indices) {
        // Skip only immediate neighbors to avoid counting normal local curvature as overlap.
        for (j in 0 until (i - 1).coerceAtLeast(0)) {
            if (segmentsLikelySameCorridor(segments[i], segments[j])) {
                flagged[i] = true
                flagged[j] = true
            }
        }
    }
    val overlapped = flagged.count { it }
    return overlapped.toDouble() / segments.size.toDouble()
}

internal fun samplePolylinePoints(points: List<List<Double>>, maxPoints: Int): List<List<Double>> {
    if (points.size <= maxPoints || maxPoints <= 0) {
        return points
    }
    val step = max(1, ceil(points.size.toDouble() / maxPoints.toDouble()).toInt())
    val sampled = mutableListOf<List<Double>>()
    for (index in points.indices step step) {
        sampled += points[index]
    }
    val lastPoint = points.last()
    val lastSample = sampled.lastOrNull()
    if (lastSample == null || lastSample.size < 2 || lastPoint.size < 2 ||
        lastSample[0] != lastPoint[0] || lastSample[1] != lastPoint[1]
    ) {
        sampled += lastPoint
    }
    return sampled
}

internal fun buildPathSegments(points: List<List<Double>>): List<PathSegment> {
    val segments = mutableListOf<PathSegment>()
    for (index in 0 until points.size - 1) {
        val from = points[index]
        val to = points[index + 1]
        if (from.size < 2 || to.size < 2) continue

        val lengthM = osmHaversineDistanceMeters(from[0], from[1], to[0], to[1])
        if (lengthM < 12.0) continue

        segments += PathSegment(
            startLat = from[0],
            startLng = from[1],
            endLat = to[0],
            endLng = to[1],
            midLat = (from[0] + to[0]) / 2.0,
            midLng = (from[1] + to[1]) / 2.0,
            lengthM = lengthM,
            bearing = bearingDegrees(from[0], from[1], to[0], to[1]),
        )
    }
    return segments
}

internal fun segmentsLikelySameCorridor(left: PathSegment, right: PathSegment): Boolean {
    val midpointToleranceMeters = 50.0
    val endpointToleranceMeters = 80.0
    val midpointDistance = osmHaversineDistanceMeters(left.midLat, left.midLng, right.midLat, right.midLng)
    if (midpointDistance > midpointToleranceMeters) return false

    val leftToRightStart = osmHaversineDistanceMeters(left.startLat, left.startLng, right.startLat, right.startLng)
    val leftToRightEnd = osmHaversineDistanceMeters(left.startLat, left.startLng, right.endLat, right.endLng)
    val rightToLeftStart = osmHaversineDistanceMeters(left.endLat, left.endLng, right.startLat, right.startLng)
    val rightToLeftEnd = osmHaversineDistanceMeters(left.endLat, left.endLng, right.endLat, right.endLng)
    if (
        min(leftToRightStart, leftToRightEnd) > endpointToleranceMeters ||
        min(rightToLeftStart, rightToLeftEnd) > endpointToleranceMeters
    ) {
        return false
    }

    var bearingDiff = abs(left.bearing - right.bearing)
    if (bearingDiff > 180.0) bearingDiff = 360.0 - bearingDiff
    if (bearingDiff > 22.0 && bearingDiff < 158.0) return false

    val maxLength = max(left.lengthM, right.lengthM)
    val minLength = min(left.lengthM, right.lengthM)
    if (minLength <= 0.0 || maxLength / minLength > 6.0) return false
    return true
}

internal data class AxisTraversal(
    val axisId: String,
    val isForward: Boolean,
)

internal data class AxisUsageSummary(
    val totalTraversals: Int,
    val uniqueAxisCount: Int,
    val conflictingAxisCount: Int,
    val reusedTraversals: Int,
    val maxAxisReuseCount: Int,
) {
    fun oppositeTraversalRatio(): Double {
        if (totalTraversals == 0) return 0.0
        return conflictingAxisCount.toDouble() / totalTraversals.toDouble()
    }

    fun reuseRatio(): Double {
        if (totalTraversals == 0) return 0.0
        return reusedTraversals.toDouble() / totalTraversals.toDouble()
    }

    fun segmentDiversityRatio(): Double {
        if (totalTraversals == 0) return 0.0
        return uniqueAxisCount.toDouble() / totalTraversals.toDouble()
    }

    fun maxAxisReuseRatio(): Double {
        if (totalTraversals == 0) return 0.0
        return maxAxisReuseCount.toDouble() / totalTraversals.toDouble()
    }
}

internal fun evaluateAxisUsage(points: List<List<Double>>): AxisUsageSummary {
    val traversals = extractAxisTraversals(points)
    if (traversals.isEmpty()) {
        return AxisUsageSummary(
            totalTraversals = 0,
            uniqueAxisCount = 0,
            conflictingAxisCount = 0,
            reusedTraversals = 0,
            maxAxisReuseCount = 0,
        )
    }
    val axisCounts = mutableMapOf<String, Int>()
    val axisDirections = mutableMapOf<String, Int>()
    var maxAxisReuseCount = 0

    traversals.forEach { traversal ->
        val count = (axisCounts[traversal.axisId] ?: 0) + 1
        axisCounts[traversal.axisId] = count
        if (count > maxAxisReuseCount) {
            maxAxisReuseCount = count
        }
        val currentDirectionMask = axisDirections[traversal.axisId] ?: 0
        val updatedDirectionMask = if (traversal.isForward) {
            currentDirectionMask or 0b01
        } else {
            currentDirectionMask or 0b10
        }
        axisDirections[traversal.axisId] = updatedDirectionMask
    }

    var conflictingAxisCount = 0
    var reusedTraversals = 0
    axisCounts.forEach { (axisId, count) ->
        if ((axisDirections[axisId] ?: 0) == 0b11) {
            conflictingAxisCount++
        }
        if (count > 1) {
            reusedTraversals += count - 1
        }
    }

    return AxisUsageSummary(
        totalTraversals = traversals.size,
        uniqueAxisCount = axisCounts.size,
        conflictingAxisCount = conflictingAxisCount,
        reusedTraversals = reusedTraversals,
        maxAxisReuseCount = maxAxisReuseCount,
    )
}

internal fun extractAxisTraversals(points: List<List<Double>>): List<AxisTraversal> {
    if (points.size < 3) return emptyList()
    return buildList(points.size - 1) {
        for (index in 0 until points.size - 1) {
            val from = points[index]
            val to = points[index + 1]
            if (from.size < 2 || to.size < 2) continue
            val fromId = quantizedPointKey(from[0], from[1])
            val toId = quantizedPointKey(to[0], to[1])
            if (fromId == toId) continue
            add(
                AxisTraversal(
                    axisId = canonicalEdgeKey(fromId, toId),
                    isForward = fromId < toId,
                ),
            )
        }
    }
}

internal fun evaluateAxisReuseOutsideStartZone(
    points: List<List<Double>>,
    start: Coordinates,
    startZoneMeters: Double,
    minOppositeMeters: Double,
): Triple<Boolean, Int, Double> {
    if (points.size < 2) return Triple(false, 0, 0.0)

    data class LocalAxisUsage(
        var count: Int = 0,
        var directionMask: Int = 0,
        var forwardMeters: Double = 0.0,
        var reverseMeters: Double = 0.0,
    )

    val axisUsage = mutableMapOf<String, LocalAxisUsage>()
    var maxReuseOutsideStart = 0
    var outsideTotalMeters = 0.0

    for (index in 0 until points.size - 1) {
        val from = points[index]
        val to = points[index + 1]
        if (from.size < 2 || to.size < 2) continue

        val midLat = (from[0] + to[0]) / 2.0
        val midLng = (from[1] + to[1]) / 2.0
        val midDistance = osmHaversineDistanceMeters(midLat, midLng, start.lat, start.lng)
        if (midDistance <= startZoneMeters) {
            // Reuse around start/finish hub is allowed.
            // Midpoint classification avoids exempting long segments that
            // cross the hub boundary and then retrace outside it.
            continue
        }

        val fromId = quantizedPointKey(from[0], from[1])
        val toId = quantizedPointKey(to[0], to[1])
        if (fromId.isBlank() || toId.isBlank() || fromId == toId) continue

        val axisId = canonicalEdgeKey(fromId, toId)
        val segmentMeters = osmHaversineDistanceMeters(from[0], from[1], to[0], to[1])
        if (segmentMeters < MIN_AXIS_SEGMENT_LENGTH_METERS) continue
        val usage = axisUsage.getOrPut(axisId) { LocalAxisUsage() }
        usage.count += 1
        usage.directionMask = if (fromId < toId) {
            usage.forwardMeters += segmentMeters
            usage.directionMask or 0b01
        } else {
            usage.reverseMeters += segmentMeters
            usage.directionMask or 0b10
        }
        outsideTotalMeters += segmentMeters
        if (usage.count > maxReuseOutsideStart) {
            maxReuseOutsideStart = usage.count
        }
    }

    var oppositeMeters = 0.0
    for (usage in axisUsage.values) {
        if (usage.directionMask == 0b11) {
            oppositeMeters += min(usage.forwardMeters, usage.reverseMeters)
        }
    }
    if (outsideTotalMeters <= 0.0) {
        return Triple(false, maxReuseOutsideStart, 0.0)
    }
    val oppositeRatio = osmClampUnit(oppositeMeters / outsideTotalMeters)
    // Ignore tiny opposite-direction artifacts caused by local snap/geometry noise.
    val minimum = max(MIN_OPPOSITE_REUSE_METERS, minOppositeMeters)
    return Triple(oppositeMeters >= minimum, maxReuseOutsideStart, oppositeRatio)
}

internal fun edgeReuseRatio(points: List<List<Double>>): Double {
    return evaluateAxisUsage(points).reuseRatio()
}

internal fun incrementRejectCount(rejectCounts: MutableMap<String, Int>, reason: String) {
    val normalizedReason = reason.trim()
    if (normalizedReason.isBlank()) {
        return
    }
    rejectCounts[normalizedReason] = (rejectCounts[normalizedReason] ?: 0) + 1
}

internal fun formatRejectCounts(rejectCounts: Map<String, Int>): String {
    if (rejectCounts.isEmpty()) {
        return "none"
    }
    return rejectCounts.entries
        .sortedWith(compareByDescending<Map.Entry<String, Int>> { entry -> entry.value }.thenBy { entry -> entry.key })
        .joinToString(", ") { entry -> "${entry.key}=${entry.value}" }
}
