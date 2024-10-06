package me.nicolas.stravastats.api.dto

import com.fasterxml.jackson.annotation.JsonIgnoreProperties
import me.nicolas.stravastats.domain.business.strava.Segment

@JsonIgnoreProperties(ignoreUnknown = true)
data class SegmentDto(
    val activityType: String,
    val averageGrade: Double,
    val city: String?,
    val climbCategory: Int,
    val country: String?,
    val distance: Double,
    val elevationHigh: Double,
    val elevationLow: Double,
    val endLatLng: List<Double>,
    val hazardous: Boolean,
    val id: Long,
    val maximumGrade: Double,
    val name: String,
    val isPrivate: Boolean,
    val resourceState: Int,
    val starred: Boolean,
    val startLatLng: List<Double>,
    val state: String?,
)

fun Segment.toDto() = SegmentDto(
    activityType = activityType,
    averageGrade = averageGrade,
    city = city,
    climbCategory = climbCategory,
    country = country,
    distance = distance,
    elevationHigh = elevationHigh,
    elevationLow = elevationLow,
    endLatLng = endLatLng,
    hazardous = hazardous,
    id = id,
    maximumGrade = maximumGrade,
    name = name,
    isPrivate = isPrivate,
    resourceState = resourceState,
    starred = starred,
    startLatLng = startLatLng,
    state = state,
)