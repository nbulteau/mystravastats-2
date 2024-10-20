package me.nicolas.stravastats.domain.business.strava.stream

import com.fasterxml.jackson.annotation.JsonProperty

data class MovingStream(
    // The sequence of moving values for this stream, as boolean values
    @JsonProperty("data")
    val `data`: List<Boolean>,
    @JsonProperty("original_size")
    val originalSize: Int,
    @JsonProperty("resolution")
    val resolution: String,
    @JsonProperty("series_type")
    val seriesType: String,
)