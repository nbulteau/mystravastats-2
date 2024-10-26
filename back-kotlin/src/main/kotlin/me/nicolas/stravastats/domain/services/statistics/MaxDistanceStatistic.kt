package me.nicolas.stravastats.domain.services.statistics

import me.nicolas.stravastats.domain.business.ActivityShort
import me.nicolas.stravastats.domain.business.strava.StravaActivity

internal class MaxDistanceStatistic(
    activities: List<StravaActivity>,
) : ActivityStatistic("Max distance", activities) {

    private val maxDistance: Double?

    init {
        val maxDistanceActivity = activities.maxByOrNull { activity -> activity.distance }
        maxDistanceActivity?.let { activity -> this.activity = ActivityShort(activity.id, activity.name, activity.type) }
        maxDistance = maxDistanceActivity?.distance
    }

    override val value: String
        get() = if (maxDistance != null) {
            "%.2f km".format(maxDistance.div(1000))
        } else {
            "Not available"
        }
}