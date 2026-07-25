package me.nicolas.stravastats.domain.services.statistics

import me.nicolas.stravastats.domain.business.strava.StravaActivity

private const val DEFAULT_PLAUSIBLE_MAX_SPEED_MS = 35.0

internal fun invalidSpeedSegmentPrefix(
    distances: List<Double>,
    times: List<Int>,
    streamDataSize: Int,
    activityType: String,
): IntArray {
    if (streamDataSize < 2) {
        return IntArray(streamDataSize + 1)
    }
    val invalidSegments = IntArray(streamDataSize)
    val threshold = plausibleSpeedThreshold(activityType)
    for (index in 1 until streamDataSize) {
        invalidSegments[index] = invalidSegments[index - 1]
        val deltaTime = times[index] - times[index - 1]
        val deltaDistance = distances[index] - distances[index - 1]
        if (deltaTime <= 0 || deltaDistance < 0.0 || deltaDistance / deltaTime > threshold) {
            invalidSegments[index]++
        }
    }
    return invalidSegments
}

internal fun hasInvalidSpeedSegment(invalidSegments: IntArray, idxStart: Int, idxEnd: Int): Boolean {
    if (idxStart < 0 || idxEnd <= idxStart || idxEnd >= invalidSegments.size) {
        return false
    }
    return invalidSegments[idxEnd] > invalidSegments[idxStart]
}

internal fun isPlausibleActivityMaxSpeed(activity: StravaActivity): Boolean {
    val maxSpeed = activity.maxSpeed.toDouble()
    return maxSpeed.isFinite() && maxSpeed > 0.0 && maxSpeed <= plausibleSpeedThreshold(activity.type) * 1.3
}

private fun plausibleSpeedThreshold(activityType: String): Double = when (activityType) {
    "Run", "TrailRun" -> 12.0
    "Hike", "Walk" -> 7.0
    "AlpineSki" -> 45.0
    else -> DEFAULT_PLAUSIBLE_MAX_SPEED_MS
}
