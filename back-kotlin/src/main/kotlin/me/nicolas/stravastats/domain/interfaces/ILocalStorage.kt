package me.nicolas.stravastats.domain.interfaces

import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.business.strava.StravaAthlete
import me.nicolas.stravastats.domain.business.strava.StravaDetailedActivity
import me.nicolas.stravastats.domain.business.strava.stream.Stream

interface ILocalStorageProvider {

    fun initLocalStorageForClientId(clientId: String)

    fun loadAthleteFromCache(clientId: String): StravaAthlete

    fun saveAthleteToCache(clientId: String, stravaAthlete: StravaAthlete)

    fun loadActivitiesFromCache(clientId: String, year: Int): List<StravaActivity>

    fun saveActivitiesToCache(clientId: String, year: Int, activities: List<StravaActivity>)

    fun loadActivitiesStreamsFromCache(clientId: String, year: Int, stravaActivity: StravaActivity): Stream?

    fun saveActivitiesStreamsToCache(clientId: String, year: Int, stravaActivity: StravaActivity, stream: Stream)

    fun buildStreamIdsSet(clientId: String, year: Int): Set<Long>

    fun isLocalCacheExistForYear(clientId: String, year: Int): Boolean

    fun loadDetailedActivityFromCache(clientId: String, year: Int, activityId: Long): StravaDetailedActivity?

    fun saveDetailedActivityToCache(clientId: String, year: Int, stravaDetailedActivity: StravaDetailedActivity)

    fun readStravaAuthentication(stravaCache: String): Triple<String?, String?, Boolean?>
}