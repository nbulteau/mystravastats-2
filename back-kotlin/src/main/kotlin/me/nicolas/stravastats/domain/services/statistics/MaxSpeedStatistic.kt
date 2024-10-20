package me.nicolas.stravastats.domain.services.statistics

import me.nicolas.stravastats.domain.business.strava.StravaActivity

internal class MaxSpeedStatistic(
    activities: List<StravaActivity>,
) : ActivityStatistic("Max speed", activities) {

    init {
        activity = activities.maxByOrNull { activity -> activity.maxSpeed }
    }

    override val value: String
        get() = if (activity != null) {
            "%.02f km/h".format(activity?.maxSpeed?.times(3.6))
        } else {
            "Not available"
        }
}