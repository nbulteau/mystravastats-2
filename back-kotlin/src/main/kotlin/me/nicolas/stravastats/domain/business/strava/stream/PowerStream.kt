package me.nicolas.stravastats.domain.business.strava.stream

import com.fasterxml.jackson.annotation.JsonProperty

data class PowerStream(
    @param:JsonProperty("data")
    val `data`: List<Int?>,
    @param:JsonProperty("original_size")
    val originalSize: Int,
    val resolution: String,
    @param:JsonProperty("series_type")
    val seriesType: String,
)