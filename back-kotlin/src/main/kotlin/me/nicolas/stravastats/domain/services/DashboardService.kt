package me.nicolas.stravastats.domain.services

import me.nicolas.stravastats.domain.business.ActivityType
import me.nicolas.stravastats.domain.business.AnnualGoalMonth
import me.nicolas.stravastats.domain.business.AnnualGoalMetric
import me.nicolas.stravastats.domain.business.AnnualGoalProgress
import me.nicolas.stravastats.domain.business.AnnualGoalStatus
import me.nicolas.stravastats.domain.business.AnnualGoalTargets
import me.nicolas.stravastats.domain.business.AnnualGoals
import me.nicolas.stravastats.domain.business.DashboardData
import me.nicolas.stravastats.domain.business.EddingtonNumber
import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.services.ActivityHelper.groupActivitiesByDay
import me.nicolas.stravastats.domain.services.activityproviders.ActivityProviderCacheIdentity
import me.nicolas.stravastats.domain.services.activityproviders.IActivityProvider
import me.nicolas.stravastats.domain.services.activityproviders.StravaActivityProvider
import me.nicolas.stravastats.domain.services.statistics.BestEffortDistanceStatistic
import org.slf4j.LoggerFactory
import org.springframework.stereotype.Service
import tools.jackson.databind.DeserializationFeature
import tools.jackson.databind.json.JsonMapper
import tools.jackson.module.kotlin.KotlinModule
import java.io.File
import java.time.LocalDate
import java.time.temporal.ChronoUnit
import kotlin.math.round
import kotlin.math.roundToInt

interface IDashboardService {
    fun getCumulativeDistancePerYear(activityTypes: Set<ActivityType>): Map<String, Map<String, Number>>

    fun getCumulativeElevationPerYear(activityTypes: Set<ActivityType>): Map<String, Map<String, Number>>

    fun getEddingtonNumber(activityTypes: Set<ActivityType>): EddingtonNumber

    fun getDashboardData(activityTypes: Set<ActivityType>): DashboardData

    fun getActivityHeatmap(activityTypes: Set<ActivityType>): Map<String, Map<String, ActivityHeatmapDay>>

    fun getAnnualGoals(year: Int, activityTypes: Set<ActivityType>): AnnualGoals

    fun saveAnnualGoals(year: Int, activityTypes: Set<ActivityType>, targets: AnnualGoalTargets): AnnualGoals
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

