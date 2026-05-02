package me.nicolas.stravastats.domain.services

import me.nicolas.stravastats.domain.business.ActivityEffort
import me.nicolas.stravastats.domain.business.ActivityShort
import me.nicolas.stravastats.domain.business.ActivityType
import me.nicolas.stravastats.domain.business.PersonalRecordTimelineEntry
import me.nicolas.stravastats.domain.business.runActivities
import me.nicolas.stravastats.domain.business.strava.*
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


interface IStatisticsService {
    fun getStatistics(activityTypes: Set<ActivityType>, year: Int?): List<Statistic>
    fun getPersonalRecordsTimeline(activityTypes: Set<ActivityType>, year: Int?, metric: String?): List<PersonalRecordTimelineEntry>
}

@Service
internal class StatisticsService(
    activityProvider: IActivityProvider,
) : IStatisticsService, AbstractStravaService(activityProvider) {

    private val logger = LoggerFactory.getLogger(StatisticsService::class.java)

    override fun getStatistics(activityTypes: Set<ActivityType>, year: Int?): List<Statistic> {
        logger.info("Compute $activityTypes statistics for ${year ?: "all years"}")

        val filteredActivities = activityProvider.getActivitiesByActivityTypeAndYear(activityTypes, year)
            .withoutDataQualityExcludedStats(activityProvider)

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
        metric: String?,
    ): List<PersonalRecordTimelineEntry> {
        logger.info("Compute personal records timeline for $activityTypes in ${year ?: "all years"}")

        val filteredActivities = activityProvider.getActivitiesByActivityTypeAndYear(activityTypes, year)
            .withoutDataQualityExcludedStats(activityProvider)
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

        if (filteredActivities.isEmpty()) return emptyList()

        val selectedMetrics = getPersonalRecordMetricDefinitions(activityTypes)
            .filter { definition -> metric.isNullOrBlank() || definition.key == metric }

        val timeline = mutableListOf<PersonalRecordTimelineEntry>()

        selectedMetrics.forEach { definition ->
            var bestEffort: ActivityEffort? = null

            filteredActivities.forEach { activity ->
                val effort = definition.effortExtractor(activity) ?: return@forEach
                if (effort.activityShort.id != activity.id) return@forEach

                val previousBest = bestEffort
                if (previousBest == null || definition.isBetter(definition.score(effort), definition.score(previousBest))) {
                    timeline += PersonalRecordTimelineEntry(
                        metricKey = definition.key,
                        metricLabel = definition.label,
                        activityDate = activity.startDateLocal,
                        value = definition.valueFormatter(effort),
                        previousValue = previousBest?.let(definition.valueFormatter),
                        improvement = previousBest?.let { previous -> definition.improvementFormatter(previous, effort) },
                        activity = effort.activityShort,
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

    // ---- Date parsing helpers ----

    private fun parseActivityDateEpochMillis(value: String?): Long? {
        if (value.isNullOrBlank()) return null
        return runCatching { OffsetDateTime.parse(value).toInstant().toEpochMilli() }
            .recoverCatching { Instant.parse(value).toEpochMilli() }
            .recoverCatching { LocalDateTime.parse(value).toInstant(ZoneOffset.UTC).toEpochMilli() }
            .getOrNull()
    }

    // ---- Statistics builders ----

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
                        stravaActivity.startDateLocal.substringBefore('T')
                    }.mapValues { it.value.sumOf { activity -> activity.distance / 1000 } }
                        .maxByOrNull { entry: Map.Entry<String, Double> -> entry.value }?.toPair()
                },
                BestDayStatistic("Max elevation in a day", hikeActivities, formatString = "%s => %.02f m") {
                    hikeActivities.groupBy { stravaActivity: StravaActivity ->
                        stravaActivity.startDateLocal.substringBefore('T')
                    }.mapValues { it.value.sumOf { activity -> activity.totalElevationGain } }
                        .maxByOrNull { entry: Map.Entry<String, Double> -> entry.value }?.toPair()
                },
            )
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

    private fun computeCommonStats(activities: List<StravaActivity>): List<Statistic> =
        listOf(
            GlobalStatistic("Nb activities", activities, { number -> "%d".format(number) }, List<StravaActivity>::size),
            GlobalStatistic("Nb actives days", activities, { number -> "%d".format(number) }) {
                activities.groupBy { stravaActivity: StravaActivity ->
                    stravaActivity.startDateLocal.substringBefore('T')
                }.count()
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

    // ---- Personal record metrics ----

    private fun getPersonalRecordMetricDefinitions(activityTypes: Set<ActivityType>): List<PersonalRecordMetricDefinition> =
        when (resolvePrimaryActivityType(activityTypes)) {
            ActivityType.Run -> buildRunMetricDefinitions()
            ActivityType.InlineSkate -> buildInlineSkateMetricDefinitions()
            ActivityType.AlpineSki -> buildAlpineSkiMetricDefinitions()
            ActivityType.Hike -> buildActivityRecordMetricDefinitions()
            else -> buildRideMetricDefinitions()
        }

    private fun buildRunMetricDefinitions(): List<PersonalRecordMetricDefinition> =
        listOf(
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

    private fun buildRideMetricDefinitions(): List<PersonalRecordMetricDefinition> =
        listOf(
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

    private fun buildAlpineSkiMetricDefinitions(): List<PersonalRecordMetricDefinition> =
        listOf(
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

    private fun buildInlineSkateMetricDefinitions(): List<PersonalRecordMetricDefinition> =
        listOf(
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

    private fun buildActivityRecordMetricDefinitions(): List<PersonalRecordMetricDefinition> =
        listOf(
            maxDistanceActivityMetric("max-distance-activity", "Max distance"),
            maxSpeedActivityMetric("max-speed-activity", "Max speed"),
            maxMovingTimeActivityMetric("max-moving-time-activity", "Max moving time"),
            maxDistanceInDayMetric("max-distance-in-a-day", "Max distance in a day"),
            maxElevationActivityMetric("max-elevation-activity", "Max elevation"),
            maxElevationInDayMetric("max-elevation-in-a-day", "Max elevation gain in a day"),
            highestPointActivityMetric("highest-point-activity", "Highest point"),
        )

    private fun maxDistanceActivityMetric(key: String, label: String): PersonalRecordMetricDefinition =
        PersonalRecordMetricDefinition(
            key = key, label = label,
            effortExtractor = { activity ->
                if (activity.distance <= 0.0) null
                else activity.toActivityEffort(scoreValue = activity.distance, secondsOverride = activity.movingTime.coerceAtLeast(1))
            },
            score = { effort -> effort.distance },
            isBetter = { score, previousScore -> score > previousScore },
            valueFormatter = { effort -> String.format(Locale.ENGLISH, "%.2f km", effort.distance / 1000.0) },
            improvementFormatter = { previous, current -> "${formatDistance(current.distance - previous.distance)} farther" },
        )

    private fun maxSpeedActivityMetric(key: String, label: String): PersonalRecordMetricDefinition =
        PersonalRecordMetricDefinition(
            key = key, label = label,
            effortExtractor = { activity ->
                if (activity.maxSpeed <= 0.0f) null
                else activity.toActivityEffort(scoreValue = activity.maxSpeed.toDouble(), secondsOverride = activity.movingTime.coerceAtLeast(1))
            },
            score = { effort -> effort.distance },
            isBetter = { score, previousScore -> score > previousScore },
            valueFormatter = { effort -> effort.distance.formatSpeed(effort.activityShort.type) },
            improvementFormatter = { previous, current ->
                String.format(Locale.ENGLISH, "%+.2f km/h", (current.distance - previous.distance) * 3.6)
            },
        )

    private fun maxMovingTimeActivityMetric(key: String, label: String): PersonalRecordMetricDefinition =
        PersonalRecordMetricDefinition(
            key = key, label = label,
            effortExtractor = { activity ->
                if (activity.movingTime <= 0) null
                else activity.toActivityEffort(scoreValue = activity.distance, secondsOverride = activity.movingTime)
            },
            score = { effort -> effort.seconds.toDouble() },
            isBetter = { score, previousScore -> score > previousScore },
            valueFormatter = { effort -> effort.seconds.formatSeconds() },
            improvementFormatter = { previous, current ->
                "${(current.seconds - previous.seconds).coerceAtLeast(0).formatSeconds()} longer"
            },
        )

    private fun maxDistanceInDayMetric(key: String, label: String): PersonalRecordMetricDefinition {
        val distanceByDay = mutableMapOf<String, Double>()
        return PersonalRecordMetricDefinition(
            key = key, label = label,
            effortExtractor = { activity ->
                if (activity.distance <= 0.0) null
                else {
                    val day = activity.startDateLocal.substringBefore('T').ifBlank { activity.startDateLocal }
                    val updatedTotal = (distanceByDay[day] ?: 0.0) + activity.distance
                    distanceByDay[day] = updatedTotal
                    activity.toActivityEffort(scoreValue = updatedTotal, secondsOverride = activity.movingTime.coerceAtLeast(1), labelOverride = day)
                }
            },
            score = { effort -> effort.distance },
            isBetter = { score, previousScore -> score > previousScore },
            valueFormatter = { effort ->
                "${String.format(Locale.ENGLISH, "%.2f km", effort.distance / 1000.0)} - ${formatRecordDay(effort.label)}"
            },
            improvementFormatter = { previous, current -> "${formatDistance(current.distance - previous.distance)} farther" },
        )
    }

    private fun maxElevationActivityMetric(key: String, label: String): PersonalRecordMetricDefinition =
        PersonalRecordMetricDefinition(
            key = key, label = label,
            effortExtractor = { activity ->
                if (activity.totalElevationGain <= 0.0) null
                else activity.toActivityEffort(scoreValue = activity.totalElevationGain, secondsOverride = activity.movingTime.coerceAtLeast(1))
            },
            score = { effort -> effort.distance },
            isBetter = { score, previousScore -> score > previousScore },
            valueFormatter = { effort -> String.format(Locale.ENGLISH, "%.2f m", effort.distance) },
            improvementFormatter = { previous, current ->
                String.format(Locale.ENGLISH, "+%.2f m", current.distance - previous.distance)
            },
        )

    private fun maxElevationInDayMetric(key: String, label: String): PersonalRecordMetricDefinition {
        val elevationByDay = mutableMapOf<String, Double>()
        return PersonalRecordMetricDefinition(
            key = key, label = label,
            effortExtractor = { activity ->
                if (activity.totalElevationGain <= 0.0) null
                else {
                    val day = activity.startDateLocal.substringBefore('T').ifBlank { activity.startDateLocal }
                    val updatedTotal = (elevationByDay[day] ?: 0.0) + activity.totalElevationGain
                    elevationByDay[day] = updatedTotal
                    activity.toActivityEffort(scoreValue = updatedTotal, secondsOverride = activity.movingTime.coerceAtLeast(1), labelOverride = day)
                }
            },
            score = { effort -> effort.distance },
            isBetter = { score, previousScore -> score > previousScore },
            valueFormatter = { effort ->
                "${String.format(Locale.ENGLISH, "%.2f m", effort.distance)} - ${formatRecordDay(effort.label)}"
            },
            improvementFormatter = { previous, current ->
                String.format(Locale.ENGLISH, "+%.2f m", current.distance - previous.distance)
            },
        )
    }

    private fun highestPointActivityMetric(key: String, label: String): PersonalRecordMetricDefinition =
        PersonalRecordMetricDefinition(
            key = key, label = label,
            effortExtractor = { activity ->
                if (activity.elevHigh <= 0.0) null
                else activity.toActivityEffort(scoreValue = activity.elevHigh, secondsOverride = activity.movingTime.coerceAtLeast(1))
            },
            score = { effort -> effort.distance },
            isBetter = { score, previousScore -> score > previousScore },
            valueFormatter = { effort -> String.format(Locale.ENGLISH, "%.2f m", effort.distance) },
            improvementFormatter = { previous, current ->
                String.format(Locale.ENGLISH, "+%.2f m", current.distance - previous.distance)
            },
        )

    private fun bestTimeForDistanceMetric(key: String, label: String, distance: Double): PersonalRecordMetricDefinition =
        PersonalRecordMetricDefinition(
            key = key, label = label,
            effortExtractor = { activity -> activity.calculateBestTimeForDistance(distance) },
            score = { effort -> effort.seconds.toDouble() },
            isBetter = { score, previousScore -> score < previousScore },
            valueFormatter = { effort -> "${effort.seconds.formatSeconds()} (${effort.getFormattedSpeedWithUnits()})" },
            improvementFormatter = { previous, current ->
                val gainedSeconds = previous.seconds - current.seconds
                "${gainedSeconds.formatSeconds()} faster"
            },
        )

    private fun bestDistanceForTimeMetric(key: String, label: String, seconds: Int): PersonalRecordMetricDefinition =
        PersonalRecordMetricDefinition(
            key = key, label = label,
            effortExtractor = { activity -> activity.calculateBestDistanceForTime(seconds) },
            score = { effort -> effort.distance },
            isBetter = { score, previousScore -> score > previousScore },
            valueFormatter = { effort -> "${formatDistance(effort.distance)} (${effort.getFormattedSpeedWithUnits()})" },
            improvementFormatter = { previous, current -> "${formatDistance(current.distance - previous.distance)} farther" },
        )

    private fun bestPowerForTimeMetric(key: String, label: String, seconds: Int): PersonalRecordMetricDefinition =
        PersonalRecordMetricDefinition(
            key = key, label = label,
            effortExtractor = { activity -> activity.calculateBestPowerForTime(seconds) },
            score = { effort -> effort.averagePower?.toDouble() ?: Double.NEGATIVE_INFINITY },
            isBetter = { score, previousScore -> score > previousScore },
            valueFormatter = { effort -> effort.getFormattedPower() },
            improvementFormatter = { previous, current ->
                val previousPower = previous.averagePower ?: 0
                val currentPower = current.averagePower ?: 0
                "+${currentPower - previousPower} W"
            },
        )

    private fun bestGradientForDistanceMetric(key: String, label: String, distance: Double): PersonalRecordMetricDefinition =
        PersonalRecordMetricDefinition(
            key = key, label = label,
            effortExtractor = { activity -> activity.calculateBestElevationForDistance(distance) },
            score = { effort -> effort.getGradient() },
            isBetter = { score, previousScore -> score > previousScore },
            valueFormatter = { effort -> effort.getFormattedGradientWithUnit() },
            improvementFormatter = { previous, current ->
                String.format(Locale.ENGLISH, "+%.2f %%", current.getGradient() - previous.getGradient())
            },
        )

    // ---- Activity/effort extension helpers ----

    private fun StravaActivity.toActivityEffort(scoreValue: Double, secondsOverride: Int): ActivityEffort =
        ActivityEffort(
            distance = scoreValue,
            seconds = secondsOverride.coerceAtLeast(1),
            deltaAltitude = totalElevationGain,
            idxStart = 0,
            idxEnd = 0,
            label = name,
            activityShort = ActivityShort(id, name, type),
        )

    private fun StravaActivity.toActivityEffort(
        scoreValue: Double,
        secondsOverride: Int,
        labelOverride: String,
    ): ActivityEffort =
        ActivityEffort(
            distance = scoreValue,
            seconds = secondsOverride.coerceAtLeast(1),
            deltaAltitude = totalElevationGain,
            idxStart = 0,
            idxEnd = 0,
            label = labelOverride,
            activityShort = ActivityShort(id, name, type),
        )

    // ---- Activity type resolution ----

    private fun resolvePrimaryActivityType(activityTypes: Set<ActivityType>): ActivityType =
        when {
            activityTypes.any { type -> type in runActivities } -> ActivityType.Run
            activityTypes.contains(ActivityType.InlineSkate) -> ActivityType.InlineSkate
            activityTypes.contains(ActivityType.Hike) || activityTypes.contains(ActivityType.Walk) -> ActivityType.Hike
            activityTypes.contains(ActivityType.AlpineSki) -> ActivityType.AlpineSki
            else -> ActivityType.Ride
        }

    // ---- Formatting helpers ----

    private fun formatDistance(distanceInMeters: Double): String =
        if (distanceInMeters >= 1000) String.format(Locale.ENGLISH, "%.2f km", distanceInMeters / 1000.0)
        else String.format(Locale.ENGLISH, "%.0f m", distanceInMeters)

    private fun formatRecordDay(day: String): String =
        runCatching { LocalDate.parse(day).format(dateFormatter) }.getOrDefault(day)

    // ---- Internal data type ----

    private data class PersonalRecordMetricDefinition(
        val key: String,
        val label: String,
        val effortExtractor: (StravaActivity) -> ActivityEffort?,
        val score: (ActivityEffort) -> Double,
        val isBetter: (Double, Double) -> Boolean,
        val valueFormatter: (ActivityEffort) -> String,
        val improvementFormatter: (ActivityEffort, ActivityEffort) -> String,
    )
}
