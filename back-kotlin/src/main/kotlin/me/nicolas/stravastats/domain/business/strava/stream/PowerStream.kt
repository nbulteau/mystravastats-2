package me.nicolas.stravastats.domain.business.strava.stream

import com.fasterxml.jackson.annotation.JsonProperty

data class PowerStream(
    @JsonProperty("data")
    val `data`: List<Int?>,
    @JsonProperty("original_size")
    val originalSize: Int,
    val resolution: String,
    @JsonProperty("series_type")
    val seriesType: String,
)