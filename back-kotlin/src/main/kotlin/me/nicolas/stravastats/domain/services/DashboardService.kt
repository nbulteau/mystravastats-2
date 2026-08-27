package me.nicolas.stravastats.domain.services

import me.nicolas.stravastats.domain.business.ActivityType
import me.nicolas.stravastats.domain.business.DashboardData
import me.nicolas.stravastats.domain.business.EddingtonBasis
import me.nicolas.stravastats.domain.business.EddingtonMetric
import me.nicolas.stravastats.domain.business.EddingtonNumber
import me.nicolas.stravastats.domain.business.EddingtonScope
import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.services.ActivityHelper.groupActivitiesByDay
import me.nicolas.stravastats.domain.services.ActivityHelper.activityYearOrNull
import me.nicolas.stravastats.domain.services.activityproviders.IActivityProvider
import me.nicolas.stravastats.domain.services.activityproviders.StravaActivityProvider
import me.nicolas.stravastats.domain.services.statistics.isPlausibleActivityMaxSpeed
import org.slf4j.LoggerFactory
import org.springframework.stereotype.Service
import java.time.LocalDate
import kotlin.math.round

interface IDashboardService {
    fun getCumulativeDistancePerYear(activityTypes: Set<ActivityType>): Map<String, Map<String, Number>>

    fun getCumulativeElevationPerYear(activityTypes: Set<ActivityType>): Map<String, Map<String, Number>>

    fun getEddingtonNumber(
        activityTypes: Set<ActivityType>,
        scope: EddingtonScope = EddingtonScope.LIFETIME,
        metric: EddingtonMetric = EddingtonMetric.DISTANCE,
        basis: EddingtonBasis = EddingtonBasis.DAYS,
        year: Int? = null,
    ): EddingtonNumber

    fun getDashboardData(activityTypes: Set<ActivityType>): DashboardData

    fun getActivityHeatmap(activityTypes: Set<ActivityType>): Map<String, Map<String, ActivityHeatmapDay>>
}

data class ActivityHeatmapActivity(
    val id: Long,
    val name: String,
    val type: String,
    val distanceKm: Double,
    val elevationGainM: Double,
    val durationSec: Int,
)

data class ActivityHeatmapDay(
    val distanceKm: Double,
    val elevationGainM: Double,
    val durationSec: Int,
    val activityCount: Int,
    val activities: List<ActivityHeatmapActivity>,
)


