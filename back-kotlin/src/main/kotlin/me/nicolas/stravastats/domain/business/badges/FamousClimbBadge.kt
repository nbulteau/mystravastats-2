package me.nicolas.stravastats.domain.business.badges

import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.business.strava.GeoCoordinate

private const val FAMOUS_CLIMB_ACTIVITY_START_RADIUS_KM = 80.0
private const val FAMOUS_CLIMB_WAYPOINT_TOLERANCE_METERS = 500


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
    val category: String,
) : Badge(label) {

    override fun check(activities: List<StravaActivity>): Pair<List<StravaActivity>, Boolean> {
        val filteredActivities = activities.filter { activity ->
            if (activity.startLatlng?.isNotEmpty() == true) {
                val distanceToStart = this.start.haversineInKM(activity.startLatlng[0], activity.startLatlng[1])
                val distanceToEnd = this.end.haversineInKM(activity.startLatlng[0], activity.startLatlng[1])
                distanceToStart < FAMOUS_CLIMB_ACTIVITY_START_RADIUS_KM || distanceToEnd < FAMOUS_CLIMB_ACTIVITY_START_RADIUS_KM
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
                if (geoCoordinateToCheck.haversineInM(coords[0], coords[1]) < FAMOUS_CLIMB_WAYPOINT_TOLERANCE_METERS) {
                    return true
                }
            }
        }

        return false
    }

    override fun toString() = name
}
