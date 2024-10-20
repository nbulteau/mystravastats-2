package me.nicolas.stravastats.domain.services.statistics

import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.utils.formatSeconds

internal class MaxMovingTimeStatistic(
    activities: List<StravaActivity>,
) : ActivityStatistic("Max moving time", activities) {

    init {
        activity = activities.maxByOrNull { activity -> activity.movingTime }
    }

    override val value: String
        get() = if (activity != null) {
            "${activity?.movingTime?.formatSeconds()}"
        } else {
            "Not available"
        }
}