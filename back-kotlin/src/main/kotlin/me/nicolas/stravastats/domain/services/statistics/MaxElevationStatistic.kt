package me.nicolas.stravastats.domain.services.statistics

import me.nicolas.stravastats.domain.business.ActivityShort
import me.nicolas.stravastats.domain.business.strava.StravaActivity

internal class MaxElevationStatistic(
    activities: List<StravaActivity>,
) : ActivityStatistic("Max elevation", activities) {

    private val maxTotalElevationGain: Double?

    init {
        val maxTotalElevationGainActivity = activities.maxByOrNull { activity -> activity.totalElevationGain }
        maxTotalElevationGainActivity?.let { activity -> this.activity = ActivityShort(activity.id, activity.name, activity.type) }
        maxTotalElevationGain = maxTotalElevationGainActivity?.totalElevationGain
    }

    override val value: String
        get() = if (maxTotalElevationGain != null) {
            "%.2f m".format(maxTotalElevationGain)
        } else {
            "Not available"
        }
}