package me.nicolas.stravastats.domain.services.statistics

import me.nicolas.stravastats.domain.business.ActivityShort
import me.nicolas.stravastats.domain.business.strava.StravaActivity

internal class MaxSpeedStatistic(
    activities: List<StravaActivity>,
) : ActivityStatistic("Max speed", activities) {

    private val maxSpeed: Float?

    init {
        val maxSpeedActivity = activities.maxByOrNull { activity -> activity.maxSpeed }
        maxSpeedActivity?.let { activity -> this.activity = ActivityShort(activity.id, activity.name, activity.type) }
        maxSpeed = maxSpeedActivity?.maxSpeed
    }

    override val value: String
        get() = if (maxSpeed != null) {
            "%.02f km/h".format(maxSpeed.times(3.6))
        } else {
            "Not available"
        }
}