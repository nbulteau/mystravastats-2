package me.nicolas.stravastats.domain.services.statistics

import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.utils.formatSpeed

internal class AverageSpeedStatistic(
activities: List<StravaActivity>,
) : Statistic("Average speed", activities) {

    private val averageSpeed: Double?

    init {
        val totalDistance = activities.sumOf { activity -> activity.distance }
        val totalMovingTime = activities.sumOf { activity -> activity.movingTime }
        averageSpeed = if (activities.isNotEmpty()) {
            if (totalMovingTime > 0) {
                totalDistance / totalMovingTime
            } else {
                0.0
            }
        } else {
            null
        }
    }

    override val value: String
        get() = averageSpeed?.formatSpeed(activities.first().type) ?: "Not available"
}