package me.nicolas.stravastats.api.dto

import me.nicolas.stravastats.domain.business.strava.SplitsMetric

data class SplitsMetricDto(
    val averageSpeed: Double,
    val averageGradeAdjustedSpeed: Double?,
    val averageHeartRate: Double,
    val distance: Double,
    val elapsedTime: Int,
    val elevationDifference: Double,
    val movingTime: Int,
    val paceZone: Int,
    val split: Int,
)

fun SplitsMetric.toDto() = SplitsMetricDto(
    averageSpeed = averageSpeed,
    averageGradeAdjustedSpeed = averageGradeAdjustedSpeed,
    averageHeartRate = averageHeartRate,
    distance = distance,
    elapsedTime = elapsedTime,
    elevationDifference = elevationDifference,
    movingTime = movingTime,
    paceZone = paceZone,
    split = split,
)