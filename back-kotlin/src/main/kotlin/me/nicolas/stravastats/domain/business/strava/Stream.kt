package me.nicolas.stravastats.domain.business.strava

import com.fasterxml.jackson.annotation.JsonProperty

data class Stream(
    val distance: Distance,
    val time: Time,
    val moving: Moving?,
    val altitude: Altitude?,
    @JsonProperty("latlng")
    val latitudeLongitude: LatitudeLongitude?,
    val watts: PowerStream?,
)