@Service
class DashboardService(
    activityProvider: IActivityProvider,
) : IDashboardService, AbstractStravaService(activityProvider) {

    private val logger = LoggerFactory.getLogger(DashboardService::class.java)

    private data class YearAccumulator(
        var nbActivities: Int = 0,
        var totalDistanceKm: Double = 0.0,
        var maxDistanceKm: Double = 0.0,
        var totalElevation: Double = 0.0,
        var maxElevation: Int = 0,
        var speedCount: Int = 0,
        var speedSum: Double = 0.0,
        var averageHeartRateCount: Int = 0,
        var averageHeartRateSum: Double = 0.0,
        var maxHeartRate: Int = 0,
        var averageWattsCount: Int = 0,
        var averageWattsSum: Int = 0,
        var maxWatts: Int = 0,
        var deviceAverageWattsCount: Int = 0,
        var deviceAverageWattsSum: Int = 0,
        var deviceMaxWatts: Int = 0,
    )

    /**
     * Get cumulative distance per year for a specific stravaActivity type.
     * It returns a map with the year as a key and the cumulative distance in km as a value.
     * @param activityTypes the stravaActivity type
     * @return a map with the year as a key and the cumulative distance in km as value
     */
    override fun getCumulativeDistancePerYear(activityTypes: Set<ActivityType>): Map<String, Map<String, Number>> {
        logger.info("Get cumulative distance per year for stravaActivity type $activityTypes")
        return getCumulativeDataPerYear(activityTypes) { activitiesByDay ->
            cumulativeDistance(activitiesByDay)
        }
    }

    override fun getCumulativeElevationPerYear(activityTypes: Set<ActivityType>): Map<String, Map<String, Number>> {
        logger.info("Get cumulative elevation per year for stravaActivity type $activityTypes")
        return getCumulativeDataPerYear(activityTypes) { activitiesByDay ->
            cumulativeElevation(activitiesByDay)
        }
    }

    /**
     * Get the Eddington number for a specific stravaActivity type.
     * @param activityTypes the stravaActivity type
     * @return the Eddington number structure
     */
    override fun getEddingtonNumber(
        activityTypes: Set<ActivityType>,
        scope: EddingtonScope,
        metric: EddingtonMetric,
        basis: EddingtonBasis,
        year: Int?,
    ): EddingtonNumber {
        logger.info("Get Eddington number for activity type $activityTypes, scope $scope, metric $metric and basis $basis")

        val excludedActivityIds = dataQualityExcludedActivityIds(activityProvider)
        val values = when (scope) {
            EddingtonScope.YEAR -> {
                if (year == null) {
                    emptyList()
                } else if (metric == EddingtonMetric.DISTANCE && basis == EddingtonBasis.DAYS && excludedActivityIds.isEmpty()) {
                    activityProvider.getActivitiesByActivityTypeByYearGroupByActiveDays(activityTypes, year).values.toList()
                } else {
                    activityProvider.getActivitiesByActivityTypeAndYear(activityTypes, year)
                        .withoutDataQualityExcludedStats(activityProvider)
                        .toEddingtonValues(metric, basis)
                }
            }
            EddingtonScope.ROLLING_12_MONTHS -> {
                val today = LocalDate.now()
                val start = today.minusYears(1)
                activityProvider.getActivitiesByActivityTypeAndYear(activityTypes)
                    .withoutDataQualityExcludedStats(activityProvider)
                    .filter { activity ->
                        val date = activity.localDate() ?: return@filter false
                        !date.isBefore(start) && !date.isAfter(today)
                    }
                    .toEddingtonValues(metric, basis)
            }
            EddingtonScope.LIFETIME -> {
                if (metric == EddingtonMetric.DISTANCE && basis == EddingtonBasis.DAYS && excludedActivityIds.isEmpty()) {
                    activityProvider.getActivitiesByActivityTypeGroupByActiveDays(activityTypes).values.toList()
                } else {
                    activityProvider.getActivitiesByActivityTypeAndYear(activityTypes)
                        .withoutDataQualityExcludedStats(activityProvider)
                        .toEddingtonValues(metric, basis)
                }
            }
        }
        val eddingtonList = computeEddingtonListFromDailyTotals(values)

        var eddingtonNumber = 0
        for (day in eddingtonList.size downTo 1) {
            if (eddingtonList[day - 1] >= day) {
                eddingtonNumber = day
                break
            }
        }

        return EddingtonNumber(eddingtonNumber, eddingtonList, scope, metric, basis)
    }

    private fun computeEddingtonListFromDailyTotals(dailyTotals: Collection<Int>): List<Int> {
        val positiveDailyTotals = dailyTotals.filter { total -> total > 0 }
        return if (positiveDailyTotals.isEmpty()) {
            emptyList()
        } else {
            val counts = IntArray(positiveDailyTotals.max()) { 0 }.toMutableList()
            if (counts.isNotEmpty()) {
                // counts = number of time we reach a distance
                positiveDailyTotals.forEach { total ->
                    for (day in total downTo 1) {
                        counts[day - 1] += 1
                    }
                }
            }
            counts
        }
    }

    override fun getDashboardData(activityTypes: Set<ActivityType>): DashboardData {
        logger.info("Get dashboard data for activity type $activityTypes")

        val activitiesByYear = activityProvider.getActivitiesByActivityTypeAndYear(activityTypes)
            .withoutDataQualityExcludedStats(activityProvider)
            .groupByValidYear("dashboard data")

        val yearlyAccumulators = activitiesByYear.mapValues { (_, activities) ->
            aggregateYear(activities)
        }

        val nbActivitiesByYear = yearlyAccumulators
            .mapValues { (_, stats) -> stats.nbActivities }
            .filter { it.value > 0 }

        val activeDaysByYear = activitiesByYear
            .mapValues { (_, activities) -> countActiveDays(activities) }
            .filter { it.value > 0 }

        val consistencyByYear = activeDaysByYear
            .mapValues { (year, activeDays) -> computeConsistencyByYear(year, activeDays) }
            .filter { it.value > 0F }

        val movingTimeByYear = activitiesByYear
            .mapValues { (_, activities) -> sumMovingTimeSeconds(activities) }
            .filter { it.value > 0 }

        val totalDistanceByYear = yearlyAccumulators
            .mapValues { (_, stats) -> stats.totalDistanceKm.toFloat() }
            .filter { it.value > 0 }

        val averageDistanceByYear = yearlyAccumulators
            .mapValues { (_, stats) ->
                (stats.totalDistanceKm / stats.nbActivities).toFloat()
            }
            .filter { it.value > 0 }

        val maxDistanceByYear = yearlyAccumulators
            .mapValues { (_, stats) -> stats.maxDistanceKm.toFloat() }
            .filter { it.value > 0 }

        val maxDistanceDateByYear = activitiesByYear
            .mapValues { (_, activities) -> maxDistanceDate(activities) }
            .filter { it.value.isNotBlank() }

        val distanceByActiveDayByYear = activitiesByYear
            .mapValues { (_, activities) -> distanceTotalsByActiveDay(activities) }

        val averageDistanceByActiveDayByYear = distanceByActiveDayByYear
            .mapValues { (_, dayTotals) -> averageDoubleValues(dayTotals).toFloat() }
            .filter { it.value > 0 }

        val maxDistanceByActiveDayByYear = distanceByActiveDayByYear
            .mapValues { (_, dayTotals) -> maxDoubleValue(dayTotals).toFloat() }
            .filter { it.value > 0 }

        val maxDistanceByActiveDayDateByYear = distanceByActiveDayByYear
            .mapValues { (_, dayTotals) -> maxDoubleValueKey(dayTotals) }
            .filter { it.value.isNotBlank() }

        val totalElevationByYear = yearlyAccumulators
            .mapValues { (_, stats) -> stats.totalElevation.toInt() }
            .filter { it.value > 0 }

        val averageElevationByYear = yearlyAccumulators
            .mapValues { (_, stats) ->
                (stats.totalElevation / stats.nbActivities).toInt()
            }
            .filter { it.value > 0 }

        val maxElevationByYear = yearlyAccumulators
            .mapValues { (_, stats) -> stats.maxElevation }
            .filter { entry -> entry.value > 0 }

        val maxElevationDateByYear = activitiesByYear
            .mapValues { (_, activities) -> maxElevationDate(activities) }
            .filter { it.value.isNotBlank() }

        val elevationByActiveDayByYear = activitiesByYear
            .mapValues { (_, activities) -> elevationTotalsByActiveDay(activities) }

        val averageElevationByActiveDayByYear = elevationByActiveDayByYear
            .mapValues { (_, dayTotals) -> averageIntValues(dayTotals) }
            .filter { it.value > 0 }

        val maxElevationByActiveDayByYear = elevationByActiveDayByYear
            .mapValues { (_, dayTotals) -> maxIntValue(dayTotals) }
            .filter { it.value > 0 }

        val maxElevationByActiveDayDateByYear = elevationByActiveDayByYear
            .mapValues { (_, dayTotals) -> maxIntValueKey(dayTotals) }
            .filter { it.value.isNotBlank() }

        val elevationEfficiencyByYear = totalDistanceByYear
            .mapNotNull { (year, distanceKm) ->
                val totalElevation = totalElevationByYear[year] ?: return@mapNotNull null
                if (distanceKm <= 0f || totalElevation <= 0) {
                    return@mapNotNull null
                }
                val value = ((totalElevation.toDouble() / distanceKm.toDouble()) * 10.0).toFloat()
                year to value
            }
            .toMap()

        val averageSpeedByYear = yearlyAccumulators
            .mapValues { (_, stats) ->
                if (stats.speedCount == 0) 0F else (stats.speedSum / stats.speedCount).toFloat()
            }
            .filter { entry -> entry.value > 0 }

        val maxSpeedActivityByYear = activitiesByYear
            .mapValues { (_, activities) -> maxSpeedActivity(activities) }

        val maxSpeedByYear = maxSpeedActivityByYear
            .mapValues { (_, activity) -> activity?.maxSpeed ?: 0F }
            .filter { entry -> entry.value > 0.0 }

        val maxSpeedDateByYear = maxSpeedActivityByYear
            .mapValues { (_, activity) -> activity?.activityDate() ?: "" }
            .filter { it.value.isNotBlank() }

        val averageHeartRateByYear = yearlyAccumulators
            .mapValues { (_, stats) ->
                if (stats.averageHeartRateCount == 0) 0 else (stats.averageHeartRateSum / stats.averageHeartRateCount).toInt()
            }
            .filter { entry -> entry.value > 0 }

        val maxHeartRateByYear = yearlyAccumulators
            .mapValues { (_, stats) -> stats.maxHeartRate }
            .filter { it.value > 0 }

        val maxHeartRateDateByYear = activitiesByYear
            .mapValues { (_, activities) -> maxHeartRateDate(activities) }
            .filter { it.value.isNotBlank() }

        val averageWattsByYear = yearlyAccumulators
            .mapValues { (_, stats) ->
                if (stats.averageWattsCount == 0) 0 else stats.averageWattsSum / stats.averageWattsCount
            }
            .filter { it.value > 0 }

        val maxWattsByYear = yearlyAccumulators
            .mapValues { (_, stats) -> stats.maxWatts }
            .filter { it.value > 0 }

        val maxWattsDateByYear = activitiesByYear
            .mapValues { (_, activities) -> maxWattsDate(activities, deviceOnly = false) }
            .filter { it.value.isNotBlank() }

        val deviceAverageWattsByYear = yearlyAccumulators
            .mapValues { (_, stats) ->
                if (stats.deviceAverageWattsCount == 0) 0 else stats.deviceAverageWattsSum / stats.deviceAverageWattsCount
            }
            .filter { it.value > 0 }

        val deviceMaxWattsByYear = yearlyAccumulators
            .mapValues { (_, stats) -> stats.deviceMaxWatts }
            .filter { it.value > 0 }

        val deviceMaxWattsDateByYear = activitiesByYear
            .mapValues { (_, activities) -> maxWattsDate(activities, deviceOnly = true) }
            .filter { it.value.isNotBlank() }

        return DashboardData(
            nbActivitiesByYear,
            activeDaysByYear,
            consistencyByYear,
            movingTimeByYear,
            totalDistanceByYear,
            averageDistanceByYear,
            maxDistanceByYear,
            maxDistanceDateByYear,
            averageDistanceByActiveDayByYear,
            maxDistanceByActiveDayByYear,
            maxDistanceByActiveDayDateByYear,
            totalElevationByYear,
            averageElevationByYear,
            maxElevationByYear,
            maxElevationDateByYear,
            averageElevationByActiveDayByYear,
            maxElevationByActiveDayByYear,
            maxElevationByActiveDayDateByYear,
            elevationEfficiencyByYear,
            averageSpeedByYear,
            maxSpeedByYear,
            maxSpeedDateByYear,
            averageHeartRateByYear,
            maxHeartRateByYear,
            maxHeartRateDateByYear,
            averageWattsByYear,
            maxWattsByYear,
            maxWattsDateByYear,
            deviceAverageWattsByYear,
            deviceMaxWattsByYear,
            deviceMaxWattsDateByYear,
        )
    }

    /**
     * Build a daily training heatmap per year.
     * Returns a map: year → (MM-DD → distance/elevation/duration and detailed activities for that day).
     */
    override fun getActivityHeatmap(activityTypes: Set<ActivityType>): Map<String, Map<String, ActivityHeatmapDay>> {
        logger.info("Get activity heatmap for activity type $activityTypes")
        return getCumulativeDataPerYear(activityTypes) { activitiesByDay ->
            activitiesByDay.mapValues { (_, dayActivities) ->
                val details = dayActivities.map { activity ->
                    val durationSec = if (activity.movingTime > 0) activity.movingTime else activity.elapsedTime
                    ActivityHeatmapActivity(
                        id = activity.id,
                        name = activity.name,
                        type = activity.sportType,
                        distanceKm = roundOneDecimal(activity.distance / 1000.0),
                        elevationGainM = roundOneDecimal(activity.totalElevationGain),
                        durationSec = durationSec,
                    )
                }
                val distanceKm = dayActivities.sumOf { it.distance / 1000.0 }
                val elevationGainM = dayActivities.sumOf { it.totalElevationGain }
                val durationSec = dayActivities.sumOf { if (it.movingTime > 0) it.movingTime else it.elapsedTime }

                ActivityHeatmapDay(
                    distanceKm = roundOneDecimal(distanceKm),
                    elevationGainM = roundOneDecimal(elevationGainM),
                    durationSec = durationSec,
                    activityCount = dayActivities.size,
                    activities = details,
                )
            }
        }
    }

    private fun aggregateYear(activities: List<StravaActivity>): YearAccumulator {
        val stats = YearAccumulator()
        for (activity in activities) {
            val distanceKm = activity.distance / 1000
            stats.nbActivities++
            stats.totalDistanceKm += distanceKm
            stats.maxDistanceKm = maxOf(stats.maxDistanceKm, distanceKm)
            stats.totalElevation += activity.totalElevationGain
            stats.maxElevation = maxOf(stats.maxElevation, activity.totalElevationGain.toInt())

            if (activity.averageSpeed > 0.0) {
                stats.speedCount++
                stats.speedSum += activity.averageSpeed
            }
            if (activity.averageHeartrate > 0.0) {
                stats.averageHeartRateCount++
                stats.averageHeartRateSum += activity.averageHeartrate
            }
            stats.maxHeartRate = maxOf(stats.maxHeartRate, activity.maxHeartrate)

            if (activity.averageWatts > 0) {
                stats.averageWattsCount++
                stats.averageWattsSum += activity.averageWatts
            }
            stats.maxWatts = maxOf(stats.maxWatts, activity.averageWatts)
            if (activity.deviceWatts && activity.averageWatts > 0) {
                stats.deviceAverageWattsCount++
                stats.deviceAverageWattsSum += activity.averageWatts
                stats.deviceMaxWatts = maxOf(stats.deviceMaxWatts, activity.averageWatts)
            }
        }
        return stats
    }

    private fun <T> getCumulativeDataPerYear(
        activityTypes: Set<ActivityType>,
        calculate: (Map<String, List<StravaActivity>>) -> Map<String, T>,
    ): Map<String, Map<String, T>> {
        val activitiesByYear = activityProvider.getActivitiesByActivityTypeAndYear(activityTypes)
            .withoutDataQualityExcludedStats(activityProvider)
            .groupByValidYear("cumulative dashboard data")
        return (StravaActivityProvider.STRAVA_FIRST_YEAR..LocalDate.now().year).mapNotNull { year ->
            activitiesByYear[year.toString()]?.let { activities ->
                val activitiesByDay = groupActivitiesByDay(activities, year)
                year.toString() to calculate(activitiesByDay)
            }
        }.toMap()
    }

    /**
     * Calculate the cumulative distance for each stravaActivity
     * @param activities list of activities
     * @return a map with the stravaActivity id as key and the cumulative distance as value
     * @see StravaActivity
     */
    private fun cumulativeDistance(activities: Map<String, List<StravaActivity>>): Map<String, Double> {
        var sum = 0.0
        return activities.mapValues { (_, activities) ->
            sum += activities.sumOf { activity -> activity.distance / 1000 }
            sum
        }
    }

    private fun cumulativeElevation(activities: Map<String, List<StravaActivity>>): Map<String, Int> {
        var sum = 0
        return activities.mapValues { (_, activities) ->
            sum += activities.sumOf { activity -> activity.totalElevationGain.toInt() }
            sum
        }
    }

    private fun List<StravaActivity>.toEddingtonValues(metric: EddingtonMetric, basis: EddingtonBasis): List<Int> {
        if (basis == EddingtonBasis.ACTIVITIES) {
            return this.map { activity -> activity.eddingtonValue(metric) }
        }
        return this
            .groupBy { activity -> activity.startDateLocal.substringBefore('T') }
            .values
            .map { activities -> activities.sumOf { activity -> activity.eddingtonValue(metric) } }
    }

    private fun StravaActivity.eddingtonValue(metric: EddingtonMetric): Int {
        return when (metric) {
            EddingtonMetric.ELEVATION -> (totalElevationGain / metric.thresholdScale).toInt()
            EddingtonMetric.DISTANCE -> (distance / 1000).toInt()
        }
    }

    private fun roundOneDecimal(value: Double): Double = round(value * 10.0) / 10.0

    private fun List<StravaActivity>.groupByValidYear(context: String): Map<String, List<StravaActivity>> {
        return mapNotNull { activity ->
            activity.activityYearOrNull()?.let { year -> year.toString() to activity }
                ?: run {
                    logger.warn("Skipping activity {} with invalid start date while grouping {}", activity.id, context)
                    null
                }
        }.groupBy(keySelector = { (year, _) -> year }, valueTransform = { (_, activity) -> activity })
    }

    private fun countActiveDays(activities: List<StravaActivity>): Int {
        return activities.mapNotNull { activity ->
            activity.startDateLocal.takeIf { it.length >= 10 }?.substring(0, 10)
        }.toSet().size
    }

    private fun distanceTotalsByActiveDay(activities: List<StravaActivity>): Map<String, Double> {
        return activities
            .filter { activity -> activity.startDateLocal.length >= 10 }
            .groupBy { activity -> activity.startDateLocal.substring(0, 10) }
            .mapValues { (_, dayActivities) -> dayActivities.sumOf { activity -> activity.distance / 1000.0 } }
    }

    private fun elevationTotalsByActiveDay(activities: List<StravaActivity>): Map<String, Int> {
        return activities
            .filter { activity -> activity.startDateLocal.length >= 10 }
            .groupBy { activity -> activity.startDateLocal.substring(0, 10) }
            .mapValues { (_, dayActivities) -> dayActivities.sumOf { activity -> activity.totalElevationGain.toInt() } }
    }

    private fun maxDistanceDate(activities: List<StravaActivity>): String {
        return activities.maxByOrNull { activity -> activity.distance }?.activityDate() ?: ""
    }

    private fun maxElevationDate(activities: List<StravaActivity>): String {
        return activities.maxByOrNull { activity -> activity.totalElevationGain }?.activityDate() ?: ""
    }

    private fun maxHeartRateDate(activities: List<StravaActivity>): String {
        return activities
            .filter { activity -> activity.maxHeartrate > 0 }
            .sortedBy { activity -> activity.activityDate() }
            .maxByOrNull { activity -> activity.maxHeartrate }
            ?.activityDate() ?: ""
    }

    private fun maxWattsDate(activities: List<StravaActivity>, deviceOnly: Boolean): String {
        return activities
            .filter { activity -> (!deviceOnly || activity.deviceWatts) && activity.averageWatts > 0 }
            .sortedBy { activity -> activity.activityDate() }
            .maxByOrNull { activity -> activity.averageWatts }
            ?.activityDate() ?: ""
    }

    private fun averageDoubleValues(values: Map<String, Double>): Double {
        return if (values.isEmpty()) 0.0 else values.values.sum() / values.size
    }

    private fun maxDoubleValue(values: Map<String, Double>): Double {
        return values.values.maxOrNull() ?: 0.0
    }

    private fun maxDoubleValueKey(values: Map<String, Double>): String {
        return values.entries
            .sortedBy { entry -> entry.key }
            .maxByOrNull { entry -> entry.value }
            ?.key ?: ""
    }

    private fun averageIntValues(values: Map<String, Int>): Int {
        return if (values.isEmpty()) 0 else values.values.sum() / values.size
    }

    private fun maxIntValue(values: Map<String, Int>): Int {
        return values.values.maxOrNull() ?: 0
    }

    private fun maxIntValueKey(values: Map<String, Int>): String {
        return values.entries
            .sortedBy { entry -> entry.key }
            .maxByOrNull { entry -> entry.value }
            ?.key ?: ""
    }

    private fun maxSpeedActivity(activities: List<StravaActivity>): StravaActivity? {
        return activities
            .filter(::isPlausibleActivityMaxSpeed)
            .maxByOrNull { activity -> activity.maxSpeed }
    }

    private fun StravaActivity.activityDate(): String {
        return startDateLocal.takeIf { it.length >= 10 }?.substring(0, 10) ?: ""
    }

    private fun sumMovingTimeSeconds(activities: List<StravaActivity>): Int {
        return activities.sumOf { activity ->
            activityMovingTimeSeconds(activity)
        }
    }

    private fun activityMovingTimeSeconds(activity: StravaActivity): Int {
        return if (activity.movingTime > 0) activity.movingTime else activity.elapsedTime
    }

    private fun StravaActivity.localDate(): LocalDate? {
        val value = startDateLocal.takeIf { it.length >= 10 } ?: startDate.takeIf { it.length >= 10 } ?: return null
        return runCatching { LocalDate.parse(value.substring(0, 10)) }.getOrNull()
    }

    private fun computeConsistencyByYear(year: String, activeDays: Int): Float {
        val yearNumber = year.toIntOrNull() ?: return 0F
        if (activeDays <= 0) return 0F
        val now = LocalDate.now()
        val scopeDays = if (yearNumber == now.year) now.dayOfYear else if (isLeapYear(yearNumber)) 366 else 365
        if (scopeDays <= 0) return 0F
        val ratio = (activeDays.toDouble() / scopeDays.toDouble()) * 100.0
        return (round(ratio * 10.0) / 10.0).toFloat()
    }

    private fun isLeapYear(year: Int): Boolean {
        if (year % 400 == 0) return true
        if (year % 100 == 0) return false
        return year % 4 == 0
    }
}
