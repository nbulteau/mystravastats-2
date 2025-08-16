package me.nicolas.stravastats.api.dto

import io.swagger.v3.oas.annotations.media.Schema
import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.services.statistics.calculateBestElevationForDistance
import me.nicolas.stravastats.domain.services.statistics.calculateBestPowerForTime
import me.nicolas.stravastats.domain.services.statistics.calculateBestTimeForDistance
import me.nicolas.stravastats.domain.utils.formatSpeed


@Schema(description = "Activity object", name = "Activity")
data class ActivityDto(
    @param:Schema(description = "Activity id")
    val id: Long = 0,
    @param:Schema(description = "Activity name")
    val name: String,
    @param:Schema(description = "Activity type")
    val type: String,
    @param:Schema(description = "Activity link to ")
    val link: String,
    @param:Schema(description = "Activity distance in meters")
    val distance: Int,
    @param:Schema(description = "Activity elapsed time in seconds")
    val elapsedTime: Int,
    @param:Schema(description = "Activity total elevation gain in meters")
    val totalElevationGain: Int,
    @param:Schema(description = "Activity average speed")
    val averageSpeed: String,
    @param:Schema(description = "Activity best time for distance for 1000m")
    val bestTimeForDistanceFor1000m: String,
    @param:Schema(description = "Activity best elevation for distance for 500m in %")
    val bestElevationForDistanceFor500m: String,
    @param:Schema(description = "Activity best elevation for distance for 1000m in %")
    val bestElevationForDistanceFor1000m: String,
    @param:Schema(description = "Activity date")
    val date: String,
    @param:Schema(description = "Activity average watts")
    val averageWatts: Int,
    @param:Schema(description = "Activity weighted average watts")
    val weightedAverageWatts: String,
    @param:Schema(description = "Activity best power for 20 minutes in watts")
    val bestPowerFor20minutes: String,
    @param:Schema(description = "Activity best power for 60 minutes in watts")
    val bestPowerFor60minutes: String,
    @param:Schema(description = "Activity FTP (Functional Threshold Power) in watts")
    val ftp: String,
)

fun StravaActivity.toDto(): ActivityDto {

    val bestPowerFor20Minutes = calculateBestPowerForTime(20 * 60)
    val bestPowerFor60Minutes = calculateBestPowerForTime(60 * 60)

    val ftp = if (bestPowerFor60Minutes != null) {
        "${bestPowerFor60Minutes.averagePower}"
    } else if (bestPowerFor20Minutes != null) {
        "${(bestPowerFor20Minutes.averagePower?.times(0.95))?.toInt()}"
    } else {
        ""
    }

    // If the activity is not uploaded, the link is not available
    val link = if (this.uploadId != 0L) "https://www.strava.com/activities/${this.id}" else ""

    return ActivityDto(
        id = this.id,
        name = this.name,
        type = this.type,
        link = link,
        distance = this.distance.toInt(),
        elapsedTime = this.elapsedTime,
        totalElevationGain = this.totalElevationGain.toInt(),
        averageSpeed = this.averageSpeed.formatSpeed(this.type),
        bestTimeForDistanceFor1000m = calculateBestTimeForDistance(1000.0)?.getFormattedSpeed() ?: "",
        bestElevationForDistanceFor500m = calculateBestElevationForDistance(500.0)?.getGradient() ?: "",
        bestElevationForDistanceFor1000m = calculateBestElevationForDistance(1000.0)?.getGradient() ?: "",
        date = this.startDateLocal,
        averageWatts = this.averageWatts,
        weightedAverageWatts = "${this.weightedAverageWatts}",
        bestPowerFor20minutes = bestPowerFor20Minutes?.getFormattedPower() ?: "",
        bestPowerFor60minutes = bestPowerFor60Minutes?.getFormattedPower() ?: "",
        ftp = ftp,
    )
}