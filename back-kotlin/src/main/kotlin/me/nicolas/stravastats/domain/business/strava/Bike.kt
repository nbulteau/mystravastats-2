package me.nicolas.stravastats.domain.business.strava

import com.fasterxml.jackson.annotation.JsonProperty

data class Bike(
    @param:JsonProperty("distance")
    val distance: Int,
    @param:JsonProperty("id")
    val id: String,
    @param:JsonProperty("name")
    val name: String,
    @param:JsonProperty("nickname")
    val nickname: String?,
    @param:JsonProperty("retired")
    val retired: Boolean?,
    @param:JsonProperty("converted_distance")
    val convertedDistance: Double,
    @param:JsonProperty("primary")
    val primary: Boolean,
    @param:JsonProperty("resource_state")
    val resourceState: Int,
)