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
    @param:Schema(description = "Average cadence")
    val averageCadence: Int,
    @param:Schema(description = "Average heartrate")
    val averageHeartrate: Int,
    @param:Schema(description = "Average power output in watts during this activity. Rides only.")
    val averageWatts: Int,
    @param:Schema(description = "Average speed")
    val averageSpeed: Float,
    val calories: Double,
    @param:Schema(description = "Whether the activity was a commute.")
    val commute: Boolean,
    @param:Schema(description = "Whether the watts are from a power meter, false if estimated.")
    val deviceWatts: Boolean = false,
    @param:Schema(description = "DistanceStream in meters.")
    var distance: Double,
    @param:Schema(description = "Elapsed time in seconds.")
    var elapsedTime: Int,
    @param:Schema(description = "Highest elevation in meters.")
    val elevHigh: Double,
    @param:Schema(description = "Activity id.")
    val id: Long,
    @param:Schema(description = "The total work done in kilojoules during this activity. Rides only.")
    val kilojoules: Double,
    @param:Schema(description = "Maximum heartrate")
    val maxHeartrate: Int,
    @param:Schema(description = "Maximum speed.")
    val maxSpeed: Float,
    @param:Schema(description = "Maximum power output in watts during this activity. Rides only.")
    val maxWatts: Int,
    @param:Schema(description = "MovingStream time in seconds.")
    val movingTime: Int,
    @param:Schema(description = "Activity name.")
    val name: String,
    @param:Schema(description = "List of activity efforts.")
    val activityEfforts: List<ActivityEffortDto>,
    @param:Schema(description = "The time at which the activity was started.")
    val startDate: String,
    @param:Schema(description = "The time at which the activity was started in the local timezone.")
    val startDateLocal: String,
    @param:Schema(description = "The start latitude and longitude of the activity.")
    val startLatlng: List<Double>?,
    @param:Schema(description = "Stream object")
    val stream: StreamDto? = null,
    @param:Schema(description = "The suffer score for the activity.")
    val sufferScore: Double?,
    @param:Schema(description = "Total descent in meters")
    val totalDescent: Double,
    @param:Schema(description = "Total elevation gain in meters.")
    val totalElevationGain: Int,
    @param:Schema(description = "Activity type")
    val type: String,
    @param:Schema(description = "Weighted average power output in watts during this activity. Rides only.")
    val weightedAverageWatts: Int,
)

fun StravaDetailedActivity.toDto(): DetailedActivityDto {

    val activityForDto = this.copy(stream = this.stream?.sanitizedForDtoComputation())
    val activityEfforts = activityForDto.buildActivityEfforts()

    return DetailedActivityDto(
        averageSpeed = activityForDto.averageSpeed.finiteFloatOrZero(),
        averageCadence = activityForDto.averageCadence.finiteIntOrZero(),
        averageHeartrate = activityForDto.averageHeartrate.finiteIntOrZero(),
        averageWatts = activityForDto.averageWatts.finiteIntOrZero(),
        calories = activityForDto.calories.finiteOrZero(),
        commute = activityForDto.commute,
        distance = activityForDto.distance.toDouble().finiteOrZero(),
        deviceWatts = activityForDto.deviceWatts,
        elapsedTime = activityForDto.elapsedTime,
        elevHigh = activityForDto.elevHigh.finiteOrZero(),
        id = activityForDto.id,
        kilojoules = activityForDto.kilojoules.finiteOrZero(),
        maxHeartrate = activityForDto.maxHeartrate,
        maxSpeed = activityForDto.maxSpeed.finiteFloatOrZero(),
        maxWatts = activityForDto.maxWatts,
        movingTime = activityForDto.movingTime,
        name = activityForDto.name,
        activityEfforts = activityEfforts.map { activityEffort -> activityEffort.toDto() },
        startDate = activityForDto.startDate,
        startDateLocal = activityForDto.startDateLocal,
        startLatlng = activityForDto.startLatLng.finiteValues(),
        sufferScore = activityForDto.sufferScore.finiteOrNull(),
        totalDescent = activityForDto.elevLow.finiteOrZero(),
        totalElevationGain = activityForDto.totalElevationGain,
        type = activityForDto.type,
        weightedAverageWatts = activityForDto.weightedAverageWatts,
        stream = activityForDto.stream?.toDto(),
    )
}

private fun Stream.sanitizedForDtoComputation(): Stream {
    return this.copy(
        distance = this.distance.copy(data = this.distance.data.finiteValues()),
        latlng = this.latlng?.copy(data = this.latlng.data.finiteCoordinateValues()),
        altitude = this.altitude?.copy(data = this.altitude.data.finiteValues()),
        velocitySmooth = this.velocitySmooth?.copy(data = this.velocitySmooth.data.finiteFloatValues()),
        gradeSmooth = this.gradeSmooth?.copy(data = this.gradeSmooth.data.finiteFloatValues()),
    )
}

data class StreamDto(
    val distance: List<Double>,
    val time: List<Int>,
    val latlng: List<List<Double>>? = null,
    val heartrate: List<Int>? = null,
    val moving: List<Boolean>? = null,
    val altitude: List<Double>? = null,
    val watts: List<Int?>? = null,
    val velocitySmooth: List<Double>? = null,
)

fun Stream.toDto(): StreamDto {
    if (this.latlng == null) {
        return StreamDto(
            distance = this.distance.data.finiteValues(),
            time = this.time.data
        )
    }

    val velocity = if (this.velocitySmooth?.data == null) {
        // Calculate velocitySmooth
        val velocitySmooth = mutableListOf<Double>()
        for (i in 0 until this.latlng.data.size - 1) {
            val current = this.latlng.data[i]
            val next = this.latlng.data[i + 1]
            if (current.size < 2 || next.size < 2) {
                velocitySmooth.add(0.0)
                continue
            }
            val lat1 = current[0].finiteOrZero()
            val lon1 = current[1].finiteOrZero()
            val lat2 = next[0].finiteOrZero()
            val lon2 = next[1].finiteOrZero()
            val distance = haversine(lat1, lon1, lat2, lon2).finiteOrZero()
            val time = this.time.data[i + 1] - this.time.data[i]
            if (time == 0) {
                velocitySmooth.add(0.0)
            } else {
                velocitySmooth.add((distance / time).finiteOrZero())
            }
        }
        velocitySmooth.smooth().finiteValues()
    } else {
        this.velocitySmooth.data.finiteFloatValues().map { it.toDouble() }
    }

    return StreamDto(
        distance = this.distance.data.finiteValues(),
        time = this.time.data,
        latlng = this.latlng.data.finiteCoordinateValues(),
        heartrate = this.heartrate?.data,
        moving = this.moving?.data,
        altitude = this.altitude?.data?.finiteValues(),
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
        id = "${this.label.hashCode()}-${this.idxStart}-${this.idxEnd}-${this.seconds}",
        label = this.label,
        distance = this.distance.finiteOrZero(),
        seconds = this.seconds,
        deltaAltitude = this.deltaAltitude.finiteOrZero(),
        idxStart = this.idxStart,
        idxEnd = this.idxEnd,
        averagePower = this.averagePower,
        description = this.getDescription(),
    )
}
