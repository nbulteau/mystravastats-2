package me.nicolas.stravastats.domain.services.statistics

import me.nicolas.stravastats.domain.business.ActivityEffort
import me.nicolas.stravastats.domain.business.ActivityShort
import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.business.strava.StravaDetailedActivity
import me.nicolas.stravastats.domain.business.strava.stream.Stream
import me.nicolas.stravastats.domain.utils.formatSeconds


internal open class BestEffortPowerStatistic(
    name: String,
    activities: List<StravaActivity>,
    private val seconds: Int,
) : ActivityStatistic(name, activities) {

    private val bestActivityEffort = activities
        .mapNotNull { activity -> activity.calculateBestPowerForTime(seconds) }
        .maxByOrNull { activityEffort -> activityEffort.distance }

    init {
        require(seconds > 10) { "DistanceStream must be > 10 seconds" }
        activity = bestActivityEffort?.activityShort
    }

    override val value: String
        get() = if (bestActivityEffort != null) {
            if (bestActivityEffort.averagePower != null) {
                "%d W".format(bestActivityEffort.averagePower)
            } else {
                "Not available"
            }
        } else {
            "Not available"
        }

    protected open fun result(bestActivityEffort: ActivityEffort) =
        if (bestActivityEffort.averagePower != null) {
            "%d W".format(bestActivityEffort.averagePower)
        } else {
            "Not available"
        }
}

fun StravaDetailedActivity.calculateBestPowerForTime(seconds: Int): ActivityEffort? {

    // no stream -> return null
    return if (stream == null || stream?.altitude == null) {
        null
    } else {
        activityEffort(this.id, this.name, this.type, this.stream!!, seconds)
    }
}

fun StravaActivity.calculateBestPowerForTime(seconds: Int): ActivityEffort? {

    // no stream -> return null
    return if (stream == null || stream?.altitude == null) {
        null
    } else {
        activityEffort(this.id, this.name, this.type, this.stream!!, seconds)
    }
}

/**
 * Sliding window best power for a given time
 * @param seconds given time
 */
private fun activityEffort(
    id: Long,
    name: String,
    type: String,
    stream: Stream,
    seconds: Int
): ActivityEffort? {
    val altitudes = stream.altitude?.data
    val watts = stream.watts?.data ?: return null

    var idxStart = 0
    var idxEnd = 0
    var maxPower = 0
    var bestEffort: ActivityEffort? = null

    val distances = stream.distance.data
    val times = stream.time.data
    val streamDataSize = distances.size

    do {
        val totalDistance = distances[idxEnd] - distances[idxStart]
        val totalAltitude = if (altitudes?.isNotEmpty() == true) {
            altitudes[idxEnd] - altitudes[idxStart]
        } else {
            0.0
        }

        val totalPower = if (watts.isNotEmpty()) {
            (idxStart..idxEnd).sumOf { watts[it] }
        } else {
            0
        }

        val totalTime = times[idxEnd] - times[idxStart]

        if (totalTime < seconds) {
            ++idxEnd
        } else {
            if (totalPower > maxPower) {
                maxPower = totalPower
                val averagePower = totalPower / (idxEnd - idxStart)
                bestEffort = ActivityEffort(
                    totalDistance, seconds, totalAltitude, idxStart, idxEnd, averagePower,
                    label = "Best power for ${seconds.formatSeconds()}",
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
