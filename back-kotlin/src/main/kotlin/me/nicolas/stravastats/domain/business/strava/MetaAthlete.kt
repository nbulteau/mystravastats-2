package me.nicolas.stravastats.domain.business.strava

import com.fasterxml.jackson.annotation.JsonIgnoreProperties

@JsonIgnoreProperties(ignoreUnknown = true)
data class MetaAthlete(
    val id: Long,
)