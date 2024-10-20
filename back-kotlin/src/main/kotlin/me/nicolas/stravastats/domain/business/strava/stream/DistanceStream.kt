package me.nicolas.stravastats.domain.business.strava.stream

import com.fasterxml.jackson.annotation.JsonProperty

data class DistanceStream(
    // The sequence of distance values for this stream, in meters
    @JsonProperty("data")
    val `data`: List<Double>,
    @JsonProperty("original_size")
    var originalSize: Int,
    @JsonProperty("resolution")
    val resolution: String,
    @JsonProperty("series_type")
    val seriesType: String,
)