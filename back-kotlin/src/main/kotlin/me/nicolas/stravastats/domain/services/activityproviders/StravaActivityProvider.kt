package me.nicolas.stravastats.domain.services.activityproviders

import kotlinx.coroutines.async
import kotlinx.coroutines.awaitAll
import kotlinx.coroutines.coroutineScope
import kotlinx.coroutines.runBlocking
import me.nicolas.stravastats.adapters.localrepositories.strava.StravaRepository
import me.nicolas.stravastats.adapters.strava.StravaApi
import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.business.strava.StravaAthlete
import me.nicolas.stravastats.domain.business.strava.StravaDetailedActivity
import me.nicolas.stravastats.domain.business.strava.stream.Stream
import me.nicolas.stravastats.domain.interfaces.IStravaApi
import me.nicolas.stravastats.domain.services.ActivityHelper.filterByActivityTypes
import me.nicolas.stravastats.domain.services.toStravaDetailedActivity
import org.slf4j.LoggerFactory
import java.time.LocalDate
import java.util.*
import kotlin.system.exitProcess
import kotlin.system.measureTimeMillis

class StravaActivityProvider(
    stravaCache: String = "strava-cache",
) : AbstractActivityProvider() {

    private val logger = LoggerFactory.getLogger(StravaActivityProvider::class.java)

    private val localStorageProvider = StravaRepository(stravaCache = stravaCache)

    private val clientId: String

    private var stravaApi: IStravaApi? = null

    init {
        localStorageProvider.readStravaAuthentication(stravaCache).let { (id, secret, useCache) ->
            if (id == null) {
                logger.error("Strava authentication not found")
                exitProcess(-1)
            }

            clientId = id
            if (useCache == true) {
                stravaAthlete = localStorageProvider.loadAthleteFromCache(clientId)
                activities = runBlocking { loadFromLocalCache(clientId) }
            } else {
                if (secret != null) {
                    localStorageProvider.initLocalStorageForClientId(clientId)
                    stravaApi = StravaApi(clientId, secret)
                    stravaAthlete = retrieveLoggedInAthlete(clientId)
                    activities = runBlocking { loadActivities(clientId) }
                } else {
                    logger.error("Strava authentication not found")
                    exitProcess(-1)
                }
            }
        }

        logger.info("ActivityService initialized with clientId=$clientId and ${activities.size} activities")
    }

    override fun getDetailedActivity(activityId: Long): Optional<StravaDetailedActivity> {
        logger.info("Get detailed activity for activity id $activityId")

        // find detailed activity in cache or retrieve from Strava
        val activity = activities.find { it.id == activityId } ?: return Optional.empty()
        val year = activity.startDate.take(4).toInt()

        // load detailed activity from cache or retrieve from Strava
        var stravaDetailedActivity = localStorageProvider.loadDetailedActivityFromCache(clientId, year, activityId)
        if (stravaApi != null && stravaDetailedActivity == null) {
            // It's not in local cache, retrieve from Strava
            val detailedActivity = stravaApi!!.getDetailedActivity(activityId)
            if (detailedActivity.isPresent) {
                localStorageProvider.saveDetailedActivityToCache(clientId, year, detailedActivity.get())
                stravaDetailedActivity = detailedActivity.get()
            }
        }

        if (stravaDetailedActivity == null) {
            // Detailed activity not found on Strava, return the activity without details
            stravaDetailedActivity = activity.toStravaDetailedActivity()
        }

        // load stream from cache or retrieve from Strava
        var stream = localStorageProvider.loadActivitiesStreamsFromCache(clientId, year, activity)
        if (stravaApi != null && stream == null) {
            stream = stravaApi!!.getActivityStream(activity)
            if (stream != null) {
                localStorageProvider.saveActivitiesStreamsToCache(clientId, year, activity, stream)
            }
        }
        stravaDetailedActivity.stream = stream

        return Optional.of(stravaDetailedActivity)
    }

    private suspend fun loadFromLocalCache(clientId: String): List<StravaActivity> = coroutineScope {
        logger.info("Load Strava activities from local cache ...")

        val loadedActivities = mutableListOf<StravaActivity>()
        val elapsed = measureTimeMillis {
            val deferredActivities = (LocalDate.now().year downTo 2010).map { year ->
                async {
                    try {
                        logger.info("Load $year activities ...")
                        localStorageProvider.loadActivitiesFromCache(clientId, year)
                    } catch (e: Exception) {
                        logger.error("Error loading activities for year $year from local cache", e)
                        emptyList()
                    }
                }
            }
            loadedActivities.addAll(deferredActivities.awaitAll().flatten())
        }
        logger.info("${loadedActivities.size} activities loaded form local cache in ${elapsed / 1000} s.")

        return@coroutineScope loadedActivities
    }

    private suspend fun loadActivities(clientId: String): List<StravaActivity> = coroutineScope {
        logger.info("Loading Strava activities ...")
        val currentYear = LocalDate.now().year
        val loadedActivities = mutableListOf<StravaActivity>()
        val elapsed = measureTimeMillis {
            val deferredActivities = (currentYear downTo 2010).map { year ->
                async {
                    try {
                        // Check if we should load from cache or API
                        if (currentYear != year
                            && localStorageProvider.isLocalCacheExistForYear(clientId, year)
                            && !shouldReloadFromStravaAPI(clientId, year)) {
                            logger.info("Loading activities for $year from cache ...")
                            val activities = localStorageProvider.loadActivitiesFromCache(clientId, year)
                            loadMissingStreamsFromCache(clientId, year, activities)
                            loadMissingStreamsFromApi(clientId, year, activities)
                        } else {
                            logger.info("Loading activities for $year from Strava API ...")
                            val activities = retrieveActivitiesFromApi(clientId, year)
                            saveActivitiesToCache(clientId, year, activities)
                            loadMissingStreamsFromCache(clientId, year, activities)
                            loadMissingStreamsFromApi(clientId, year, activities)
                        }
                    } catch (exception: Exception) {
                        logger.error("Error loading activities for year $year", exception)
                        emptyList()
                    }
                }
            }
            loadedActivities.addAll(deferredActivities.awaitAll().flatten())
        }
        logger.info("${loadedActivities.size} activities loaded in ${elapsed / 1000} s.")
        return@coroutineScope loadedActivities
    }

    // Determines if activities should be reloaded from Strava API
    private fun shouldReloadFromStravaAPI(clientId: String, year: Int): Boolean {
        // If the file is older than 17 August 2025, it needs to be reloaded
        return localStorageProvider.getLocalCacheLastModified(clientId, year) < 1755408900L // 17 August 2025 in milliseconds
    }

    // Loads missing streams from the cache
    private fun loadMissingStreamsFromCache(
        clientId: String,
        year: Int,
        activities: List<StravaActivity>
    ): List<StravaActivity> {
        activities
            // Filter activities that do not have a stream
            .filter { it.stream == null }
            .forEach { activity ->
                val stream = localStorageProvider.loadActivitiesStreamsFromCache(clientId, year, activity)
                activity.stream = stream
            }

        return activities
    }

    // Loads missing streams from API
    private fun loadMissingStreamsFromApi(
        clientId: String,
        year: Int,
        activities: List<StravaActivity>
    ): List<StravaActivity> {
        activities
            // Filter activities that do not have a stream
            .filter { activity -> activity.stream == null }
            .forEach { activity ->
                stravaApi?.getActivityStream(activity)?.let { stream ->
                    localStorageProvider.saveActivitiesStreamsToCache(clientId, year, activity, stream)
                    activity.stream = stream
                }
            }

        return activities
    }

    // Retrieves activities from Strava API
    private fun retrieveActivitiesFromApi(clientId: String, year: Int): List<StravaActivity> {
        return stravaApi?.getActivities(year)?.filterByActivityTypes() ?: emptyList()
    }

    // Saves activities to cache
    private fun saveActivitiesToCache(clientId: String, year: Int, activities: List<StravaActivity>) {
        localStorageProvider.saveActivitiesToCache(clientId, year, activities)
    }

    private fun loadActivitiesStreams(clientId: String, year: Int, activities: List<StravaActivity>) {

        // stream id file list
        val streamIdsSet = localStorageProvider.buildStreamIdsSet(clientId, year)

        activities.forEach { activity ->
            val stream: Stream?

            if (streamIdsSet.contains(activity.id)) {
                stream = localStorageProvider.loadActivitiesStreamsFromCache(clientId, year, activity)
            } else {
                if (stravaApi != null) {
                    stream = stravaApi!!.getActivityStream(activity)
                    if (stream != null) {
                        localStorageProvider.saveActivitiesStreamsToCache(clientId, year, activity, stream)
                    } else {
                        logger.warn("Stream for activity ${activity.id} not found in Strava API")
                    }
                } else {
                    stream = null
                }
            }

            // Clean stream
            activity.stream = stream
        }
    }

    private fun retrieveLoggedInAthlete(clientId: String): StravaAthlete {
        logger.info("Load stravaAthlete with id $clientId description from Strava")

        return if (stravaApi != null) {
            val athlete = stravaApi!!.retrieveLoggedInAthlete()
            if (athlete.isPresent) {
                localStorageProvider.saveAthleteToCache(clientId, athlete.get())
            }
            athlete.get()
        } else {
            localStorageProvider.loadAthleteFromCache(clientId)
        }
    }

    private fun retrieveActivities(clientId: String, year: Int): List<StravaActivity> {
        logger.info("Load activities from Strava for year $year")

        if (stravaApi != null) {
            val retriedActivities = stravaApi!!.getActivities(year).filterByActivityTypes()

            localStorageProvider.saveActivitiesToCache(clientId, year, retriedActivities)

            this.loadActivitiesStreams(clientId, year, retriedActivities)

            logger.info("${retriedActivities.size} activities loaded")

            return retriedActivities
        }

        return emptyList()
    }
}