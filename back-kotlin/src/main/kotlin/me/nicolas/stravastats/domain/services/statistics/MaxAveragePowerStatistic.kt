package me.nicolas.stravastats.domain.services.statistics

import me.nicolas.stravastats.domain.business.strava.Activity

internal class MaxAveragePowerStatistic(
    activities: List<Activity>,
) : ActivityStatistic("Average power", activities) {

    init {
        activity = activities.maxByOrNull { activity -> activity.maxSpeed }
    }

    override val value: String
        get() = if (activity != null) {
            "%.02f W".format(activity?.averageWatts)
        } else {
            "Not available"
        }
}