package me.nicolas.stravastats.domain.services.statistics

import me.nicolas.stravastats.domain.business.strava.Activity

internal class HighestPointStatistic(
    activities: List<Activity>,
) : ActivityStatistic("Highest point", activities) {

    init {
        activity = activities.maxByOrNull { activity -> activity.elevHigh }
    }

    override val value: String
        get() = if (activity != null) {
            "%.2f m".format(activity!!.elevHigh)
        } else {
            "Not available"
        }
}