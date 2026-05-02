package me.nicolas.stravastats.domain.services.activityproviders

import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import me.nicolas.stravastats.domain.business.ActivityType
import me.nicolas.stravastats.domain.business.strava.StravaActivity
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
import me.nicolas.stravastats.domain.utils.formatSeconds
import org.slf4j.LoggerFactory
import java.nio.file.Files
import java.time.Instant
import java.util.Locale
import java.util.concurrent.atomic.AtomicBoolean

/**
 * Handles the warmup pipeline that pre-computes best-effort statistics and year
 * summaries, and manages the on-disk cache manifest.
 *
 * Extracted from StravaActivityProvider to isolate the warmup responsibility.
 */
internal class StravaWarmupPipeline(
    private val cacheRoot: String,
    private val clientId: String,
) {
    private val logger = LoggerFactory.getLogger(StravaWarmupPipeline::class.java)

    private val manifestLock = Any()
    private val warmupInProgress = AtomicBoolean(false)

    @Volatile
    private var cacheManifest: CacheManifest = CacheManifestStore.defaultManifest("unknown")

    // --- Public state accessors ---

    /** Thread-safe snapshot of the current cache manifest. */
    fun manifestSnapshot(): CacheManifest = synchronized(manifestLock) { cacheManifest }

    /** Returns true if a warmup run is currently in progress. */
    fun isWarmupInProgress(): Boolean = warmupInProgress.get()

    // --- Lifecycle ---

    /**
     * Loads the cache manifest and the best-effort cache from disk.
     * Must be called once during startup before [runWarmupPipeline].
     */
    fun initialize() {
        val loadedManifest = CacheManifestStore.load(cacheRoot, clientId)
            ?: CacheManifestStore.defaultManifest(clientId)

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

        runCatching { CacheManifestStore.save(cacheRoot, updatedManifest) }
            .onFailure { exception -> logger.error("Unable to save cache manifest", exception) }

        synchronized(manifestLock) { cacheManifest = updatedManifest }

        logger.info("Loaded best-effort cache: {} entries", loadedEntries)
    }

    // --- Warmup pipeline ---

    /**
     * Runs the three-stage warmup pipeline (year summaries -> best efforts -> advanced metrics).
     * Concurrent calls while a warmup is in progress are silently dropped.
     *
     * @param reason Human-readable label used in log messages.
     * @param activities Snapshot of activities to compute metrics from.
     */
    suspend fun runWarmupPipeline(reason: String, activities: List<StravaActivity>) {
        if (!warmupInProgress.compareAndSet(false, true)) {
            return
        }
        try {
            if (activities.isEmpty()) return
            logger.info("Warmup started ({})", reason)

            val yearSummaries = computeWarmupYearSummaries(activities)
            val preparedYears = yearSummaries.map { it.year }.sortedDescending()

            var payload = WarmupSummariesFile(athleteId = clientId, yearSummaries = yearSummaries)
            persistWarmupArtifacts(payload, "ready", "pending", "pending", preparedYears)

            payload = payload.copy(majorBestEfforts = precomputeMajorBestEfforts(activities))
            persistWarmupArtifacts(payload, "ready", "ready", "pending", preparedYears)

            payload = payload.copy(advancedMetrics = precomputeAdvancedMetrics(activities))
            persistWarmupArtifacts(payload, "ready", "ready", "ready", preparedYears)

            logger.info("Warmup completed ({})", reason)
        } catch (exception: Exception) {
            logger.error("Warmup failed ({})", reason, exception)
        } finally {
            warmupInProgress.set(false)
        }
    }

    // --- Diagnostics ---

    /** Returns the manifest and files sub-sections for [StravaActivityProvider.getCacheDiagnostics]. */
    fun diagnosticsSection(): Map<String, Any?> {
        val snap = synchronized(manifestLock) { cacheManifest }
        val manifestPath = CacheManifestStore.manifestPath(cacheRoot, clientId)
        val bestEffortPath = CacheManifestStore.bestEffortCachePath(cacheRoot, clientId, snap)
        val warmupPath = CacheManifestStore.warmupSummariesPath(cacheRoot, clientId, snap)
        return mapOf(
            "manifest" to mapOf(
                "schemaVersion" to snap.schemaVersion,
                "updatedAt" to snap.updatedAt,
                "bestEffortCache" to mapOf(
                    "algoVersion" to snap.bestEffortCache.algoVersion,
                    "entriesPersisted" to snap.bestEffortCache.entries,
                    "entriesInMemory" to BestEffortCache.size(),
                    "file" to snap.bestEffortCache.file,
                    "lastPersistedAt" to snap.bestEffortCache.lastPersistedAt,
                ),
                "warmup" to mapOf(
                    "algoVersion" to snap.warmup.algoVersion,
                    "file" to snap.warmup.file,
                    "priority1" to snap.warmup.priority1,
                    "priority2" to snap.warmup.priority2,
                    "priority3" to snap.warmup.priority3,
                    "preparedYears" to snap.warmup.preparedYears,
                    "lastRunAt" to snap.warmup.lastRunAt,
                ),
            ),
            "files" to mapOf(
                "manifest" to fileDiagnostics(manifestPath),
                "bestEffortCache" to fileDiagnostics(bestEffortPath),
                "warmupSummaries" to fileDiagnostics(warmupPath),
            ),
        )
    }

    // --- Private helpers ---

    private suspend fun persistWarmupArtifacts(
        payload: WarmupSummariesFile,
        priority1: String,
        priority2: String,
        priority3: String,
        preparedYears: List<Int>,
    ) {
        val snap = synchronized(manifestLock) { cacheManifest }
        val bestEffortPath = CacheManifestStore.bestEffortCachePath(cacheRoot, clientId, snap)
        val entries = withContext(Dispatchers.IO) { BestEffortCache.saveToDisk(bestEffortPath) }
        val updated = snap.copy(
            updatedAt = Instant.now().toString(),
            bestEffortCache = snap.bestEffortCache.copy(entries = entries, lastPersistedAt = Instant.now().toString()),
            warmup = snap.warmup.copy(
                priority1 = priority1, priority2 = priority2, priority3 = priority3,
                preparedYears = preparedYears, lastRunAt = Instant.now().toString(),
            )
        )
        withContext(Dispatchers.IO) {
            CacheManifestStore.saveWarmupSummaries(cacheRoot, clientId, payload, updated)
            CacheManifestStore.save(cacheRoot, updated)
        }
        synchronized(manifestLock) { cacheManifest = updated }
    }

    private fun computeWarmupYearSummaries(activities: List<StravaActivity>): List<WarmupYearSummary> {
        val summaries = mutableMapOf<Int, MutableWarmupYearSummary>()
        val allYears = MutableWarmupYearSummary(year = 0)
        activities.forEach { activity ->
            val year = resolveYearFromDateString(activity.startDateLocal)
            summaries.getOrPut(year) { MutableWarmupYearSummary(year = year) }.accept(activity)
            allYears.accept(activity)
        }
        return buildList {
            add(allYears.toPublic())
            addAll(summaries.values.map { it.toPublic() })
        }.sortedByDescending { it.year }
    }

    private fun precomputeMajorBestEfforts(activities: List<StravaActivity>): List<WarmupMetricSummary> {
        val ride = filterActivitiesForWarmup(activities, "ride")
        val run = filterActivitiesForWarmup(activities, "run")
        return buildList {
            computeBestTimeDistanceMetric("ride", ride, 1000.0)?.let { add(it) }
            computeBestTimeDistanceMetric("ride", ride, 5000.0)?.let { add(it) }
            computeBestDistanceTimeMetric("ride", ride, 20 * 60)?.let { add(it) }
            computeBestDistanceTimeMetric("ride", ride, 60 * 60)?.let { add(it) }
            computeBestTimeDistanceMetric("run", run, 1000.0)?.let { add(it) }
            computeBestTimeDistanceMetric("run", run, 5000.0)?.let { add(it) }
            computeBestDistanceTimeMetric("run", run, 20 * 60)?.let { add(it) }
            computeBestDistanceTimeMetric("run", run, 60 * 60)?.let { add(it) }
        }
    }

    private fun precomputeAdvancedMetrics(activities: List<StravaActivity>): List<WarmupMetricSummary> {
        val ride = filterActivitiesForWarmup(activities, "ride")
        return buildList {
            computeBestElevationMetric("ride", ride, 1000.0)?.let { add(it) }
            computeBestElevationMetric("ride", ride, 5000.0)?.let { add(it) }
            computeBestPowerMetric("ride", ride, 20 * 60)?.let { add(it) }
            computeBestPowerMetric("ride", ride, 60 * 60)?.let { add(it) }
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

    private fun computeBestTimeDistanceMetric(group: String, activities: List<StravaActivity>, distance: Double): WarmupMetricSummary? {
        val best = activities.mapNotNull { it.calculateBestTimeForDistance(distance) }.minByOrNull { it.seconds }
            ?: return null
        return WarmupMetricSummary(group, "best-time-distance", distance.toString(),
            "${best.seconds.formatSeconds()} => ${best.getFormattedSpeedWithUnits()}", best.activityShort.id)
    }

    private fun computeBestDistanceTimeMetric(group: String, activities: List<StravaActivity>, seconds: Int): WarmupMetricSummary? {
        val best = activities.mapNotNull { it.calculateBestDistanceForTime(seconds) }.maxByOrNull { it.distance }
            ?: return null
        val label = if (best.distance >= 1000.0) "%.2f km".format(Locale.ENGLISH, best.distance / 1000.0)
                    else "%.0f m".format(Locale.ENGLISH, best.distance)
        return WarmupMetricSummary(group, "best-distance-time", seconds.toString(),
            "$label => ${best.getFormattedSpeedWithUnits()}", best.activityShort.id)
    }

    private fun computeBestPowerMetric(group: String, activities: List<StravaActivity>, seconds: Int): WarmupMetricSummary? {
        val best = activities.mapNotNull { it.calculateBestPowerForTime(seconds) }.maxByOrNull { it.distance }
            ?: return null
        val power = best.averagePower ?: return null
        return WarmupMetricSummary(group, "best-power-time", seconds.toString(), "$power W", best.activityShort.id)
    }

    private fun computeBestElevationMetric(group: String, activities: List<StravaActivity>, distance: Double): WarmupMetricSummary? {
        val best = activities.mapNotNull { it.calculateBestElevationForDistance(distance) }.maxByOrNull { it.deltaAltitude }
            ?: return null
        return WarmupMetricSummary(group, "best-elevation-distance", distance.toString(),
            "${best.seconds.formatSeconds()} => ${best.getFormattedGradient()}%", best.activityShort.id)
    }

    private fun fileDiagnostics(path: java.nio.file.Path): Map<String, Any?> {
        if (!Files.exists(path)) return mapOf("path" to path.toString(), "exists" to false)
        return mapOf(
            "path" to path.toString(), "exists" to true,
            "sizeBytes" to Files.size(path),
            "lastModified" to Files.getLastModifiedTime(path).toInstant().toString(),
        )
    }

    /**
     * Mutable accumulator used only inside [computeWarmupYearSummaries].
     * Not a data class: mutable fields, never used as map key or copied via copy().
     */
    private class MutableWarmupYearSummary(
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
        fun toPublic() = WarmupYearSummary(year, activityCount, totalDistanceKm, totalElevationM, elapsedSeconds)
    }
}

