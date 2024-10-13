package me.nicolas.stravastats.domain.services.statistics

import me.nicolas.stravastats.domain.business.strava.StravaActivity

internal class MaxDistanceStatistic(
    activities: List<StravaActivity>,
) : ActivityStatistic("Max distance", activities) {

    init {
        stravaActivity = activities.maxByOrNull { activity -> activity.distance }
    }

    override val value: String
        get() = if (stravaActivity != null) {
            "%.2f km".format(stravaActivity?.distance?.div(1000))
        } else {
            "Not available"
        }
}