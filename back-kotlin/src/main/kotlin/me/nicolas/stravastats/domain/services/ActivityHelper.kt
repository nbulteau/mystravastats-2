package me.nicolas.stravastats.domain.services

import me.nicolas.stravastats.domain.business.strava.Activity
import me.nicolas.stravastats.domain.business.strava.ActivityType

object ActivityHelper {




    /**
     * Remove activities that are not in the list of activity types to consider (i.e. Run, Ride, Hike, etc.)
     * @return a list of activities filtered by type
     * @see Activity
     */
    fun List<Activity>.filterByActivityTypes() = this.filter { activity ->
        ActivityType.entries.any { activity.type == it.name }
    }
}

