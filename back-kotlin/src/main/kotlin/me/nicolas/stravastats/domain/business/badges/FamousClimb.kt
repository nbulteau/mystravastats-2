package me.nicolas.stravastats.domain.business.badges

import me.nicolas.stravastats.domain.business.strava.GeoCoordinate


data class FamousClimb(
    val name: String,
    val country: String,
    val massif: String,
    val topOfTheAscent: Int,
    val geoCoordinate: GeoCoordinate,
    val alternatives: List<Alternative> = listOf(),
)

data class Alternative(
    val name: String,
    val geoCoordinate: GeoCoordinate,
    val routeCheckpoints: List<GeoCoordinate> = emptyList(),
    val summitToleranceMeters: Int = 0,
    val length: Double,
    val totalAscent: Int,
    val difficulty: Int,
    val category: String = "",
    val averageGradient: Double = 0.0,
    val minimumAltitude: Int = 0,
    val maximumGradient: Double = 0.0,
    val sourceUrl: String = "",
)
