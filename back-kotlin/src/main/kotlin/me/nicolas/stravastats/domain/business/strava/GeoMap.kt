package me.nicolas.stravastats.domain.business.strava


import com.fasterxml.jackson.annotation.JsonProperty

data class GeoMap(
    @JsonProperty("id")
    val id: String,
    val polyline: String?,
    @JsonProperty("resource_state")
    val resourceState: Int,
    @JsonProperty("summary_polyline")
    val summaryPolyline: String?,
)