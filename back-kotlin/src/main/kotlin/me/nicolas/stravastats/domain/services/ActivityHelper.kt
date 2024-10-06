package me.nicolas.stravastats.domain.services

import me.nicolas.stravastats.domain.business.strava.Activity
import me.nicolas.stravastats.domain.business.strava.ActivityType
import me.nicolas.stravastats.domain.utils.inDateTimeFormatter
import java.time.LocalDate
import java.time.LocalDateTime
import java.time.Month
import java.time.format.TextStyle
import java.time.temporal.WeekFields
import java.util.*

object ActivityHelper {
    
    /**
     * Group activities by year
     * @param activities list of activities
     * @return a map with the year as key and the list of activities as value
     * @see Activity
     */
    fun groupActivitiesByYear(activities: List<Activity>): Map<String, List<Activity>> {
        val activitiesByYear =
            activities.groupBy { activity -> activity.startDateLocal.subSequence(0, 4).toString() }.toMutableMap()

        // Add years without activities
        if (activitiesByYear.isNotEmpty()) {
            val min = activitiesByYear.keys.minOf { it.toInt() }
            val max = activitiesByYear.keys.maxOf { it.toInt() }
            for (year in min..max) {
                if (!activitiesByYear.contains("$year")) {
                    activitiesByYear["$year"] = emptyList()
                }
            }
        }
        return activitiesByYear.toSortedMap()
    }

    /**
     * Group activities by month
     * @param activities list of activities
     * @return a map with the month as key and the list of activities as value
     * @see Activity
     */
    fun groupActivitiesByMonth(activities: List<Activity>): Map<String, List<Activity>> {
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
    fun groupActivitiesByWeek(activities: List<Activity>): Map<String, List<Activity>> {

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
    fun groupActivitiesByDay(activities: List<Activity>, year: Int): Map<String, List<Activity>> {
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
    fun cumulativeDistance(activities: Map<String, List<Activity>>): Map<String, Double> {
        var sum = 0.0
        return activities.mapValues { (_, activities) ->
            sum += activities.sumOf { activity -> activity.distance / 1000 }; sum
        }
    }

    /**
     * Remove activities that are not in the list of activity types to consider (i.e. Run, Ride, Hike, etc.)
     * @return a list of activities filtered by type
     * @see Activity
     */
    fun List<Activity>.filterActivities() = this.filter { activity ->
        ActivityType.entries.any { activity.type == it.name }
    }
}

