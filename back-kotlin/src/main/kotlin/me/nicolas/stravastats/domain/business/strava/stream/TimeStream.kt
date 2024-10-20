package me.nicolas.stravastats.domain.business.strava.stream

import com.fasterxml.jackson.annotation.JsonProperty

data class TimeStream(
    // The sequence of time values for this stream, in seconds
    @JsonProperty("data")
    val `data`: List<Int>,
    @JsonProperty("original_size")
    var originalSize: Int,
    @JsonProperty("resolution")
    val resolution: String,
    @JsonProperty("series_type")
    val seriesType: String,
)