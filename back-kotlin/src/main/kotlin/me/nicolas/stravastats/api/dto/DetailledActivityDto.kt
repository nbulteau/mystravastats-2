package me.nicolas.stravastats.api.dto

import io.swagger.v3.oas.annotations.media.Schema
import me.nicolas.stravastats.domain.business.ActivityEffort
import me.nicolas.stravastats.domain.business.strava.StravaDetailedActivity
import me.nicolas.stravastats.domain.business.strava.stream.Stream
import me.nicolas.stravastats.domain.services.ActivityHelper.buildActivityEfforts
import me.nicolas.stravastats.domain.services.ActivityHelper.smooth
import kotlin.math.*

@Schema(description = "Detailed activity object", name = "DetailedActivity")
data class DetailedActivityDto(
    @Schema(description = "Average cadence")
    val averageCadence: Int,
    @Schema(description = "Average heartrate")
    val averageHeartrate: Int,
    @Schema(description = "Average power output in watts during this activity. Rides only.")
    val averageWatts: Int,
    @Schema(description = "Average speed")
    val averageSpeed: Float,
    val calories: Double,
    @Schema(description = "Whether the activity was a commute.")
    val commute: Boolean,
    @Schema(description = "Whether the watts are from a power meter, false if estimated.")
    val deviceWatts: Boolean = false,
    @Schema(description = "DistanceStream in meters.")
    var distance: Double,
    @Schema(description = "Elapsed time in seconds.")
    var elapsedTime: Int,
    @Schema(description = "Highest elevation in meters.")
    val elevHigh: Double,
    @Schema(description = "Activity id.")
    val id: Long,
    @Schema(description = "The total work done in kilojoules during this activity. Rides only.")
    val kilojoules: Double,
    @Schema(description = "Maximum heartrate")
    val maxHeartrate: Int,
    @Schema(description = "Maximum speed.")
    val maxSpeed: Float,
    @Schema(description = "Maximum power output in watts during this activity. Rides only.")
    val maxWatts: Int,
    @Schema(description = "MovingStream time in seconds.")
    val movingTime: Int,
    @Schema(description = "Activity name.")
    val name: String,
    @Schema(description = "List of activity efforts.")
    val activityEfforts: List<ActivityEffortDto>,
    @Schema(description = "The time at which the activity was started.")
    val startDate: String,
    @Schema(description = "The time at which the activity was started in the local timezone.")
    val startDateLocal: String,
    @Schema(description = "The start latitude and longitude of the activity.")
    val startLatlng: List<Double>?,
    @Schema(description = "Stream object")
    val stream: StreamDto? = null,
    @Schema(description = "The suffer score for the activity.")
    val sufferScore: Double?,
    @Schema(description = "Total descent in meters")
    val totalDescent: Double,
    @Schema(description = "Total elevation gain in meters.")
    val totalElevationGain: Int,
    @Schema(description = "Activity type")
    val type: String,
    @Schema(description = "Weighted average power output in watts during this activity. Rides only.")
    val weightedAverageWatts: Int,
)

fun StravaDetailedActivity.toDto(): DetailedActivityDto {

    val activityEfforts = buildActivityEfforts()

    return DetailedActivityDto(
        averageSpeed = this.averageSpeed.toFloat(),
        averageCadence = this.averageCadence.toInt(),
        averageHeartrate = this.averageHeartrate.toInt(),
        averageWatts = this.averageWatts.toInt(),
        calories = this.calories,
        commute = this.commute,
        distance = this.distance.toDouble(),
        deviceWatts = this.deviceWatts,
        elapsedTime = this.elapsedTime,
        elevHigh = this.elevHigh,
        id = this.id,
        kilojoules = this.kilojoules,
        maxHeartrate = this.maxHeartrate,
        maxSpeed = this.maxSpeed.toFloat(),
        maxWatts = this.maxWatts,
        movingTime = this.movingTime,
        name = this.name,
        activityEfforts = activityEfforts.map { it.toDto() },
        startDate = this.startDate,
        startDateLocal = this.startDateLocal,
        startLatlng = this.startLatLng,
        sufferScore = this.sufferScore,
        totalDescent = this.elevLow,
        totalElevationGain = this.totalElevationGain,
        type = this.type,
        weightedAverageWatts = this.weightedAverageWatts,
        stream = this.stream?.toDto(),
    )
}

data class StreamDto(
    val distance: List<Double>,
    val time: List<Int>,
    val latlng: List<List<Double>>? = null,
    val moving: List<Boolean>? = null,
    val altitude: List<Double>? = null,
    val watts: List<Int>? = null,
    val velocitySmooth: List<Double>? = null,
)

fun Stream.toDto(): StreamDto {
    if (this.latlng == null) {
        return StreamDto(
            distance = this.distance.data,
            time = this.time.data
        )
    }

    val velocity = if (this.velocitySmooth?.data == null) {
        // Calculate velocitySmooth
        val velocitySmooth = mutableListOf<Double>()
        for (i in 0 until this.latlng.data.size - 1) {
            val (lat1, lon1) = this.latlng.data[i]
            val (lat2, lon2) = this.latlng.data[i + 1]
            val distance = haversine(lat1, lon1, lat2, lon2)
            val time = this.time.data[i + 1] - this.time.data[i]
            if (time == 0) {
                velocitySmooth.add(0.0)
            } else {
                velocitySmooth.add(distance / time)
            }
        }
        velocitySmooth.smooth()
    } else {
        this.velocitySmooth.data.map { it.toDouble() }
    }

    return StreamDto(
        distance = this.distance.data,
        time = this.time.data,
        latlng = this.latlng.data,
        moving = this.moving?.data,
        altitude = this.altitude?.data,
        watts = this.watts?.data,
        velocitySmooth = velocity,
    )
}

fun haversine(lat1: Double, lon1: Double, lat2: Double, lon2: Double): Double {
    val earthRadius = 6371e3 // Earth radius in meters
    val phi1 = lat1 * PI / 180
    val phi2 = lat2 * PI / 180
    val deltaPhi = (lat2 - lat1) * PI / 180
    val deltaLambda = (lon2 - lon1) * PI / 180

    val a = sin(deltaPhi / 2).pow(2) + cos(phi1) * cos(phi2) * sin(deltaLambda / 2).pow(2)
    val c = 2 * atan2(sqrt(a), sqrt(1 - a))

    return earthRadius * c
}

data class ActivityEffortDto(
    val id: String,
    val label: String,
    val distance: Double,
    val seconds: Int,
    val deltaAltitude: Double,
    val idxStart: Int,
    val idxEnd: Int,
    val averagePower: Int? = null,
    val description: String,
)

fun ActivityEffort.toDto(): ActivityEffortDto {
    return ActivityEffortDto(
        id = this.label.hashCode().toString(),
        label = this.label,
        distance = this.distance,
        seconds = this.seconds,
        deltaAltitude = this.deltaAltitude,
        idxStart = this.idxStart,
        idxEnd = this.idxEnd,
        averagePower = this.averagePower,
        description = this.getDescription(),
    )
}
