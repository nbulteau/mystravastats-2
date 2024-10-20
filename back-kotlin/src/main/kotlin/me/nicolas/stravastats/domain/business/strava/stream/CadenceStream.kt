package me.nicolas.stravastats.domain.business.strava.stream

import com.fasterxml.jackson.annotation.JsonProperty

data class CadenceStream(
    // The sequence of cadence values for this stream, in rotations per minute
    @JsonProperty("data")
    val `data`: List<Int>,
    // The number of data points in this stream
    @JsonProperty("original_size")
    var originalSize: Int,
    @JsonProperty("resolution")
    // The level of detail (sampling) in which this stream was returned May take one of the following values: low, medium, high
    val resolution: String,
    @JsonProperty("series_type")
    // The base series used in the case the stream was downsampled May take one of the following values: distance, time
    val seriesType: String,
)