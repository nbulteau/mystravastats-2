package me.nicolas.stravastats.api.dto

data class MapTrackDto(
    val activityId: Long,
    val activityName: String,
    val activityDate: String,
    val activityType: String,
    val distanceKm: Double,
    val elevationGainM: Double,
    val coordinates: List<List<Double>>,
)

