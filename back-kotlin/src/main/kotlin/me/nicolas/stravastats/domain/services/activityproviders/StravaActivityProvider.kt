package me.nicolas.stravastats.domain.services.activityproviders

import kotlinx.coroutines.*
import kotlinx.coroutines.sync.Semaphore
import kotlinx.coroutines.sync.withPermit
import me.nicolas.stravastats.adapters.localrepositories.strava.StravaRepository
import me.nicolas.stravastats.adapters.strava.StravaApi
import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.business.strava.StravaAthlete
import me.nicolas.stravastats.domain.business.strava.StravaDetailedActivity
import me.nicolas.stravastats.domain.interfaces.ILocalStorageProvider
import me.nicolas.stravastats.domain.interfaces.IStravaApi
import me.nicolas.stravastats.domain.services.ActivityHelper.filterByActivityTypes
import me.nicolas.stravastats.domain.services.statistics.BestEffortCache
import me.nicolas.stravastats.domain.services.toStravaDetailedActivity
import me.nicolas.stravastats.domain.utils.GenericCache
import me.nicolas.stravastats.domain.utils.SoftCache
import org.slf4j.LoggerFactory
import java.time.LocalDate
import java.util.*
import java.util.concurrent.TimeUnit
import java.util.concurrent.atomic.AtomicBoolean
import kotlin.system.measureTimeMillis

