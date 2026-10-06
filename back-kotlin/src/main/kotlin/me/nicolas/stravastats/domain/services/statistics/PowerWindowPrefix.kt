package me.nicolas.stravastats.domain.services.statistics

import me.nicolas.stravastats.domain.business.strava.stream.PowerTimeline

/** Duration-weighted means; never infer time from a sample count. */
internal class PowerWindowPrefix(samples: List<Int?>?, times: List<Int>) {
    private val timeline = PowerTimeline(samples.orEmpty(), times)
    fun average(start: Int, end: Int): Double? = timeline.averageIndices(start, end)
}
