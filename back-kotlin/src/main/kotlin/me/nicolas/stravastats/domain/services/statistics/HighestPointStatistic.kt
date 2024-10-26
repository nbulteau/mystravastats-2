package me.nicolas.stravastats.domain.services.statistics

import me.nicolas.stravastats.domain.business.ActivityShort
import me.nicolas.stravastats.domain.business.strava.StravaActivity

internal class HighestPointStatistic(
    activities: List<StravaActivity>,
) : ActivityStatistic("Highest point", activities) {

    private val highestElevation: Double?

    init {
        val highestElevationActivity = activities.maxByOrNull { activity -> activity.elevHigh }
        highestElevationActivity?.let { activity -> this.activity = ActivityShort(activity.id, activity.name, activity.type) }
        highestElevation = highestElevationActivity?.elevHigh

    }

    override val value: String
        get() = if (highestElevation != null) {
            "%.2f m".format(highestElevation)
        } else {
            "Not available"
        }
}