class StravaActivityProvider(
    // allow injection for testability; accept the interface publicly to avoid exposing internal implementation
    localStorageProvider: ILocalStorageProvider? = null,
    private var stravaApi: IStravaApi? = null,
    stravaCache: String = "strava-cache",
) : AbstractActivityProvider(), AutoCloseable {

    // Internally keep a concrete reference (default to StravaRepository when none injected)
    private val storageProvider: ILocalStorageProvider = localStorageProvider ?: StravaRepository(stravaCache)

    private val logger = LoggerFactory.getLogger(StravaActivityProvider::class.java)

    private val clientId: String

    // keep auth info for deferred initialization
    private val authSecret: String?
    private val useCacheAuth: Boolean?
    private val streamIdsCache: GenericCache<Int, Set<Long>> = SoftCache()
    private val startupScope = CoroutineScope(SupervisorJob() + Dispatchers.IO)
    private val backgroundRefreshStarted = AtomicBoolean(false)

    companion object {
        // Reload a year's cache if it is older than this duration (avoids a fixed hardcoded date)
        private val CACHE_MAX_AGE_MS: Long = TimeUnit.DAYS.toMillis(365L)
        private const val MAX_CONCURRENT_STREAM_LOADS = 8
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

        // Load athlete from cache immediately if cache flag is set (this is cheap/local)
        if (useCacheAuth == true) {
            stravaAthlete = storageProvider.loadAthleteFromCache(clientId)
        }

        logger.info("ActivityService prepared with clientId=$clientId (initial loading deferred)")
    }

    /** Cancels the background coroutine scope when the bean is destroyed. */
    override fun close() {
        logger.info("Shutting down StravaActivityProvider background scope")
        startupScope.cancel()
    }

    suspend fun initializeAndLoadActivities() = coroutineScope {
        storageProvider.initLocalStorageForClientId(clientId)
        stravaAthlete = storageProvider.loadAthleteFromCache(clientId)

        // Fast startup path: load only from local cache first.
        activities = loadFromLocalCache()
        logger.info("ActivityService initialized with clientId=$clientId and ${activities.size} activities (cache-first)")

        // If cache mode is forced, never hit Strava API at startup.
        if (useCacheAuth == true) {
            return@coroutineScope
        }

        // No credentials: keep cache-only behavior.
        if (authSecret == null) {
            logger.warn("No Strava credentials found; keeping cache-only startup mode")
            return@coroutineScope
        }

        // First start (empty cache): fallback to the full bootstrap to keep a functional first run.
        if (activities.isEmpty()) {
            logger.info("No activities found in cache; bootstrapping from Strava API")
            stravaAthlete = retrieveLoggedInAthlete()
            activities = loadActivities()
            logger.info("ActivityService initialized with clientId=$clientId and ${activities.size} activities (from Strava)")
            return@coroutineScope
        }

        launchBackgroundDataRefresh()
    }

    override fun getDetailedActivity(activityId: Long): Optional<StravaDetailedActivity> {
        logger.info("Get detailed activity for activity id $activityId")

        // find detailed activity in cache or retrieve from Strava
        val activity = getActivity(activityId).orElse(null) ?: return Optional.empty()
        val year = resolveActivityYear(activity)
        val api = stravaApi ?: createStravaApiIfNeeded()

        // load detailed activity from cache or retrieve from Strava
        var stravaDetailedActivity = loadDetailedActivityFromCacheAnyYear(activityId, year)
        if (api != null && stravaDetailedActivity == null) {
            // It's not in local cache, retrieve from Strava
            val detailedActivity = api.getDetailedActivity(activityId)
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
        if (api != null && stream == null) {
            stream = api.getActivityStream(activity)
            if (stream != null) {
                storageProvider.saveActivitiesStreamsToCache(clientId, year, activity, stream)
            }
        }
        stravaDetailedActivity.stream = stream

        return Optional.of(stravaDetailedActivity)
    }

    override fun getCachedDetailedActivity(activityId: Long): Optional<StravaDetailedActivity> {
        val activity = getActivity(activityId).orElse(null) ?: return Optional.empty()
        val year = resolveActivityYear(activity)
        val cached = loadDetailedActivityFromCacheAnyYear(activityId, year)

        return Optional.ofNullable(cached)
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

    // Determines if activities should be reloaded from Strava API (cache is older than CACHE_MAX_AGE_MS)
    private fun shouldReloadFromStravaAPI(year: Int): Boolean {
        val lastModified = storageProvider.getLocalCacheLastModified(clientId, year)
        return (System.currentTimeMillis() - lastModified) > CACHE_MAX_AGE_MS
    }

    // Loads missing streams from the cache
    private fun loadMissingStreamsFromCache(
        year: Int,
        activities: List<StravaActivity>
    ): List<StravaActivity> {
        val cachedStreamIds = getCachedStreamIds(year)
        activities
            // Filter activities that do not have a stream
            .filter { activity -> activity.stream == null && cachedStreamIds.contains(activity.id) }
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
        val api = stravaApi ?: createStravaApiIfNeeded() ?: return@coroutineScope activities
        val semaphore = Semaphore(MAX_CONCURRENT_STREAM_LOADS)

        val deferred = activities
            .filter { activity -> activity.stream == null }
            .map { activity ->
                async(Dispatchers.IO) {
                    semaphore.withPermit {
                        try {
                            api.getActivityStream(activity)?.let { stream ->
                                storageProvider.saveActivitiesStreamsToCache(clientId, year, activity, stream)
                                activity.stream = stream
                            }
                        } catch (exception: Exception) {
                            logger.error("Error loading stream for activity ${activity.id}", exception)
                        }
                    }
                    activity
                }
            }

        deferred.awaitAll()
        return@coroutineScope activities
    }

    // Retrieves activities from Strava API
    private fun retrieveActivitiesFromApi(year: Int): List<StravaActivity> {
        val api = stravaApi ?: createStravaApiIfNeeded() ?: return emptyList()
        return api.getActivities(year).filterByActivityTypes()
    }

    // Saves activities to cache
    private fun saveActivitiesToCache(year: Int, activities: List<StravaActivity>) {
        storageProvider.saveActivitiesToCache(clientId, year, activities)
    }

    private fun getCachedStreamIds(year: Int): Set<Long> {
        return streamIdsCache[year] ?: storageProvider.buildStreamIdsSet(clientId, year).also { streamIds ->
            streamIdsCache[year] = streamIds
        }
    }

    private fun retrieveLoggedInAthlete(): StravaAthlete {
        logger.info("Load stravaAthlete with id $clientId description from Strava")
        val api = stravaApi ?: createStravaApiIfNeeded()

        return if (api != null) {
            val athlete = api.retrieveLoggedInAthlete()
            if (athlete.isPresent) {
                storageProvider.saveAthleteToCache(clientId, athlete.get())
            }
            athlete.get()
        } else {
            storageProvider.loadAthleteFromCache(clientId)
        }
    }

    @Synchronized
    private fun createStravaApiIfNeeded(): IStravaApi? {
        val existing = stravaApi
        if (existing != null) {
            return existing
        }
        val secret = authSecret ?: return null
        return StravaApi(clientId, secret).also { created ->
            stravaApi = created
        }
    }

    private suspend fun refreshCurrentYearActivitiesInBackground(currentYear: Int) {
        val refreshedActivities = retrieveActivitiesFromApi(currentYear)
        if (refreshedActivities.isEmpty()) {
            logger.info("No current-year updates received from Strava for $currentYear")
            return
        }

        saveActivitiesToCache(currentYear, refreshedActivities)
        loadMissingStreamsFromCache(currentYear, refreshedActivities)
        loadMissingStreamsFromApi(currentYear, refreshedActivities)
        streamIdsCache.remove(currentYear)

        val existingActivities = activities
        val mergedActivities = existingActivities
            .filterNot { activity -> activity.startDateLocal.take(4).toIntOrNull() == currentYear }
            .plus(refreshedActivities)
            .sortedBy { activity -> activity.startDateLocal }

        activities = mergedActivities

        // Invalidate best-effort cache so refreshed activities are reflected in statistics
        BestEffortCache.clear()

        logger.info("Background refresh merged current-year activities: {} total activities in memory", activities.size)
    }

    private suspend fun backfillMissingStreamsInBackground() = coroutineScope {
        val activitiesByYear = activities
            .filter { activity -> activity.stream == null }
            .groupBy { activity -> activity.startDateLocal.take(4).toIntOrNull() ?: LocalDate.now().year }

        if (activitiesByYear.isEmpty()) {
            logger.info("All cached activities already have streams; skipping stream backfill")
            return@coroutineScope
        }

        val tasks = activitiesByYear.map { (year, yearActivities) ->
            async(Dispatchers.IO) {
                loadMissingStreamsFromCache(year, yearActivities)
                loadMissingStreamsFromApi(year, yearActivities)
                streamIdsCache.remove(year)
            }
        }

        tasks.awaitAll()
    }

    private fun launchBackgroundDataRefresh() {
        if (!backgroundRefreshStarted.compareAndSet(false, true)) {
            return
        }

        startupScope.launch {
            try {
                logger.info("Background data refresh started")
                stravaAthlete = retrieveLoggedInAthlete()

                val currentYear = LocalDate.now().year
                refreshCurrentYearActivitiesInBackground(currentYear)
                backfillMissingStreamsInBackground()

                logger.info("Background data refresh completed")
            } catch (exception: Exception) {
                logger.error("Background data refresh failed", exception)
            } finally {
                backgroundRefreshStarted.set(false)
            }
        }
    }

    private fun resolveActivityYear(activity: StravaActivity): Int {
        return activity.startDateLocal.take(4).toIntOrNull()
            ?: activity.startDate.take(4).toIntOrNull()
            ?: LocalDate.now().year
    }

    private fun loadDetailedActivityFromCacheAnyYear(activityId: Long, preferredYear: Int): StravaDetailedActivity? {
        val yearsToTry = buildList {
            if (preferredYear >= 2010) {
                add(preferredYear)
            }
            for (year in LocalDate.now().year downTo 2010) {
                if (year != preferredYear) {
                    add(year)
                }
            }
        }

        yearsToTry.forEach { year ->
            val cached = storageProvider.loadDetailedActivityFromCache(clientId, year, activityId)
            if (cached != null) {
                return cached
            }
        }
        return null
    }
}
