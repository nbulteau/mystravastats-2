package me.nicolas.stravastats.domain.services.statistics

import me.nicolas.stravastats.domain.business.strava.StravaActivity

internal class MaxElevationStatistic(
    activities: List<StravaActivity>,
) : ActivityStatistic("Max elevation", activities) {

    init {
        activity = activities.maxByOrNull { activity -> activity.totalElevationGain }
    }

    override val value: String
        get() = if (activity != null) {
            "%.2f m".format(activity?.totalElevationGain)
        } else {
            "Not available"
        }
}