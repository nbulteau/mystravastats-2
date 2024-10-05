package me.nicolas.stravastats.domain.services

import me.nicolas.stravastats.domain.business.EddingtonNumber
import me.nicolas.stravastats.domain.business.Period
import me.nicolas.stravastats.domain.business.strava.Activity
import me.nicolas.stravastats.domain.business.strava.ActivityType
import org.slf4j.LoggerFactory
import org.springframework.stereotype.Component
import java.time.LocalDate

interface IChartsService {
    fun getDistanceByPeriodByActivityTypeByYear(
        activityType: ActivityType,
        year: Int,
        period: Period,
    ): List<Pair<String, Double>>

    fun getElevationByPeriodByActivityTypeByYear(
        activityType: ActivityType,
        year: Int,
        period: Period,
    ): List<Pair<String, Double>>

    fun getAverageSpeedByPeriodByActivityTypeByYear(
        activityType: ActivityType,
        year: Int,
        period: Period
    ): List<Pair<String, Double>>

    fun getCumulativeDistancePerYear(activityType: ActivityType): Map<String, Map<String, Double>>

    fun getEddingtonNumber(activityType: ActivityType): EddingtonNumber
}

@Component
internal class ChartsService(
    stravaProxy: StravaProxy,
) : IChartsService, AbstractStravaService(stravaProxy) {

    private val logger = LoggerFactory.getLogger(ChartsService::class.java)

    /**
     * Get distance by period by activity type by year.
     * It returns a list of pair with the period as key and the distance in km as value.
     * @param activityType the activity type
     * @param year the year
     * @param period the period (days, weeks or months)
     * @return a list of pair with the period as key and the distance in km as value
     */
    override fun getDistanceByPeriodByActivityTypeByYear(
        activityType: ActivityType,
        year: Int,
        period: Period,
    ): List<Pair<String, Double>> {
        logger.info("Get distance by $period by activity ($activityType) type by year ($year)")

        val activitiesByPeriod = this.activitiesByPeriod(activityType, year, period)
        return activitiesByPeriod.mapValues { (_, activities) ->
            activities.sumOf { activity ->
                activity.distance / 1000
            }
        }.toList()
    }

    /**
     * Get elevation by period by activity type by year.
     * It returns a list of pair with the period as key and the elevation in meters as value.
     * @param activityType the activity type
     * @param year the year
     * @param period the period (days, weeks or months)
     * @return a list of pair with the period as key and the elevation in meters as value
     */
    override fun getElevationByPeriodByActivityTypeByYear(
        activityType: ActivityType,
        year: Int,
        period: Period,
    ): List<Pair<String, Double>> {
        logger.info("Get elevation by $period by activity ($activityType) type by year ($year)")

        val activitiesByPeriod = activitiesByPeriod(activityType, year, period)
        return activitiesByPeriod.mapValues { (_, activities) ->
            activities.sumOf { activity ->
                activity.totalElevationGain
            }
        }.toList()
    }

    /**
     * Get average speed by period by activity type by year.
     * It returns a list of pair with the period as key and the average speed in km/h as value.
     * @param activityType the activity type
     * @param year the year
     * @param period the period (days, weeks or months)
     * @return a list of pair with the period as key and the average speed in km/h as value
     */
    override fun getAverageSpeedByPeriodByActivityTypeByYear(
        activityType: ActivityType,
        year: Int,
        period: Period
    ): List<Pair<String, Double>> {
        logger.info("Get average speed by $period by activity ($activityType) type by year ($year)")

        val activitiesByPeriod = activitiesByPeriod(activityType, year, period)
        return activitiesByPeriod.mapValues { (_, activities) ->
            if (activities.isEmpty()) {
                0.0
            } else {
                activities.sumOf { activity -> activity.averageSpeed } / activities.size
            }
        }.toList()
    }

    /**
     * Get cumulative distance per year for a specific activity type.
     * It returns a map with the year as key and the cumulative distance in km as value.
     * @param activityType the activity type
     * @return a map with the year as key and the cumulative distance in km as value
     */
    override fun getCumulativeDistancePerYear(activityType: ActivityType): Map<String, Map<String, Double>> {
        logger.info("Get cumulative distance per year for activity type $activityType")

        val activitiesByYear = stravaProxy.getActivitiesByActivityTypeGroupByYear(activityType)

        return (2010..LocalDate.now().year).mapNotNull { year ->
            val cumulativeDistance = if (activitiesByYear[year.toString()] != null) {
                val activitiesByDay = ActivityHelper.groupActivitiesByDay(activitiesByYear[year.toString()]!!, year)
                ActivityHelper.cumulativeDistance(activitiesByDay)
            } else {
                null
            }
            cumulativeDistance?.let { year.toString() to it }
        }.toMap()
    }

    /**
     * Get the Eddington number for a specific activity type.
     * @param activityType the activity type
     * @return the Eddington number structure
     */
    override fun getEddingtonNumber(activityType: ActivityType): EddingtonNumber {
        logger.info("Get Eddington number for activity type $activityType")

        val activitiesByActiveDays = stravaProxy.getActivitiesByActivityTypeGroupByActiveDays(activityType)

        val eddingtonList: List<Int> = if (activitiesByActiveDays.isEmpty()) {
            emptyList()
        } else {
            val counts = IntArray(activitiesByActiveDays.maxOf { entry -> entry.value }) { 0 }.toMutableList()
            if (counts.isNotEmpty()) {
                // counts = number of time we reach a distance
                activitiesByActiveDays.forEach { entry: Map.Entry<String, Int> ->
                    for (day in entry.value downTo 1) {
                        counts[day - 1] += 1
                    }
                }
            }
            counts
        }

        var eddingtonNumber = 0
        for (day in eddingtonList.size downTo 1) {
            if (eddingtonList[day - 1] >= day) {
                eddingtonNumber = day
                break
            }
        }

        return EddingtonNumber(eddingtonNumber, eddingtonList)
    }

    /**
     * Get filtered activities by activity type, year and period.
     * @param activityType the activity type
     * @param year the year
     * @param period the period
     * @return a map with the period as key and the list of activities as value
     */
    private fun activitiesByPeriod(
        activityType: ActivityType,
        year: Int,
        period: Period,
    ): Map<String, List<Activity>> {
        val filteredActivities = stravaProxy.getFilteredActivitiesByActivityTypeAndYear(activityType, year)

        val activitiesByPeriod = when (period) {
            Period.MONTHS -> ActivityHelper.groupActivitiesByMonth(filteredActivities)
            Period.WEEKS -> ActivityHelper.groupActivitiesByWeek(filteredActivities)
            Period.DAYS -> ActivityHelper.groupActivitiesByDay(filteredActivities, year)
        }
        return activitiesByPeriod
    }
}