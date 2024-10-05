package me.nicolas.stravastats.domain.interfaces

import me.nicolas.stravastats.domain.business.strava.Activity
import me.nicolas.stravastats.domain.business.strava.Athlete
import me.nicolas.stravastats.domain.business.strava.DetailledActivity
import me.nicolas.stravastats.domain.business.strava.Stream
import java.io.File

interface ILocalStorageProvider {

    fun loadAthleteFromCache(clientId: String): Athlete?

    fun saveAthleteToCache(clientId: String, athlete: Athlete)

    fun loadActivitiesFromCache(clientId: String, year: Int): List<Activity>

    fun saveActivitiesToCache(clientId: String, year: Int, activities: List<Activity>)

    fun loadDetailedActivityFromCache(clientId: String, year: Int, activityId: Long): DetailledActivity?

    fun saveDetailedActivityToCache(clientId: String, year: Int, detailedActivity: DetailledActivity)

    fun loadActivitiesStreamsFromCache(clientId: String, year: Int, activity: Activity): Stream?

    fun saveActivitiesStreamsToCache(clientId: String, year: Int, activity: Activity, stream: Stream)

    fun buildStreamIdsSet(clientId: String, year: Int): Set<Long>

    fun isLocalCacheExistForYear(clientId: String, year: Int): Boolean
}