package me.nicolas.stravastats.domain.business.strava.stream

import com.fasterxml.jackson.annotation.JsonProperty

data class TimeStream(
    // The sequence of time values for this stream, in seconds
    @param:JsonProperty("data")
    val `data`: List<Int>,
    @param:JsonProperty("original_size")
    var originalSize: Int,
    @param:JsonProperty("resolution")
    val resolution: String,
    @param:JsonProperty("series_type")
    val seriesType: String,
)