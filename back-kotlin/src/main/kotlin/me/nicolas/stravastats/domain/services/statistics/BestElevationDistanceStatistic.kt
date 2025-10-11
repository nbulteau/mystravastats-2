package me.nicolas.stravastats.domain.services.statistics

import me.nicolas.stravastats.domain.business.ActivityEffort
import me.nicolas.stravastats.domain.business.ActivityShort
import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.business.strava.StravaDetailedActivity
import me.nicolas.stravastats.domain.business.strava.stream.Stream


internal open class BestElevationDistanceStatistic(
    name: String,
    activities: List<StravaActivity>,
    private val distance: Double,
) : ActivityStatistic(name, activities) {

    private val bestActivityEffort = activities
        .mapNotNull { activity -> activity.calculateBestElevationForDistance(distance) }
        .maxByOrNull { activityEffort -> activityEffort.deltaAltitude }

    init {
        require(distance > 100) { "DistanceStream must be > 100 meters" }
        activity = bestActivityEffort?.activityShort
    }

    override val value: String
        get() = bestActivityEffort?.getFormattedGradientWithUnit() ?: "Not available"
}

fun StravaActivity.calculateBestElevationForDistance(distance: Double): ActivityEffort? {

    // no stream -> return null
    return if (stream == null || stream?.altitude == null) {
        null
    } else {
        activityEffort(this.id, this.name, this.type, this.stream!!, distance)
    }
}

fun StravaDetailedActivity.calculateBestElevationForDistance(distance: Double): ActivityEffort? {

    // no stream -> return null
    return if (stream == null || stream?.altitude == null) {
        null
    } else {
        activityEffort(this.id, this.name, this.type, this.stream!!, distance)
    }
}

/**
 * Sliding window looking for best elevation gain for a given distance.
 * @param distance given distance.
 */
private fun activityEffort(
    id: Long,
    name: String,
    type: String,
    stream: Stream,
    distance: Double,
): ActivityEffort? {
    var idxStart = 0
    var idxEnd = 0
    var bestElevation = Double.MIN_VALUE
    var bestEffort: ActivityEffort? = null

    val distances = stream.distance.data
    val times = stream.time.data
    val altitudes = stream.altitude?.data!!
    val nonNullWatts: List<Int>? = stream.watts?.data?.map { it ?: 0 }

    val streamDataSize = stream.distance.originalSize

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
                val averagePower = nonNullWatts?.let {
                    (idxStart..idxEnd).sumOf { nonNullWatts[it] } / (idxEnd - idxStart)
                }
                bestEffort = ActivityEffort(
                    distance, totalTime, bestElevation, idxStart, idxEnd, averagePower,
                    label = "Best gradient for ${distance.toInt()}m",
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
