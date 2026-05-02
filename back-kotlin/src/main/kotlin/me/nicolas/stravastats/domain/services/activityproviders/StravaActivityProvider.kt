package me.nicolas.stravastats.domain.services.activityproviders

import kotlinx.coroutines.*
import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.sync.Semaphore
import kotlinx.coroutines.sync.withLock
import kotlinx.coroutines.sync.withPermit
import me.nicolas.stravastats.adapters.localrepositories.strava.StravaRepository
import me.nicolas.stravastats.adapters.strava.StravaApi
import me.nicolas.stravastats.adapters.strava.StravaRateLimitException
import me.nicolas.stravastats.domain.business.HeartRateZoneSettings
import me.nicolas.stravastats.domain.business.ActivityType
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
import java.time.Instant
import java.util.concurrent.ConcurrentHashMap
import java.util.concurrent.TimeUnit
import java.util.concurrent.atomic.AtomicBoolean
import kotlin.system.measureTimeMillis
import kotlin.time.Duration.Companion.milliseconds

class StravaActivityProvider(
    // Allow injection for testability; accept the interface publicly to avoid exposing internal implementation
    localStorageProvider: ILocalStorageProvider? = null,
    private var stravaApi: IStravaApi? = null,
    stravaCache: String = "strava-cache",
) : AbstractActivityProvider(), AutoCloseable {

    // Internally keep a concrete reference (default to StravaRepository when none injected)
    private val storageProvider: ILocalStorageProvider = localStorageProvider ?: StravaRepository(stravaCache)

    private val logger = LoggerFactory.getLogger(StravaActivityProvider::class.java)

    private val clientId: String

    // Keep auth info for deferred initialization
    private val authSecret: String?
    private val useCacheAuth: Boolean?
    private val streamIdsCache: GenericCache<Int, Set<Long>> = SoftCache()
    private val startupScope = CoroutineScope(SupervisorJob() + Dispatchers.IO)
    private val backgroundRefreshStarted = AtomicBoolean(false)
    /** Coroutine-safe mutex replacing @Synchronized on createStravaApiIfNeeded. */
    private val stravaApiMutex = Mutex()
    /** Dedicated lock object for heartRateZoneSettings to avoid using `this` as a monitor. */
    private val heartRateSettingsLock = Any()
    private val cacheRoot = stravaCache

    // Extracted responsibility: rate limiting
    private val rateLimiter = StravaRateLimiter()

    // Extracted responsibility: warmup pipeline + manifest management
    private val warmupPipeline: StravaWarmupPipeline

    @Volatile
    private var heartRateZoneSettings: HeartRateZoneSettings = HeartRateZoneSettings()

    companion object {
        // Reload a year's cache if it is older than this duration
        private val CACHE_MAX_AGE_MS: Long = TimeUnit.DAYS.toMillis(365L)

        /**
         * Maximum number of stream requests sent to the Strava API in parallel.
         * Keeping this low avoids exhausting the Strava quota during backfill.
         */
        private const val MAX_CONCURRENT_STREAM_LOADS = 8

        /**
         * Delay between consecutive detailed-activity requests during the backfill phase (ms).
         * 1 500 ms ≈ 40 requests/min, well within Strava's 200 requests/15 min limit.
         */
        private const val DETAILED_BACKFILL_REQUEST_DELAY_MS = 1_500L

        /** Earliest year from which Strava activity data is considered available. */
        const val STRAVA_FIRST_YEAR = 2010
    }

    init {
        val (id, secret, useCache) = storageProvider.readStravaAuthentication(stravaCache)
        if (id == null) {
            throw IllegalStateException("Strava authentication not found")
        }

        clientId = id
        authSecret = secret
        useCacheAuth = useCache
        warmupPipeline = StravaWarmupPipeline(cacheRoot, clientId)

        // Load athlete from cache immediately if cache flag is set (cheap/local)
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
        heartRateZoneSettings = storageProvider.loadHeartRateZoneSettings(clientId)

        // Load manifest and best-effort cache from disk
        warmupPipeline.initialize()

        // Fast startup path: load only from local cache first
        activities = loadFromLocalCache()
        logger.info("ActivityService initialized with clientId=$clientId and ${activities.size} activities (cache-first)")

        // If cache mode is forced, never hit Strava API at startup
        if (useCacheAuth == true) {
            launchBackgroundWarmup("cache-only startup")
            return@coroutineScope
        }

        // No credentials: keep cache-only behavior
        if (authSecret == null) {
            logger.warn("No Strava credentials found; keeping cache-only startup mode")
            launchBackgroundWarmup("no-credentials startup")
            return@coroutineScope
        }

        // First start (empty cache): fallback to the full bootstrap
        if (activities.isEmpty()) {
            logger.info("No activities found in cache; bootstrapping from Strava API")
            stravaAthlete = retrieveLoggedInAthlete()
            activities = loadActivities()
            logger.info("ActivityService initialized with clientId=$clientId and ${activities.size} activities (from Strava)")
            launchBackgroundWarmup("first bootstrap")
            return@coroutineScope
        }

        launchBackgroundWarmup("cache-first startup")
        launchBackgroundDataRefresh()
    }

    override fun getDetailedActivity(activityId: Long): StravaDetailedActivity? {
        logger.info("Get detailed activity for activity id $activityId")

        // Find detailed activity in cache or retrieve from Strava
        val activity = getActivity(activityId)
        if (activity == null) {
            val cachedDetailed = loadDetailedActivityFromCacheAnyYear(activityId, LocalDate.now().year)
            if (cachedDetailed != null) {
                logger.info("Detailed activity $activityId loaded from cache without base activity metadata")
                return cachedDetailed
            }
            val api = if (rateLimiter.isActive()) null else stravaApi ?: runBlocking(Dispatchers.IO) { createStravaApiIfNeeded() }
            if (api != null) {
                try {
                    val detailed = api.getDetailedActivityFailFastOnRateLimit(activityId)
                    if (detailed != null) {
                        val year = resolveDetailedActivityYear(detailed)
                        storageProvider.saveDetailedActivityToCache(clientId, year, detailed)
                        return detailed
                    }
                } catch (exception: StravaRateLimitException) {
                    rateLimiter.mark("detailed activity $activityId", exception)
                }
            }
            return null
        }
        val year = resolveActivityYear(activity)

        // Load detailed activity from cache or retrieve from Strava
        var stravaDetailedActivity = loadDetailedActivityFromCacheAnyYear(activityId, year)
        val cacheHit = stravaDetailedActivity != null
        var stream = storageProvider.loadActivitiesStreamsFromCache(clientId, year, activity)
        var api: IStravaApi? = if (rateLimiter.isActive()) null else stravaApi
        val needsApiCall = stravaDetailedActivity == null || stream == null
        if (needsApiCall && api == null) {
            api = runBlocking(Dispatchers.IO) { createStravaApiIfNeeded() }
        }

        if (api != null && stravaDetailedActivity == null) {
            try {
                stravaDetailedActivity = api.getDetailedActivityFailFastOnRateLimit(activityId)
            } catch (exception: StravaRateLimitException) {
                rateLimiter.mark("detailed activity $activityId", exception)
            }
        }

        if (stravaDetailedActivity == null) {
            stravaDetailedActivity = activity.toStravaDetailedActivity()
        }

        if (api != null && stream == null) {
            try {
                stream = api.getActivityStreamFailFastOnRateLimit(activity)
                if (stream != null) {
                    storageProvider.saveActivitiesStreamsToCache(clientId, year, activity, stream)
                }
            } catch (exception: StravaRateLimitException) {
                rateLimiter.mark("stream for activity ${activity.id}", exception)
            }
        }

        // copy() preserves immutability — stream is now a val constructor parameter
        val enrichedActivity = stravaDetailedActivity.copy(stream = stream)
        if (!cacheHit) {
            storageProvider.saveDetailedActivityToCache(clientId, year, enrichedActivity)
        }

        return enrichedActivity
    }

    override fun getCachedDetailedActivity(activityId: Long): StravaDetailedActivity? {
        val activity = getActivity(activityId) ?: return null
        val year = resolveActivityYear(activity)
        return loadDetailedActivityFromCacheAnyYear(activityId, year)
    }

    override fun getHeartRateZoneSettings(): HeartRateZoneSettings = heartRateZoneSettings

    override fun saveHeartRateZoneSettings(settings: HeartRateZoneSettings): HeartRateZoneSettings {
        // Use a dedicated lock object instead of @Synchronized (which locks on `this`, incompatible with coroutine usage)
        synchronized(heartRateSettingsLock) {
            heartRateZoneSettings = settings
            storageProvider.saveHeartRateZoneSettings(clientId, settings)
            return heartRateZoneSettings
        }
    }

    private suspend fun loadFromLocalCache(): List<StravaActivity> = coroutineScope {
        logger.info("Load Strava activities from local cache ...")

        val loadedActivities = mutableListOf<StravaActivity>()
        val elapsed = measureTimeMillis {
            val deferredActivities = (LocalDate.now().year downTo STRAVA_FIRST_YEAR).map { year ->
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
            val deferredActivities = (currentYear downTo STRAVA_FIRST_YEAR).map { year ->
                async(Dispatchers.IO) {
                    try {
                        if (currentYear != year
                            && storageProvider.isLocalCacheExistForYear(clientId, year)
                            && !shouldReloadFromStravaAPI(year)) {
                            logger.info("Loading activities for $year from cache ...")
                            val cached = storageProvider.loadActivitiesFromCache(clientId, year)
                            val withCacheStreams = loadMissingStreamsFromCache(year, cached)
                            loadMissingStreamsFromApi(year, withCacheStreams)
                        } else {
                            logger.info("Loading activities for $year from Strava API ...")
                            val fromApi = retrieveActivitiesFromApi(year)
                            saveActivitiesToCache(year, fromApi)
                            val withCacheStreams = loadMissingStreamsFromCache(year, fromApi)
                            loadMissingStreamsFromApi(year, withCacheStreams)
                        }
                    } catch (exception: Exception) {
                        logger.error("Error loading activities for year $year", exception)
                        emptyList<StravaActivity>()
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

    // Returns a new list where activities with a cached stream are replaced by copy(stream=...)
    private fun loadMissingStreamsFromCache(year: Int, activities: List<StravaActivity>): List<StravaActivity> {
        val cachedStreamIds = getCachedStreamIds(year)
        return activities.map { activity ->
            if (activity.stream == null && cachedStreamIds.contains(activity.id)) {
                val stream = storageProvider.loadActivitiesStreamsFromCache(clientId, year, activity)
                if (stream != null) activity.copy(stream = stream) else activity
            } else {
                activity
            }
        }
    }

    // Returns a new list where activities are replaced by stream-enriched copies fetched from the API (parallelized)
    internal suspend fun loadMissingStreamsFromApi(
        year: Int,
        activities: List<StravaActivity>,
    ): List<StravaActivity> = coroutineScope {
        if (rateLimiter.isActive()) {
            return@coroutineScope activities
        }
        val api = stravaApi ?: createStravaApiIfNeeded() ?: return@coroutineScope activities
        val semaphore = Semaphore(MAX_CONCURRENT_STREAM_LOADS)
        val stopRequested = AtomicBoolean(false)
        // Thread-safe map: activity id -> stream-enriched copy
        val enrichedById = ConcurrentHashMap<Long, StravaActivity>()

        val deferred = activities
            .filter { activity -> activity.stream == null }
            .map { activity ->
                async(Dispatchers.IO) {
                    semaphore.withPermit {
                        try {
                            if (stopRequested.get() || rateLimiter.isActive()) {
                                return@withPermit
                            }
                            api.getActivityStreamFailFastOnRateLimit(activity)?.let { stream ->
                                storageProvider.saveActivitiesStreamsToCache(clientId, year, activity, stream)
                                enrichedById[activity.id] = activity.copy(stream = stream)
                            }
                        } catch (exception: StravaRateLimitException) {
                            stopRequested.set(true)
                            rateLimiter.mark("stream backfill activity ${activity.id}", exception)
                        } catch (exception: Exception) {
                            logger.error("Error loading stream for activity ${activity.id}", exception)
                        }
                    }
                }
            }

        deferred.awaitAll()
        // Return a new list replacing activities that got streams
        return@coroutineScope activities.map { activity -> enrichedById[activity.id] ?: activity }
    }

    // Retrieves activities from Strava API
    private suspend fun retrieveActivitiesFromApi(year: Int, failFastOnRateLimit: Boolean = true): List<StravaActivity> {
        if (rateLimiter.isActive()) {
            throw StravaRateLimitException("strava rate limit reached (cooldown active)")
        }
        val api = stravaApi ?: createStravaApiIfNeeded() ?: return emptyList()
        val activities = try {
            if (failFastOnRateLimit) api.getActivitiesFailFastOnRateLimit(year) else api.getActivities(year)
        } catch (exception: StravaRateLimitException) {
            rateLimiter.mark("activities year $year", exception)
            throw exception
        }
        return activities.filterByActivityTypes()
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

    private suspend fun retrieveLoggedInAthlete(): StravaAthlete {
        logger.info("Load stravaAthlete with id $clientId description from Strava")
        val api = if (rateLimiter.isActive()) null else stravaApi ?: createStravaApiIfNeeded()

        return if (api != null) {
            try {
                val athlete = api.retrieveLoggedInAthlete()
                if (athlete != null) {
                    storageProvider.saveAthleteToCache(clientId, athlete)
                }
                athlete ?: storageProvider.loadAthleteFromCache(clientId)
            } catch (exception: StravaRateLimitException) {
                rateLimiter.mark("athlete", exception)
                storageProvider.loadAthleteFromCache(clientId)
            } catch (exception: Exception) {
                logger.error("Unable to retrieve athlete from Strava API", exception)
                storageProvider.loadAthleteFromCache(clientId)
            }
        } else {
            storageProvider.loadAthleteFromCache(clientId)
        }
    }

    private suspend fun createStravaApiIfNeeded(): IStravaApi? = stravaApiMutex.withLock {
        if (rateLimiter.isActive()) {
            return@withLock null
        }
        val existing = stravaApi
        if (existing != null) {
            return@withLock existing
        }
        val secret = authSecret ?: return@withLock null
        try {
            StravaApi(clientId, secret).also { created ->
                stravaApi = created
            }
        } catch (exception: Exception) {
            logger.error("Failed to initialize Strava API (token fetch error): ${exception.message}", exception)
            logger.warn("Switching to cache-only mode: no API calls will be made until next restart")
            null
        }
    }

    private suspend fun refreshAllYearsActivitiesInBackground(startYear: Int): Boolean {
        for (year in startYear downTo STRAVA_FIRST_YEAR) {
            val refreshedActivities = try {
                retrieveActivitiesFromApi(year, failFastOnRateLimit = true)
            } catch (_: StravaRateLimitException) {
                logger.warn("Background refresh stopped at year {} due to Strava rate limit", year)
                return true
            } catch (exception: Exception) {
                logger.error("Background refresh failed for year $year", exception)
                continue
            }

            saveActivitiesToCache(year, refreshedActivities)

            // Enrich with streams before merging into the in-memory list
            val activitiesWithStreams = if (refreshedActivities.isNotEmpty()) {
                val withCacheStreams = loadMissingStreamsFromCache(year, refreshedActivities)
                loadMissingStreamsFromApi(year, withCacheStreams)
            } else {
                refreshedActivities
            }
            streamIdsCache.remove(year)

            val existingActivities = activities
            val mergedActivities = existingActivities
                .filterNot { activity -> resolveYearFromDateString(activity.startDateLocal) == year }
                .plus(activitiesWithStreams)
                .sortedBy { activity -> activity.startDateLocal }

            activities = mergedActivities

            // Invalidate touched activities from the best-effort cache
            val invalidatedActivityIds = existingActivities
                .filter { activity -> resolveYearFromDateString(activity.startDateLocal) == year }
                .map { activity -> activity.id }
                .toMutableSet()
                .apply { addAll(activitiesWithStreams.map { activity -> activity.id }) }
            val removedEntries = BestEffortCache.invalidateActivities(invalidatedActivityIds)
            if (removedEntries > 0) {
                logger.info("Invalidated {} best-effort cache entries after year {} refresh", removedEntries, year)
            }

            logger.info(
                "Background refresh merged year {} activities ({} total activities in memory)",
                year, activities.size,
            )
        }

        return false
    }

    private suspend fun backfillMissingStreamsInBackground() = coroutineScope {
        val activitiesByYear = activities
            .filter { activity -> activity.stream == null }
            .groupBy { activity -> resolveYearFromDateString(activity.startDateLocal) }

        if (activitiesByYear.isEmpty()) {
            logger.info("All cached activities already have streams; skipping stream backfill")
            return@coroutineScope
        }

        val years = activitiesByYear.keys.sortedDescending()
        for (year in years) {
            if (rateLimiter.isActive()) {
                logger.info("Stream backfill stopped early due to Strava rate limit")
                return@coroutineScope
            }
            val yearActivities = activitiesByYear[year] ?: continue
            val withCacheStreams = loadMissingStreamsFromCache(year, yearActivities)
            val enrichedActivities = loadMissingStreamsFromApi(year, withCacheStreams)
            streamIdsCache.remove(year)

            // Propagate stream-enriched copies back into the in-memory activities list
            val enrichedById = enrichedActivities.filter { it.stream != null }.associateBy { it.id }
            if (enrichedById.isNotEmpty()) {
                activities = activities.map { activity -> enrichedById[activity.id] ?: activity }
            }
        }
    }

    private suspend fun backfillMissingDetailedActivitiesInBackground(startYear: Int): Boolean = coroutineScope {
        if (rateLimiter.isActive()) {
            logger.info("Detailed backfill skipped: Strava rate limit is active")
            return@coroutineScope true
        }
        val api = stravaApi ?: createStravaApiIfNeeded()
        if (api == null) {
            logger.info("Detailed backfill skipped: Strava API unavailable")
            return@coroutineScope false
        }

        val activitiesByYear = activities
            .asSequence()
            .mapNotNull { activity ->
                val year = resolveActivityYear(activity)
                if (year !in STRAVA_FIRST_YEAR..startYear) return@mapNotNull null
                if (storageProvider.loadDetailedActivityFromCache(clientId, year, activity.id) != null) return@mapNotNull null
                year to activity
            }
            .groupBy(keySelector = { (year, _) -> year }, valueTransform = { (_, activity) -> activity })

        val missingCount = activitiesByYear.values.sumOf { it.size }
        if (missingCount == 0) {
            logger.info("All cached activities already have detailed payloads; skipping detailed backfill")
            return@coroutineScope false
        }

        logger.info("Detailed backfill started for {} missing activities", missingCount)
        var totalLoaded = 0
        var firstRequest = true

        for (year in startYear downTo STRAVA_FIRST_YEAR) {
            val yearActivities = activitiesByYear[year]
                ?.sortedByDescending { activity -> activity.startDateLocal }
                ?: continue

            var loadedForYear = 0
            for (activity in yearActivities) {
                if (!firstRequest) {
                    delay(DETAILED_BACKFILL_REQUEST_DELAY_MS.milliseconds)
                }
                firstRequest = false

                try {
                    if (rateLimiter.isActive()) {
                        return@coroutineScope true
                    }
                    val detailedActivity = api.getDetailedActivityFailFastOnRateLimit(activity.id)
                    if (detailedActivity != null) {
                        storageProvider.saveDetailedActivityToCache(clientId, year, detailedActivity)
                        loadedForYear += 1
                        totalLoaded += 1
                    }
                } catch (exception: StravaRateLimitException) {
                    rateLimiter.mark("detailed backfill activity ${activity.id}", exception)
                    logger.warn("Detailed backfill stopped at year {} for activity {} due to rate limit", year, activity.id)
                    return@coroutineScope true
                } catch (exception: Exception) {
                    logger.error("Unable to backfill detailed activity ${activity.id}", exception)
                }
            }

            if (loadedForYear > 0) {
                logger.info("Detailed backfill cached {} activities for year {}", loadedForYear, year)
            }
        }

        logger.info("Detailed backfill completed ({} activities cached)", totalLoaded)
        return@coroutineScope false
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
                val stoppedByRateLimit = refreshAllYearsActivitiesInBackground(currentYear)
                if (stoppedByRateLimit) {
                    logger.info("Background data refresh stopped early due to Strava rate limit")
                } else {
                    backfillMissingStreamsInBackground()
                    val detailedStopped = backfillMissingDetailedActivitiesInBackground(currentYear)
                    if (detailedStopped) {
                        logger.info("Background detailed backfill stopped early due to Strava rate limit")
                    }
                }
                warmupPipeline.runWarmupPipeline("post-refresh", activities.toList())

                logger.info("Background data refresh completed")
            } catch (exception: Exception) {
                logger.error("Background data refresh failed", exception)
            } finally {
                backgroundRefreshStarted.set(false)
            }
        }
    }

    private fun launchBackgroundWarmup(reason: String) {
        startupScope.launch {
            warmupPipeline.runWarmupPipeline(reason, activities.toList())
        }
    }

    override fun getCacheDiagnostics(): Map<String, Any?> {
        val basicDiagnostics = basicCacheDiagnostics(
            provider = "strava",
            sourcePathKey = "cacheRoot",
            sourcePath = cacheRoot,
        )

        return basicDiagnostics + mapOf(
            "timestamp" to Instant.now().toString(),
            "athleteId" to clientId,
            "refresh" to mapOf(
                "backgroundInProgress" to backgroundRefreshStarted.get(),
                "warmupInProgress" to warmupPipeline.isWarmupInProgress(),
            ),
            "rateLimit" to mapOf(
                "active" to rateLimiter.isActive(),
                "untilEpochMs" to rateLimiter.untilEpochMs(),
            ),
        ) + warmupPipeline.diagnosticsSection()
    }

    override fun cacheIdentity(): ActivityProviderCacheIdentity {
        return ActivityProviderCacheIdentity(
            cacheRoot = cacheRoot,
            athleteId = clientId,
        )
    }

    private fun resolveActivityYear(activity: StravaActivity): Int =
        resolveYearFromDateString(activity.startDateLocal)
            .takeIf { it > 0 }
            ?: resolveYearFromDateString(activity.startDate)

    private fun resolveDetailedActivityYear(activity: StravaDetailedActivity): Int =
        resolveYearFromDateString(activity.startDateLocal)
            .takeIf { it > 0 }
            ?: resolveYearFromDateString(activity.startDate)

    private fun loadDetailedActivityFromCacheAnyYear(activityId: Long, preferredYear: Int): StravaDetailedActivity? {
        val yearsToTry = buildList {
            if (preferredYear >= STRAVA_FIRST_YEAR) {
                add(preferredYear)
            }
            for (year in LocalDate.now().year downTo STRAVA_FIRST_YEAR) {
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
