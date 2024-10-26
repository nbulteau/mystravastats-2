package me.nicolas.stravastats.domain.services.statistics

import me.nicolas.stravastats.domain.business.ActivityShort
import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.utils.formatSeconds

internal class MaxMovingTimeStatistic(
    activities: List<StravaActivity>,
) : ActivityStatistic("Max moving time", activities) {

    private val maxMovingTime: Int?

    init {
        val maxMovingTimeActivity = activities.maxByOrNull { activity -> activity.movingTime }
        maxMovingTimeActivity?.let { activity -> this.activity = ActivityShort(activity.id, activity.name, activity.type) }
        maxMovingTime = maxMovingTimeActivity?.movingTime
    }

    override val value: String
        get() = if (maxMovingTime != null) {
            maxMovingTime.formatSeconds()
        } else {
            "Not available"
        }
}