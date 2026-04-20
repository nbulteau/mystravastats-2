package me.nicolas.stravastats.domain.services.routing

import me.nicolas.stravastats.domain.business.ActivityType
import me.nicolas.stravastats.domain.business.strava.StravaActivity
import java.time.Instant
import java.time.OffsetDateTime
import java.util.Locale
import kotlin.math.atan2
import kotlin.math.cos
import kotlin.math.exp
import kotlin.math.ln
import kotlin.math.sin
import kotlin.math.sqrt

private const val DEFAULT_HISTORY_HALF_LIFE_DAYS = 75.0
private const val HISTORY_AXIS_NODE_PRECISION = 4
private const val HISTORY_ZONE_PRECISION = 2
private const val MIN_HISTORY_SEGMENT_LENGTH_METERS = 25.0

fun buildRoutingHistoryProfile(
    activities: List<StravaActivity>,
    routeType: String?,
    now: Instant = Instant.now(),
    halfLifeDays: Double = DEFAULT_HISTORY_HALF_LIFE_DAYS,
): RoutingHistoryProfile? {
    if (activities.isEmpty()) {
        return null
    }
    val normalizedRouteType = normalizeHistoryRouteType(routeType)
    val normalizedHalfLifeDays = if (halfLifeDays > 0.0) halfLifeDays else DEFAULT_HISTORY_HALF_LIFE_DAYS

    val axisScores = mutableMapOf<String, Double>()
    val zoneScores = mutableMapOf<String, Double>()
    var activityCount = 0
    var segmentCount = 0
    var latestActivityEpochMs = 0L

    for (activity in activities) {
        if (!historyRouteTypeMatchesActivity(normalizedRouteType, activity)) {
            continue
        }
        val points = extractHistoryTrackPoints(activity)
        if (points.size < 2) {
            continue
        }

        val activityWeight = historyRecencyWeight(activity, now, normalizedHalfLifeDays)
        if (activityWeight <= 0.0) {
            continue
        }

        var activityContributed = false
        for (index in 1 until points.size) {
            val from = points[index - 1]
            val to = points[index]
            val segmentLengthM = haversineDistanceMeters(from.first, from.second, to.first, to.second)
            if (!segmentLengthM.isFinite() || segmentLengthM < MIN_HISTORY_SEGMENT_LENGTH_METERS) {
                continue
            }

            val axisId = historyAxisKey(from.first, from.second, to.first, to.second)
            val zoneId = historyZoneKey((from.first + to.first) / 2.0, (from.second + to.second) / 2.0)
            val contribution = segmentLengthM * activityWeight

            axisScores[axisId] = axisScores.getOrDefault(axisId, 0.0) + contribution
            zoneScores[zoneId] = zoneScores.getOrDefault(zoneId, 0.0) + contribution
            segmentCount += 1
            activityContributed = true
        }

        if (!activityContributed) {
            continue
        }
        activityCount += 1

        parseHistoryActivityInstant(activity)?.toEpochMilli()?.let { epochMs ->
            if (epochMs > latestActivityEpochMs) {
                latestActivityEpochMs = epochMs
            }
        }
    }

    if (activityCount == 0 || segmentCount == 0 || axisScores.isEmpty()) {
        return null
    }

    return RoutingHistoryProfile(
        routeType = normalizedRouteType,
        halfLifeDays = normalizedHalfLifeDays.toInt(),
        activityCount = activityCount,
        segmentCount = segmentCount,
        axisScores = axisScores.toMap(),
        zoneScores = zoneScores.toMap(),
        latestActivityEpochMs = latestActivityEpochMs,
    )
}

private data class HistoryPoint(val first: Double, val second: Double)

private fun normalizeHistoryRouteType(routeType: String?): String {
    return when (routeType.orEmpty().trim().uppercase(Locale.getDefault())) {
        "RIDE", "MTB", "GRAVEL", "RUN", "TRAIL", "HIKE" -> routeType.orEmpty().trim().uppercase(Locale.getDefault())
        else -> "RIDE"
    }
}

private fun historyRouteTypeMatchesActivity(routeType: String, activity: StravaActivity): Boolean {
    val activityType = resolveHistoryActivityType(activity) ?: return false
    return when (routeType) {
        "GRAVEL" -> activityType == ActivityType.GravelRide
        "MTB" -> activityType == ActivityType.MountainBikeRide
        "RUN" -> activityType == ActivityType.Run
        "TRAIL" -> activityType == ActivityType.TrailRun
        "HIKE" -> activityType == ActivityType.Hike || activityType == ActivityType.Walk
        "RIDE" -> activityType == ActivityType.Ride || activityType == ActivityType.Commute || activityType == ActivityType.VirtualRide
        else -> activityType == ActivityType.Ride
    }
}

private fun resolveHistoryActivityType(activity: StravaActivity): ActivityType? {
    val raw = activity.sportType.trim().ifBlank { activity.type.trim() }
    return ActivityType.entries.firstOrNull { candidate -> candidate.name.equals(raw, ignoreCase = true) }
}

private fun extractHistoryTrackPoints(activity: StravaActivity): List<HistoryPoint> {
    val rawPoints = activity.stream?.latlng?.data.orEmpty()
    if (rawPoints.size < 2) {
        return emptyList()
    }
    return rawPoints.mapNotNull { point ->
        if (point.size < 2) {
            return@mapNotNull null
        }
        val lat = point[0]
        val lng = point[1]
        if (!isFiniteCoordinate(lat, lng)) {
            return@mapNotNull null
        }
        HistoryPoint(lat, lng)
    }.let { sanitized ->
        if (sanitized.size < 2) emptyList() else sanitized
    }
}

private fun parseHistoryActivityInstant(activity: StravaActivity): Instant? {
    val raw = activity.startDateLocal.trim().ifBlank { activity.startDate.trim() }
    if (raw.isBlank()) {
        return null
    }
    return runCatching { Instant.parse(raw) }.getOrElse {
        runCatching { OffsetDateTime.parse(raw).toInstant() }.getOrNull()
    }
}

private fun historyRecencyWeight(activity: StravaActivity, now: Instant, halfLifeDays: Double): Double {
    val normalizedHalfLifeDays = if (halfLifeDays > 0.0) halfLifeDays else DEFAULT_HISTORY_HALF_LIFE_DAYS
    val activityInstant = parseHistoryActivityInstant(activity) ?: return 1.0
    if (activityInstant.isAfter(now)) {
        return 1.0
    }
    val ageDays = (now.toEpochMilli() - activityInstant.toEpochMilli()) / 86_400_000.0
    val exponent = -ln(2.0) * ageDays / normalizedHalfLifeDays
    return exp(exponent)
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

private fun isFiniteCoordinate(lat: Double, lng: Double): Boolean {
    if (!lat.isFinite() || !lng.isFinite()) {
        return false
    }
    return lat in -90.0..90.0 && lng in -180.0..180.0
}

private fun haversineDistanceMeters(lat1: Double, lng1: Double, lat2: Double, lng2: Double): Double {
    val earthRadius = 6_371_000.0
    val dLat = Math.toRadians(lat2 - lat1)
    val dLng = Math.toRadians(lng2 - lng1)
    val sinLat = sin(dLat / 2.0)
    val sinLng = sin(dLng / 2.0)
    val a = sinLat * sinLat + cos(Math.toRadians(lat1)) * cos(Math.toRadians(lat2)) * sinLng * sinLng
    val c = 2.0 * atan2(sqrt(a), sqrt(1.0 - a))
    return earthRadius * c
}
