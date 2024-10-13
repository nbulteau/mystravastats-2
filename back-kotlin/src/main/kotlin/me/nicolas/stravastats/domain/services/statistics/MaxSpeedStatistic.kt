package me.nicolas.stravastats.domain.services.statistics

import me.nicolas.stravastats.domain.business.strava.StravaActivity

internal class MaxSpeedStatistic(
    activities: List<StravaActivity>,
) : ActivityStatistic("Max speed", activities) {

    init {
        stravaActivity = activities.maxByOrNull { activity -> activity.maxSpeed }
    }

    override val value: String
        get() = if (stravaActivity != null) {
            "%.02f km/h".format(stravaActivity?.maxSpeed?.times(3.6))
        } else {
            "Not available"
        }
}