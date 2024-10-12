package me.nicolas.stravastats.api.dto

import io.swagger.v3.oas.annotations.media.Schema
import me.nicolas.stravastats.domain.business.strava.Activity
import me.nicolas.stravastats.domain.business.strava.Stream
import me.nicolas.stravastats.domain.services.statistics.calculateBestElevationForDistance
import me.nicolas.stravastats.domain.services.statistics.calculateBestPowerForTime
import me.nicolas.stravastats.domain.services.statistics.calculateBestTimeForDistance

@Schema(description = "Detailed activity object", name = "Activity")
data class DetailedActivityDto(
    @Schema(description = "Activity name")
    val name: String,
    @Schema(description = "Activity type")
    val type: String,
    @Schema(description = "Activity link to Strava")
    val link: String,
    @Schema(description = "Activity distance in meters")
    val distance: Int,
    @Schema(description = "Activity elapsed time in seconds")
    val elapsedTime: Int,
    @Schema(description = "Activity total elevation gain in meters")
    val totalElevationGain: Int,
    @Schema(description = "Activity total descent in meters")
    val totalDescent: Int,
    @Schema(description = "Activity average speed in m/s")
    val averageSpeed: Double,
    @Schema(description = "Activity best time for distance for 1000m in m/s")
    val bestTimeForDistanceFor1000m: Double,
    @Schema(description = "Activity best elevation for distance for 500m in %")
    val bestElevationForDistanceFor500m: Double,
    @Schema(description = "Activity best elevation for distance for 1000m in %")
    val bestElevationForDistanceFor1000m: Double,
    @Schema(description = "Activity date")
    val date: String,
    @Schema(description = "Activity average watts")
    val averageWatts: Int,
    @Schema(description = "Activity weighted average watts")
    val weightedAverageWatts: String,
    @Schema(description = "Activity best power for 20 minutes in watts")
    val bestPowerFor20minutes: String,
    @Schema(description = "Activity best power for 60 minutes in watts")
    val bestPowerFor60minutes: String,
    @Schema(description = "Activity FTP (Functional Threshold Power) in watts")
    val ftp: String,
    val stream: StreamDto? = null
)

fun Activity.toDetailedActivityDto(): DetailedActivityDto {

    val bestPowerFor20Minutes = calculateBestPowerForTime(20 * 60)
    val bestPowerFor60Minutes = calculateBestPowerForTime(60 * 60)

    val ftp = if (bestPowerFor60Minutes != null) {
        "${bestPowerFor60Minutes.averagePower}"
    } else if (bestPowerFor20Minutes != null) {
        "${(bestPowerFor20Minutes.averagePower?.times(0.95))?.toInt()}"
    } else {
        ""
    }
    return DetailedActivityDto(
        name = this.name,
        type = this.type,
        link = "https://www.strava.com/activities/${this.id}",
        distance = this.distance.toInt(),
        elapsedTime = this.elapsedTime,
        totalElevationGain = this.totalElevationGain.toInt(),
        totalDescent = calculateTotalDescentGain().toInt(),
        averageSpeed = this.averageSpeed,
        bestTimeForDistanceFor1000m = calculateBestTimeForDistance(1000.0)?.getMSSpeed()?.toDouble() ?: Double.NaN,
        bestElevationForDistanceFor500m = calculateBestElevationForDistance(500.0)?.getGradient()?.toDouble()
            ?: Double.NaN,
        bestElevationForDistanceFor1000m = calculateBestElevationForDistance(1000.0)?.getGradient()?.toDouble()
            ?: Double.NaN,
        date = this.startDateLocal,
        averageWatts = this.averageWatts.toInt(),
        weightedAverageWatts = "${this.weightedAverageWatts}",
        bestPowerFor20minutes = bestPowerFor20Minutes?.getFormattedPower() ?: "",
        bestPowerFor60minutes = bestPowerFor60Minutes?.getFormattedPower() ?: "",
        ftp = ftp,
        stream = this.stream?.toDto(),
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