    private val objectMapper = JsonMapper.builder()
        .addModule(KotlinModule.Builder().build())
        .disable(DeserializationFeature.FAIL_ON_NULL_FOR_PRIMITIVES)
        .build()

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
    )

    private data class AnnualGoalsCacheFile(
        val goals: Map<String, AnnualGoalTargets> = emptyMap(),
    )

    private data class AnnualGoalMetricDefinition(
        val metric: AnnualGoalMetric,
        val label: String,
        val unit: String,
        val requiredPaceUnit: String,
        val current: Double,
        val target: Double?,
        val monthlyValues: List<Double>,
        val last30DaysValue: Double,
    )

    private data class AnnualGoalCurrentValues(
        val distanceKm: Double,
        val elevationMeters: Double,
        val activities: Double,
        val activeDays: Double,
        val eddington: Double,
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
    override fun getEddingtonNumber(activityTypes: Set<ActivityType>): EddingtonNumber {
        logger.info("Get Eddington number for activity type $activityTypes")

        val excludedActivityIds = dataQualityExcludedActivityIds(activityProvider)
        val activitiesByActiveDays = if (excludedActivityIds.isEmpty()) {
            activityProvider.getActivitiesByActivityTypeGroupByActiveDays(activityTypes)
        } else {
            activityProvider.getActivitiesByActivityTypeAndYear(activityTypes)
                .withoutDataQualityExcludedStats(activityProvider)
                .groupBy { activity -> activity.startDateLocal.substringBefore('T') }
                .mapValues { (_, activities) -> activities.sumOf { activity -> activity.distance / 1000 }.roundToInt() }
        }
        val eddingtonList = computeEddingtonListFromDailyTotals(activitiesByActiveDays.values)

        var eddingtonNumber = 0
        for (day in eddingtonList.size downTo 1) {
            if (eddingtonList[day - 1] >= day) {
                eddingtonNumber = day
                break
            }
        }

        return EddingtonNumber(eddingtonNumber, eddingtonList)
    }

    override fun getAnnualGoals(year: Int, activityTypes: Set<ActivityType>): AnnualGoals {
        logger.info("Get annual goals for year $year and activity type $activityTypes")
        val targets = loadAnnualGoalTargets(year, activityTypes).normalize()
        val activities = activityProvider.getActivitiesByActivityTypeAndYear(activityTypes, year)
            .withoutDataQualityExcludedStats(activityProvider)
        return buildAnnualGoals(year, activityTypeKey(activityTypes), targets, activities, LocalDate.now())
    }

    override fun saveAnnualGoals(
        year: Int,
        activityTypes: Set<ActivityType>,
        targets: AnnualGoalTargets,
    ): AnnualGoals {
        logger.info("Save annual goals for year $year and activity type $activityTypes")
        val normalizedTargets = targets.normalize()
        saveAnnualGoalTargets(year, activityTypes, normalizedTargets)
        val activities = activityProvider.getActivitiesByActivityTypeAndYear(activityTypes, year)
            .withoutDataQualityExcludedStats(activityProvider)
        return buildAnnualGoals(year, activityTypeKey(activityTypes), normalizedTargets, activities, LocalDate.now())
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
            .groupBy { activity -> activity.startDateLocal.subSequence(0, 4).toString() }

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

        val maxSpeedByYear = activitiesByYear
            .mapValues { (_, activities) ->
                (BestEffortDistanceStatistic("", activities, 200.0).getSpeed())!!.toFloat()
            }
            .filter { entry -> entry.value > 0.0 }

        val averageHeartRateByYear = yearlyAccumulators
            .mapValues { (_, stats) ->
                if (stats.averageHeartRateCount == 0) 0 else (stats.averageHeartRateSum / stats.averageHeartRateCount).toInt()
            }
            .filter { entry -> entry.value > 0 }

        val maxHeartRateByYear = yearlyAccumulators
            .mapValues { (_, stats) -> stats.maxHeartRate }
            .filter { it.value > 0 }

        val averageWattsByYear = yearlyAccumulators
            .mapValues { (_, stats) ->
                if (stats.averageWattsCount == 0) 0 else stats.averageWattsSum / stats.averageWattsCount
            }
            .filter { it.value > 0 }

        val maxWattsByYear = yearlyAccumulators
            .mapValues { (_, stats) -> stats.maxWatts }
            .filter { it.value > 0 }

        return DashboardData(
            nbActivitiesByYear,
            activeDaysByYear,
            consistencyByYear,
            movingTimeByYear,
            totalDistanceByYear,
            averageDistanceByYear,
            maxDistanceByYear,
            totalElevationByYear,
            averageElevationByYear,
            maxElevationByYear,
            elevationEfficiencyByYear,
            averageSpeedByYear,
            maxSpeedByYear,
            averageHeartRateByYear,
            maxHeartRateByYear,
            averageWattsByYear,
            maxWattsByYear
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
        }
        return stats
    }

    private fun <T> getCumulativeDataPerYear(
        activityTypes: Set<ActivityType>,
        calculate: (Map<String, List<StravaActivity>>) -> Map<String, T>,
    ): Map<String, Map<String, T>> {
        val activitiesByYear = activityProvider.getActivitiesByActivityTypeAndYear(activityTypes)
            .withoutDataQualityExcludedStats(activityProvider)
            .groupBy { activity -> activity.startDateLocal.subSequence(0, 4).toString() }
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

    private fun roundOneDecimal(value: Double): Double = round(value * 10.0) / 10.0

    private fun countActiveDays(activities: List<StravaActivity>): Int {
        return activities.mapNotNull { activity ->
            activity.startDateLocal.takeIf { it.length >= 10 }?.substring(0, 10)
        }.toSet().size
    }

    private fun sumMovingTimeSeconds(activities: List<StravaActivity>): Int {
        return activities.sumOf { activity ->
            activityMovingTimeSeconds(activity)
        }
    }

    private fun activityMovingTimeSeconds(activity: StravaActivity): Int {
        return if (activity.movingTime > 0) activity.movingTime else activity.elapsedTime
    }

    private fun StravaActivity.annualGoalDate(): LocalDate? {
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

    private fun buildAnnualGoals(
        year: Int,
        activityTypeKey: String,
        targets: AnnualGoalTargets,
        activities: List<StravaActivity>,
        today: LocalDate,
    ): AnnualGoals {
        val current = annualGoalCurrentValues(activities)
        val monthlyValues = annualGoalMonthlyValues(activities)
        val (last30DaysValues, last30DaysWindowDays) = annualGoalLast30DaysValues(year, activities, today)
        val definitions = listOf(
            AnnualGoalMetricDefinition(
                metric = AnnualGoalMetric.DISTANCE_KM,
                label = "Distance",
                unit = "km",
                requiredPaceUnit = "km/day",
                current = current.distanceKm,
                target = targets.distanceKm,
                monthlyValues = monthlyValues.getValue(AnnualGoalMetric.DISTANCE_KM),
                last30DaysValue = last30DaysValues.distanceKm,
            ),
            AnnualGoalMetricDefinition(
                metric = AnnualGoalMetric.ELEVATION_METERS,
                label = "Elevation",
                unit = "m",
                requiredPaceUnit = "m/day",
                current = current.elevationMeters,
                target = targets.elevationMeters?.toDouble(),
                monthlyValues = monthlyValues.getValue(AnnualGoalMetric.ELEVATION_METERS),
                last30DaysValue = last30DaysValues.elevationMeters,
            ),
            AnnualGoalMetricDefinition(
                metric = AnnualGoalMetric.ACTIVITIES,
                label = "Activities",
                unit = "activities",
                requiredPaceUnit = "activities/day",
                current = current.activities,
                target = targets.activities?.toDouble(),
                monthlyValues = monthlyValues.getValue(AnnualGoalMetric.ACTIVITIES),
                last30DaysValue = last30DaysValues.activities,
            ),
            AnnualGoalMetricDefinition(
                metric = AnnualGoalMetric.ACTIVE_DAYS,
                label = "Active days",
                unit = "days",
                requiredPaceUnit = "days/day",
                current = current.activeDays,
                target = targets.activeDays?.toDouble(),
                monthlyValues = monthlyValues.getValue(AnnualGoalMetric.ACTIVE_DAYS),
                last30DaysValue = last30DaysValues.activeDays,
            ),
            AnnualGoalMetricDefinition(
                metric = AnnualGoalMetric.EDDINGTON,
                label = "Eddington",
                unit = "level",
                requiredPaceUnit = "level/day",
                current = current.eddington,
                target = targets.eddington?.toDouble(),
                monthlyValues = monthlyValues.getValue(AnnualGoalMetric.EDDINGTON),
                last30DaysValue = last30DaysValues.eddington,
            ),
        )

        return AnnualGoals(
            year = year,
            activityTypeKey = activityTypeKey,
            targets = targets,
            progress = definitions.map { definition ->
                buildAnnualGoalProgress(year, today, definition, last30DaysWindowDays)
            },
        )
    }

    private fun annualGoalCurrentValues(activities: List<StravaActivity>): AnnualGoalCurrentValues {
        val dailyDistanceTotals = activities
            .filter { activity -> activity.startDateLocal.length >= 10 }
            .groupBy { activity -> activity.startDateLocal.substringBefore('T') }
            .mapValues { (_, dayActivities) -> dayActivities.sumOf { activity -> (activity.distance / 1000).toInt() } }
        val eddingtonList = computeEddingtonListFromDailyTotals(dailyDistanceTotals.values)
        val eddington = (eddingtonList.size downTo 1)
            .firstOrNull { day -> eddingtonList[day - 1] >= day }
            ?: 0

        return AnnualGoalCurrentValues(
            distanceKm = activities.sumOf { activity -> activity.distance / 1000.0 },
            elevationMeters = activities.sumOf { activity -> activity.totalElevationGain },
            activities = activities.size.toDouble(),
            activeDays = countActiveDays(activities).toDouble(),
            eddington = eddington.toDouble(),
        )
    }

    private fun annualGoalMonthlyValues(activities: List<StravaActivity>): Map<AnnualGoalMetric, List<Double>> {
        val values = AnnualGoalMetric.entries.associateWith { MutableList(12) { 0.0 } }
        val activeDaysByMonth = List(12) { mutableSetOf<String>() }
        val dailyDistanceByMonth = List(12) { mutableMapOf<String, Int>() }

        activities.forEach { activity ->
            val activityDate = activity.annualGoalDate() ?: return@forEach
            val monthIndex = activityDate.monthValue - 1
            val day = activityDate.toString()
            values.getValue(AnnualGoalMetric.DISTANCE_KM)[monthIndex] += activity.distance / 1000.0
            values.getValue(AnnualGoalMetric.ELEVATION_METERS)[monthIndex] += activity.totalElevationGain
            values.getValue(AnnualGoalMetric.ACTIVITIES)[monthIndex] += 1.0
            activeDaysByMonth[monthIndex].add(day)
            dailyDistanceByMonth[monthIndex][day] =
                (dailyDistanceByMonth[monthIndex][day] ?: 0) + (activity.distance / 1000.0).toInt()
        }

        for (monthIndex in 0 until 12) {
            values.getValue(AnnualGoalMetric.ACTIVE_DAYS)[monthIndex] = activeDaysByMonth[monthIndex].size.toDouble()
            val eddingtonList = computeEddingtonListFromDailyTotals(dailyDistanceByMonth[monthIndex].values)
            val eddington = (eddingtonList.size downTo 1)
                .firstOrNull { day -> eddingtonList[day - 1] >= day }
                ?: 0
            values.getValue(AnnualGoalMetric.EDDINGTON)[monthIndex] = eddington.toDouble()
        }

        return values
    }

    private data class AnnualGoalTrendWindow(
        val start: LocalDate,
        val end: LocalDate,
        val days: Int,
    )

    private fun annualGoalLast30DaysValues(
        year: Int,
        activities: List<StravaActivity>,
        today: LocalDate,
    ): Pair<AnnualGoalCurrentValues, Int> {
        val window = annualGoalLast30DaysWindow(year, today) ?: return AnnualGoalCurrentValues(
            distanceKm = 0.0,
            elevationMeters = 0.0,
            activities = 0.0,
            activeDays = 0.0,
            eddington = 0.0,
        ) to 0

        val filtered = activities.filter { activity ->
            val activityDate = activity.annualGoalDate() ?: return@filter false
            !activityDate.isBefore(window.start) && !activityDate.isAfter(window.end)
        }
        return annualGoalCurrentValues(filtered) to window.days
    }

    private fun annualGoalLast30DaysWindow(year: Int, today: LocalDate): AnnualGoalTrendWindow? {
        if (year > today.year) return null
        val yearStart = LocalDate.of(year, 1, 1)
        val windowEnd = if (year == today.year) today else LocalDate.of(year, 12, 31)
        val windowStart = maxOf(yearStart, windowEnd.minusDays(29))
        val windowDays = ChronoUnit.DAYS.between(windowStart, windowEnd).toInt() + 1
        return AnnualGoalTrendWindow(windowStart, windowEnd, windowDays)
    }

    private fun buildAnnualGoalProgress(
        year: Int,
        today: LocalDate,
        definition: AnnualGoalMetricDefinition,
        last30DaysWindowDays: Int,
    ): AnnualGoalProgress {
        val elapsedDays = elapsedDaysForAnnualGoal(year, today)
        val remainingDays = remainingDaysForAnnualGoal(year, today)
        val expectedProgressPercent = annualExpectedProgressPercent(year, today)
        val projectedEndOfYear = if (year == today.year && elapsedDays > 0) {
            definition.current / elapsedDays.toDouble() * daysInYear(year).toDouble()
        } else {
            definition.current
        }
        val last30DaysWeeklyPace = if (last30DaysWindowDays > 0) {
            definition.last30DaysValue / last30DaysWindowDays.toDouble() * 7.0
        } else {
            0.0
        }

        val target = definition.target ?: 0.0
        if (target <= 0.0) {
            return AnnualGoalProgress(
                metric = definition.metric,
                label = definition.label,
                unit = definition.unit,
                current = roundAnnualGoalValue(definition.current),
                target = 0.0,
                progressPercent = 0.0,
                expectedProgressPercent = roundAnnualGoalValue(expectedProgressPercent),
                projectedEndOfYear = roundAnnualGoalValue(projectedEndOfYear),
                requiredPace = 0.0,
                requiredPaceUnit = definition.requiredPaceUnit,
                requiredWeeklyPace = 0.0,
                last30Days = roundAnnualGoalValue(definition.last30DaysValue),
                last30DaysWeeklyPace = roundAnnualGoalValue(last30DaysWeeklyPace),
                weeklyPaceGap = 0.0,
                suggestedTarget = null,
                monthly = buildAnnualGoalMonthlyProgress(year, definition.monthlyValues, 0.0),
                status = AnnualGoalStatus.NOT_SET,
            )
        }

        val progressPercent = definition.current / target * 100.0
        val requiredPace = if (remainingDays > 0) {
            maxOf(target - definition.current, 0.0) / remainingDays.toDouble()
        } else {
            0.0
        }
        val requiredWeeklyPace = requiredPace * 7.0
        val weeklyPaceGap = maxOf(requiredWeeklyPace - last30DaysWeeklyPace, 0.0)
        val suggestedTarget = suggestedAnnualGoalTarget(
            year = year,
            today = today,
            target = target,
            projectedEndOfYear = projectedEndOfYear,
            progressPercent = progressPercent,
            expectedProgressPercent = expectedProgressPercent,
        )

        return AnnualGoalProgress(
            metric = definition.metric,
            label = definition.label,
            unit = definition.unit,
            current = roundAnnualGoalValue(definition.current),
            target = roundAnnualGoalValue(target),
            progressPercent = roundAnnualGoalValue(progressPercent),
            expectedProgressPercent = roundAnnualGoalValue(expectedProgressPercent),
            projectedEndOfYear = roundAnnualGoalValue(projectedEndOfYear),
            requiredPace = roundAnnualGoalValue(requiredPace),
            requiredPaceUnit = definition.requiredPaceUnit,
            requiredWeeklyPace = roundAnnualGoalValue(requiredWeeklyPace),
            last30Days = roundAnnualGoalValue(definition.last30DaysValue),
            last30DaysWeeklyPace = roundAnnualGoalValue(last30DaysWeeklyPace),
            weeklyPaceGap = roundAnnualGoalValue(weeklyPaceGap),
            suggestedTarget = suggestedTarget,
            monthly = buildAnnualGoalMonthlyProgress(year, definition.monthlyValues, target),
            status = annualGoalStatus(progressPercent, expectedProgressPercent),
        )
    }

    private fun suggestedAnnualGoalTarget(
        year: Int,
        today: LocalDate,
        target: Double,
        projectedEndOfYear: Double,
        progressPercent: Double,
        expectedProgressPercent: Double,
    ): Double? {
        if (year != today.year || target <= 0.0 || projectedEndOfYear <= 0.0) return null
        if (progressPercent >= expectedProgressPercent - 5.0) return null
        if (projectedEndOfYear >= target * 0.9) return null
        return roundAnnualGoalValue(projectedEndOfYear)
    }

    private fun buildAnnualGoalMonthlyProgress(
        year: Int,
        monthlyValues: List<Double>,
        target: Double,
    ): List<AnnualGoalMonth> {
        var cumulative = 0.0
        return (1..12).map { month ->
            val value = monthlyValues.getOrElse(month - 1) { 0.0 }
            cumulative += value
            val expectedCumulative = if (target > 0.0) {
                target * LocalDate.of(year, month, 1).withDayOfMonth(
                    LocalDate.of(year, month, 1).lengthOfMonth()
                ).dayOfYear.toDouble() / daysInYear(year).toDouble()
            } else {
                0.0
            }
            AnnualGoalMonth(
                month = month,
                value = roundAnnualGoalValue(value),
                cumulative = roundAnnualGoalValue(cumulative),
                expectedCumulative = roundAnnualGoalValue(expectedCumulative),
            )
        }
    }

    private fun annualGoalStatus(progressPercent: Double, expectedProgressPercent: Double): AnnualGoalStatus {
        return when {
            progressPercent >= expectedProgressPercent + 5.0 -> AnnualGoalStatus.AHEAD
            progressPercent >= expectedProgressPercent - 5.0 -> AnnualGoalStatus.ON_TRACK
            else -> AnnualGoalStatus.BEHIND
        }
    }

    private fun elapsedDaysForAnnualGoal(year: Int, today: LocalDate): Int {
        return when {
            year < today.year -> daysInYear(year)
            year > today.year -> 0
            else -> today.dayOfYear
        }
    }

    private fun remainingDaysForAnnualGoal(year: Int, today: LocalDate): Int {
        return when {
            year < today.year -> 0
            year > today.year -> daysInYear(year)
            else -> daysInYear(year) - today.dayOfYear
        }
    }

    private fun annualExpectedProgressPercent(year: Int, today: LocalDate): Double {
        val elapsedDays = elapsedDaysForAnnualGoal(year, today)
        if (elapsedDays <= 0) return 0.0
        return elapsedDays.toDouble() / daysInYear(year).toDouble() * 100.0
    }

    private fun daysInYear(year: Int): Int = if (isLeapYear(year)) 366 else 365

    private fun loadAnnualGoalTargets(year: Int, activityTypes: Set<ActivityType>): AnnualGoalTargets {
        val file = annualGoalsFile()
        if (!file.exists()) {
            return AnnualGoalTargets()
        }
        return runCatching {
            objectMapper.readValue(file, AnnualGoalsCacheFile::class.java)
                .goals[annualGoalTargetsKey(year, activityTypes)]
                ?: AnnualGoalTargets()
        }.getOrElse { exception ->
            logger.warn("Unable to read annual goals from ${file.absolutePath}", exception)
            AnnualGoalTargets()
        }
    }

    private fun saveAnnualGoalTargets(year: Int, activityTypes: Set<ActivityType>, targets: AnnualGoalTargets) {
        val file = annualGoalsFile()
        val current = if (file.exists()) {
            runCatching {
                objectMapper.readValue(file, AnnualGoalsCacheFile::class.java)
            }.getOrElse { AnnualGoalsCacheFile() }
        } else {
            AnnualGoalsCacheFile()
        }
        val updated = AnnualGoalsCacheFile(
            goals = current.goals + (annualGoalTargetsKey(year, activityTypes) to targets),
        )
        file.parentFile.mkdirs()
        objectMapper.writerWithDefaultPrettyPrinter().writeValue(file, updated)
    }

    private fun annualGoalsFile(): File {
        val identity = activityProvider.cacheIdentity() ?: fallbackCacheIdentity()
        val athleteDirectory = File(identity.cacheRoot, "strava-${identity.athleteId}")
        return File(athleteDirectory, "annual-goals-${identity.athleteId}.json")
    }

    private fun fallbackCacheIdentity(): ActivityProviderCacheIdentity {
        val athleteId = runCatching { activityProvider.athlete().id.toString() }.getOrDefault("local")
        return ActivityProviderCacheIdentity(cacheRoot = "strava-cache", athleteId = athleteId)
    }

    private fun annualGoalTargetsKey(year: Int, activityTypes: Set<ActivityType>): String {
        return "$year:${activityTypeKey(activityTypes)}"
    }

    private fun activityTypeKey(activityTypes: Set<ActivityType>): String {
        return activityTypes.map { activityType -> activityType.name }.sorted().joinToString("_")
    }

    private fun AnnualGoalTargets.normalize(): AnnualGoalTargets {
        return AnnualGoalTargets(
            distanceKm = distanceKm?.takeIf { it > 0.0 },
            elevationMeters = elevationMeters?.takeIf { it > 0 },
            movingTimeSeconds = null,
            activities = activities?.takeIf { it > 0 },
            activeDays = activeDays?.takeIf { it > 0 },
            eddington = eddington?.takeIf { it > 0 },
        )
    }

    private fun roundAnnualGoalValue(value: Double): Double = round(value * 10.0) / 10.0
}
