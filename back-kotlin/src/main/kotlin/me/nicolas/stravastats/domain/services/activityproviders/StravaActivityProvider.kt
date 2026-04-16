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
import me.nicolas.stravastats.domain.services.cache.CacheManifest
import me.nicolas.stravastats.domain.services.cache.CacheManifestStore
import me.nicolas.stravastats.domain.services.cache.WarmupMetricSummary
import me.nicolas.stravastats.domain.services.cache.WarmupSummariesFile
import me.nicolas.stravastats.domain.services.cache.WarmupYearSummary
import me.nicolas.stravastats.domain.services.statistics.BestEffortCache
import me.nicolas.stravastats.domain.services.statistics.calculateBestDistanceForTime
import me.nicolas.stravastats.domain.services.statistics.calculateBestElevationForDistance
import me.nicolas.stravastats.domain.services.statistics.calculateBestPowerForTime
import me.nicolas.stravastats.domain.services.statistics.calculateBestTimeForDistance
import me.nicolas.stravastats.domain.services.toStravaDetailedActivity
import me.nicolas.stravastats.domain.utils.GenericCache
import me.nicolas.stravastats.domain.utils.SoftCache
import me.nicolas.stravastats.domain.utils.formatSeconds
import org.slf4j.LoggerFactory
import java.nio.file.Files
import java.time.LocalDate
import java.time.Instant
import java.util.Locale
import java.util.concurrent.TimeUnit
import java.util.concurrent.atomic.AtomicBoolean
import java.util.concurrent.atomic.AtomicLong
import kotlin.system.measureTimeMillis
import kotlin.time.Duration.Companion.milliseconds

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
    private val warmupInProgress = AtomicBoolean(false)
    private val rateLimitUntilMs = AtomicLong(0L)
    private val cacheRoot = stravaCache
    private val manifestLock = Any()
    /** Coroutine-safe mutex replacing @Synchronized on createStravaApiIfNeeded. */
    private val stravaApiMutex = Mutex()
    @Volatile
    private var cacheManifest: CacheManifest = CacheManifestStore.defaultManifest("unknown")
    @Volatile
    private var heartRateZoneSettings: HeartRateZoneSettings = HeartRateZoneSettings()

    companion object {
        // Reload a year's cache if it is older than this duration (avoids a fixed hardcoded date)
        private val CACHE_MAX_AGE_MS: Long = TimeUnit.DAYS.toMillis(365L)
        private const val MAX_CONCURRENT_STREAM_LOADS = 8
        private const val DETAILED_BACKFILL_REQUEST_DELAY_MS = 1_500L
        private const val RATE_LIMIT_COOLDOWN_MS = 15 * 60 * 1_000L
        /** Earliest year from which Strava activity data is considered available. */
        const val STRAVA_FIRST_YEAR = 2010
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
        cacheManifest = CacheManifestStore.load(cacheRoot, clientId) ?: CacheManifestStore.defaultManifest(clientId)

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
        heartRateZoneSettings = storageProvider.loadHeartRateZoneSettings(clientId)
        loadPersistentCacheArtifacts()

        // Fast startup path: load only from local cache first.
        activities = loadFromLocalCache()
        logger.info("ActivityService initialized with clientId=$clientId and ${activities.size} activities (cache-first)")

        // If cache mode is forced, never hit Strava API at startup.
        if (useCacheAuth == true) {
            launchBackgroundWarmup("cache-only startup")
            return@coroutineScope
        }

        // No credentials: keep cache-only behavior.
        if (authSecret == null) {
            logger.warn("No Strava credentials found; keeping cache-only startup mode")
            launchBackgroundWarmup("no-credentials startup")
            return@coroutineScope
        }

        // First start (empty cache): fallback to the full bootstrap to keep a functional first run.
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

        // find detailed activity in cache or retrieve from Strava
        val activity = getActivity(activityId)
        if (activity == null) {
            val cachedDetailed = loadDetailedActivityFromCacheAnyYear(activityId, LocalDate.now().year)
            if (cachedDetailed != null) {
                logger.info("Detailed activity $activityId loaded from cache without base activity metadata")
                return cachedDetailed
            }
            val api = if (isRateLimitActive()) null else stravaApi ?: runBlocking { createStravaApiIfNeeded() }
            if (api != null) {
                try {
                    val detailed = api.getDetailedActivityFailFastOnRateLimit(activityId)
                    if (detailed != null) {
                        val year = resolveDetailedActivityYear(detailed)
                        storageProvider.saveDetailedActivityToCache(clientId, year, detailed)
                        return detailed
                    }
                } catch (exception: StravaRateLimitException) {
                    markRateLimitActive("detailed activity $activityId", exception)
                }
            }
            return null
        }
        val year = resolveActivityYear(activity)

        // load detailed activity from cache or retrieve from Strava
        var stravaDetailedActivity = loadDetailedActivityFromCacheAnyYear(activityId, year)
        val cacheHit = stravaDetailedActivity != null
        var stream = storageProvider.loadActivitiesStreamsFromCache(clientId, year, activity)
        var api: IStravaApi? = if (isRateLimitActive()) null else stravaApi
        val needsApiCall = stravaDetailedActivity == null || stream == null
        if (needsApiCall && api == null) {
            api = runBlocking { createStravaApiIfNeeded() }
        }

        if (api != null && stravaDetailedActivity == null) {
            // It's not in local cache, retrieve from Strava
            try {
                stravaDetailedActivity = api.getDetailedActivityFailFastOnRateLimit(activityId)
            } catch (exception: StravaRateLimitException) {
                markRateLimitActive("detailed activity $activityId", exception)
            }
        }

        if (stravaDetailedActivity == null) {
            // Detailed activity not found on Strava, return the activity without details
            stravaDetailedActivity = activity.toStravaDetailedActivity()
        }

        if (api != null && stream == null) {
            try {
                stream = api.getActivityStreamFailFastOnRateLimit(activity)
                if (stream != null) {
                    storageProvider.saveActivitiesStreamsToCache(clientId, year, activity, stream)
                }
            } catch (exception: StravaRateLimitException) {
                markRateLimitActive("stream for activity ${activity.id}", exception)
            }
        }
        stravaDetailedActivity.stream = stream
        if (!cacheHit) {
            storageProvider.saveDetailedActivityToCache(clientId, year, stravaDetailedActivity)
        }

        return stravaDetailedActivity
    }

    override fun getCachedDetailedActivity(activityId: Long): StravaDetailedActivity? {
        val activity = getActivity(activityId) ?: return null
        val year = resolveActivityYear(activity)
        return loadDetailedActivityFromCacheAnyYear(activityId, year)
    }

    override fun getHeartRateZoneSettings(): HeartRateZoneSettings {
        return heartRateZoneSettings
    }

    @Synchronized
    override fun saveHeartRateZoneSettings(settings: HeartRateZoneSettings): HeartRateZoneSettings {
        heartRateZoneSettings = settings
        storageProvider.saveHeartRateZoneSettings(clientId, settings)
        return heartRateZoneSettings
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
        if (isRateLimitActive()) {
            return@coroutineScope activities
        }
        val api = stravaApi ?: createStravaApiIfNeeded() ?: return@coroutineScope activities
        val semaphore = Semaphore(MAX_CONCURRENT_STREAM_LOADS)
        val stopRequested = AtomicBoolean(false)

        val deferred = activities
            .filter { activity -> activity.stream == null }
            .map { activity ->
                async(Dispatchers.IO) {
                    semaphore.withPermit {
                        try {
                            if (stopRequested.get() || isRateLimitActive()) {
                                return@withPermit
                            }
                            api.getActivityStreamFailFastOnRateLimit(activity)?.let { stream ->
                                storageProvider.saveActivitiesStreamsToCache(clientId, year, activity, stream)
                                activity.stream = stream
                            }
                        } catch (exception: StravaRateLimitException) {
                            stopRequested.set(true)
                            markRateLimitActive("stream backfill activity ${activity.id}", exception)
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
    private suspend fun retrieveActivitiesFromApi(year: Int, failFastOnRateLimit: Boolean = true): List<StravaActivity> {
        if (isRateLimitActive()) {
            throw StravaRateLimitException("strava rate limit reached (cooldown active)")
        }
        val api = stravaApi ?: createStravaApiIfNeeded() ?: return emptyList()
        val activities = try {
            if (failFastOnRateLimit) {
                api.getActivitiesFailFastOnRateLimit(year)
            } else {
                api.getActivities(year)
            }
        } catch (exception: StravaRateLimitException) {
            markRateLimitActive("activities year $year", exception)
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
        val api = if (isRateLimitActive()) null else stravaApi ?: createStravaApiIfNeeded()

        return if (api != null) {
            try {
                val athlete = api.retrieveLoggedInAthlete()
                if (athlete != null) {
                    storageProvider.saveAthleteToCache(clientId, athlete)
                }
                athlete ?: storageProvider.loadAthleteFromCache(clientId)
            } catch (exception: StravaRateLimitException) {
                markRateLimitActive("athlete", exception)
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
        if (isRateLimitActive()) {
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
            logger.warn("Switching to cache-only mode: activities will be loaded from cache but no API calls will be made until next restart")
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
            if (refreshedActivities.isNotEmpty()) {
                loadMissingStreamsFromCache(year, refreshedActivities)
                loadMissingStreamsFromApi(year, refreshedActivities)
            }
            streamIdsCache.remove(year)

            val existingActivities = activities
            val mergedActivities = existingActivities
                .filterNot { activity -> resolveYearFromDateString(activity.startDateLocal) == year }
                .plus(refreshedActivities)
                .sortedBy { activity -> activity.startDateLocal }

            activities = mergedActivities

            // Invalidate only touched activities so the cache keeps unaffected entries.
            val invalidatedActivityIds = existingActivities
                .filter { activity -> resolveYearFromDateString(activity.startDateLocal) == year }
                .map { activity -> activity.id }
                .toMutableSet()
                .apply { addAll(refreshedActivities.map { activity -> activity.id }) }
            val removedEntries = BestEffortCache.invalidateActivities(invalidatedActivityIds)
            if (removedEntries > 0) {
                logger.info("Invalidated {} best-effort cache entries after year {} refresh", removedEntries, year)
            }

            logger.info(
                "Background refresh merged year {} activities ({} total activities in memory)",
                year,
                activities.size
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
            if (isRateLimitActive()) {
                logger.info("Stream backfill stopped early due to Strava rate limit")
                return@coroutineScope
            }
            val yearActivities = activitiesByYear[year] ?: continue
            loadMissingStreamsFromCache(year, yearActivities)
            loadMissingStreamsFromApi(year, yearActivities)
            streamIdsCache.remove(year)
        }
    }

    private suspend fun backfillMissingDetailedActivitiesInBackground(startYear: Int): Boolean = coroutineScope {
        if (isRateLimitActive()) {
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
                if (year !in STRAVA_FIRST_YEAR..startYear) {
                    return@mapNotNull null
                }
                if (storageProvider.loadDetailedActivityFromCache(clientId, year, activity.id) != null) {
                    return@mapNotNull null
                }
                year to activity
            }
            .groupBy(keySelector = { (year, _) -> year }, valueTransform = { (_, activity) -> activity })

        val missingCount = activitiesByYear.values.sumOf { yearActivities -> yearActivities.size }
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
                    if (isRateLimitActive()) {
                        return@coroutineScope true
                    }
                    val detailedActivity = api.getDetailedActivityFailFastOnRateLimit(activity.id)
                    if (detailedActivity != null) {
                        storageProvider.saveDetailedActivityToCache(clientId, year, detailedActivity)
                        loadedForYear += 1
                        totalLoaded += 1
                    }
                } catch (exception: StravaRateLimitException) {
                    markRateLimitActive("detailed backfill activity ${activity.id}", exception)
                    logger.warn(
                        "Detailed backfill stopped at year {} for activity {} due to Strava rate limit",
                        year,
                        activity.id
                    )
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
                    val detailedStoppedByRateLimit = backfillMissingDetailedActivitiesInBackground(currentYear)
                    if (detailedStoppedByRateLimit) {
                        logger.info("Background detailed backfill stopped early due to Strava rate limit")
                    }
                }
                runWarmupPipeline("post-refresh")

                logger.info("Background data refresh completed")
            } catch (exception: Exception) {
                logger.error("Background data refresh failed", exception)
            } finally {
                backgroundRefreshStarted.set(false)
            }
        }
    }

    private fun loadPersistentCacheArtifacts() {
        val loadedManifest = CacheManifestStore.load(cacheRoot, clientId) ?: CacheManifestStore.defaultManifest(clientId)
        val loadedEntries = runCatching {
            BestEffortCache.loadFromDisk(CacheManifestStore.bestEffortCachePath(cacheRoot, clientId, loadedManifest))
        }.getOrElse { exception ->
            logger.error("Unable to load best-effort cache from disk", exception)
            BestEffortCache.clear()
            0
        }

        val updatedManifest = loadedManifest.copy(
            bestEffortCache = loadedManifest.bestEffortCache.copy(
                entries = loadedEntries,
                lastPersistedAt = loadedManifest.bestEffortCache.lastPersistedAt ?: Instant.now().toString(),
            )
        )

        synchronized(manifestLock) {
            cacheManifest = updatedManifest
            runCatching { CacheManifestStore.save(cacheRoot, updatedManifest) }
                .onFailure { exception -> logger.error("Unable to save cache manifest", exception) }
        }

        logger.info("Loaded best-effort cache: {} entries", loadedEntries)
    }

    private fun launchBackgroundWarmup(reason: String) {
        startupScope.launch {
            runWarmupPipeline(reason)
        }
    }

    private suspend fun runWarmupPipeline(reason: String) = coroutineScope {
        if (!warmupInProgress.compareAndSet(false, true)) {
            return@coroutineScope
        }

        try {
            val snapshot = activities.toList()
            if (snapshot.isEmpty()) {
                return@coroutineScope
            }

            logger.info("Warmup started ({})", reason)

            val yearSummaries = computeWarmupYearSummaries(snapshot)
            val preparedYears = yearSummaries.map { summary -> summary.year }.sortedDescending()

            var warmupPayload = WarmupSummariesFile(
                athleteId = clientId,
                yearSummaries = yearSummaries,
            )

            persistWarmupArtifacts(
                payload = warmupPayload,
                priority1 = "ready",
                priority2 = "pending",
                priority3 = "pending",
                preparedYears = preparedYears,
            )

            warmupPayload = warmupPayload.copy(
                majorBestEfforts = precomputeMajorBestEfforts(snapshot)
            )
            persistWarmupArtifacts(
                payload = warmupPayload,
                priority1 = "ready",
                priority2 = "ready",
                priority3 = "pending",
                preparedYears = preparedYears,
            )

            warmupPayload = warmupPayload.copy(
                advancedMetrics = precomputeAdvancedMetrics(snapshot)
            )
            persistWarmupArtifacts(
                payload = warmupPayload,
                priority1 = "ready",
                priority2 = "ready",
                priority3 = "ready",
                preparedYears = preparedYears,
            )

            logger.info("Warmup completed ({})", reason)
        } catch (exception: Exception) {
            logger.error("Warmup failed ({})", reason, exception)
        } finally {
            warmupInProgress.set(false)
        }
    }

    private fun persistWarmupArtifacts(
        payload: WarmupSummariesFile,
        priority1: String,
        priority2: String,
        priority3: String,
        preparedYears: List<Int>,
    ) {
        synchronized(manifestLock) {
            val entries = BestEffortCache.saveToDisk(
                CacheManifestStore.bestEffortCachePath(cacheRoot, clientId, cacheManifest)
            )

            val updatedManifest = cacheManifest.copy(
                updatedAt = Instant.now().toString(),
                bestEffortCache = cacheManifest.bestEffortCache.copy(
                    entries = entries,
                    lastPersistedAt = Instant.now().toString(),
                ),
                warmup = cacheManifest.warmup.copy(
                    priority1 = priority1,
                    priority2 = priority2,
                    priority3 = priority3,
                    preparedYears = preparedYears,
                    lastRunAt = Instant.now().toString(),
                )
            )

            CacheManifestStore.saveWarmupSummaries(cacheRoot, clientId, payload, updatedManifest)
            CacheManifestStore.save(cacheRoot, updatedManifest)
            cacheManifest = updatedManifest
        }
    }

    override fun getCacheDiagnostics(): Map<String, Any?> {
        val manifestSnapshot = synchronized(manifestLock) { cacheManifest }
        val manifestPath = CacheManifestStore.manifestPath(cacheRoot, clientId)
        val bestEffortPath = CacheManifestStore.bestEffortCachePath(cacheRoot, clientId, manifestSnapshot)
        val warmupPath = CacheManifestStore.warmupSummariesPath(cacheRoot, clientId, manifestSnapshot)

        return mapOf(
            "timestamp" to Instant.now().toString(),
            "athleteId" to clientId,
            "rateLimit" to mapOf(
                "active" to isRateLimitActive(),
                "untilEpochMs" to rateLimitUntilMs.get(),
            ),
            "manifest" to mapOf(
                "schemaVersion" to manifestSnapshot.schemaVersion,
                "updatedAt" to manifestSnapshot.updatedAt,
                "bestEffortCache" to mapOf(
                    "algoVersion" to manifestSnapshot.bestEffortCache.algoVersion,
                    "entriesPersisted" to manifestSnapshot.bestEffortCache.entries,
                    "entriesInMemory" to BestEffortCache.size(),
                    "file" to manifestSnapshot.bestEffortCache.file,
                    "lastPersistedAt" to manifestSnapshot.bestEffortCache.lastPersistedAt,
                ),
                "warmup" to mapOf(
                    "algoVersion" to manifestSnapshot.warmup.algoVersion,
                    "file" to manifestSnapshot.warmup.file,
                    "priority1" to manifestSnapshot.warmup.priority1,
                    "priority2" to manifestSnapshot.warmup.priority2,
                    "priority3" to manifestSnapshot.warmup.priority3,
                    "preparedYears" to manifestSnapshot.warmup.preparedYears,
                    "lastRunAt" to manifestSnapshot.warmup.lastRunAt,
                ),
            ),
            "files" to mapOf(
                "manifest" to fileDiagnostics(manifestPath),
                "bestEffortCache" to fileDiagnostics(bestEffortPath),
                "warmupSummaries" to fileDiagnostics(warmupPath),
            ),
        )
    }

    override fun cacheIdentity(): ActivityProviderCacheIdentity {
        return ActivityProviderCacheIdentity(
            cacheRoot = cacheRoot,
            athleteId = clientId,
        )
    }

    private fun fileDiagnostics(path: java.nio.file.Path): Map<String, Any?> {
        if (!Files.exists(path)) {
            return mapOf(
                "path" to path.toString(),
                "exists" to false,
            )
        }

        return mapOf(
            "path" to path.toString(),
            "exists" to true,
            "sizeBytes" to Files.size(path),
            "lastModified" to Files.getLastModifiedTime(path).toInstant().toString(),
        )
    }

    private fun isRateLimitActive(): Boolean {
        val untilMs = rateLimitUntilMs.get()
        return untilMs > System.currentTimeMillis()
    }

    private fun markRateLimitActive(source: String, throwable: Throwable? = null) {
        val now = System.currentTimeMillis()
        val until = now + RATE_LIMIT_COOLDOWN_MS
        val previous = rateLimitUntilMs.getAndUpdate { current -> maxOf(current, until) }
        if (previous > now) {
            return
        }

        logger.warn(
            "Strava rate limit detected ({}). Switching to immediate cache-only mode until {}",
            source,
            Instant.ofEpochMilli(until)
        )
        if (throwable != null) {
            logger.debug("Rate limit trigger details for '{}'", source, throwable)
        }
    }

    private fun computeWarmupYearSummaries(activities: List<StravaActivity>): List<WarmupYearSummary> {
        val summaries = mutableMapOf<Int, MutableWarmupYearSummary>()
        val allYears = MutableWarmupYearSummary(year = 0)

        activities.forEach { activity ->
            val year = resolveActivityYear(activity)
            val summary = summaries.getOrPut(year) { MutableWarmupYearSummary(year = year) }
            summary.accept(activity)
            allYears.accept(activity)
        }

        return buildList {
            add(allYears.toPublic())
            addAll(summaries.values.map { summary -> summary.toPublic() })
        }.sortedByDescending { summary -> summary.year }
    }

    private fun precomputeMajorBestEfforts(activities: List<StravaActivity>): List<WarmupMetricSummary> {
        val rideActivities = filterActivitiesForWarmup(activities, "ride")
        val runActivities = filterActivitiesForWarmup(activities, "run")

        return buildList {
            computeBestTimeDistanceMetric("ride", rideActivities, 1000.0)?.let { add(it) }
            computeBestTimeDistanceMetric("ride", rideActivities, 5000.0)?.let { add(it) }
            computeBestDistanceTimeMetric("ride", rideActivities, 20 * 60)?.let { add(it) }
            computeBestDistanceTimeMetric("ride", rideActivities, 60 * 60)?.let { add(it) }
            computeBestTimeDistanceMetric("run", runActivities, 1000.0)?.let { add(it) }
            computeBestTimeDistanceMetric("run", runActivities, 5000.0)?.let { add(it) }
            computeBestDistanceTimeMetric("run", runActivities, 20 * 60)?.let { add(it) }
            computeBestDistanceTimeMetric("run", runActivities, 60 * 60)?.let { add(it) }
        }
    }

    private fun precomputeAdvancedMetrics(activities: List<StravaActivity>): List<WarmupMetricSummary> {
        val rideActivities = filterActivitiesForWarmup(activities, "ride")
        return buildList {
            computeBestElevationMetric("ride", rideActivities, 1000.0)?.let { add(it) }
            computeBestElevationMetric("ride", rideActivities, 5000.0)?.let { add(it) }
            computeBestPowerMetric("ride", rideActivities, 20 * 60)?.let { add(it) }
            computeBestPowerMetric("ride", rideActivities, 60 * 60)?.let { add(it) }
        }
    }

    private fun filterActivitiesForWarmup(activities: List<StravaActivity>, group: String): List<StravaActivity> {
        return activities.filter { activity ->
            when (group) {
                "run" -> activity.sportType == ActivityType.Run.name || activity.sportType == ActivityType.TrailRun.name
                "ride" -> activity.sportType == ActivityType.Ride.name
                        || activity.sportType == ActivityType.GravelRide.name
                        || activity.sportType == ActivityType.MountainBikeRide.name
                        || activity.sportType == ActivityType.VirtualRide.name
                else -> false
            }
        }
    }

    private fun computeBestTimeDistanceMetric(
        group: String,
        activities: List<StravaActivity>,
        distance: Double,
    ): WarmupMetricSummary? {
        val bestEffort = activities
            .mapNotNull { activity -> activity.calculateBestTimeForDistance(distance) }
            .minByOrNull { effort -> effort.seconds }
            ?: return null

        return WarmupMetricSummary(
            activityGroup = group,
            metric = "best-time-distance",
            target = distance.toString(),
            value = "${bestEffort.seconds.formatSeconds()} => ${bestEffort.getFormattedSpeedWithUnits()}",
            activityId = bestEffort.activityShort.id,
        )
    }

    private fun computeBestDistanceTimeMetric(
        group: String,
        activities: List<StravaActivity>,
        seconds: Int,
    ): WarmupMetricSummary? {
        val bestEffort = activities
            .mapNotNull { activity -> activity.calculateBestDistanceForTime(seconds) }
            .maxByOrNull { effort -> effort.distance }
            ?: return null

        val distanceLabel = if (bestEffort.distance >= 1000.0) {
            "%.2f km".format(Locale.ENGLISH, bestEffort.distance / 1000.0)
        } else {
            "%.0f m".format(Locale.ENGLISH, bestEffort.distance)
        }

        return WarmupMetricSummary(
            activityGroup = group,
            metric = "best-distance-time",
            target = seconds.toString(),
            value = "$distanceLabel => ${bestEffort.getFormattedSpeedWithUnits()}",
            activityId = bestEffort.activityShort.id,
        )
    }

    private fun computeBestPowerMetric(
        group: String,
        activities: List<StravaActivity>,
        seconds: Int,
    ): WarmupMetricSummary? {
        val bestEffort = activities
            .mapNotNull { activity -> activity.calculateBestPowerForTime(seconds) }
            .maxByOrNull { effort -> effort.distance }
            ?: return null
        val power = bestEffort.averagePower ?: return null

        return WarmupMetricSummary(
            activityGroup = group,
            metric = "best-power-time",
            target = seconds.toString(),
            value = "$power W",
            activityId = bestEffort.activityShort.id,
        )
    }

    private fun computeBestElevationMetric(
        group: String,
        activities: List<StravaActivity>,
        distance: Double,
    ): WarmupMetricSummary? {
        val bestEffort = activities
            .mapNotNull { activity -> activity.calculateBestElevationForDistance(distance) }
            .maxByOrNull { effort -> effort.deltaAltitude }
            ?: return null

        return WarmupMetricSummary(
            activityGroup = group,
            metric = "best-elevation-distance",
            target = distance.toString(),
            value = "${bestEffort.seconds.formatSeconds()} => ${bestEffort.getFormattedGradient()}%",
            activityId = bestEffort.activityShort.id,
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

    private data class MutableWarmupYearSummary(
        val year: Int,
        var activityCount: Int = 0,
        var totalDistanceKm: Double = 0.0,
        var totalElevationM: Double = 0.0,
        var elapsedSeconds: Int = 0,
    ) {
        fun accept(activity: StravaActivity) {
            activityCount += 1
            totalDistanceKm += activity.distance / 1000.0
            totalElevationM += activity.totalElevationGain
            elapsedSeconds += activity.elapsedTime
        }

        fun toPublic(): WarmupYearSummary = WarmupYearSummary(
            year = year,
            activityCount = activityCount,
            totalDistanceKm = totalDistanceKm,
            totalElevationM = totalElevationM,
            elapsedSeconds = elapsedSeconds,
        )
    }
}
