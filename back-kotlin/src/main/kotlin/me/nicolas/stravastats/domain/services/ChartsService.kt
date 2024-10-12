package me.nicolas.stravastats.domain.services

import me.nicolas.stravastats.domain.business.EddingtonNumber
import me.nicolas.stravastats.domain.business.Period
import me.nicolas.stravastats.domain.business.strava.Activity
import me.nicolas.stravastats.domain.business.strava.ActivityType
import me.nicolas.stravastats.domain.services.activityproviders.IActivityProvider
import me.nicolas.stravastats.domain.utils.inDateTimeFormatter
import org.slf4j.LoggerFactory
import org.springframework.stereotype.Component
import java.time.LocalDate
import java.time.LocalDateTime
import java.time.Month
import java.time.format.TextStyle
import java.time.temporal.WeekFields
import java.util.*

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

        val activitiesByYear = activityProvider.getActivitiesByActivityTypeGroupByYear(activityType)

        return (2010..LocalDate.now().year).mapNotNull { year ->
            val cumulativeDistance = if (activitiesByYear[year.toString()] != null) {
                val activitiesByDay = groupActivitiesByDay(activitiesByYear[year.toString()]!!, year)
                cumulativeDistance(activitiesByDay)
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

        val activitiesByActiveDays = activityProvider.getActivitiesByActivityTypeGroupByActiveDays(activityType)

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
        val filteredActivities = activityProvider.getActivitiesByActivityTypeAndYear(activityType, year)

        val activitiesByPeriod = when (period) {
            Period.MONTHS -> groupActivitiesByMonth(filteredActivities)
            Period.WEEKS -> groupActivitiesByWeek(filteredActivities)
            Period.DAYS -> groupActivitiesByDay(filteredActivities, year)
        }
        return activitiesByPeriod
    }

    /**
     * Group activities by month
     * @param activities list of activities
     * @return a map with the month as key and the list of activities as value
     * @see Activity
     */
    private fun groupActivitiesByMonth(activities: List<Activity>): Map<String, List<Activity>> {
        val activitiesByMonth =
            activities.groupBy { activity -> activity.startDateLocal.subSequence(5, 7).toString() }.toMutableMap()

        // Add months without activities
        for (month in (1..12)) {
            if (!activitiesByMonth.contains("$month".padStart(2, '0'))) {
                activitiesByMonth["$month".padStart(2, '0')] = emptyList()
            }
        }

        return activitiesByMonth.toSortedMap().mapKeys { (key, _) ->
            Month.of(key.toInt()).getDisplayName(TextStyle.FULL_STANDALONE, Locale.getDefault())
        }.toMap()
    }

    /**
     * Group activities by week
     * @param activities list of activities
     * @return a map with the week as key and the list of activities as value
     * @see Activity
     */
    private fun groupActivitiesByWeek(activities: List<Activity>): Map<String, List<Activity>> {

        val activitiesByWeek = activities.groupBy { activity ->
            val week = LocalDateTime.parse(activity.startDateLocal, inDateTimeFormatter)
                .get(WeekFields.of(Locale.getDefault()).weekOfYear())
            "$week".padStart(2, '0')
        }.toMutableMap()

        // Add weeks without activities
        for (week in (1..52)) {
            if (!activitiesByWeek.contains("$week".padStart(2, '0'))) {
                activitiesByWeek["$week".padStart(2, '0')] = emptyList()
            }
        }

        return activitiesByWeek.toSortedMap()
    }

    /**
     * Group activities by day
     * @param activities list of activities
     * @return a map with the day as key and the list of activities as value
     * @see Activity
     */
    private fun groupActivitiesByDay(activities: List<Activity>, year: Int): Map<String, List<Activity>> {
        val activitiesByDay =
            activities.groupBy { activity -> activity.startDateLocal.subSequence(5, 10).toString() }.toMutableMap()

        // Add days without activities
        var currentDate = LocalDate.ofYearDay(year, 1)
        for (i in (0..365 + if (currentDate.isLeapYear) 1 else 0)) {
            currentDate = currentDate.plusDays(1L)
            val dayString =
                "${currentDate.monthValue}".padStart(2, '0') + "-" + "${currentDate.dayOfMonth}".padStart(2, '0')
            if (!activitiesByDay.containsKey(dayString)) {
                activitiesByDay[dayString] = emptyList()
            }
        }

        return activitiesByDay.toSortedMap()
    }

    /**
     * Calculate the cumulative distance for each activity
     * @param activities list of activities
     * @return a map with the activity id as key and the cumulative distance as value
     * @see Activity
     */
    private fun cumulativeDistance(activities: Map<String, List<Activity>>): Map<String, Double> {
        var sum = 0.0
        return activities.mapValues { (_, activities) ->
            sum += activities.sumOf { activity -> activity.distance / 1000 }; sum
        }
    }
}