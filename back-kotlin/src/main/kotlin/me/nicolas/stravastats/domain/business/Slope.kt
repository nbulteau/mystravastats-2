package me.nicolas.stravastats.domain.business

// Define a data class for a slope segment
data class Slope(
    val type: SlopeType, // "Ascent", "Descent", or "Plateau"
    val startIndex: Int,
    val endIndex: Int,
    val startAltitude: Double,
    val endAltitude: Double,
    val grade: Double, // Percentage grade of the slope
    val maxGrade: Double, // Maximum grade within the slope segment
    val distance: Double, // Distance over which the slope is measured
    val duration: Int, // Duration of the slope segment in seconds
    val averageSpeed: Double, // Average speed over the slope segment
)

enum class SlopeType {
    ASCENT,
    DESCENT,
    PLATEAU
}
