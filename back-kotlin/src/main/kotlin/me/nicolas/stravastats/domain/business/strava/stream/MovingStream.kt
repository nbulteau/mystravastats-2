package me.nicolas.stravastats.domain.business.strava.stream

import com.fasterxml.jackson.annotation.JsonProperty

data class MovingStream(
    // The sequence of moving values for this stream, as boolean values
    @param:JsonProperty("data")
    val `data`: List<Boolean>,
    @param:JsonProperty("original_size")
    val originalSize: Int,
    @param:JsonProperty("resolution")
    val resolution: String,
    @param:JsonProperty("series_type")
    val seriesType: String,
)