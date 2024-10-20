package me.nicolas.stravastats.api.dto

import io.swagger.v3.oas.annotations.media.Schema
import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.services.statistics.calculateBestElevationForDistance
import me.nicolas.stravastats.domain.services.statistics.calculateBestPowerForTime
import me.nicolas.stravastats.domain.services.statistics.calculateBestTimeForDistance


@Schema(description = "StravaActivity object", name = "StravaActivity")
data class ActivityDto(
    @Schema(description = "StravaActivity name")
    val name: String,
    @Schema(description = "StravaActivity type")
    val type: String,
    @Schema(description = "StravaActivity link to Strava")
    val link: String,
    @Schema(description = "StravaActivity distance in meters")
    val distance: Int,
    @Schema(description = "StravaActivity elapsed time in seconds")
    val elapsedTime: Int,
    @Schema(description = "StravaActivity total elevation gain in meters")
    val totalElevationGain: Int,
    @Schema(description = "StravaActivity average speed in m/s")
    val averageSpeed: Double,
    @Schema(description = "StravaActivity best time for distance for 1000m in m/s")
    val bestTimeForDistanceFor1000m: Double,
    @Schema(description = "StravaActivity best elevation for distance for 500m in %")
    val bestElevationForDistanceFor500m: Double,
    @Schema(description = "StravaActivity best elevation for distance for 1000m in %")
    val bestElevationForDistanceFor1000m: Double,
    @Schema(description = "StravaActivity date")
    val date: String,
    @Schema(description = "StravaActivity average watts")
    val averageWatts: Int,
    @Schema(description = "StravaActivity weighted average watts")
    val weightedAverageWatts: String,
    @Schema(description = "StravaActivity best power for 20 minutes in watts")
    val bestPowerFor20minutes: String,
    @Schema(description = "StravaActivity best power for 60 minutes in watts")
    val bestPowerFor60minutes: String,
    @Schema(description = "StravaActivity FTP (Functional Threshold Power) in watts")
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
    val link = if (this.uploadId == 0L) "https://www.strava.com/activities/${this.id}" else ""

    return ActivityDto(
        name = this.name,
        type = this.type,
        link = link,
        distance = this.distance.toInt(),
        elapsedTime = this.elapsedTime,
        totalElevationGain = this.totalElevationGain.toInt(),
        averageSpeed = this.averageSpeed,
        bestTimeForDistanceFor1000m = calculateBestTimeForDistance(1000.0)?.getMSSpeed()?.toDouble() ?: Double.NaN,
        bestElevationForDistanceFor500m = calculateBestElevationForDistance(500.0)?.getGradient()?.toDouble()
            ?: Double.NaN,
        bestElevationForDistanceFor1000m = calculateBestElevationForDistance(1000.0)?.getGradient()?.toDouble()
            ?: Double.NaN,
        date = this.startDateLocal,
        averageWatts = this.averageWatts,
        weightedAverageWatts = "${this.weightedAverageWatts}",
        bestPowerFor20minutes = bestPowerFor20Minutes?.getFormattedPower() ?: "",
        bestPowerFor60minutes = bestPowerFor60Minutes?.getFormattedPower() ?: "",
        ftp = ftp,
    )
}