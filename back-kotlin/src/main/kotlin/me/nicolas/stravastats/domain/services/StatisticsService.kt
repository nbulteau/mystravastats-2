package me.nicolas.stravastats.domain.services

import me.nicolas.stravastats.domain.business.ActivityEffort
import me.nicolas.stravastats.domain.business.ActivityShort
import me.nicolas.stravastats.domain.business.ActivityType
import me.nicolas.stravastats.domain.business.PersonalRecordTimelineEntry
import me.nicolas.stravastats.domain.business.SegmentClimbAttempt
import me.nicolas.stravastats.domain.business.SegmentClimbProgression
import me.nicolas.stravastats.domain.business.SegmentClimbTargetSummary
import me.nicolas.stravastats.domain.business.SegmentSummary
import me.nicolas.stravastats.domain.business.runActivities
import me.nicolas.stravastats.domain.business.strava.*
import me.nicolas.stravastats.domain.services.cache.SegmentAnalysisCacheEntryFile
import me.nicolas.stravastats.domain.services.cache.SegmentAnalysisCacheFile
import me.nicolas.stravastats.domain.services.cache.SegmentAnalysisCacheStore
import me.nicolas.stravastats.domain.services.cache.SegmentAttemptRawSnapshot
import me.nicolas.stravastats.domain.services.activityproviders.IActivityProvider
import me.nicolas.stravastats.domain.services.statistics.*
import me.nicolas.stravastats.domain.utils.dateFormatter
import me.nicolas.stravastats.domain.utils.formatSeconds
import me.nicolas.stravastats.domain.utils.formatSpeed
import org.slf4j.LoggerFactory
import org.springframework.stereotype.Service
import java.time.Instant
import java.time.LocalDate
import java.time.LocalDateTime
import java.time.OffsetDateTime
import java.time.ZoneOffset
import java.util.Locale
import kotlin.math.abs
import kotlin.math.roundToInt
import kotlin.math.sqrt


interface IStatisticsService {
    fun getStatistics(activityTypes: Set<ActivityType>, year: Int?): List<Statistic>
    fun getPersonalRecordsTimeline(activityTypes: Set<ActivityType>, year: Int?, metric: String?): List<PersonalRecordTimelineEntry>
    fun getSegmentClimbProgression(
        activityTypes: Set<ActivityType>,
        year: Int?,
        metric: String?,
        targetType: String?,
        targetId: Long?
    ): SegmentClimbProgression
    fun listSegments(
        activityTypes: Set<ActivityType>,
        year: Int?,
        metric: String?,
        query: String?,
        from: String?,
        to: String?,
    ): List<SegmentClimbTargetSummary>
    fun getSegmentEfforts(
        activityTypes: Set<ActivityType>,
        year: Int?,
        metric: String?,
        segmentId: Long,
        from: String?,
        to: String?,
    ): List<SegmentClimbAttempt>
    fun getSegmentSummary(
        activityTypes: Set<ActivityType>,
        year: Int?,
        metric: String?,
        segmentId: Long,
        from: String?,
        to: String?,
    ): SegmentSummary?
}

