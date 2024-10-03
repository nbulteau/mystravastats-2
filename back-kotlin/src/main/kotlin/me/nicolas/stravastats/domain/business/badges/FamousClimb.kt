package me.nicolas.stravastats.domain.business.badges

import me.nicolas.stravastats.domain.business.strava.GeoCoordinate


data class FamousClimb(
    val name: String,
    val topOfTheAscent: Int,
    val geoCoordinate: GeoCoordinate,
    val alternatives: List<Alternative> = listOf(),
)

data class Alternative(
    val name: String,
    val geoCoordinate: GeoCoordinate,
    val length: Double,
    val totalAscent: Int,
    val difficulty: Int,
    val averageGradient: Double,
)
