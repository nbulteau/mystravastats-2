package me.nicolas.stravastats.domain.services.statistics

import me.nicolas.stravastats.domain.business.ActivityEffort
import me.nicolas.stravastats.domain.business.ActivityShort
import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.business.strava.StravaDetailedActivity
import me.nicolas.stravastats.domain.business.strava.stream.Stream
import me.nicolas.stravastats.domain.utils.formatSeconds


internal open class BestEffortDistanceStatistic(
    name: String,
    activities: List<StravaActivity>,
    private val distance: Double,
) : ActivityStatistic(name, activities) {

    private val bestActivityEffort = activities
        .mapNotNull { activity -> activity.calculateBestTimeForDistance(distance) }
        .minByOrNull { activityEffort -> activityEffort.seconds }

    init {
        require(distance > 100) { "DistanceStream must be > 100 meters" }
        activity = bestActivityEffort?.activityShort
    }

    override val value: String
        get() = if (bestActivityEffort != null) {
            "${bestActivityEffort.seconds.formatSeconds()} => ${bestActivityEffort.getFormattedSpeed()}"
        } else {
            "Not available"
        }
}


fun StravaActivity.calculateBestTimeForDistance(distance: Double): ActivityEffort? {

    // no stream -> return null
    return if (stream == null || stream?.distance == null) {
        null
    } else {
        activityEffort(this.id, this.name, this.type, this.stream!!, distance)
    }
}

fun StravaDetailedActivity.calculateBestTimeForDistance(distance: Double): ActivityEffort? {

    // no stream -> return null
    return if (this.stream == null || this.stream?.altitude == null) {
        null
    } else {
        activityEffort(this.id, this.name, this.type, this.stream!!, distance)
    }
}

/**
 * Sliding window best time for a given distance.
 * @param distance given distance.
 */
private fun activityEffort(
    id: Long,
    name: String,
    type: String,
    stream: Stream,
    distance: Double
): ActivityEffort? {
    var idxStart = 0
    var idxEnd = 0
    var bestTime = Double.MAX_VALUE
    var bestEffort: ActivityEffort? = null

    val distances = stream.distance.data
    val times = stream.time.data
    val altitudes = stream.altitude?.data ?: emptyList()
    val nonNullWatts = stream.watts?.data?.map { it ?: 0 }

    val streamDataSize = distances.size

    while (idxEnd < streamDataSize) {
        val totalDistance = distances[idxEnd] - distances[idxStart]
        val totalTime = times[idxEnd] - times[idxStart]
        val totalAltitude = altitudes.getOrNull(idxEnd)?.minus(altitudes.getOrNull(idxStart) ?: 0.0) ?: 0.0

        if (totalDistance < distance - 0.5) {
            idxEnd++
        } else {
            val estimatedTimeForDistance = distance / totalDistance * totalTime
            if (estimatedTimeForDistance < bestTime && estimatedTimeForDistance > 1) {
                bestTime = estimatedTimeForDistance
                val averagePower = nonNullWatts?.subList(idxStart, idxEnd + 1)?.average()?.toInt()
                bestEffort = ActivityEffort(
                    distance, bestTime.toInt(), totalAltitude, idxStart, idxEnd, averagePower,
                    label = "Best speed for ${distance.toInt()}m",
                    activityShort = ActivityShort(
                        id = id,
                        name = name,
                        type = type
                    )
                )
            }
            idxStart++
        }
    }

    return bestEffort
}
