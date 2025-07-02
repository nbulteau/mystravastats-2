package me.nicolas.stravastats.domain.business.strava

import com.fasterxml.jackson.annotation.JsonProperty

data class SplitsMetric(
    @param:JsonProperty("average_speed")
    val averageSpeed: Double,
    @param:JsonProperty("average_grade_adjusted_speed")
    val averageGradeAdjustedSpeed: Double?,
    @param:JsonProperty("average_heartrate")
    val averageHeartRate: Double,
    val distance: Double,
    @param:JsonProperty("elapsed_time")
    val elapsedTime: Int,
    @param:JsonProperty("elevation_difference")
    val elevationDifference: Double,
    @param:JsonProperty("moving_time")
    val movingTime: Int,
    @param:JsonProperty("pace_zone")
    val paceZone: Int,
    val split: Int,
)