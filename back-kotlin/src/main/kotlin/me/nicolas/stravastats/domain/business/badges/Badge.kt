package me.nicolas.stravastats.domain.business.badges

import me.nicolas.stravastats.domain.business.strava.StravaActivity

sealed class Badge(
    open val label: String,
) {
    abstract fun check(activities: List<StravaActivity>): Pair<List<StravaActivity>, Boolean>

    override fun toString(): String {
        return label
    }
}



