package me.nicolas.stravastats.domain.services

import me.nicolas.stravastats.domain.business.ActivityType
import me.nicolas.stravastats.domain.business.DashboardData
import me.nicolas.stravastats.domain.business.EddingtonNumber
import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.services.ActivityHelper.groupActivitiesByDay
import me.nicolas.stravastats.domain.services.activityproviders.IActivityProvider
import me.nicolas.stravastats.domain.services.statistics.BestEffortDistanceStatistic
import org.slf4j.LoggerFactory
import org.springframework.stereotype.Service
import java.time.LocalDate
import kotlin.math.round

interface IDashboardService {
    fun getCumulativeDistancePerYear(activityTypes: Set<ActivityType>): Map<String, Map<String, Number>>

    fun getCumulativeElevationPerYear(activityTypes: Set<ActivityType>): Map<String, Map<String, Number>>

    fun getEddingtonNumber(activityTypes: Set<ActivityType>): EddingtonNumber

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

        val activitiesByActiveDays = activityProvider.getActivitiesByActivityTypeGroupByActiveDays(activityTypes)
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
            .groupBy { activity -> activity.startDateLocal.subSequence(0, 4).toString() }

        val yearlyAccumulators = activitiesByYear.mapValues { (_, activities) ->
            aggregateYear(activities)
        }

        val nbActivitiesByYear = yearlyAccumulators
            .mapValues { (_, stats) -> stats.nbActivities }
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
            totalDistanceByYear,
            averageDistanceByYear,
            maxDistanceByYear,
            totalElevationByYear,
            averageElevationByYear,
            maxElevationByYear,
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
        val activitiesByYear = activityProvider.getActivitiesByActivityTypeGroupByYear(activityTypes)
        return (2010..LocalDate.now().year).mapNotNull { year ->
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
}
