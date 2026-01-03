package me.nicolas.stravastats.domain.services.activityproviders

import kotlinx.coroutines.async
import kotlinx.coroutines.awaitAll
import kotlinx.coroutines.coroutineScope
import kotlinx.coroutines.Dispatchers
import me.nicolas.stravastats.adapters.strava.StravaApi
import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.business.strava.StravaAthlete
import me.nicolas.stravastats.domain.business.strava.StravaDetailedActivity
import me.nicolas.stravastats.domain.business.strava.stream.Stream
import me.nicolas.stravastats.domain.interfaces.ILocalStorageProvider
import me.nicolas.stravastats.domain.interfaces.IStravaApi
import me.nicolas.stravastats.adapters.localrepositories.strava.StravaRepository
import me.nicolas.stravastats.domain.services.ActivityHelper.filterByActivityTypes
import me.nicolas.stravastats.domain.services.toStravaDetailedActivity
import org.slf4j.LoggerFactory
import java.time.Instant
import java.time.LocalDate
import java.util.*
import kotlin.system.measureTimeMillis

class StravaActivityProvider(
    // allow injection for testability; accept the interface publicly to avoid exposing internal implementation
    localStorageProvider: ILocalStorageProvider? = null,
    private var stravaApi: IStravaApi? = null,
    stravaCache: String = "strava-cache",
) : AbstractActivityProvider() {

    // Internally keep a concrete reference (default to StravaRepository when none injected)
    private val storageProvider: ILocalStorageProvider = localStorageProvider ?: StravaRepository(stravaCache)

    private val logger = LoggerFactory.getLogger(StravaActivityProvider::class.java)

    private val clientId: String

    // keep auth info for deferred initialization
    private val authSecret: String?
    private val useCacheAuth: Boolean?

    companion object {
        // Threshold instant: 2025-08-17T00:00:00Z (replace with desired date/time)
        private val CACHE_RELOAD_THRESHOLD: Instant = Instant.parse("2025-08-17T00:00:00Z")
    }

    init {
        // read authentication but do not perform blocking loads here
        val (id, secret, useCache) = storageProvider.readStravaAuthentication(stravaCache)
        if (id == null) {
            // Throw instead of exiting the process to let the application decide how to handle it
            throw IllegalStateException("Strava authentication not found")
        }

        clientId = id
        authSecret = secret
        useCacheAuth = useCache

        // if an API implementation wasn't injected but we have credentials, create the default API
        if (stravaApi == null && authSecret != null) {
            stravaApi = StravaApi(clientId, authSecret)
        }

        // Load athlete from cache immediately if cache flag is set (this is cheap/local)
        if (useCacheAuth == true) {
            stravaAthlete = storageProvider.loadAthleteFromCache(clientId)
        }

        logger.info("ActivityService prepared with clientId=$clientId (initial loading deferred)" )
    }

    suspend fun initializeAndLoadActivities() = coroutineScope {
        // If configured to use cache, load from local cache
        if (useCacheAuth == true) {
            activities = loadFromLocalCache()
            logger.info("ActivityService initialized with clientId=$clientId and ${activities.size} activities (from cache)")
            return@coroutineScope
        }

        // If we have credentials, initialize storage and load from Strava
        if (authSecret != null) {
            storageProvider.initLocalStorageForClientId(clientId)
            // retrieve athlete and activities from Strava
            stravaAthlete = retrieveLoggedInAthlete()
            activities = loadActivities()
            logger.info("ActivityService initialized with clientId=$clientId and ${activities.size} activities (from Strava)")
            return@coroutineScope
        }

        throw IllegalStateException("No valid Strava authentication available to load activities")
    }

    override fun getDetailedActivity(activityId: Long): Optional<StravaDetailedActivity> {
        logger.info("Get detailed activity for activity id $activityId")

        // find detailed activity in cache or retrieve from Strava
        val activity = activities.find { it.id == activityId } ?: return Optional.empty()
        val year = activity.startDate.take(4).toInt()

        // load detailed activity from cache or retrieve from Strava
        var stravaDetailedActivity = storageProvider.loadDetailedActivityFromCache(clientId, year, activityId)
        if (stravaApi != null && stravaDetailedActivity == null) {
            // It's not in local cache, retrieve from Strava
            val detailedActivity = stravaApi!!.getDetailedActivity(activityId)
            if (detailedActivity.isPresent) {
                storageProvider.saveDetailedActivityToCache(clientId, year, detailedActivity.get())
                stravaDetailedActivity = detailedActivity.get()
            }
        }

        if (stravaDetailedActivity == null) {
            // Detailed activity not found on Strava, return the activity without details
            stravaDetailedActivity = activity.toStravaDetailedActivity()
        }

        // load stream from cache or retrieve from Strava
        var stream = storageProvider.loadActivitiesStreamsFromCache(clientId, year, activity)
        if (stravaApi != null && stream == null) {
            stream = stravaApi!!.getActivityStream(activity)
            if (stream != null) {
                storageProvider.saveActivitiesStreamsToCache(clientId, year, activity, stream)
            }
        }
        stravaDetailedActivity.stream = stream

        return Optional.of(stravaDetailedActivity)
    }

    private suspend fun loadFromLocalCache(): List<StravaActivity> = coroutineScope {
        logger.info("Load Strava activities from local cache ...")

        val loadedActivities = mutableListOf<StravaActivity>()
        val elapsed = measureTimeMillis {
            val deferredActivities = (LocalDate.now().year downTo 2010).map { year ->
                async(Dispatchers.IO) {
                    try {
                        logger.info("Load $year activities ...")
                        storageProvider.loadActivitiesFromCache(clientId, year)
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

    private suspend fun loadActivities(): List<StravaActivity> = coroutineScope {
        logger.info("Loading Strava activities ...")
        val currentYear = LocalDate.now().year
        val loadedActivities = mutableListOf<StravaActivity>()
        val elapsed = measureTimeMillis {
            val deferredActivities = (currentYear downTo 2010).map { year ->
                async(Dispatchers.IO) {
                    try {
                        // Check if we should load from cache or API
                        if (currentYear != year
                            && storageProvider.isLocalCacheExistForYear(clientId, year)
                            && !shouldReloadFromStravaAPI(year)) {
                            logger.info("Loading activities for $year from cache ...")
                            val activities = storageProvider.loadActivitiesFromCache(clientId, year)
                            loadMissingStreamsFromCache(year, activities)
                            // now parallelized
                            loadMissingStreamsFromApi(year, activities)
                        } else {
                            logger.info("Loading activities for $year from Strava API ...")
                            val activities = retrieveActivitiesFromApi(year)
                            saveActivitiesToCache(year, activities)
                            loadMissingStreamsFromCache(year, activities)
                            // now parallelized
                            loadMissingStreamsFromApi(year, activities)
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
    private fun shouldReloadFromStravaAPI(year: Int): Boolean {
        // If the file is older than CACHE_RELOAD_THRESHOLD, it needs to be reloaded
        return storageProvider.getLocalCacheLastModified(clientId, year) < CACHE_RELOAD_THRESHOLD.toEpochMilli()
    }

    // Loads missing streams from the cache
    private fun loadMissingStreamsFromCache(
        year: Int,
        activities: List<StravaActivity>
    ): List<StravaActivity> {
        activities
            // Filter activities that do not have a stream
            .filter { it.stream == null }
            .forEach { activity ->
                val stream = storageProvider.loadActivitiesStreamsFromCache(clientId, year, activity)
                activity.stream = stream
            }

        return activities
    }

    // Loads missing streams from API (parallelized)
    suspend fun loadMissingStreamsFromApi(
        year: Int,
        activities: List<StravaActivity>
    ): List<StravaActivity> = coroutineScope {
        val api = stravaApi ?: return@coroutineScope activities

        val deferred = activities
            .filter { activity -> activity.stream == null }
            .map { activity ->
                async(Dispatchers.IO) {
                    try {
                        api.getActivityStream(activity)?.let { stream ->
                            storageProvider.saveActivitiesStreamsToCache(clientId, year, activity, stream)
                            activity.stream = stream
                        }
                    } catch (exception: Exception) {
                        logger.error("Error loading stream for activity ${activity.id}", exception)
                    }
                    activity
                }
            }

        deferred.awaitAll()
        return@coroutineScope activities
    }

    // Retrieves activities from Strava API
    private fun retrieveActivitiesFromApi(year: Int): List<StravaActivity> {
        return stravaApi?.getActivities(year)?.filterByActivityTypes() ?: emptyList()
    }

    // Saves activities to cache
    private fun saveActivitiesToCache(year: Int, activities: List<StravaActivity>) {
        storageProvider.saveActivitiesToCache(clientId, year, activities)
    }

    private fun loadActivitiesStreams(year: Int, activities: List<StravaActivity>) {

        // stream id file list
        val streamIdsSet = storageProvider.buildStreamIdsSet(clientId, year)

        activities.forEach { activity ->
            val stream: Stream?

            if (streamIdsSet.contains(activity.id)) {
                stream = storageProvider.loadActivitiesStreamsFromCache(clientId, year, activity)
            } else {
                if (stravaApi != null) {
                    stream = stravaApi!!.getActivityStream(activity)
                    if (stream != null) {
                        storageProvider.saveActivitiesStreamsToCache(clientId, year, activity, stream)
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

    private fun retrieveLoggedInAthlete(): StravaAthlete {
        logger.info("Load stravaAthlete with id $clientId description from Strava")

        return if (stravaApi != null) {
            val athlete = stravaApi!!.retrieveLoggedInAthlete()
            if (athlete.isPresent) {
                storageProvider.saveAthleteToCache(clientId, athlete.get())
            }
            athlete.get()
        } else {
            storageProvider.loadAthleteFromCache(clientId)
        }
    }

}
