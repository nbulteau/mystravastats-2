package me.nicolas.stravastats.domain.business.badges

import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.business.strava.GeoCoordinate


data class FamousClimbBadge(
    override val label: String,
    val name: String,
    val topOfTheAscent: Int,
    val start: GeoCoordinate,
    val end: GeoCoordinate,
    val length: Double,
    val totalAscent: Int,
    val averageGradient: Double,
    val difficulty: Int,
) : Badge(label) {

    override fun check(activities: List<StravaActivity>): Pair<List<StravaActivity>, Boolean> {
        val filteredActivities = activities.filter { activity ->
            if (activity.startLatlng?.isNotEmpty() == true) {
                this.start.haversineInKM(activity.startLatlng[0], activity.startLatlng[1]) < 50
            } else {
                false
            }
        }.filter { activity ->
            check(activity, this.start) && check(activity, this.end)
        }

        return Pair(filteredActivities, filteredActivities.isNotEmpty())
    }

    private fun check(stravaActivity: StravaActivity, geoCoordinateToCheck: GeoCoordinate): Boolean {
        if (stravaActivity.stream != null && stravaActivity.stream?.latlng != null) {
            for (coords in stravaActivity.stream?.latlng?.data!!) {
                if (geoCoordinateToCheck.match(coords[0], coords[1])) {
                    return true
                }
            }
        }

        return false
    }

    override fun toString() = name
}

