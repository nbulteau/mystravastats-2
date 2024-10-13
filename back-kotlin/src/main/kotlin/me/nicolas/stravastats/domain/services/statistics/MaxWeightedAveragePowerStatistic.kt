package me.nicolas.stravastats.domain.services.statistics

import me.nicolas.stravastats.domain.business.strava.StravaActivity

/**
 * Weighted Average Power (WAP) is a power calculation that takes into account the variability of your power output.
 */
internal class MaxWeightedAveragePowerStatistic(
    activities: List<StravaActivity>,
) : ActivityStatistic("Weighted average power", activities) {

    init {
        stravaActivity = activities.maxByOrNull { activity -> activity.weightedAverageWatts }
    }

    override val value: String
        get() = if (stravaActivity != null) {
            "%d W".format(stravaActivity?.weightedAverageWatts)
        } else {
            "Not available"
        }
}