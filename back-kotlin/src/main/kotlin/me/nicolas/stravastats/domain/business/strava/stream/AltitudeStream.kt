package me.nicolas.stravastats.domain.business.strava.stream

import com.fasterxml.jackson.annotation.JsonProperty

data class AltitudeStream(
    // The sequence of altitude values for this stream, in meters
    @JsonProperty("data")
    val `data`: List<Double>,
    // The number of data points in this stream
    @JsonProperty("original_size")
    var originalSize: Int,
    @JsonProperty("resolution")
    // The level of detail (sampling) in which this stream was returned May take one of the following values: low, medium, high
    val resolution: String,
    @JsonProperty("series_type")
    // The base series used in the case the stream was downsampled May take one of the following values: distance, time
    val seriesType: String,
) {
    constructor(data: List<Double>) : this(data.toMutableList(), data.size, "high", "distance")
}