@Service
internal class StatisticsService(
    activityProvider: IActivityProvider,
) : IStatisticsService, AbstractStravaService(activityProvider) {

    companion object {
        private const val SEGMENT_CACHE_MAX_ENTRIES = 256
        private const val SEGMENT_CACHE_TTL_SECONDS = 30 * 60L
        private const val SEGMENT_FALLBACK_CACHE_TTL_SECONDS = 45L
        private const val SEGMENT_ANALYSIS_ALGO_VERSION = "direction-v2"
        private const val SEGMENT_DIRECTION_MIN_ALTITUDE_DELTA_M = 3.0
        private const val SEGMENT_DIRECTION_MIN_GRADE_PERCENT = 0.5
    }

    private val logger = LoggerFactory.getLogger(StatisticsService::class.java)
    private val segmentCacheIdentity by lazy(LazyThreadSafetyMode.NONE) {
        runCatching { activityProvider.cacheIdentity() }.getOrNull()
    }
    private val segmentCacheLock = Any()
    @Volatile
    private var segmentCacheLoaded = false
    private val segmentAttemptsCache = mutableMapOf<String, SegmentAttemptsCacheEntry>()

    override fun getStatistics(activityTypes: Set<ActivityType>, year: Int?): List<Statistic> {
        logger.info("Compute $activityTypes statistics for ${year ?: "all years"}")

        val filteredActivities = activityProvider.getActivitiesByActivityTypeAndYear(activityTypes, year)

        return when (resolvePrimaryActivityType(activityTypes)) {
            ActivityType.Run -> computeRunStatistics(filteredActivities)
            ActivityType.InlineSkate -> computeInlineSkateStatistics(filteredActivities)
            ActivityType.Hike -> computeHikeStatistics(filteredActivities)
            ActivityType.AlpineSki -> computeAlpineSkiStatistics(filteredActivities)
            else -> computeRideStatistics(filteredActivities)
        }
    }

    override fun getPersonalRecordsTimeline(
        activityTypes: Set<ActivityType>,
        year: Int?,
        metric: String?
    ): List<PersonalRecordTimelineEntry> {
        logger.info("Compute personal records timeline for $activityTypes in ${year ?: "all years"}")

        val filteredActivities = activityProvider.getActivitiesByActivityTypeAndYear(activityTypes, year)
            .sortedWith(
                compareBy<StravaActivity> { activity ->
                    extractSortableDay(activity.startDateLocal)
                        ?: extractSortableDay(activity.startDate)
                        ?: "9999-12-31"
                }.thenBy { activity ->
                    parseActivityDateEpochMillis(activity.startDateLocal)
                        ?: parseActivityDateEpochMillis(activity.startDate)
                        ?: Long.MAX_VALUE
                }.thenBy { activity ->
                    activity.startDateLocal.ifBlank { activity.startDate }
                }
            )

        if (filteredActivities.isEmpty()) {
            return emptyList()
        }

        val selectedMetrics = getPersonalRecordMetricDefinitions(activityTypes)
            .filter { definition -> metric.isNullOrBlank() || definition.key == metric }

        val timeline = mutableListOf<PersonalRecordTimelineEntry>()

        selectedMetrics.forEach { definition ->
            var bestEffort: ActivityEffort? = null

            filteredActivities.forEach { activity ->
                val effort = definition.effortExtractor(activity) ?: return@forEach
                if (effort.activityShort.id != activity.id) {
                    return@forEach
                }

                val previousBest = bestEffort
                if (previousBest == null || definition.isBetter(definition.score(effort), definition.score(previousBest))) {
                    timeline += PersonalRecordTimelineEntry(
                        metricKey = definition.key,
                        metricLabel = definition.label,
                        activityDate = activity.startDateLocal,
                        value = definition.valueFormatter(effort),
                        previousValue = previousBest?.let(definition.valueFormatter),
                        improvement = previousBest?.let { previous -> definition.improvementFormatter(previous, effort) },
                        activity = effort.activityShort
                    )
                    bestEffort = effort
                }
            }
        }

        return timeline.sortedWith(
            compareBy<PersonalRecordTimelineEntry> { entry ->
                extractSortableDay(entry.activityDate) ?: "9999-12-31"
            }.thenBy { entry ->
                parseActivityDateEpochMillis(entry.activityDate) ?: Long.MAX_VALUE
            }.thenBy { entry ->
                entry.activityDate
            }
        )
    }

    private fun extractSortableDay(value: String?): String? {
        if (value.isNullOrBlank()) {
            return null
        }
        val normalized = value.trim()
        if (normalized.length < 10) {
            return null
        }
        val day = normalized.substring(0, 10)
        return runCatching { LocalDate.parse(day).toString() }.getOrNull()
    }

    private fun parseActivityDateEpochMillis(value: String?): Long? {
        if (value.isNullOrBlank()) {
            return null
        }

        return runCatching { OffsetDateTime.parse(value).toInstant().toEpochMilli() }
            .recoverCatching { Instant.parse(value).toEpochMilli() }
            .recoverCatching { LocalDateTime.parse(value).toInstant(ZoneOffset.UTC).toEpochMilli() }
            .getOrNull()
    }

    override fun getSegmentClimbProgression(
        activityTypes: Set<ActivityType>,
        year: Int?,
        metric: String?,
        targetType: String?,
        targetId: Long?
    ): SegmentClimbProgression {
        val resolvedMetric = parseSegmentMetric(metric)
        val resolvedTargetTypeFilter = parseSegmentTargetType(targetType)
        logger.info(
            "Compute segment/climb progression for {} in {} with metric={} targetType={} targetId={}",
            activityTypes,
            year ?: "all years",
            resolvedMetric,
            resolvedTargetTypeFilter,
            targetId
        )

        val filteredActivities = activityProvider.getActivitiesByActivityTypeAndYear(activityTypes, year)
            .sortedBy { activity -> activity.startDateLocal }

        val rawAttempts = filteredActivities.flatMap { activity ->
            val detailedActivity = activityProvider.getCachedDetailedActivity(activity.id)
                .orElseGet { activityProvider.getDetailedActivity(activity.id).orElse(null) }
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
                        activity = ActivityShort(activity.id, activity.name, activity.type)
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
                attempts = emptyList()
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
            attempts = selectedAttempts
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
        if (rawAttempts.isEmpty()) {
            return emptyList()
        }

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
        if (attempts.isEmpty()) {
            return emptyList()
        }
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
        if (attempts.isEmpty()) {
            return null
        }

        val progressionAttempts = buildAttempts(attempts, resolvedMetric)
        val topEfforts = rankTopEfforts(progressionAttempts, resolvedMetric, 3)
        return SegmentSummary(
            metric = resolvedMetric.name,
            segment = buildTargetSummary(attempts, resolvedMetric),
            personalRecord = topEfforts.firstOrNull(),
            topEfforts = topEfforts,
        )
    }

    private fun computeRunStatistics(runActivities: List<StravaActivity>): List<Statistic> {

        val statistics = computeCommonStats(runActivities).toMutableList()

        statistics.addAll(
            listOf(
                CooperStatistic(runActivities),
                VO2maxStatistic(runActivities),
                BestEffortDistanceStatistic("Best 200 m", runActivities, 200.0),
                BestEffortDistanceStatistic("Best 400 m", runActivities, 400.0),
                BestEffortDistanceStatistic("Best 1000 m", runActivities, 1000.0),
                BestEffortDistanceStatistic("Best 5000 m", runActivities, 5000.0),
                BestEffortDistanceStatistic("Best 10000 m", runActivities, 10000.0),
                BestEffortDistanceStatistic("Best half Marathon", runActivities, 21097.0),
                BestEffortDistanceStatistic("Best Marathon", runActivities, 42195.0),
                BestEffortTimeStatistic("Best 1 h", runActivities, 60 * 60),
                BestEffortTimeStatistic("Best 2 h", runActivities, 2 * 60 * 60),
                BestEffortTimeStatistic("Best 3 h", runActivities, 3 * 60 * 60),
                BestEffortTimeStatistic("Best 4 h", runActivities, 4 * 60 * 60),
                BestEffortTimeStatistic("Best 5 h", runActivities, 5 * 60 * 60),
                BestEffortTimeStatistic("Best 6 h", runActivities, 6 * 60 * 60),
            )
        )

        return statistics
    }

    private fun computeRideStatistics(rideActivities: List<StravaActivity>): List<Statistic> {

        val statistics = computeCommonStats(rideActivities).toMutableList()
        statistics.addAll(
            listOf(
                MaxSpeedStatistic(rideActivities),
                MaxMovingTimeStatistic(rideActivities),
                BestEffortDistanceStatistic("Best 250 m", rideActivities, 250.0),
                BestEffortDistanceStatistic("Best 500 m", rideActivities, 500.0),
                BestEffortDistanceStatistic("Best 1000 m", rideActivities, 1000.0),
                BestEffortDistanceStatistic("Best 5 km", rideActivities, 5000.0),
                BestEffortDistanceStatistic("Best 10 km", rideActivities, 10000.0),
                BestEffortDistanceStatistic("Best 20 km", rideActivities, 20000.0),
                BestEffortDistanceStatistic("Best 50 km", rideActivities, 50000.0),
                BestEffortDistanceStatistic("Best 100 km", rideActivities, 100000.0),
                BestEffortTimeStatistic("Best 30 min", rideActivities, 30 * 60),
                BestEffortTimeStatistic("Best 1 h", rideActivities, 60 * 60),
                BestEffortTimeStatistic("Best 2 h", rideActivities, 2 * 60 * 60),
                BestEffortTimeStatistic("Best 3 h", rideActivities, 3 * 60 * 60),
                BestEffortTimeStatistic("Best 4 h", rideActivities, 4 * 60 * 60),
                BestEffortTimeStatistic("Best 5 h", rideActivities, 5 * 60 * 60),
                BestElevationDistanceStatistic("Max gradient for 250 m", rideActivities, 250.0),
                BestElevationDistanceStatistic("Max gradient for 500 m", rideActivities, 500.0),
                BestElevationDistanceStatistic("Max gradient for 1000 m", rideActivities, 1000.0),
                BestElevationDistanceStatistic("Max gradient for 5 km", rideActivities, 5000.0),
                BestElevationDistanceStatistic("Max gradient for 10 km", rideActivities, 10000.0),
                BestElevationDistanceStatistic("Max gradient for 20 km", rideActivities, 20000.0),
                BestEffortPowerStatistic("Best average power for 20 min", rideActivities, 20 * 60),
                BestEffortPowerStatistic("Best average power for 1 h", rideActivities, 60 * 60),
            )
        )
        return statistics
    }

    private fun computeAlpineSkiStatistics(filteredActivities: List<StravaActivity>): List<Statistic> {
        val statistics = computeCommonStats(filteredActivities).toMutableList()
        statistics.addAll(
            listOf(
                MaxSpeedStatistic(filteredActivities),
                MaxMovingTimeStatistic(filteredActivities),
                BestEffortDistanceStatistic("Best 250 m", filteredActivities, 250.0),
                BestEffortDistanceStatistic("Best 500 m", filteredActivities, 500.0),
                BestEffortDistanceStatistic("Best 1000 m", filteredActivities, 1000.0),
                BestEffortDistanceStatistic("Best 5 km", filteredActivities, 5000.0),
                BestEffortDistanceStatistic("Best 10 km", filteredActivities, 10000.0),
                BestEffortDistanceStatistic("Best 20 km", filteredActivities, 20000.0),
                BestEffortDistanceStatistic("Best 50 km", filteredActivities, 50000.0),
                BestEffortDistanceStatistic("Best 100 km", filteredActivities, 100000.0),
                BestEffortTimeStatistic("Best 30 min", filteredActivities, 30 * 60),
                BestEffortTimeStatistic("Best 1 h", filteredActivities, 60 * 60),
                BestEffortTimeStatistic("Best 2 h", filteredActivities, 2 * 60 * 60),
                BestEffortTimeStatistic("Best 3 h", filteredActivities, 3 * 60 * 60),
                BestEffortTimeStatistic("Best 4 h", filteredActivities, 4 * 60 * 60),
                BestEffortTimeStatistic("Best 5 h", filteredActivities, 5 * 60 * 60),
            )
        )
        return statistics
    }

    private fun computeHikeStatistics(hikeActivities: List<StravaActivity>): List<Statistic> {

        val statistics = computeCommonStats(hikeActivities).toMutableList()

        statistics.addAll(
            listOf(
                BestDayStatistic(
                "Max distance in a day", hikeActivities, formatString = "%s => %.02f km"
            ) {
                hikeActivities.groupBy { stravaActivity: StravaActivity ->
                    stravaActivity.startDateLocal.substringBefore(
                        'T'
                    )
                }.mapValues { it.value.sumOf { activity -> activity.distance / 1000 } }
                    .maxByOrNull { entry: Map.Entry<String, Double> -> entry.value }?.toPair()
            }, BestDayStatistic("Max elevation in a day", hikeActivities, formatString = "%s => %.02f m") {
                hikeActivities.groupBy { stravaActivity: StravaActivity ->
                    stravaActivity.startDateLocal.substringBefore(
                        'T'
                    )
                }.mapValues { it.value.sumOf { activity -> activity.totalElevationGain } }
                    .maxByOrNull { entry: Map.Entry<String, Double> -> entry.value }?.toPair()
            })
        )

        return statistics
    }

    private fun computeInlineSkateStatistics(inlineSkateActivities: List<StravaActivity>): List<Statistic> {

        val statistics = computeCommonStats(inlineSkateActivities).toMutableList()

        statistics.addAll(
            listOf(
                BestEffortDistanceStatistic("Best 200 m", inlineSkateActivities, 200.0),
                BestEffortDistanceStatistic("Best 400 m", inlineSkateActivities, 400.0),
                BestEffortDistanceStatistic("Best 1000 m", inlineSkateActivities, 1000.0),
                BestEffortDistanceStatistic("Best 10000 m", inlineSkateActivities, 10000.0),
                BestEffortDistanceStatistic("Best half Marathon", inlineSkateActivities, 21097.0),
                BestEffortDistanceStatistic("Best Marathon", inlineSkateActivities, 42195.0),
                BestEffortTimeStatistic("Best 1 h", inlineSkateActivities, 60 * 60),
                BestEffortTimeStatistic("Best 2 h", inlineSkateActivities, 2 * 60 * 60),
                BestEffortTimeStatistic("Best 3 h", inlineSkateActivities, 3 * 60 * 60),
                BestEffortTimeStatistic("Best 4 h", inlineSkateActivities, 4 * 60 * 60),
            )
        )

        return statistics
    }

    private fun computeCommonStats(activities: List<StravaActivity>): List<Statistic> {

        return listOf(
            GlobalStatistic("Nb activities", activities, { number -> "%d".format(number) }, List<StravaActivity>::size),

            GlobalStatistic("Nb actives days", activities, { number -> "%d".format(number) }) {
                activities.groupBy { stravaActivity: StravaActivity -> stravaActivity.startDateLocal.substringBefore('T') }
                    .count()
            },
            MaxStreakStatistic(activities),
            GlobalStatistic("Total distance", activities, { number -> "%.2f km".format(number) }, {
                activities.sumOf { stravaActivity: StravaActivity -> stravaActivity.distance } / 1000
            }),
            GlobalStatistic("Elapsed time", activities, { number -> number.toInt().formatSeconds() }, {
                activities.sumOf { stravaActivity: StravaActivity -> stravaActivity.elapsedTime }
            }),
            GlobalStatistic("Total elevation", activities, { number -> "%.2f m".format(number) }, {
                activities.sumOf { stravaActivity: StravaActivity -> stravaActivity.totalElevationGain }
            }),
            GlobalStatistic("Km by activity", activities, { number -> "%.2f km".format(number) }, {
                if (activities.isEmpty()) 0.0
                else activities.sumOf { stravaActivity: StravaActivity -> stravaActivity.distance }
                    .div(activities.size) / 1000
            }),

            AverageSpeedStatistic(activities),
            MaxDistanceStatistic(activities),
            MaxDistanceInADayStatistic(activities),
            MaxElevationStatistic(activities),
            MaxElevationInADayStatistic(activities),
            HighestPointStatistic(activities),
            MaxMovingTimeStatistic(activities),
            MostActiveMonthStatistic(activities),
            EddingtonStatistic(activities),
        )
    }

    private fun isFavoriteEffort(effort: StravaSegmentEffort): Boolean {
        return effort.segment.starred || effort.segment.climbCategory > 0 || (effort.prRank != null && effort.prRank <= 3)
    }

    private fun isCandidateEffort(effort: StravaSegmentEffort): Boolean {
        if (effort.segment.id == 0L) {
            return false
        }
        if (effort.segment.name.isBlank()) {
            return false
        }
        return effort.elapsedTime > 0 && effort.distance > 0.0
    }

    private fun collectSegmentAttempts(
        activityTypes: Set<ActivityType>,
        year: Int?,
        from: String?,
        to: String?,
    ): List<SegmentAttemptRaw> {
        val fromDate = parseDateFilter(from)
        val toDate = parseDateFilter(to)

        val filteredActivities = activityProvider.getActivitiesByActivityTypeAndYear(activityTypes, year)
            .sortedBy { activity -> activity.startDateLocal }

        val cacheKey = buildSegmentAttemptsCacheKey(
            activityTypes = activityTypes,
            year = year,
            from = from,
            to = to,
            activitySignature = computeSegmentActivitiesSignature(filteredActivities),
        )
        getSegmentAttemptsFromCache(cacheKey)?.let { cachedAttempts ->
            return cachedAttempts
        }

        val segmentEffortAttempts = filteredActivities.flatMap { activity ->
            val detailedActivity = activityProvider.getCachedDetailedActivity(activity.id)
                .orElseGet { activityProvider.getDetailedActivity(activity.id).orElse(null) }
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
                        activity = ActivityShort(activity.id, activity.name, activity.type)
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
                val day = extractSortableDay(activity.startDateLocal)?.let { LocalDate.parse(it) } ?: return@filter false
                if (fromDate != null && day.isBefore(fromDate)) {
                    return@filter false
                }
                if (toDate != null && day.isAfter(toDate)) {
                    return@filter false
                }
                activity.name.isNotBlank()
            }
            .groupBy { activity -> activity.name.trim().lowercase(Locale.getDefault()) }
            .filterValues { grouped -> grouped.size >= 2 }

        return groupedByName.flatMap { (normalizedName, groupedActivities) ->
            val displayName = groupedActivities.firstOrNull()?.name?.trim().orEmpty()
            val targetId = routeNameBasedTargetId(normalizedName)

            groupedActivities.mapNotNull { activity ->
                val elapsedSeconds = activity.elapsedTime
                if (elapsedSeconds <= 0) {
                    return@mapNotNull null
                }
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
        return -nameKey.hashCode().toLong().let { hash -> kotlin.math.abs(hash) + 1L }
    }

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
        if (attempts.isEmpty()) {
            return
        }

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
        if (segmentCacheLoaded) {
            return
        }

        synchronized(segmentCacheLock) {
            if (segmentCacheLoaded) {
                return
            }

            segmentCacheLoaded = true
            val identity = segmentCacheIdentity ?: return
            val payload = SegmentAnalysisCacheStore.load(identity.cacheRoot, identity.athleteId) ?: return

            val now = Instant.now()
            payload.entries.forEach { entry ->
                val createdAt = runCatching { Instant.parse(entry.createdAt) }.getOrNull() ?: now
                val expiresAt = runCatching { Instant.parse(entry.expiresAt) }.getOrNull() ?: return@forEach
                if (!expiresAt.isAfter(now)) {
                    return@forEach
                }
                val attempts = entry.attempts.mapNotNull { snapshot -> snapshot.toRawOrNull() }
                if (attempts.isEmpty()) {
                    return@forEach
                }
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

        if (segmentAttemptsCache.size <= SEGMENT_CACHE_MAX_ENTRIES) {
            return
        }

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

    private fun parseDateFilter(value: String?): LocalDate? {
        if (value.isNullOrBlank()) {
            return null
        }
        return runCatching { LocalDate.parse(value.trim()) }.getOrNull()
    }

    private fun matchesDateFilter(value: String, fromDate: LocalDate?, toDate: LocalDate?): Boolean {
        val day = extractSortableDay(value)?.let { dayValue -> LocalDate.parse(dayValue) } ?: return false
        if (fromDate != null && day.isBefore(fromDate)) {
            return false
        }
        if (toDate != null && day.isAfter(toDate)) {
            return false
        }
        return true
    }

    private fun computeSpeedKph(distanceInMeters: Double, elapsedTimeSeconds: Int): Double {
        if (distanceInMeters <= 0.0 || elapsedTimeSeconds <= 0) {
            return 0.0
        }
        val metersPerSecond = distanceInMeters / elapsedTimeSeconds.toDouble()
        return metersPerSecond * 3.6
    }

    private fun splitAttemptsByDirection(attempts: List<SegmentAttemptRaw>): List<SegmentAttemptRaw> {
        if (attempts.isEmpty()) {
            return emptyList()
        }

        return attempts
            .groupBy { attempt -> attempt.targetId }
            .values
            .flatMap { groupedAttempts ->
                val baseTargetId = groupedAttempts.first().targetId
                if (baseTargetId <= 0L) {
                    return@flatMap groupedAttempts
                }

                val hasAscent = groupedAttempts.any { attempt -> attempt.direction == SegmentDirection.ASCENT }
                val hasDescent = groupedAttempts.any { attempt -> attempt.direction == SegmentDirection.DESCENT }
                if (!hasAscent || !hasDescent) {
                    return@flatMap groupedAttempts
                }

                groupedAttempts.map { attempt ->
                    val resolvedDirection = when (attempt.direction) {
                        SegmentDirection.ASCENT -> SegmentDirection.ASCENT
                        SegmentDirection.DESCENT -> SegmentDirection.DESCENT
                        SegmentDirection.UNKNOWN ->
                            resolveDirectionFromAverageGrade(attempt.averageGrade).takeIf { it != SegmentDirection.UNKNOWN }
                                ?: SegmentDirection.ASCENT
                    }
                    attempt.copy(
                        targetId = directionAwareTargetId(baseTargetId, resolvedDirection),
                        targetName = directionAwareTargetName(attempt.targetName, resolvedDirection),
                    )
                }
            }
    }

    private fun resolveSegmentDirection(
        effort: StravaSegmentEffort,
        activity: StravaActivity,
        detailedActivity: StravaDetailedActivity,
    ): SegmentDirection {
        resolveDirectionFromAltitudeStream(detailedActivity, effort)?.let { direction ->
            return direction
        }

        resolveDirectionFromLabels(activity.name, effort.name, effort.segment.name)?.let { direction ->
            return direction
        }

        return resolveDirectionFromAverageGrade(effort.segment.averageGrade)
    }

    private fun resolveDirectionFromAltitudeStream(
        detailedActivity: StravaDetailedActivity,
        effort: StravaSegmentEffort,
    ): SegmentDirection? {
        val altitudeData = detailedActivity.stream?.altitude?.data ?: return null
        if (altitudeData.isEmpty()) {
            return null
        }
        if (effort.startIndex < 0 || effort.endIndex < 0) {
            return null
        }
        if (effort.startIndex >= altitudeData.size || effort.endIndex >= altitudeData.size) {
            return null
        }
        if (effort.startIndex == effort.endIndex) {
            return null
        }

        val altitudeDelta = altitudeData[effort.endIndex] - altitudeData[effort.startIndex]
        if (abs(altitudeDelta) < SEGMENT_DIRECTION_MIN_ALTITUDE_DELTA_M) {
            return null
        }
        return if (altitudeDelta > 0.0) SegmentDirection.ASCENT else SegmentDirection.DESCENT
    }

    private fun resolveDirectionFromLabels(vararg labels: String): SegmentDirection? {
        val ascentKeywords = listOf("montee", "ascent", "climb", "uphill")
        val descentKeywords = listOf("descente", "descent", "downhill")

        labels.forEach { label ->
            val normalized = normalizeDirectionLabel(label)
            if (normalized.isBlank()) {
                return@forEach
            }
            if (descentKeywords.any { keyword -> normalized.contains(keyword) }) {
                return SegmentDirection.DESCENT
            }
            if (ascentKeywords.any { keyword -> normalized.contains(keyword) }) {
                return SegmentDirection.ASCENT
            }
        }

        return null
    }

    private fun normalizeDirectionLabel(label: String): String {
        if (label.isBlank()) {
            return ""
        }
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
            .replace("’", "'")
            .replace("-", " ")
            .split(Regex("\\s+"))
            .filter { token -> token.isNotBlank() }
            .joinToString(" ")
    }

    private fun resolveDirectionFromAverageGrade(averageGrade: Double): SegmentDirection {
        if (abs(averageGrade) < SEGMENT_DIRECTION_MIN_GRADE_PERCENT) {
            return SegmentDirection.UNKNOWN
        }
        return if (averageGrade > 0.0) SegmentDirection.ASCENT else SegmentDirection.DESCENT
    }

    private fun directionAwareTargetId(baseTargetId: Long, direction: SegmentDirection): Long {
        if (baseTargetId <= 0L) {
            return baseTargetId
        }
        return when (direction) {
            SegmentDirection.ASCENT -> -(baseTargetId * 10L + 1L)
            SegmentDirection.DESCENT -> -(baseTargetId * 10L + 2L)
            SegmentDirection.UNKNOWN -> baseTargetId
        }
    }

    private fun directionAwareTargetName(baseTargetName: String, direction: SegmentDirection): String {
        return when (direction) {
            SegmentDirection.ASCENT -> {
                if (baseTargetName.contains("(ascent)")) baseTargetName else "$baseTargetName (ascent)"
            }

            SegmentDirection.DESCENT -> {
                if (baseTargetName.contains("(descent)")) baseTargetName else "$baseTargetName (descent)"
            }

            SegmentDirection.UNKNOWN -> baseTargetName
        }
    }

    private fun buildAttempts(attempts: List<SegmentAttemptRaw>, metric: SegmentMetric): List<SegmentClimbAttempt> {
        val sortedAttempts = attempts.sortedBy { attempt -> attempt.activityDate }
        val personalRankByEffortId = attempts
            .sortedWith { left, right ->
                when (metric) {
                    SegmentMetric.TIME -> {
                        if (left.elapsedTimeSeconds != right.elapsedTimeSeconds) {
                            left.elapsedTimeSeconds.compareTo(right.elapsedTimeSeconds)
                        } else {
                            left.activityDate.compareTo(right.activityDate)
                        }
                    }
                    SegmentMetric.SPEED -> {
                        if (left.speedKph != right.speedKph) {
                            right.speedKph.compareTo(left.speedKph)
                        } else {
                            left.activityDate.compareTo(right.activityDate)
                        }
                    }
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
            if (setsNewPr) {
                bestSoFar = currentMetricValue
            }

            val closeToPr = when (metric) {
                SegmentMetric.TIME -> !setsNewPr && currentMetricValue <= bestValue * 1.03
                SegmentMetric.SPEED -> !setsNewPr && currentMetricValue >= bestValue * 0.97
            }

            val deltaToPr = when (metric) {
                SegmentMetric.TIME -> {
                    val delta = (currentMetricValue - bestValue).roundToInt()
                    if (delta <= 0) "PR"
                    else "+${delta.formatSeconds()}"
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
                activity = attempt.activity
            )
        }
    }

    private fun buildTargetSummary(attempts: List<SegmentAttemptRaw>, metric: SegmentMetric): SegmentClimbTargetSummary {
        val progressionAttempts = buildAttempts(attempts, metric)
        val latestAttempt = progressionAttempts.last()
        val bestAttempt = when (metric) {
            SegmentMetric.TIME -> progressionAttempts.minBy { attempt -> attempt.elapsedTimeSeconds }
            SegmentMetric.SPEED -> progressionAttempts.maxBy { attempt -> attempt.speedKph }
        }
        val averageSpeedKph = progressionAttempts.map { attempt -> attempt.speedKph }.average()

        val consistency = when (metric) {
            SegmentMetric.TIME -> {
                val values = progressionAttempts.map { attempt -> attempt.elapsedTimeSeconds.toDouble() }
                consistencyLabel(values)
            }

            SegmentMetric.SPEED -> {
                val values = progressionAttempts.map { attempt -> attempt.speedKph }
                consistencyLabel(values)
            }
        }

        val recentTrend = when (metric) {
            SegmentMetric.TIME -> {
                val values = progressionAttempts.map { attempt -> attempt.elapsedTimeSeconds.toDouble() }
                trendLabel(values, lowerIsBetter = true, unit = "time")
            }

            SegmentMetric.SPEED -> {
                val values = progressionAttempts.map { attempt -> attempt.speedKph }
                trendLabel(values, lowerIsBetter = false, unit = "speed")
            }
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
            recentTrend = recentTrend
        )
    }

    private fun rankTopEfforts(
        attempts: List<SegmentClimbAttempt>,
        metric: SegmentMetric,
        limit: Int,
    ): List<SegmentClimbAttempt> {
        if (attempts.isEmpty() || limit <= 0) {
            return emptyList()
        }
        val ranked = attempts.sortedWith { left, right ->
            when (metric) {
                SegmentMetric.TIME -> {
                    if (left.elapsedTimeSeconds != right.elapsedTimeSeconds) {
                        left.elapsedTimeSeconds.compareTo(right.elapsedTimeSeconds)
                    } else {
                        left.activityDate.compareTo(right.activityDate)
                    }
                }
                SegmentMetric.SPEED -> {
                    if (left.speedKph != right.speedKph) {
                        right.speedKph.compareTo(left.speedKph)
                    } else {
                        left.activityDate.compareTo(right.activityDate)
                    }
                }
            }
        }
        return ranked.take(limit)
    }

    private fun consistencyLabel(values: List<Double>): String {
        if (values.size < 3) {
            return "-"
        }
        val mean = values.average()
        if (mean == 0.0) {
            return "-"
        }
        val variance = values.map { value -> (value - mean) * (value - mean) }.average()
        val coefficientOfVariation = sqrt(variance) / mean * 100.0
        return String.format(Locale.ENGLISH, "CV %.1f%%", coefficientOfVariation)
    }

    private fun trendLabel(values: List<Double>, lowerIsBetter: Boolean, unit: String): String {
        if (values.size < 6) {
            return "Not enough data"
        }
        val recentAverage = values.takeLast(3).average()
        val previousAverage = values.dropLast(3).takeLast(3).average()
        if (previousAverage == 0.0) {
            return "Stable"
        }
        val ratio = (recentAverage - previousAverage) / previousAverage
        val isImproving = if (lowerIsBetter) ratio < 0 else ratio > 0
        val percentage = kotlin.math.abs(ratio * 100.0)
        return when {
            percentage < 1.0 -> "Stable"
            isImproving -> String.format(Locale.ENGLISH, "Improving %.1f%% (%s)", percentage, unit)
            else -> String.format(Locale.ENGLISH, "Declining %.1f%% (%s)", percentage, unit)
        }
    }

    private fun parseSegmentMetric(metric: String?): SegmentMetric {
        return SegmentMetric.entries.firstOrNull { candidate ->
            candidate.name.equals(metric, ignoreCase = true)
        } ?: SegmentMetric.TIME
    }

    private fun parseSegmentTargetType(targetType: String?): SegmentTargetType {
        return SegmentTargetType.entries.firstOrNull { candidate ->
            candidate.name.equals(targetType, ignoreCase = true)
        } ?: SegmentTargetType.ALL
    }

    private fun resolvePrimaryActivityType(activityTypes: Set<ActivityType>): ActivityType {
        return when {
            activityTypes.any { type -> type in runActivities } -> ActivityType.Run
            activityTypes.contains(ActivityType.InlineSkate) -> ActivityType.InlineSkate
            activityTypes.contains(ActivityType.Hike) -> ActivityType.Hike
            activityTypes.contains(ActivityType.AlpineSki) -> ActivityType.AlpineSki
            else -> ActivityType.Ride
        }
    }

    private data class PersonalRecordMetricDefinition(
        val key: String,
        val label: String,
        val effortExtractor: (StravaActivity) -> ActivityEffort?,
        val score: (ActivityEffort) -> Double,
        val isBetter: (Double, Double) -> Boolean,
        val valueFormatter: (ActivityEffort) -> String,
        val improvementFormatter: (ActivityEffort, ActivityEffort) -> String,
    )

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

    private enum class SegmentMetric {
        TIME,
        SPEED
    }

    private enum class SegmentTargetType {
        ALL,
        SEGMENT,
        CLIMB
    }

    private enum class SegmentDirection {
        UNKNOWN,
        ASCENT,
        DESCENT,
    }

    private fun getPersonalRecordMetricDefinitions(activityTypes: Set<ActivityType>): List<PersonalRecordMetricDefinition> {
        return when (resolvePrimaryActivityType(activityTypes)) {
            ActivityType.Run -> buildRunMetricDefinitions()
            ActivityType.InlineSkate -> buildInlineSkateMetricDefinitions()
            ActivityType.AlpineSki -> buildAlpineSkiMetricDefinitions()
            ActivityType.Hike -> buildActivityRecordMetricDefinitions()
            else -> buildRideMetricDefinitions()
        }
    }

    private fun buildRunMetricDefinitions(): List<PersonalRecordMetricDefinition> {
        return listOf(
            bestTimeForDistanceMetric("best-time-200m", "Best 200 m", 200.0),
            bestTimeForDistanceMetric("best-time-400m", "Best 400 m", 400.0),
            bestTimeForDistanceMetric("best-time-1000m", "Best 1000 m", 1000.0),
            bestTimeForDistanceMetric("best-time-5000m", "Best 5000 m", 5000.0),
            bestTimeForDistanceMetric("best-time-10000m", "Best 10000 m", 10000.0),
            bestTimeForDistanceMetric("best-time-half-marathon", "Best half Marathon", 21097.0),
            bestTimeForDistanceMetric("best-time-marathon", "Best Marathon", 42195.0),
            bestDistanceForTimeMetric("best-distance-1h", "Best 1 h", 60 * 60),
            bestDistanceForTimeMetric("best-distance-2h", "Best 2 h", 2 * 60 * 60),
            bestDistanceForTimeMetric("best-distance-3h", "Best 3 h", 3 * 60 * 60),
            bestDistanceForTimeMetric("best-distance-4h", "Best 4 h", 4 * 60 * 60),
            bestDistanceForTimeMetric("best-distance-5h", "Best 5 h", 5 * 60 * 60),
            bestDistanceForTimeMetric("best-distance-6h", "Best 6 h", 6 * 60 * 60),
        ) + buildActivityRecordMetricDefinitions()
    }

    private fun buildRideMetricDefinitions(): List<PersonalRecordMetricDefinition> {
        return listOf(
            bestTimeForDistanceMetric("best-time-250m", "Best 250 m", 250.0),
            bestTimeForDistanceMetric("best-time-500m", "Best 500 m", 500.0),
            bestTimeForDistanceMetric("best-time-1000m", "Best 1000 m", 1000.0),
            bestTimeForDistanceMetric("best-time-5km", "Best 5 km", 5000.0),
            bestTimeForDistanceMetric("best-time-10km", "Best 10 km", 10000.0),
            bestTimeForDistanceMetric("best-time-20km", "Best 20 km", 20000.0),
            bestTimeForDistanceMetric("best-time-50km", "Best 50 km", 50000.0),
            bestTimeForDistanceMetric("best-time-100km", "Best 100 km", 100000.0),
            bestDistanceForTimeMetric("best-distance-30min", "Best 30 min", 30 * 60),
            bestDistanceForTimeMetric("best-distance-1h", "Best 1 h", 60 * 60),
            bestDistanceForTimeMetric("best-distance-2h", "Best 2 h", 2 * 60 * 60),
            bestDistanceForTimeMetric("best-distance-3h", "Best 3 h", 3 * 60 * 60),
            bestDistanceForTimeMetric("best-distance-4h", "Best 4 h", 4 * 60 * 60),
            bestDistanceForTimeMetric("best-distance-5h", "Best 5 h", 5 * 60 * 60),
            bestGradientForDistanceMetric("best-gradient-250m", "Max gradient for 250 m", 250.0),
            bestGradientForDistanceMetric("best-gradient-500m", "Max gradient for 500 m", 500.0),
            bestGradientForDistanceMetric("best-gradient-1000m", "Max gradient for 1000 m", 1000.0),
            bestGradientForDistanceMetric("best-gradient-5km", "Max gradient for 5 km", 5000.0),
            bestGradientForDistanceMetric("best-gradient-10km", "Max gradient for 10 km", 10000.0),
            bestGradientForDistanceMetric("best-gradient-20km", "Max gradient for 20 km", 20000.0),
            bestPowerForTimeMetric("best-power-20min", "Best average power for 20 min", 20 * 60),
            bestPowerForTimeMetric("best-power-1h", "Best average power for 1 h", 60 * 60),
        ) + buildActivityRecordMetricDefinitions()
    }

    private fun buildAlpineSkiMetricDefinitions(): List<PersonalRecordMetricDefinition> {
        return listOf(
            bestTimeForDistanceMetric("best-time-250m", "Best 250 m", 250.0),
            bestTimeForDistanceMetric("best-time-500m", "Best 500 m", 500.0),
            bestTimeForDistanceMetric("best-time-1000m", "Best 1000 m", 1000.0),
            bestTimeForDistanceMetric("best-time-5km", "Best 5 km", 5000.0),
            bestTimeForDistanceMetric("best-time-10km", "Best 10 km", 10000.0),
            bestTimeForDistanceMetric("best-time-20km", "Best 20 km", 20000.0),
            bestTimeForDistanceMetric("best-time-50km", "Best 50 km", 50000.0),
            bestTimeForDistanceMetric("best-time-100km", "Best 100 km", 100000.0),
            bestDistanceForTimeMetric("best-distance-30min", "Best 30 min", 30 * 60),
            bestDistanceForTimeMetric("best-distance-1h", "Best 1 h", 60 * 60),
            bestDistanceForTimeMetric("best-distance-2h", "Best 2 h", 2 * 60 * 60),
            bestDistanceForTimeMetric("best-distance-3h", "Best 3 h", 3 * 60 * 60),
            bestDistanceForTimeMetric("best-distance-4h", "Best 4 h", 4 * 60 * 60),
            bestDistanceForTimeMetric("best-distance-5h", "Best 5 h", 5 * 60 * 60),
        ) + buildActivityRecordMetricDefinitions()
    }

    private fun buildInlineSkateMetricDefinitions(): List<PersonalRecordMetricDefinition> {
        return listOf(
            bestTimeForDistanceMetric("best-time-200m", "Best 200 m", 200.0),
            bestTimeForDistanceMetric("best-time-400m", "Best 400 m", 400.0),
            bestTimeForDistanceMetric("best-time-1000m", "Best 1000 m", 1000.0),
            bestTimeForDistanceMetric("best-time-10000m", "Best 10000 m", 10000.0),
            bestTimeForDistanceMetric("best-time-half-marathon", "Best half Marathon", 21097.0),
            bestTimeForDistanceMetric("best-time-marathon", "Best Marathon", 42195.0),
            bestDistanceForTimeMetric("best-distance-1h", "Best 1 h", 60 * 60),
            bestDistanceForTimeMetric("best-distance-2h", "Best 2 h", 2 * 60 * 60),
            bestDistanceForTimeMetric("best-distance-3h", "Best 3 h", 3 * 60 * 60),
            bestDistanceForTimeMetric("best-distance-4h", "Best 4 h", 4 * 60 * 60),
        ) + buildActivityRecordMetricDefinitions()
    }

    private fun buildActivityRecordMetricDefinitions(): List<PersonalRecordMetricDefinition> {
        return listOf(
            maxDistanceActivityMetric("max-distance-activity", "Max distance"),
            maxSpeedActivityMetric("max-speed-activity", "Max speed"),
            maxMovingTimeActivityMetric("max-moving-time-activity", "Max moving time"),
            maxDistanceInDayMetric("max-distance-in-a-day", "Max distance in a day"),
            maxElevationActivityMetric("max-elevation-activity", "Max elevation"),
            maxElevationInDayMetric("max-elevation-in-a-day", "Max elevation gain in a day"),
            highestPointActivityMetric("highest-point-activity", "Highest point"),
        )
    }

    private fun maxDistanceActivityMetric(key: String, label: String): PersonalRecordMetricDefinition {
        return PersonalRecordMetricDefinition(
            key = key,
            label = label,
            effortExtractor = { activity ->
                if (activity.distance <= 0.0) null
                else activity.toActivityEffort(scoreValue = activity.distance, secondsOverride = activity.movingTime.coerceAtLeast(1))
            },
            score = { effort -> effort.distance },
            isBetter = { score, previousScore -> score > previousScore },
            valueFormatter = { effort -> String.format(Locale.ENGLISH, "%.2f km", effort.distance / 1000.0) },
            improvementFormatter = { previous, current ->
                "${formatDistance(current.distance - previous.distance)} farther"
            }
        )
    }

    private fun maxSpeedActivityMetric(key: String, label: String): PersonalRecordMetricDefinition {
        return PersonalRecordMetricDefinition(
            key = key,
            label = label,
            effortExtractor = { activity ->
                if (activity.maxSpeed <= 0.0f) null
                else activity.toActivityEffort(scoreValue = activity.maxSpeed.toDouble(), secondsOverride = activity.movingTime.coerceAtLeast(1))
            },
            score = { effort -> effort.distance },
            isBetter = { score, previousScore -> score > previousScore },
            valueFormatter = { effort -> effort.distance.formatSpeed(effort.activityShort.type) },
            improvementFormatter = { previous, current ->
                String.format(Locale.ENGLISH, "%+.2f km/h", (current.distance - previous.distance) * 3.6)
            }
        )
    }

    private fun maxMovingTimeActivityMetric(key: String, label: String): PersonalRecordMetricDefinition {
        return PersonalRecordMetricDefinition(
            key = key,
            label = label,
            effortExtractor = { activity ->
                if (activity.movingTime <= 0) null
                else activity.toActivityEffort(scoreValue = activity.distance, secondsOverride = activity.movingTime)
            },
            score = { effort -> effort.seconds.toDouble() },
            isBetter = { score, previousScore -> score > previousScore },
            valueFormatter = { effort -> effort.seconds.formatSeconds() },
            improvementFormatter = { previous, current ->
                "${(current.seconds - previous.seconds).coerceAtLeast(0).formatSeconds()} longer"
            }
        )
    }

    private fun maxDistanceInDayMetric(key: String, label: String): PersonalRecordMetricDefinition {
        val distanceByDay = mutableMapOf<String, Double>()
        return PersonalRecordMetricDefinition(
            key = key,
            label = label,
            effortExtractor = { activity ->
                if (activity.distance <= 0.0) {
                    null
                } else {
                    val day = activity.startDateLocal.substringBefore('T').ifBlank { activity.startDateLocal }
                    val updatedTotal = (distanceByDay[day] ?: 0.0) + activity.distance
                    distanceByDay[day] = updatedTotal
                    activity.toActivityEffort(
                        scoreValue = updatedTotal,
                        secondsOverride = activity.movingTime.coerceAtLeast(1),
                        labelOverride = day
                    )
                }
            },
            score = { effort -> effort.distance },
            isBetter = { score, previousScore -> score > previousScore },
            valueFormatter = { effort ->
                "${String.format(Locale.ENGLISH, "%.2f km", effort.distance / 1000.0)} - ${formatRecordDay(effort.label)}"
            },
            improvementFormatter = { previous, current ->
                "${formatDistance(current.distance - previous.distance)} farther"
            }
        )
    }

    private fun maxElevationActivityMetric(key: String, label: String): PersonalRecordMetricDefinition {
        return PersonalRecordMetricDefinition(
            key = key,
            label = label,
            effortExtractor = { activity ->
                if (activity.totalElevationGain <= 0.0) null
                else activity.toActivityEffort(scoreValue = activity.totalElevationGain, secondsOverride = activity.movingTime.coerceAtLeast(1))
            },
            score = { effort -> effort.distance },
            isBetter = { score, previousScore -> score > previousScore },
            valueFormatter = { effort -> String.format(Locale.ENGLISH, "%.2f m", effort.distance) },
            improvementFormatter = { previous, current ->
                String.format(Locale.ENGLISH, "+%.2f m", current.distance - previous.distance)
            }
        )
    }

    private fun maxElevationInDayMetric(key: String, label: String): PersonalRecordMetricDefinition {
        val elevationByDay = mutableMapOf<String, Double>()
        return PersonalRecordMetricDefinition(
            key = key,
            label = label,
            effortExtractor = { activity ->
                if (activity.totalElevationGain <= 0.0) {
                    null
                } else {
                    val day = activity.startDateLocal.substringBefore('T').ifBlank { activity.startDateLocal }
                    val updatedTotal = (elevationByDay[day] ?: 0.0) + activity.totalElevationGain
                    elevationByDay[day] = updatedTotal
                    activity.toActivityEffort(
                        scoreValue = updatedTotal,
                        secondsOverride = activity.movingTime.coerceAtLeast(1),
                        labelOverride = day
                    )
                }
            },
            score = { effort -> effort.distance },
            isBetter = { score, previousScore -> score > previousScore },
            valueFormatter = { effort ->
                "${String.format(Locale.ENGLISH, "%.2f m", effort.distance)} - ${formatRecordDay(effort.label)}"
            },
            improvementFormatter = { previous, current ->
                String.format(Locale.ENGLISH, "+%.2f m", current.distance - previous.distance)
            }
        )
    }

    private fun highestPointActivityMetric(key: String, label: String): PersonalRecordMetricDefinition {
        return PersonalRecordMetricDefinition(
            key = key,
            label = label,
            effortExtractor = { activity ->
                if (activity.elevHigh <= 0.0) null
                else activity.toActivityEffort(scoreValue = activity.elevHigh, secondsOverride = activity.movingTime.coerceAtLeast(1))
            },
            score = { effort -> effort.distance },
            isBetter = { score, previousScore -> score > previousScore },
            valueFormatter = { effort -> String.format(Locale.ENGLISH, "%.2f m", effort.distance) },
            improvementFormatter = { previous, current ->
                String.format(Locale.ENGLISH, "+%.2f m", current.distance - previous.distance)
            }
        )
    }

    private fun StravaActivity.toActivityEffort(scoreValue: Double, secondsOverride: Int): ActivityEffort {
        return ActivityEffort(
            distance = scoreValue,
            seconds = secondsOverride.coerceAtLeast(1),
            deltaAltitude = this.totalElevationGain,
            idxStart = 0,
            idxEnd = 0,
            label = this.name,
            activityShort = ActivityShort(this.id, this.name, this.type),
        )
    }

    private fun StravaActivity.toActivityEffort(scoreValue: Double, secondsOverride: Int, labelOverride: String): ActivityEffort {
        return ActivityEffort(
            distance = scoreValue,
            seconds = secondsOverride.coerceAtLeast(1),
            deltaAltitude = this.totalElevationGain,
            idxStart = 0,
            idxEnd = 0,
            label = labelOverride,
            activityShort = ActivityShort(this.id, this.name, this.type),
        )
    }

    private fun bestTimeForDistanceMetric(key: String, label: String, distance: Double): PersonalRecordMetricDefinition {
        return PersonalRecordMetricDefinition(
            key = key,
            label = label,
            effortExtractor = { activity -> activity.calculateBestTimeForDistance(distance) },
            score = { effort -> effort.seconds.toDouble() },
            isBetter = { score, previousScore -> score < previousScore },
            valueFormatter = { effort -> "${effort.seconds.formatSeconds()} (${effort.getFormattedSpeedWithUnits()})" },
            improvementFormatter = { previous, current ->
                val gainedSeconds = previous.seconds - current.seconds
                "${gainedSeconds.formatSeconds()} faster"
            }
        )
    }

    private fun bestDistanceForTimeMetric(key: String, label: String, seconds: Int): PersonalRecordMetricDefinition {
        return PersonalRecordMetricDefinition(
            key = key,
            label = label,
            effortExtractor = { activity -> activity.calculateBestDistanceForTime(seconds) },
            score = { effort -> effort.distance },
            isBetter = { score, previousScore -> score > previousScore },
            valueFormatter = { effort -> "${formatDistance(effort.distance)} (${effort.getFormattedSpeedWithUnits()})" },
            improvementFormatter = { previous, current ->
                "${formatDistance(current.distance - previous.distance)} farther"
            }
        )
    }

    private fun bestPowerForTimeMetric(key: String, label: String, seconds: Int): PersonalRecordMetricDefinition {
        return PersonalRecordMetricDefinition(
            key = key,
            label = label,
            effortExtractor = { activity -> activity.calculateBestPowerForTime(seconds) },
            score = { effort -> effort.averagePower?.toDouble() ?: Double.NEGATIVE_INFINITY },
            isBetter = { score, previousScore -> score > previousScore },
            valueFormatter = { effort -> effort.getFormattedPower() },
            improvementFormatter = { previous, current ->
                val previousPower = previous.averagePower ?: 0
                val currentPower = current.averagePower ?: 0
                "+${currentPower - previousPower} W"
            }
        )
    }

    private fun bestGradientForDistanceMetric(key: String, label: String, distance: Double): PersonalRecordMetricDefinition {
        return PersonalRecordMetricDefinition(
            key = key,
            label = label,
            effortExtractor = { activity -> activity.calculateBestElevationForDistance(distance) },
            score = { effort -> effort.getGradient() },
            isBetter = { score, previousScore -> score > previousScore },
            valueFormatter = { effort -> effort.getFormattedGradientWithUnit() },
            improvementFormatter = { previous, current ->
                val deltaGradient = current.getGradient() - previous.getGradient()
                String.format(Locale.ENGLISH, "+%.2f %%", deltaGradient)
            }
        )
    }

    private fun formatDistance(distanceInMeters: Double): String {
        return if (distanceInMeters >= 1000) {
            String.format(Locale.ENGLISH, "%.2f km", distanceInMeters / 1000.0)
        } else {
            String.format(Locale.ENGLISH, "%.0f m", distanceInMeters)
        }
    }

    private fun formatRecordDay(day: String): String {
        return runCatching { LocalDate.parse(day).format(dateFormatter) }
            .getOrDefault(day)
    }
}
