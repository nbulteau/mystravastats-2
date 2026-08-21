package me.nicolas.stravastats.domain.services.statistics

import me.nicolas.stravastats.domain.business.strava.StravaActivity

private const val DEFAULT_PLAUSIBLE_SPEED_MS = 35.0

private fun plausibleSpeedThreshold(activityType: String): Double = when (activityType) {
    "Run", "TrailRun" -> 12.0
    "Hike", "Walk" -> 7.0
    "AlpineSki" -> 45.0
    else -> DEFAULT_PLAUSIBLE_SPEED_MS
}

internal fun isPlausibleActivityMaxSpeed(activity: StravaActivity): Boolean {
    val speed = activity.maxSpeed.toDouble()
    return speed.isFinite() && speed > 0.0 && speed <= plausibleSpeedThreshold(activity.type) * 1.3
}

internal fun invalidSpeedSegmentPrefix(
    distances: List<Double>,
    times: List<Int>,
    size: Int,
    activityType: String,
): IntArray {
    val prefix = IntArray(size)
    val threshold = plausibleSpeedThreshold(activityType)
    for (index in 1 until size) {
        prefix[index] = prefix[index - 1]
        val deltaDistance = distances[index] - distances[index - 1]
        val deltaSeconds = times[index] - times[index - 1]
        val invalid = !deltaDistance.isFinite() ||
            deltaDistance < 0.0 ||
            deltaSeconds < 0 ||
            (deltaSeconds == 0 && deltaDistance > 0.5) ||
            (deltaSeconds > 0 && deltaDistance / deltaSeconds > threshold)
        if (invalid) {
            prefix[index]++
        }
    }
    return prefix
}

internal fun hasInvalidSpeedSegment(prefix: IntArray, start: Int, end: Int): Boolean {
    if (start < 0 || end <= start || end >= prefix.size) {
        return false
    }
    return prefix[end] - prefix[start] > 0
}
