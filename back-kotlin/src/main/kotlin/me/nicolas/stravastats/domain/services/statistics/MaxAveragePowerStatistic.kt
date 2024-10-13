package me.nicolas.stravastats.domain.services.statistics

import me.nicolas.stravastats.domain.business.strava.StravaActivity

internal class MaxAveragePowerStatistic(
    activities: List<StravaActivity>,
) : ActivityStatistic("Average power", activities) {

    init {
        stravaActivity = activities.maxByOrNull { activity -> activity.maxSpeed }
    }

    override val value: String
        get() = if (stravaActivity != null) {
            "%.02f W".format(stravaActivity?.averageWatts)
        } else {
            "Not available"
        }
}