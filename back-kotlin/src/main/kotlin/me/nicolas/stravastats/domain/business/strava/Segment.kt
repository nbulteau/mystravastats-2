package me.nicolas.stravastats.domain.business.strava

import com.fasterxml.jackson.annotation.JsonIgnoreProperties
import com.fasterxml.jackson.annotation.JsonProperty

@JsonIgnoreProperties(ignoreUnknown = true)
data class Segment(
    @param:JsonProperty("activity_type")
    val activityType: String,
    @param:JsonProperty("average_grade")
    val averageGrade: Double,
    val city: String?,
    @param:JsonProperty("climb_category")
    val climbCategory: Int,
    val country: String?,
    val distance: Double,
    @param:JsonProperty("elevation_high")
    val elevationHigh: Double,
    @param:JsonProperty("elevation_low")
    val elevationLow: Double,
    @param:JsonProperty("end_latlng")
    val endLatLng: List<Double>,
    val hazardous: Boolean,
    val id: Long,
    @param:JsonProperty("maximum_grade")
    val maximumGrade: Double,
    val name: String,
    @param:JsonProperty("private")
    val isPrivate: Boolean,
    @param:JsonProperty("resource_state")
    val resourceState: Int,
    val starred: Boolean,
    @param:JsonProperty("start_latlng")
    val startLatLng: List<Double>,
    val state: String?,
)