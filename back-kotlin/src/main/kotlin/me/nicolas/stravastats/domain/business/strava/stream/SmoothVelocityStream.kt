package me.nicolas.stravastats.domain.business.strava.stream

import com.fasterxml.jackson.annotation.JsonProperty

data class SmoothVelocityStream(
    // The sequence of velocity values for this stream, in meters per second
    @param:JsonProperty("data")
    val `data`: List<Float>,
    @param:JsonProperty("original_size")
    var originalSize: Int,
    @param:JsonProperty("resolution")
    val resolution: String,
    @param:JsonProperty("series_type")
    val seriesType: String,
)