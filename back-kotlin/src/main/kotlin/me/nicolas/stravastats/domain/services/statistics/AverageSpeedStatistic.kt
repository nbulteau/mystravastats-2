package me.nicolas.stravastats.domain.services.statistics

import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.utils.formatSpeed

internal class AverageSpeedStatistic(
activities: List<StravaActivity>,
) : Statistic("Average speed", activities) {

    private val averageSpeed: Double?

    init {
        val totalAverageSpeed = activities.sumOf { activity -> activity.averageSpeed }
        averageSpeed = if (activities.isNotEmpty()) {
            totalAverageSpeed.div(activities.size)
        } else {
            null
        }
    }

    override val value: String
        get() = averageSpeed?.formatSpeed(activities.first().type) ?: "Not available"
}