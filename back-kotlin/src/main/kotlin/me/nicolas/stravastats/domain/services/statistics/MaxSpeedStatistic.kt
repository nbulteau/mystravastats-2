package me.nicolas.stravastats.domain.services.statistics

import me.nicolas.stravastats.domain.business.ActivityShort
import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.utils.formatSpeed

internal class MaxSpeedStatistic(
    activities: List<StravaActivity>,
) : ActivityStatistic("Max speed", activities) {

    private val maxSpeed: Double?

    init {
        val maxSpeedActivity = activities.maxByOrNull { activity -> activity.maxSpeed }
        maxSpeedActivity?.let { activity -> this.activity = ActivityShort(activity.id, activity.name, activity.type) }
        maxSpeed = maxSpeedActivity?.maxSpeed?.toDouble()
    }

    override val value: String
        get() = maxSpeed?.formatSpeed(activities.first().type) ?: "Not available"

}