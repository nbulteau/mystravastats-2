package me.nicolas.stravastats.domain.services.statistics

import me.nicolas.stravastats.domain.business.ActivityShort
import me.nicolas.stravastats.domain.business.strava.StravaActivity

internal class MaxAveragePowerStatistic(
    activities: List<StravaActivity>,
) : ActivityStatistic("Average power", activities) {

    private val averageWatts: Int?

    init {
        val maxAverageWattsActivity = activities.maxByOrNull { activity -> activity.averageWatts }
        maxAverageWattsActivity?.let { activity -> this.activity = ActivityShort(activity.id, activity.name, activity.type) }
        averageWatts = maxAverageWattsActivity?.averageWatts
    }

    override val value: String
        get() = if (averageWatts != null) {
            "%.02f W".format(averageWatts)
        } else {
            "Not available"
        }
}