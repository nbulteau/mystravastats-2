package me.nicolas.stravastats.domain.business.badges

import me.nicolas.stravastats.domain.business.strava.Activity

data class BadgeSet(val name: String, val badges: List<Badge>) {

    fun check(activities: List<Activity>): List<Triple<Badge, Activity?, Boolean>> {
        return badges.map { badge ->
            val (activity, isCompleted) = badge.check(activities)
            Triple(badge, activity, isCompleted)
        }
    }
}