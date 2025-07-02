package me.nicolas.stravastats.domain.business.strava


import com.fasterxml.jackson.annotation.JsonProperty

data class GeoMap(
    @param:JsonProperty("id")
    val id: String,
    val polyline: String?,
    @param:JsonProperty("resource_state")
    val resourceState: Int,
    @param:JsonProperty("summary_polyline")
    val summaryPolyline: String?,
)