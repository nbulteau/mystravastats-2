package me.nicolas.stravastats.domain.services.statistics

import me.nicolas.stravastats.domain.business.ActivityEffort
import me.nicolas.stravastats.domain.business.ActivityShort
import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.business.strava.StravaDetailedActivity
import me.nicolas.stravastats.domain.business.strava.stream.Stream
import me.nicolas.stravastats.domain.utils.formatSeconds


internal open class BestEffortTimeStatistic(
    name: String,
    activities: List<StravaActivity>,
    private val seconds: Int,
) : ActivityStatistic(name, activities) {

    private val bestActivityEffort = activities
        .mapNotNull { activity -> activity.calculateBestDistanceForTime(seconds) }
        .maxByOrNull { activityEffort -> activityEffort.distance }

    init {
        require(seconds > 10) { "DistanceStream must be > 10 seconds" }
        activity = bestActivityEffort?.activityShort
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

fun StravaActivity.calculateBestDistanceForTime(seconds: Int): ActivityEffort? {
    // no stream -> return null
    return if (stream == null || stream?.altitude == null) {
        null
    } else {
        activityEffort(this.id, this.name, this.type, this.stream!!, seconds)
    }
}

fun StravaDetailedActivity.calculateBestDistanceForTime(seconds: Int): ActivityEffort? {
    // no stream -> return null
    return if (stream == null || stream?.altitude == null) {
        null
    } else {
        activityEffort(this.id, this.name, this.type, this.stream!!, seconds)
    }
}

/**
 * Sliding window best distance for a given time
 * @param seconds given time
 */
private fun activityEffort(
    id: Long,
    name: String,
    type: String,
    stream: Stream,
    seconds: Int
): ActivityEffort? {
    var idxStart = 0
    var idxEnd = 0
    var maxDist = 0.0
    var bestEffort: ActivityEffort? = null

    val distances = stream.distance.data
    val times = stream.time.data
    val altitudes = stream.altitude?.data!!

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
                val averagePower = stream.watts?.data?.let { watts ->
                    (idxStart..idxEnd).sumOf { watts[it] } / (idxEnd - idxStart)
                }
                bestEffort = ActivityEffort(
                    maxDist, seconds, totalAltitude, idxStart, idxEnd, averagePower,
                    label = "Best distance for ${seconds.formatSeconds()}",
                    activityShort = ActivityShort(
                        id = id,
                        name = name,
                        type = type
                    )
                )
            }
            ++idxStart
        }
    } while (idxEnd < streamDataSize)

    return bestEffort
}
