package me.nicolas.stravastats.domain.business.strava.stream

import com.fasterxml.jackson.annotation.JsonProperty

data class SmoothGradeStream(
    // The sequence of grade values for this stream, as percents of a grade (ex. 10.0 for 10.0%)
    @param:JsonProperty("data")
    val `data`: List<Float>,
    @param:JsonProperty("original_size")
    var originalSize: Int,
    @param:JsonProperty("resolution")
    val resolution: String,
    @param:JsonProperty("series_type")
    val seriesType: String,
)