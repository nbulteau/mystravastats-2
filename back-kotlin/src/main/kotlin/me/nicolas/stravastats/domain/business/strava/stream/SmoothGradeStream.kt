package me.nicolas.stravastats.domain.business.strava.stream

import com.fasterxml.jackson.annotation.JsonProperty

data class SmoothGradeStream(
    // The sequence of grade values for this stream, as percents of a grade (ex. 10.0 for 10.0%)
    @JsonProperty("data")
    val `data`: List<Float>,
    @JsonProperty("original_size")
    var originalSize: Int,
    @JsonProperty("resolution")
    val resolution: String,
    @JsonProperty("series_type")
    val seriesType: String,
)