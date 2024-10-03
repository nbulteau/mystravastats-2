package me.nicolas.stravastats.domain.business.strava

import com.fasterxml.jackson.annotation.JsonProperty

data class Achievement(
    @JsonProperty("effort_count")
    val effortCount: Int,
    val rank: Int,
    val type: String,
    @JsonProperty("type_id")
    val typeId: Int,
)