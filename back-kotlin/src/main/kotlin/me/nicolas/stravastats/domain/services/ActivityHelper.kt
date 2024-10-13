package me.nicolas.stravastats.domain.services

import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.business.ActivityType

object ActivityHelper {




    /**
     * Remove activities that are not in the list of stravaActivity types to consider (i.e. Run, Ride, Hike, etc.)
     * @return a list of activities filtered by type
     * @see StravaActivity
     */
    fun List<StravaActivity>.filterByActivityTypes() = this.filter { activity ->
        ActivityType.entries.any { activity.type == it.name }
    }
}

