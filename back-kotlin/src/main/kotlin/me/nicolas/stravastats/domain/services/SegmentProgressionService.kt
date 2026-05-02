package me.nicolas.stravastats.domain.services

import me.nicolas.stravastats.domain.business.ActivityShort
import me.nicolas.stravastats.domain.business.ActivityType
import me.nicolas.stravastats.domain.business.SegmentClimbAttempt
import me.nicolas.stravastats.domain.business.SegmentClimbProgression
import me.nicolas.stravastats.domain.business.SegmentClimbTargetSummary
import me.nicolas.stravastats.domain.business.SegmentSummary
import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.business.strava.StravaDetailedActivity
import me.nicolas.stravastats.domain.business.strava.StravaSegmentEffort
import me.nicolas.stravastats.domain.services.activityproviders.IActivityProvider
import me.nicolas.stravastats.domain.services.cache.SegmentAnalysisCacheEntryFile
import me.nicolas.stravastats.domain.services.cache.SegmentAnalysisCacheFile
import me.nicolas.stravastats.domain.services.cache.SegmentAnalysisCacheStore
import me.nicolas.stravastats.domain.services.cache.SegmentAttemptRawSnapshot
import me.nicolas.stravastats.domain.utils.formatSeconds
import org.slf4j.LoggerFactory
import org.springframework.stereotype.Service
import java.time.Instant
import java.time.LocalDate
import java.util.Locale
import kotlin.math.abs
import kotlin.math.roundToInt
import kotlin.math.sqrt

