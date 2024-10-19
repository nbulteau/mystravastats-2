package me.nicolas.stravastats.api.dto

import io.swagger.v3.oas.annotations.media.Schema
import me.nicolas.stravastats.domain.business.ActivityEffort
import me.nicolas.stravastats.domain.business.DetailedActivity
import me.nicolas.stravastats.domain.business.strava.Stream

@Schema(description = "Detailed activity object", name = "DetailedActivity")
data class DetailedActivityDto(
    @Schema(description = "Average speed")
    val averageSpeed: Double,
    @Schema(description = "Average cadence")
    val averageCadence: Double,
    @Schema(description = "Average heartrate")
    val averageHeartrate: Double,
    @Schema(description = "Maximum heartrate")
    val maxHeartrate: Double,
    @Schema(description = "Average power output in watts during this activity. Rides only.")
    val averageWatts: Int,
    @Schema(description = "Whether the activity was a commute.")
    val commute: Boolean,
    @Schema(description = "Distance in meters.")
    var distance: Double,
    @Schema(description = "Whether the watts are from a power meter, false if estimated.")
    val deviceWatts: Boolean = false,
    @Schema(description = "Elapsed time in seconds.")
    var elapsedTime: Int,
    @Schema(description = "Highest elevation in meters.")
    val elevHigh: Double,
    @Schema(description = "Activity id.")
    val id: Long,
    @Schema(description = "The total work done in kilojoules during this activity. Rides only.")
    val kilojoules: Double,
    @Schema(description = "Maximum speed.")
    val maxSpeed: Double,
    @Schema(description = "Moving time in seconds.")
    val movingTime: Int,
    @Schema(description = "Activity name.")
    val name: String,
    @Schema(description = "The time at which the activity was started.")
    val startDate: String,
    @Schema(description = "The time at which the activity was started in the local timezone.")
    val startDateLocal: String,
    @Schema(description = "The start latitude and longitude of the activity.")
    val startLatlng: List<Double>?,
    @Schema(description = "Total descent in meters")
    val totalDescent: Double,
    @Schema(description = "Total elevation gain in meters.")
    val totalElevationGain: Double,
    @Schema(description = "Activity type")
    val type: String,
    @Schema(description = "Weighted average power output in watts during this activity. Rides only.")
    val weightedAverageWatts: Int,
    @Schema(description = "Stream object")
    val stream: StreamDto? = null,
    @Schema(description = "Map of activity efforts")
    val activityEfforts: List<ActivityEffortDto>,
)

fun DetailedActivity.toDto(): DetailedActivityDto {
    return DetailedActivityDto(
        averageSpeed = this.averageSpeed,
        averageCadence = this.averageCadence,
        averageHeartrate = this.averageHeartrate,
        maxHeartrate = this.maxHeartrate,
        averageWatts = this.averageWatts,
        commute = this.commute,
        distance = this.distance,
        deviceWatts = this.deviceWatts,
        elapsedTime = this.elapsedTime,
        elevHigh = this.elevHigh,
        id = this.id,
        kilojoules = this.kilojoules,
        maxSpeed = this.maxSpeed,
        movingTime = this.movingTime,
        name = this.name,
        startDate = this.startDate,
        startDateLocal = this.startDateLocal,
        startLatlng = this.startLatlng,
        totalDescent = this.totalDescent,
        totalElevationGain = this.totalElevationGain,
        type = this.type,
        weightedAverageWatts = this.weightedAverageWatts,
        stream = this.stream?.toDto(),
        activityEfforts = this.activityEfforts.mapNotNull { (key, value) -> value?.toDto(key) }
    )
}

data class StreamDto(
    val distance: List<Double>,
    val time: List<Int>,
    val moving: List<Boolean>?,
    val altitude: List<Double>?,
    val latitudeLongitude: List<List<Double>>?,
    val watts: List<Int>?,
)

fun Stream.toDto(): StreamDto {
    return StreamDto(
        distance = this.distance.data,
        time = this.time.data,
        moving = this.moving?.data,
        altitude = this.altitude?.data,
        latitudeLongitude = this.latitudeLongitude?.data,
        watts = this.watts?.data,
    )
}

data class ActivityEffortDto(
    val key: String,
    val distance: Double,
    val seconds: Int,
    val deltaAltitude: Double,
    val idxStart: Int,
    val idxEnd: Int,
    val averagePower: Int? = null,
    val description: String,
)

fun ActivityEffort.toDto(key: String): ActivityEffortDto {
    return ActivityEffortDto(
        key = key,
        distance = this.distance,
        seconds = this.seconds,
        deltaAltitude = this.deltaAltitude,
        idxStart = this.idxStart,
        idxEnd = this.idxEnd,
        averagePower = this.averagePower,
        description = this.description,
    )
}