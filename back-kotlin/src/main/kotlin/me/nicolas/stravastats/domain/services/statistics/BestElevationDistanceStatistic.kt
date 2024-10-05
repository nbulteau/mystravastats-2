package me.nicolas.stravastats.domain.services.statistics

import me.nicolas.stravastats.domain.business.strava.Activity
import me.nicolas.stravastats.domain.business.strava.ActivityEffort


internal open class BestElevationDistanceStatistic(
    name: String,
    activities: List<Activity>,
    private val distance: Double,
) : ActivityStatistic(name, activities) {

    private val bestActivityEffort = activities
        .mapNotNull { activity -> activity.calculateBestElevationForDistance(distance) }
        .maxByOrNull { activityEffort -> activityEffort.deltaAltitude }

    init {
        require(distance > 100) { "Distance must be > 100 meters" }
        activity = bestActivityEffort?.activity
    }

    override val value: String
        get() = bestActivityEffort?.getFormattedGradient() ?: "Not available"
}

/**
 * Sliding window looking for best elevation gain for a given distance.
 * @param distance given distance.
 */
fun Activity.calculateBestElevationForDistance(distance: Double): ActivityEffort? {

    // no stream -> return null
    if (stream == null || stream?.altitude == null) {
        return null
    }

    var idxStart = 0
    var idxEnd = 0
    var bestElevation = Double.MIN_VALUE
    var bestEffort: ActivityEffort? = null

    val distances = this.stream?.distance?.data!!
    val times = this.stream?.time?.data!!
    val altitudes = this.stream?.altitude?.data!!

    val streamDataSize = this.stream?.distance?.originalSize!!

    do {
        val totalDistance = distances[idxEnd] - distances[idxStart]
        val totalAltitude = if (altitudes.isNotEmpty()) {
            altitudes[idxEnd] - altitudes[idxStart]
        } else {
            0.0
        }
        val totalTime = times[idxEnd] - times[idxStart]

        if (totalDistance < distance - 0.5) { // 999.6 m will count towards 1 km
            ++idxEnd
        } else {
            if (totalAltitude > bestElevation) {
                bestElevation = totalAltitude
                bestEffort = ActivityEffort(this, distance, totalTime, bestElevation, idxStart, idxEnd)
            }
            ++idxStart
        }
    } while (idxEnd < streamDataSize)

    return bestEffort
}