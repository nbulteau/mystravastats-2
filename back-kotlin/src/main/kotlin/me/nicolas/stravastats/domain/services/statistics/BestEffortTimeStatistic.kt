package me.nicolas.stravastats.domain.services.statistics

import me.nicolas.stravastats.domain.business.ActivityEffort
import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.utils.formatSeconds
import me.nicolas.stravastats.domain.utils.formatSpeed


internal open class BestEffortTimeStatistic(
    name: String,
    activities: List<StravaActivity>,
    private val seconds: Int,
) : ActivityStatistic(name, activities) {

    private val bestActivityEffort = activities
        .mapNotNull { activity -> activity.calculateBestDistanceForTime(seconds) }
        .maxByOrNull { activityEffort -> activityEffort.distance }

    init {
        require(seconds > 10) { "Distance must be > 10 seconds" }
        stravaActivity = bestActivityEffort?.stravaActivity
    }

    override val value: String
        get() = if (bestActivityEffort != null) {
            if (bestActivityEffort.distance > 1000) {
                "%.2f km => ${bestActivityEffort.getFormattedSpeed()}".format(bestActivityEffort.distance / 1000)
            } else {
                "%.0f m => ${bestActivityEffort.getFormattedSpeed()}".format(bestActivityEffort.distance)
            }
        } else {
            "Not available"
        }

    protected open fun result(bestActivityEffort: ActivityEffort) =
        if (bestActivityEffort.distance > 1000) {
            "%.2f km => ${bestActivityEffort.getFormattedSpeed()}".format(bestActivityEffort.distance / 1000)
        } else {
            "%.0f m => ${bestActivityEffort.getFormattedSpeed()}".format(bestActivityEffort.distance)
        }
}

/**
 * Sliding window best distance for a given time
 * @param seconds given time
 */
fun StravaActivity.calculateBestDistanceForTime(seconds: Int): ActivityEffort? {

    // no stream -> return null
    if (stream == null || stream?.altitude == null) {
        return null
    }

    var idxStart = 0
    var idxEnd = 0
    var maxDist = 0.0
    var bestEffort: ActivityEffort? = null

    val distances = this.stream?.distance?.data!!
    val times = this.stream?.time?.data!!
    val altitudes = this.stream?.altitude?.data!!

    val streamDataSize = distances.size

    do {
        val totalDistance = distances[idxEnd] - distances[idxStart]
        val totalAltitude = if (altitudes.isNotEmpty()) {
            altitudes[idxEnd] - altitudes[idxStart]
        } else {
            0.0
        }
        val totalTime = times[idxEnd] - times[idxStart]

        if (totalTime < seconds) {
            ++idxEnd
        } else {
            val estimatedDistanceForTime = totalDistance / totalTime * seconds

            if (estimatedDistanceForTime > maxDist) {
                maxDist = estimatedDistanceForTime
                val speed = maxDist / totalTime
                bestEffort = ActivityEffort(
                    this, maxDist, seconds, totalAltitude, idxStart, idxEnd,
                    null,
                    description = "Best distance for ${seconds.formatSeconds()}: %.2f km => ${speed.formatSpeed(this.type)}".format(maxDist / 1000)
                )
            }
            ++idxStart
        }
    } while (idxEnd < streamDataSize)

    return bestEffort
}
