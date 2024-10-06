package me.nicolas.stravastats.domain.business.badges

import me.nicolas.stravastats.domain.business.strava.Activity

sealed class Badge(
    open val label: String,
) {
    abstract fun check(activities: List<Activity>): Pair<List<Activity>, Boolean>

    override fun toString(): String {
        return label
    }
}



