package me.nicolas.stravastats.domain.business.strava.stream

import com.fasterxml.jackson.annotation.JsonProperty

data class SmoothVelocityStream(
    // The sequence of velocity values for this stream, in meters per second
    @JsonProperty("data")
    val `data`: List<Float>,
    @JsonProperty("original_size")
    var originalSize: Int,
    @JsonProperty("resolution")
    val resolution: String,
    @JsonProperty("series_type")
    val seriesType: String,
)