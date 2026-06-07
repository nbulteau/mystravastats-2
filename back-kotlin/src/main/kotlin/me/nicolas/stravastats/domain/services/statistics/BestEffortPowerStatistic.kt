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
    val activityStream = stream ?: return null
    if (activityStream.altitude == null) {
        return null
    }
    return BestEffortCache.getOrCompute(this.id, "best-power-time-v2", seconds.toString(), activityStream) {
        activityEffort(this.id, this.name, this.type, activityStream, seconds)
    }
}

fun StravaDetailedActivity.calculateBestPowerForDistance(distance: Double): ActivityEffort? {
    val activityStream = stream ?: return null
    if (activityStream.altitude == null) {
        return null
    }
    return BestEffortCache.getOrCompute(this.id, "best-power-distance-v1", distance.toString(), activityStream) {
        activityPowerDistanceEffort(this.id, this.name, this.type, activityStream, distance)
    }
}

fun StravaActivity.calculateBestPowerForTime(seconds: Int): ActivityEffort? {
    val activityStream = stream ?: return null
    if (activityStream.altitude == null) {
        return null
    }
    return BestEffortCache.getOrCompute(this.id, "best-power-time-v2", seconds.toString(), activityStream) {
        activityEffort(this.id, this.name, this.type, activityStream, seconds)
    }
}

fun StravaActivity.calculateBestPowerForDistance(distance: Double): ActivityEffort? {
    val activityStream = stream ?: return null
    if (activityStream.altitude == null) {
        return null
    }
    return BestEffortCache.getOrCompute(this.id, "best-power-distance-v1", distance.toString(), activityStream) {
        activityPowerDistanceEffort(this.id, this.name, this.type, activityStream, distance)
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
    val altitudes = stream.altitude?.data ?: emptyList()

    stream.watts?.data ?: return null

    val nonNullWatts: List<Int> = stream.watts.data.map { it ?: 0 }

    var idxStart = 0
    var idxEnd = 0
    var maxPower = 0
    var bestEffort: ActivityEffort? = null

    val distances = stream.distance.data
    val times = stream.time.data
    val streamDataSize = minOf(distances.size, times.size, altitudes.size, nonNullWatts.size)
    if (streamDataSize < 2) {
        return null
    }

    var currentPower = 0
    val elevationPrefix = ElevationGainLossPrefix.from(altitudes, streamDataSize)

    while (idxEnd < streamDataSize) {
        val totalDistance = distances[idxEnd] - distances[idxStart]
        val totalAltitude = if (altitudes.isNotEmpty()) {
            altitudes[idxEnd] - altitudes[idxStart]
        } else {
            0.0
        }

        currentPower += nonNullWatts[idxEnd]

        val totalTime = times[idxEnd] - times[idxStart]

        if (totalTime < seconds) {
            ++idxEnd
        } else {
            if (currentPower > maxPower) {
                maxPower = currentPower
                val averagePower = currentPower / (idxEnd - idxStart + 1)
                val elevation = elevationPrefix.between(idxStart, idxEnd)
                bestEffort = ActivityEffort(
                    totalDistance, seconds, totalAltitude, idxStart, idxEnd, averagePower,
                    label = "Best power for ${seconds.formatSeconds()}",
                    activityShort = ActivityShort(
                        id = id,
                        name = name,
                        type = type
                    ),
                    elevationGain = elevation?.gain,
                    elevationLoss = elevation?.loss,
                )
            }
            currentPower -= nonNullWatts[idxStart]
            ++idxStart
            ++idxEnd
        }
    }

    return bestEffort
}

/**
 * Sliding window best average power for a given distance.
 * @param distance given distance in meters
 */
private fun activityPowerDistanceEffort(
    id: Long,
    name: String,
    type: String,
    stream: Stream,
    distance: Double
): ActivityEffort? {
    val altitudes = stream.altitude?.data ?: emptyList()
    val nonNullWatts = stream.watts?.data?.map { it ?: 0 } ?: return null

    var idxStart = 0
    var idxEnd = 0
    var bestAveragePower = 0.0
    var bestEffort: ActivityEffort? = null

    val distances = stream.distance.data
    val times = stream.time.data
    val streamDataSize = minOf(distances.size, times.size, altitudes.size, nonNullWatts.size)
    if (streamDataSize < 2) {
        return null
    }

    val elevationPrefix = ElevationGainLossPrefix.from(altitudes, streamDataSize)

    while (idxEnd < streamDataSize && idxStart < streamDataSize) {
        val totalDistance = distances[idxEnd] - distances[idxStart]
        val totalTime = times[idxEnd] - times[idxStart]
        val totalAltitude = altitudes[idxEnd] - altitudes[idxStart]

        if (totalDistance < distance - 0.5) {
            idxEnd++
        } else {
            if (totalTime <= 0) {
                idxStart++
                if (idxEnd < idxStart) {
                    idxEnd = idxStart
                }
                continue
            }
            val averagePower = nonNullWatts.subList(idxStart, idxEnd + 1).average()
            if (averagePower > bestAveragePower) {
                bestAveragePower = averagePower
                val estimatedTimeForDistance = distance / totalDistance * totalTime
                val elevation = elevationPrefix.between(idxStart, idxEnd)
                bestEffort = ActivityEffort(
                    distance, estimatedTimeForDistance.toInt(), totalAltitude, idxStart, idxEnd, averagePower.toInt(),
                    label = "Best Power for ${distance.toInt()} m",
                    activityShort = ActivityShort(
                        id = id,
                        name = name,
                        type = type
                    ),
                    elevationGain = elevation?.gain,
                    elevationLoss = elevation?.loss,
                )
            }
            idxStart++
            if (idxEnd < idxStart) {
                idxEnd = idxStart
            }
        }
    }

    return bestEffort
}
