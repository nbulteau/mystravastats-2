package me.nicolas.stravastats.domain.interfaces

import me.nicolas.stravastats.domain.business.strava.StravaActivity

interface IYearActivityStorageProvider {
    fun loadActivitiesFromCache(year: Int): List<StravaActivity>
}

