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

data class MapPassagesDto(
    val segments: List<MapPassageSegmentDto>,
    val includedActivities: Int,
    val excludedActivities: Int,
    val missingStreamActivities: Int,
    val resolutionMeters: Int,
    val minPassageCount: Int,
    val omittedSegments: Int,
)

data class MapPassageSegmentDto(
    val coordinates: List<List<Double>>,
    val passageCount: Int,
    val activityCount: Int,
    val distanceKm: Double,
    val activityTypeCounts: Map<String, Int>,
)
