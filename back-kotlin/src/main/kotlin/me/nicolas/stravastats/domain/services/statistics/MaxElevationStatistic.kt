package me.nicolas.stravastats.domain.services.statistics

import me.nicolas.stravastats.domain.business.strava.StravaActivity

internal class MaxElevationStatistic(
    activities: List<StravaActivity>,
) : ActivityStatistic("Max elevation", activities) {

    init {
        stravaActivity = activities.maxByOrNull { activity -> activity.totalElevationGain }
    }

    override val value: String
        get() = if (stravaActivity != null) {
            "%.2f m".format(stravaActivity?.totalElevationGain)
        } else {
            "Not available"
        }
}