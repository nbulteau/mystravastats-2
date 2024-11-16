package me.nicolas.stravastats.domain.services

import me.nicolas.stravastats.domain.business.ActivityType
import me.nicolas.stravastats.domain.business.Period
import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.services.ActivityHelper.groupActivitiesByDay
import me.nicolas.stravastats.domain.services.ActivityHelper.groupActivitiesByMonth
import me.nicolas.stravastats.domain.services.ActivityHelper.groupActivitiesByWeek
import me.nicolas.stravastats.domain.services.activityproviders.IActivityProvider
import org.slf4j.LoggerFactory
import org.springframework.stereotype.Component

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

    fun getAverageCadenceByPeriodByActivityTypeByYear(
        activityType: ActivityType,
    ): Map<String, Int>

}

@Component
internal class ChartsService(
    activityProvider: IActivityProvider,
) : IChartsService, AbstractStravaService(activityProvider) {

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
        logger.info("Get distance by $period by sctivity ($activityType) type by year ($year)")

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

    override fun getAverageCadenceByPeriodByActivityTypeByYear(
        activityType: ActivityType,
    ): Map<String, Int> {
        val filteredActivities = activityProvider.getActivitiesByActivityTypeAndYear(activityType)

        return filteredActivities
            .groupBy { activity -> activity.startDateLocal.substringBefore('T') }
            .mapValues { (_, activities) -> activities.sumOf { activity -> activity.averageCadence * 2 } }
            .mapValues { entry -> entry.value.toInt() }
            .toMap()
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
    ): Map<String, List<StravaActivity>> {
        val filteredActivities = activityProvider.getActivitiesByActivityTypeAndYear(activityType, year)

        val activitiesByPeriod = when (period) {
            Period.MONTHS -> groupActivitiesByMonth(filteredActivities)
            Period.WEEKS -> groupActivitiesByWeek(filteredActivities)
            Period.DAYS -> groupActivitiesByDay(filteredActivities, year)
        }
        return activitiesByPeriod
    }
}