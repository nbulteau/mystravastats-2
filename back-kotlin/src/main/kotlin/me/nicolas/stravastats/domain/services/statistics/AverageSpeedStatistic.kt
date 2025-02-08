package me.nicolas.stravastats.domain.services.statistics

import me.nicolas.stravastats.domain.business.strava.StravaActivity

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
        get() = if (averageSpeed != null) {
            "%.2f km/h".format(averageSpeed)
        } else {
            "Not available"
        }
}