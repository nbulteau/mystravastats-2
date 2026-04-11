package me.nicolas.stravastats.domain.services

import me.nicolas.stravastats.domain.business.ActivityEffort
import me.nicolas.stravastats.domain.business.ActivityShort
import me.nicolas.stravastats.domain.business.ActivityType
import me.nicolas.stravastats.domain.business.PersonalRecordTimelineEntry
import me.nicolas.stravastats.domain.business.SegmentClimbAttempt
import me.nicolas.stravastats.domain.business.SegmentClimbProgression
import me.nicolas.stravastats.domain.business.SegmentClimbTargetSummary
import me.nicolas.stravastats.domain.business.runActivities
import me.nicolas.stravastats.domain.business.strava.*
import me.nicolas.stravastats.domain.services.activityproviders.IActivityProvider
import me.nicolas.stravastats.domain.services.statistics.*
import me.nicolas.stravastats.domain.utils.formatSeconds
import org.slf4j.LoggerFactory
import org.springframework.stereotype.Service
import java.util.Locale
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
}

@Service
internal class StatisticsService(
    activityProvider: IActivityProvider,
) : IStatisticsService, AbstractStravaService(activityProvider) {

    private val logger = LoggerFactory.getLogger(StatisticsService::class.java)

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
            .sortedBy { activity -> activity.startDateLocal }

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

        return timeline.sortedBy { entry -> entry.activityDate }
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
                        targetId = effort.segment.id,
                        targetName = effort.segment.name,
                        targetType = effortTargetType,
                        climbCategory = effort.segment.climbCategory,
                        distance = effort.distance,
                        averageGrade = effort.segment.averageGrade,
                        elapsedTimeSeconds = effort.elapsedTime,
                        speedKph = computeSpeedKph(effort.distance, effort.elapsedTime),
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

        if (rawAttempts.isEmpty()) {
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

        val attemptsByTarget = rawAttempts.groupBy { raw -> raw.targetId }
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

    private fun computeSpeedKph(distanceInMeters: Double, elapsedTimeSeconds: Int): Double {
        if (distanceInMeters <= 0.0 || elapsedTimeSeconds <= 0) {
            return 0.0
        }
        val metersPerSecond = distanceInMeters / elapsedTimeSeconds.toDouble()
        return metersPerSecond * 3.6
    }

    private fun buildAttempts(attempts: List<SegmentAttemptRaw>, metric: SegmentMetric): List<SegmentClimbAttempt> {
        val sortedAttempts = attempts.sortedBy { attempt -> attempt.activityDate }
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
                speedKph = attempt.speedKph,
                distance = attempt.distance,
                averageGrade = attempt.averageGrade,
                elevationGain = (attempt.distance * attempt.averageGrade) / 100.0,
                prRank = attempt.prRank,
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
        val targetId: Long,
        val targetName: String,
        val targetType: SegmentTargetType,
        val climbCategory: Int,
        val distance: Double,
        val averageGrade: Double,
        val elapsedTimeSeconds: Int,
        val speedKph: Double,
        val activityDate: String,
        val prRank: Int?,
        val activity: ActivityShort,
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

    private fun getPersonalRecordMetricDefinitions(activityTypes: Set<ActivityType>): List<PersonalRecordMetricDefinition> {
        return when (resolvePrimaryActivityType(activityTypes)) {
            ActivityType.Run -> buildRunMetricDefinitions()
            ActivityType.InlineSkate -> buildInlineSkateMetricDefinitions()
            ActivityType.AlpineSki -> buildAlpineSkiMetricDefinitions()
            ActivityType.Hike -> emptyList()
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
        )
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
        )
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
        )
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
}
