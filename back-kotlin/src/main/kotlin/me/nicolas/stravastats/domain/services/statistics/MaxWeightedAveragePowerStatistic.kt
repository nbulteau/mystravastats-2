package me.nicolas.stravastats.domain.services.statistics

import me.nicolas.stravastats.domain.business.ActivityShort
import me.nicolas.stravastats.domain.business.strava.StravaActivity

/**
 * Weighted Average Power (WAP) is a power calculation that takes into account the variability of your power output.
 */
internal class MaxWeightedAveragePowerStatistic(
    activities: List<StravaActivity>,
) : ActivityStatistic("Weighted average power", activities) {

    private val maxWeightedAverageWatts: Int?

    init {
        val maxWeightedAverageWattsActivity = activities.maxByOrNull { activity -> activity.weightedAverageWatts }
        maxWeightedAverageWattsActivity?.let{ activity -> this.activity = ActivityShort(activity.id, activity.name, activity.type) }
        maxWeightedAverageWatts = maxWeightedAverageWattsActivity?.weightedAverageWatts
    }

    override val value: String
        get() = if (maxWeightedAverageWatts != null) {
            "%d W".format(maxWeightedAverageWatts)
        } else {
            "Not available"
        }
}