@Service
internal class SegmentProgressionService(
    activityProvider: IActivityProvider,
) : ISegmentProgressionService, AbstractStravaService(activityProvider) {

    companion object {
        private const val SEGMENT_CACHE_MAX_ENTRIES = 256
        private const val SEGMENT_CACHE_TTL_SECONDS = 30 * 60L
        private const val SEGMENT_FALLBACK_CACHE_TTL_SECONDS = 45L
        private const val SEGMENT_ANALYSIS_ALGO_VERSION = "direction-v2"
        private const val SEGMENT_DIRECTION_MIN_ALTITUDE_DELTA_M = 3.0
        private const val SEGMENT_DIRECTION_MIN_GRADE_PERCENT = 0.5
    }

    private val logger = LoggerFactory.getLogger(SegmentProgressionService::class.java)

    private val segmentCacheIdentity by lazy(LazyThreadSafetyMode.NONE) {
        runCatching { activityProvider.cacheIdentity() }.getOrNull()
    }
    private val segmentCacheLock = Any()

    @Volatile
    private var segmentCacheLoaded = false
    private val segmentAttemptsCache = mutableMapOf<String, SegmentAttemptsCacheEntry>()

    // ---- Public API ----

    override fun getSegmentClimbProgression(
        activityTypes: Set<ActivityType>,
        year: Int?,
        metric: String?,
        targetType: String?,
        targetId: Long?,
    ): SegmentClimbProgression {
        val resolvedMetric = parseSegmentMetric(metric)
        val resolvedTargetTypeFilter = parseSegmentTargetType(targetType)
        logger.info(
            "Compute segment/climb progression for {} in {} with metric={} targetType={} targetId={}",
            activityTypes,
            year ?: "all years",
            resolvedMetric,
            resolvedTargetTypeFilter,
            targetId,
        )

        val filteredActivities = activityProvider.getActivitiesByActivityTypeAndYear(activityTypes, year)
            .withoutDataQualityExcludedStats(activityProvider)
            .sortedBy { activity -> activity.startDateLocal }

        val rawAttempts = filteredActivities.flatMap { activity ->
            val detailedActivity = activityProvider.getCachedDetailedActivity(activity.id)
                ?: activityProvider.getDetailedActivity(activity.id)
                ?: return@flatMap emptyList()

            detailedActivity.segmentEfforts
                .asSequence()
                .filter { effort -> isFavoriteEffort(effort) }
                .map { effort ->
                    val effortTargetType =
                        if (effort.segment.climbCategory > 0) SegmentTargetType.CLIMB else SegmentTargetType.SEGMENT
                    SegmentAttemptRaw(
                        effortId = effort.id,
                        targetId = effort.segment.id,
                        targetName = effort.segment.name,
                        targetType = effortTargetType,
                        direction = resolveSegmentDirection(effort, activity, detailedActivity),
                        climbCategory = effort.segment.climbCategory,
                        distance = effort.distance,
                        averageGrade = effort.segment.averageGrade,
                        elapsedTimeSeconds = effort.elapsedTime,
                        movingTimeSeconds = effort.movingTime,
                        speedKph = computeSpeedKph(effort.distance, effort.elapsedTime),
                        averagePowerWatts = effort.averageWatts,
                        averageHeartRate = effort.averageHeartRate,
                        activityDate = effort.startDateLocal,
                        prRank = effort.prRank,
                        activity = ActivityShort(activity.id, activity.name, activity.type),
                    )
                }
                .filter { raw ->
                    resolvedTargetTypeFilter == SegmentTargetType.ALL || raw.targetType == resolvedTargetTypeFilter
                }
                .toList()
        }

        val directionAwareRawAttempts = splitAttemptsByDirection(rawAttempts)

        if (directionAwareRawAttempts.isEmpty()) {
            if (year != null) {
                // Fallback for UX: when selected year has no segment data yet, reuse all-years data.
                return getSegmentClimbProgression(activityTypes, null, metric, targetType, targetId)
            }
            return SegmentClimbProgression(
                metric = resolvedMetric.name,
                targetTypeFilter = resolvedTargetTypeFilter.name,
                weatherContextAvailable = false,
                targets = emptyList(),
                selectedTargetId = null,
                selectedTargetType = null,
                attempts = emptyList(),
            )
        }

        val attemptsByTarget = directionAwareRawAttempts.groupBy { raw -> raw.targetId }
        val targetSummaries = attemptsByTarget.values
            .map { attempts -> buildTargetSummary(attempts, resolvedMetric) }
            .sortedWith(
                compareByDescending<SegmentClimbTargetSummary> { summary -> summary.attemptsCount }
                    .thenBy { summary -> summary.targetName.lowercase(Locale.getDefault()) }
            )

        val selectedTarget = targetSummaries.firstOrNull { summary -> summary.targetId == targetId }
            ?: targetSummaries.firstOrNull()

        val selectedAttempts = selectedTarget?.let { summary ->
            val attempts = attemptsByTarget[summary.targetId].orEmpty()
            buildAttempts(attempts, resolvedMetric)
        } ?: emptyList()

        return SegmentClimbProgression(
            metric = resolvedMetric.name,
            targetTypeFilter = resolvedTargetTypeFilter.name,
            weatherContextAvailable = false,
            targets = targetSummaries,
            selectedTargetId = selectedTarget?.targetId,
            selectedTargetType = selectedTarget?.targetType,
            attempts = selectedAttempts,
        )
    }

    override fun listSegments(
        activityTypes: Set<ActivityType>,
        year: Int?,
        metric: String?,
        query: String?,
        from: String?,
        to: String?,
    ): List<SegmentClimbTargetSummary> {
        val resolvedMetric = parseSegmentMetric(metric)
        val queryFilter = query.orEmpty().trim().lowercase(Locale.getDefault())
        val rawAttempts = collectSegmentAttempts(activityTypes, year, from, to)
        if (rawAttempts.isEmpty()) return emptyList()

        return rawAttempts
            .groupBy { attempt -> attempt.targetId }
            .values
            .asSequence()
            .filter { attempts -> attempts.size >= 2 }
            .map { attempts -> buildTargetSummary(attempts, resolvedMetric) }
            .filter { summary ->
                queryFilter.isBlank() || summary.targetName.lowercase(Locale.getDefault()).contains(queryFilter)
            }
            .sortedWith(
                compareByDescending<SegmentClimbTargetSummary> { summary -> summary.attemptsCount }
                    .thenBy { summary -> summary.targetName.lowercase(Locale.getDefault()) }
            )
            .toList()
    }

    override fun getSegmentEfforts(
        activityTypes: Set<ActivityType>,
        year: Int?,
        metric: String?,
        segmentId: Long,
        from: String?,
        to: String?,
    ): List<SegmentClimbAttempt> {
        val resolvedMetric = parseSegmentMetric(metric)
        val attempts = collectSegmentAttempts(activityTypes, year, from, to)
            .filter { attempt -> attempt.targetId == segmentId }
        if (attempts.isEmpty()) return emptyList()
        return buildAttempts(attempts, resolvedMetric)
    }

    override fun getSegmentSummary(
        activityTypes: Set<ActivityType>,
        year: Int?,
        metric: String?,
        segmentId: Long,
        from: String?,
        to: String?,
    ): SegmentSummary? {
        val resolvedMetric = parseSegmentMetric(metric)
        val attempts = collectSegmentAttempts(activityTypes, year, from, to)
            .filter { attempt -> attempt.targetId == segmentId }
        if (attempts.isEmpty()) return null

        val progressionAttempts = buildAttempts(attempts, resolvedMetric)
        val topEfforts = rankTopEfforts(progressionAttempts, resolvedMetric, 3)
        return SegmentSummary(
            metric = resolvedMetric.name,
            segment = buildTargetSummary(attempts, resolvedMetric),
            personalRecord = topEfforts.firstOrNull(),
            topEfforts = topEfforts,
        )
    }

    // ---- Segment collection and cache ----

    private fun collectSegmentAttempts(
        activityTypes: Set<ActivityType>,
        year: Int?,
        from: String?,
        to: String?,
    ): List<SegmentAttemptRaw> {
        val fromDate = parseDateFilter(from)
        val toDate = parseDateFilter(to)

        val filteredActivities = activityProvider.getActivitiesByActivityTypeAndYear(activityTypes, year)
            .withoutDataQualityExcludedStats(activityProvider)
            .sortedBy { activity -> activity.startDateLocal }

        val cacheKey = buildSegmentAttemptsCacheKey(
            activityTypes = activityTypes,
            year = year,
            from = from,
            to = to,
            activitySignature = computeSegmentActivitiesSignature(filteredActivities),
        )
        getSegmentAttemptsFromCache(cacheKey)?.let { cachedAttempts -> return cachedAttempts }

        val segmentEffortAttempts = filteredActivities.flatMap { activity ->
            val detailedActivity = activityProvider.getCachedDetailedActivity(activity.id)
                ?: activityProvider.getDetailedActivity(activity.id)
                ?: return@flatMap emptyList()

            detailedActivity.segmentEfforts
                .asSequence()
                .filter { effort -> isCandidateEffort(effort) }
                .filter { effort -> matchesDateFilter(effort.startDateLocal, fromDate, toDate) }
                .map { effort ->
                    val effortTargetType =
                        if (effort.segment.climbCategory > 0) SegmentTargetType.CLIMB else SegmentTargetType.SEGMENT
                    SegmentAttemptRaw(
                        effortId = effort.id,
                        targetId = effort.segment.id,
                        targetName = effort.segment.name,
                        targetType = effortTargetType,
                        direction = resolveSegmentDirection(effort, activity, detailedActivity),
                        climbCategory = effort.segment.climbCategory,
                        distance = effort.distance,
                        averageGrade = effort.segment.averageGrade,
                        elapsedTimeSeconds = effort.elapsedTime,
                        movingTimeSeconds = effort.movingTime,
                        speedKph = computeSpeedKph(effort.distance, effort.elapsedTime),
                        averagePowerWatts = effort.averageWatts,
                        averageHeartRate = effort.averageHeartRate,
                        activityDate = effort.startDateLocal,
                        prRank = effort.prRank,
                        activity = ActivityShort(activity.id, activity.name, activity.type),
                    )
                }
                .toList()
        }

        val directionAwareSegmentEfforts = splitAttemptsByDirection(segmentEffortAttempts)

        if (directionAwareSegmentEfforts.isNotEmpty()) {
            storeSegmentAttemptsInCache(cacheKey, directionAwareSegmentEfforts, fallbackUsed = false)
            return directionAwareSegmentEfforts
        }

        // Cache-only fallback:
        // If there is no cached detailed activity (thus no segment_efforts),
        // expose progression by repeated route names so the view remains usable.
        return buildRouteNameFallbackAttempts(filteredActivities, fromDate, toDate)
            .also { fallbackAttempts ->
                if (fallbackAttempts.isNotEmpty()) {
                    storeSegmentAttemptsInCache(cacheKey, fallbackAttempts, fallbackUsed = true)
                }
            }
    }

    private fun buildRouteNameFallbackAttempts(
        activities: List<StravaActivity>,
        fromDate: LocalDate?,
        toDate: LocalDate?,
    ): List<SegmentAttemptRaw> {
        val groupedByName = activities
            .filter { activity ->
                val day = extractSortableDay(activity.startDateLocal)?.let { LocalDate.parse(it) }
                    ?: return@filter false
                if (fromDate != null && day.isBefore(fromDate)) return@filter false
                if (toDate != null && day.isAfter(toDate)) return@filter false
                activity.name.isNotBlank()
            }
            .groupBy { activity -> activity.name.trim().lowercase(Locale.getDefault()) }
            .filterValues { grouped -> grouped.size >= 2 }

        return groupedByName.flatMap { (normalizedName, groupedActivities) ->
            val displayName = groupedActivities.firstOrNull()?.name?.trim().orEmpty()
            val targetId = routeNameBasedTargetId(normalizedName)

            groupedActivities.mapNotNull { activity ->
                val elapsedSeconds = activity.elapsedTime
                if (elapsedSeconds <= 0) return@mapNotNull null
                val movingSeconds = if (activity.movingTime > 0) activity.movingTime else elapsedSeconds
                val averageGrade = if (activity.distance > 0.0) {
                    (activity.totalElevationGain / activity.distance) * 100.0
                } else {
                    0.0
                }

                SegmentAttemptRaw(
                    effortId = activity.id,
                    targetId = targetId,
                    targetName = displayName,
                    targetType = SegmentTargetType.SEGMENT,
                    climbCategory = 0,
                    distance = activity.distance,
                    averageGrade = averageGrade,
                    elapsedTimeSeconds = elapsedSeconds,
                    movingTimeSeconds = movingSeconds,
                    speedKph = computeSpeedKph(activity.distance, movingSeconds),
                    averagePowerWatts = activity.averageWatts.toDouble(),
                    averageHeartRate = activity.averageHeartrate,
                    activityDate = activity.startDateLocal,
                    prRank = null,
                    activity = ActivityShort(activity.id, activity.name, activity.type),
                )
            }
        }
    }

    private fun routeNameBasedTargetId(nameKey: String): Long {
        // Keep fallback IDs negative to avoid collisions with real Strava segment IDs.
        return -nameKey.hashCode().toLong().let { hash -> abs(hash) + 1L }
    }

    // ---- Cache management ----

    private fun buildSegmentAttemptsCacheKey(
        activityTypes: Set<ActivityType>,
        year: Int?,
        from: String?,
        to: String?,
        activitySignature: Long,
    ): String {
        val sortedTypes = activityTypes.map { type -> type.name }.sorted().joinToString(",")
        val normalizedFrom = from?.trim().takeUnless { value -> value.isNullOrBlank() } ?: "none"
        val normalizedTo = to?.trim().takeUnless { value -> value.isNullOrBlank() } ?: "none"
        val normalizedYear = year?.toString() ?: "all"

        return listOf(
            "algo:$SEGMENT_ANALYSIS_ALGO_VERSION",
            "types:$sortedTypes",
            "year:$normalizedYear",
            "from:$normalizedFrom",
            "to:$normalizedTo",
            "activities:${activitySignature.toULong().toString(16)}",
        ).joinToString("|")
    }

    private fun computeSegmentActivitiesSignature(activities: List<StravaActivity>): Long {
        var hash = 1469598103934665603L
        val prime = 1099511628211L

        activities.forEach { activity ->
            hash = (hash xor activity.id) * prime
            hash = (hash xor activity.startDateLocal.hashCode().toLong()) * prime
            hash = (hash xor activity.name.hashCode().toLong()) * prime
            hash = (hash xor activity.distance.toBits()) * prime
            hash = (hash xor activity.totalElevationGain.toBits()) * prime
            hash = (hash xor activity.elapsedTime.toLong()) * prime
            hash = (hash xor activity.movingTime.toLong()) * prime
            hash = (hash xor activity.sportType.hashCode().toLong()) * prime
            hash = (hash xor activity.type.hashCode().toLong()) * prime
        }

        return hash
    }

    private fun getSegmentAttemptsFromCache(cacheKey: String): List<SegmentAttemptRaw>? {
        ensureSegmentCacheLoaded()
        val now = Instant.now()

        synchronized(segmentCacheLock) {
            val entry = segmentAttemptsCache[cacheKey] ?: return null
            if (!entry.expiresAt.isAfter(now)) {
                segmentAttemptsCache.remove(cacheKey)
                persistSegmentAttemptsCacheLocked()
                return null
            }
            return entry.attempts.map { attempt -> attempt.copy() }
        }
    }

    private fun storeSegmentAttemptsInCache(
        cacheKey: String,
        attempts: List<SegmentAttemptRaw>,
        fallbackUsed: Boolean,
    ) {
        if (attempts.isEmpty()) return

        ensureSegmentCacheLoaded()
        val now = Instant.now()
        val expiresAt = now.plusSeconds(
            if (fallbackUsed) SEGMENT_FALLBACK_CACHE_TTL_SECONDS else SEGMENT_CACHE_TTL_SECONDS
        )

        synchronized(segmentCacheLock) {
            segmentAttemptsCache[cacheKey] = SegmentAttemptsCacheEntry(
                createdAt = now,
                expiresAt = expiresAt,
                fallbackUsed = fallbackUsed,
                attempts = attempts.map { attempt -> attempt.copy() },
            )
            trimSegmentCacheLocked(now)
            persistSegmentAttemptsCacheLocked()
        }
    }

    private fun ensureSegmentCacheLoaded() {
        if (segmentCacheLoaded) return

        synchronized(segmentCacheLock) {
            if (segmentCacheLoaded) return

            segmentCacheLoaded = true
            val identity = segmentCacheIdentity ?: return
            val payload = SegmentAnalysisCacheStore.load(identity.cacheRoot, identity.athleteId) ?: return

            val now = Instant.now()
            payload.entries.forEach { entry ->
                val createdAt = runCatching { Instant.parse(entry.createdAt) }.getOrNull() ?: now
                val expiresAt = runCatching { Instant.parse(entry.expiresAt) }.getOrNull() ?: return@forEach
                if (!expiresAt.isAfter(now)) return@forEach
                val attempts = entry.attempts.mapNotNull { snapshot -> snapshot.toRawOrNull() }
                if (attempts.isEmpty()) return@forEach
                segmentAttemptsCache[entry.key] = SegmentAttemptsCacheEntry(
                    createdAt = createdAt,
                    expiresAt = expiresAt,
                    fallbackUsed = entry.fallbackUsed,
                    attempts = attempts,
                )
            }
        }
    }

    private fun trimSegmentCacheLocked(now: Instant) {
        segmentAttemptsCache.entries.removeIf { (_, entry) -> !entry.expiresAt.isAfter(now) }

        if (segmentAttemptsCache.size <= SEGMENT_CACHE_MAX_ENTRIES) return

        val keysToRemove = segmentAttemptsCache.entries
            .sortedByDescending { (_, entry) -> entry.createdAt }
            .drop(SEGMENT_CACHE_MAX_ENTRIES)
            .map { (key, _) -> key }

        keysToRemove.forEach { key -> segmentAttemptsCache.remove(key) }
    }

    private fun persistSegmentAttemptsCacheLocked() {
        val identity = segmentCacheIdentity ?: return
        val now = Instant.now()

        val entries = segmentAttemptsCache.entries
            .filter { (_, entry) -> entry.expiresAt.isAfter(now) }
            .map { (key, entry) ->
                SegmentAnalysisCacheEntryFile(
                    key = key,
                    createdAt = entry.createdAt.toString(),
                    expiresAt = entry.expiresAt.toString(),
                    fallbackUsed = entry.fallbackUsed,
                    attempts = entry.attempts.map { attempt -> attempt.toSnapshot() },
                )
            }

        SegmentAnalysisCacheStore.save(
            identity.cacheRoot,
            identity.athleteId,
            SegmentAnalysisCacheFile(
                athleteId = identity.athleteId,
                entries = entries,
            ),
        )
    }

    // ---- Snapshot serialization helpers ----

    private fun SegmentAttemptRaw.toSnapshot(): SegmentAttemptRawSnapshot = SegmentAttemptRawSnapshot(
        effortId = effortId,
        targetId = targetId,
        targetName = targetName,
        targetType = targetType.name,
        climbCategory = climbCategory,
        distance = distance,
        averageGrade = averageGrade,
        elapsedTimeSeconds = elapsedTimeSeconds,
        movingTimeSeconds = movingTimeSeconds,
        speedKph = speedKph,
        averagePowerWatts = averagePowerWatts,
        averageHeartRate = averageHeartRate,
        activityDate = activityDate,
        prRank = prRank,
        activity = activity,
    )

    private fun SegmentAttemptRawSnapshot.toRawOrNull(): SegmentAttemptRaw? {
        val resolvedTargetType = SegmentTargetType.entries.firstOrNull { candidate ->
            candidate.name.equals(targetType, ignoreCase = true)
        } ?: return null

        return SegmentAttemptRaw(
            effortId = effortId,
            targetId = targetId,
            targetName = targetName,
            targetType = resolvedTargetType,
            climbCategory = climbCategory,
            distance = distance,
            averageGrade = averageGrade,
            elapsedTimeSeconds = elapsedTimeSeconds,
            movingTimeSeconds = movingTimeSeconds,
            speedKph = speedKph,
            averagePowerWatts = averagePowerWatts,
            averageHeartRate = averageHeartRate,
            activityDate = activityDate,
            prRank = prRank,
            activity = activity,
        )
    }

    // ---- Direction resolution ----

    private fun isFavoriteEffort(effort: StravaSegmentEffort): Boolean =
        effort.segment.starred || effort.segment.climbCategory > 0 || (effort.prRank != null && effort.prRank <= 3)

    private fun isCandidateEffort(effort: StravaSegmentEffort): Boolean {
        if (effort.segment.id == 0L) return false
        if (effort.segment.name.isBlank()) return false
        return effort.elapsedTime > 0 && effort.distance > 0.0
    }

    private fun resolveSegmentDirection(
        effort: StravaSegmentEffort,
        activity: StravaActivity,
        detailedActivity: StravaDetailedActivity,
    ): SegmentDirection {
        resolveDirectionFromAltitudeStream(detailedActivity, effort)?.let { return it }
        resolveDirectionFromLabels(activity.name, effort.name, effort.segment.name)?.let { return it }
        return resolveDirectionFromAverageGrade(effort.segment.averageGrade)
    }

    private fun resolveDirectionFromAltitudeStream(
        detailedActivity: StravaDetailedActivity,
        effort: StravaSegmentEffort,
    ): SegmentDirection? {
        val altitudeData = detailedActivity.stream?.altitude?.data ?: return null
        if (altitudeData.isEmpty()) return null
        if (effort.startIndex < 0 || effort.endIndex < 0) return null
        if (effort.startIndex >= altitudeData.size || effort.endIndex >= altitudeData.size) return null
        if (effort.startIndex == effort.endIndex) return null

        val altitudeDelta = altitudeData[effort.endIndex] - altitudeData[effort.startIndex]
        if (abs(altitudeDelta) < SEGMENT_DIRECTION_MIN_ALTITUDE_DELTA_M) return null
        return if (altitudeDelta > 0.0) SegmentDirection.ASCENT else SegmentDirection.DESCENT
    }

    private fun resolveDirectionFromLabels(vararg labels: String): SegmentDirection? {
        val ascentKeywords = listOf("montee", "ascent", "climb", "uphill")
        val descentKeywords = listOf("descente", "descent", "downhill")

        labels.forEach { label ->
            val normalized = normalizeDirectionLabel(label)
            if (normalized.isBlank()) return@forEach
            if (descentKeywords.any { keyword -> normalized.contains(keyword) }) return SegmentDirection.DESCENT
            if (ascentKeywords.any { keyword -> normalized.contains(keyword) }) return SegmentDirection.ASCENT
        }
        return null
    }

    private fun normalizeDirectionLabel(label: String): String {
        if (label.isBlank()) return ""
        return label.trim()
            .lowercase(Locale.getDefault())
            .replace("é", "e")
            .replace("è", "e")
            .replace("ê", "e")
            .replace("ë", "e")
            .replace("à", "a")
            .replace("â", "a")
            .replace("ä", "a")
            .replace("î", "i")
            .replace("ï", "i")
            .replace("ô", "o")
            .replace("ö", "o")
            .replace("ù", "u")
            .replace("û", "u")
            .replace("ü", "u")
            .replace("ç", "c")
            .replace("'", "'")
            .replace("-", " ")
            .split(Regex("\\s+"))
            .filter { token -> token.isNotBlank() }
            .joinToString(" ")
    }

    private fun resolveDirectionFromAverageGrade(averageGrade: Double): SegmentDirection {
        if (abs(averageGrade) < SEGMENT_DIRECTION_MIN_GRADE_PERCENT) return SegmentDirection.UNKNOWN
        return if (averageGrade > 0.0) SegmentDirection.ASCENT else SegmentDirection.DESCENT
    }

    private fun directionAwareTargetId(baseTargetId: Long, direction: SegmentDirection): Long {
        if (baseTargetId <= 0L) return baseTargetId
        return when (direction) {
            SegmentDirection.ASCENT -> -(baseTargetId * 10L + 1L)
            SegmentDirection.DESCENT -> -(baseTargetId * 10L + 2L)
            SegmentDirection.UNKNOWN -> baseTargetId
        }
    }

    private fun directionAwareTargetName(baseTargetName: String, direction: SegmentDirection): String =
        when (direction) {
            SegmentDirection.ASCENT ->
                if (baseTargetName.contains("(ascent)")) baseTargetName else "$baseTargetName (ascent)"
            SegmentDirection.DESCENT ->
                if (baseTargetName.contains("(descent)")) baseTargetName else "$baseTargetName (descent)"
            SegmentDirection.UNKNOWN -> baseTargetName
        }

    private fun splitAttemptsByDirection(attempts: List<SegmentAttemptRaw>): List<SegmentAttemptRaw> {
        if (attempts.isEmpty()) return emptyList()

        return attempts
            .groupBy { attempt -> attempt.targetId }
            .values
            .flatMap { groupedAttempts ->
                val baseTargetId = groupedAttempts.first().targetId
                if (baseTargetId <= 0L) return@flatMap groupedAttempts

                val hasAscent = groupedAttempts.any { attempt -> attempt.direction == SegmentDirection.ASCENT }
                val hasDescent = groupedAttempts.any { attempt -> attempt.direction == SegmentDirection.DESCENT }
                if (!hasAscent || !hasDescent) return@flatMap groupedAttempts

                groupedAttempts.map { attempt ->
                    val resolvedDirection = when (attempt.direction) {
                        SegmentDirection.ASCENT -> SegmentDirection.ASCENT
                        SegmentDirection.DESCENT -> SegmentDirection.DESCENT
                        SegmentDirection.UNKNOWN ->
                            resolveDirectionFromAverageGrade(attempt.averageGrade)
                                .takeIf { it != SegmentDirection.UNKNOWN }
                                ?: SegmentDirection.ASCENT
                    }
                    attempt.copy(
                        targetId = directionAwareTargetId(baseTargetId, resolvedDirection),
                        targetName = directionAwareTargetName(attempt.targetName, resolvedDirection),
                    )
                }
            }
    }

    // ---- Score and ranking helpers ----

    private fun buildAttempts(attempts: List<SegmentAttemptRaw>, metric: SegmentMetric): List<SegmentClimbAttempt> {
        val sortedAttempts = attempts.sortedBy { attempt -> attempt.activityDate }
        val personalRankByEffortId = attempts
            .sortedWith { left, right ->
                when (metric) {
                    SegmentMetric.TIME ->
                        if (left.elapsedTimeSeconds != right.elapsedTimeSeconds)
                            left.elapsedTimeSeconds.compareTo(right.elapsedTimeSeconds)
                        else left.activityDate.compareTo(right.activityDate)
                    SegmentMetric.SPEED ->
                        if (left.speedKph != right.speedKph)
                            right.speedKph.compareTo(left.speedKph)
                        else left.activityDate.compareTo(right.activityDate)
                }
            }
            .mapIndexedNotNull { index, attempt ->
                if (attempt.effortId <= 0L) null else attempt.effortId to (index + 1)
            }
            .toMap()

        val bestValue = when (metric) {
            SegmentMetric.TIME -> sortedAttempts.minOf { attempt -> attempt.elapsedTimeSeconds }.toDouble()
            SegmentMetric.SPEED -> sortedAttempts.maxOf { attempt -> attempt.speedKph }
        }

        var bestSoFar = when (metric) {
            SegmentMetric.TIME -> Double.POSITIVE_INFINITY
            SegmentMetric.SPEED -> Double.NEGATIVE_INFINITY
        }

        return sortedAttempts.map { attempt ->
            val currentMetricValue = when (metric) {
                SegmentMetric.TIME -> attempt.elapsedTimeSeconds.toDouble()
                SegmentMetric.SPEED -> attempt.speedKph
            }
            val setsNewPr = when (metric) {
                SegmentMetric.TIME -> currentMetricValue < bestSoFar
                SegmentMetric.SPEED -> currentMetricValue > bestSoFar
            }
            if (setsNewPr) bestSoFar = currentMetricValue

            val closeToPr = when (metric) {
                SegmentMetric.TIME -> !setsNewPr && currentMetricValue <= bestValue * 1.03
                SegmentMetric.SPEED -> !setsNewPr && currentMetricValue >= bestValue * 0.97
            }

            val deltaToPr = when (metric) {
                SegmentMetric.TIME -> {
                    val delta = (currentMetricValue - bestValue).roundToInt()
                    if (delta <= 0) "PR" else "+${delta.formatSeconds()}"
                }
                SegmentMetric.SPEED -> {
                    val delta = bestValue - currentMetricValue
                    if (delta <= 0.0) "PR"
                    else String.format(Locale.ENGLISH, "-%.1f%%", (delta / bestValue) * 100.0)
                }
            }

            SegmentClimbAttempt(
                targetId = attempt.targetId,
                targetName = attempt.targetName,
                targetType = attempt.targetType.name,
                activityDate = attempt.activityDate,
                elapsedTimeSeconds = attempt.elapsedTimeSeconds,
                movingTimeSeconds = attempt.movingTimeSeconds,
                speedKph = attempt.speedKph,
                distance = attempt.distance,
                averageGrade = attempt.averageGrade,
                elevationGain = (attempt.distance * attempt.averageGrade) / 100.0,
                averagePowerWatts = attempt.averagePowerWatts,
                averageHeartRate = attempt.averageHeartRate,
                prRank = attempt.prRank,
                personalRank = personalRankByEffortId[attempt.effortId],
                setsNewPr = setsNewPr,
                closeToPr = closeToPr,
                deltaToPr = deltaToPr,
                weatherSummary = null,
                activity = attempt.activity,
            )
        }
    }

    private fun buildTargetSummary(
        attempts: List<SegmentAttemptRaw>,
        metric: SegmentMetric,
    ): SegmentClimbTargetSummary {
        val progressionAttempts = buildAttempts(attempts, metric)
        val latestAttempt = progressionAttempts.last()
        val bestAttempt = when (metric) {
            SegmentMetric.TIME -> progressionAttempts.minBy { attempt -> attempt.elapsedTimeSeconds }
            SegmentMetric.SPEED -> progressionAttempts.maxBy { attempt -> attempt.speedKph }
        }
        val averageSpeedKph = progressionAttempts.map { attempt -> attempt.speedKph }.average()

        val consistency = when (metric) {
            SegmentMetric.TIME -> consistencyLabel(progressionAttempts.map { attempt -> attempt.elapsedTimeSeconds.toDouble() })
            SegmentMetric.SPEED -> consistencyLabel(progressionAttempts.map { attempt -> attempt.speedKph })
        }

        val recentTrend = when (metric) {
            SegmentMetric.TIME -> trendLabel(
                progressionAttempts.map { attempt -> attempt.elapsedTimeSeconds.toDouble() },
                lowerIsBetter = true,
                unit = "time",
            )
            SegmentMetric.SPEED -> trendLabel(
                progressionAttempts.map { attempt -> attempt.speedKph },
                lowerIsBetter = false,
                unit = "speed",
            )
        }

        val bestValue = when (metric) {
            SegmentMetric.TIME -> bestAttempt.elapsedTimeSeconds.formatSeconds()
            SegmentMetric.SPEED -> String.format(Locale.ENGLISH, "%.1f km/h", bestAttempt.speedKph)
        }
        val latestValue = when (metric) {
            SegmentMetric.TIME -> latestAttempt.elapsedTimeSeconds.formatSeconds()
            SegmentMetric.SPEED -> String.format(Locale.ENGLISH, "%.1f km/h", latestAttempt.speedKph)
        }

        return SegmentClimbTargetSummary(
            targetId = latestAttempt.targetId,
            targetName = latestAttempt.targetName,
            targetType = latestAttempt.targetType,
            climbCategory = attempts.first().climbCategory,
            distance = attempts.first().distance,
            averageGrade = attempts.first().averageGrade,
            attemptsCount = progressionAttempts.size,
            bestValue = bestValue,
            latestValue = latestValue,
            consistency = consistency,
            averagePacing = String.format(Locale.ENGLISH, "%.1f km/h", averageSpeedKph),
            closeToPrCount = progressionAttempts.count { attempt -> attempt.closeToPr },
            recentTrend = recentTrend,
        )
    }

    private fun rankTopEfforts(
        attempts: List<SegmentClimbAttempt>,
        metric: SegmentMetric,
        limit: Int,
    ): List<SegmentClimbAttempt> {
        if (attempts.isEmpty() || limit <= 0) return emptyList()
        return attempts
            .sortedWith { left, right ->
                when (metric) {
                    SegmentMetric.TIME ->
                        if (left.elapsedTimeSeconds != right.elapsedTimeSeconds)
                            left.elapsedTimeSeconds.compareTo(right.elapsedTimeSeconds)
                        else left.activityDate.compareTo(right.activityDate)
                    SegmentMetric.SPEED ->
                        if (left.speedKph != right.speedKph)
                            right.speedKph.compareTo(left.speedKph)
                        else left.activityDate.compareTo(right.activityDate)
                }
            }
            .take(limit)
    }

    private fun consistencyLabel(values: List<Double>): String {
        if (values.size < 3) return "-"
        val mean = values.average()
        if (mean == 0.0) return "-"
        val variance = values.map { value -> (value - mean) * (value - mean) }.average()
        val cv = sqrt(variance) / mean * 100.0
        return String.format(Locale.ENGLISH, "CV %.1f%%", cv)
    }

    private fun trendLabel(values: List<Double>, lowerIsBetter: Boolean, unit: String): String {
        if (values.size < 6) return "Not enough data"
        val recentAverage = values.takeLast(3).average()
        val previousAverage = values.dropLast(3).takeLast(3).average()
        if (previousAverage == 0.0) return "Stable"
        val ratio = (recentAverage - previousAverage) / previousAverage
        val isImproving = if (lowerIsBetter) ratio < 0 else ratio > 0
        val percentage = abs(ratio * 100.0)
        return when {
            percentage < 1.0 -> "Stable"
            isImproving -> String.format(Locale.ENGLISH, "Improving %.1f%% (%s)", percentage, unit)
            else -> String.format(Locale.ENGLISH, "Declining %.1f%% (%s)", percentage, unit)
        }
    }

    // ---- Parsing helpers ----

    private fun parseDateFilter(value: String?): LocalDate? {
        if (value.isNullOrBlank()) return null
        return runCatching { LocalDate.parse(value.trim()) }.getOrNull()
    }

    private fun matchesDateFilter(value: String, fromDate: LocalDate?, toDate: LocalDate?): Boolean {
        val day = extractSortableDay(value)?.let { LocalDate.parse(it) } ?: return false
        if (fromDate != null && day.isBefore(fromDate)) return false
        if (toDate != null && day.isAfter(toDate)) return false
        return true
    }

    private fun computeSpeedKph(distanceInMeters: Double, elapsedTimeSeconds: Int): Double {
        if (distanceInMeters <= 0.0 || elapsedTimeSeconds <= 0) return 0.0
        return (distanceInMeters / elapsedTimeSeconds.toDouble()) * 3.6
    }

    private fun parseSegmentMetric(metric: String?): SegmentMetric =
        SegmentMetric.entries.firstOrNull { candidate ->
            candidate.name.equals(metric, ignoreCase = true)
        } ?: SegmentMetric.TIME

    private fun parseSegmentTargetType(targetType: String?): SegmentTargetType =
        SegmentTargetType.entries.firstOrNull { candidate ->
            candidate.name.equals(targetType, ignoreCase = true)
        } ?: SegmentTargetType.ALL

    // ---- Internal data types ----

    private data class SegmentAttemptRaw(
        val effortId: Long,
        val targetId: Long,
        val targetName: String,
        val targetType: SegmentTargetType,
        val direction: SegmentDirection = SegmentDirection.UNKNOWN,
        val climbCategory: Int,
        val distance: Double,
        val averageGrade: Double,
        val elapsedTimeSeconds: Int,
        val movingTimeSeconds: Int,
        val speedKph: Double,
        val averagePowerWatts: Double,
        val averageHeartRate: Double,
        val activityDate: String,
        val prRank: Int?,
        val activity: ActivityShort,
    )

    private data class SegmentAttemptsCacheEntry(
        val createdAt: Instant,
        val expiresAt: Instant,
        val fallbackUsed: Boolean,
        val attempts: List<SegmentAttemptRaw>,
    )

    private enum class SegmentMetric { TIME, SPEED }

    private enum class SegmentTargetType { ALL, SEGMENT, CLIMB }

    private enum class SegmentDirection { UNKNOWN, ASCENT, DESCENT }
}

