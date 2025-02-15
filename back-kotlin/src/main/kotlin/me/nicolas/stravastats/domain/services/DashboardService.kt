package me.nicolas.stravastats.domain.services

import me.nicolas.stravastats.domain.business.ActivityType
import me.nicolas.stravastats.domain.business.DashboardData
import me.nicolas.stravastats.domain.business.EddingtonNumber
import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.services.ActivityHelper.groupActivitiesByDay
import me.nicolas.stravastats.domain.services.activityproviders.IActivityProvider
import org.slf4j.LoggerFactory
import org.springframework.stereotype.Service
import java.time.LocalDate
import java.time.ZoneOffset

interface IDashboardService {
    fun getCumulativeDistancePerYear(activityType: ActivityType): Map<String, Map<String, Number>>

    fun getCumulativeElevationPerYear(activityType: ActivityType): Map<String, Map<String, Number>>

    fun getEddingtonNumber(activityType: ActivityType): EddingtonNumber

    fun getDashboardData(activityType: ActivityType): DashboardData
}


@Service
class DashboardService(
    activityProvider: IActivityProvider,
) : IDashboardService, AbstractStravaService(activityProvider) {

    private val logger = LoggerFactory.getLogger(DashboardService::class.java)

    /**
     * Get cumulative distance per year for a specific stravaActivity type.
     * It returns a map with the year as key and the cumulative distance in km as value.
     * @param activityType the stravaActivity type
     * @return a map with the year as key and the cumulative distance in km as value
     */
    override fun getCumulativeDistancePerYear(activityType: ActivityType): Map<String, Map<String, Number>> {
        logger.info("Get cumulative distance per year for stravaActivity type $activityType")
        return getCumulativeDataPerYear(activityType) { activitiesByDay ->
            cumulativeDistance(activitiesByDay)
        }
    }

    override fun getCumulativeElevationPerYear(activityType: ActivityType): Map<String, Map<String, Number>> {
        logger.info("Get cumulative elevation per year for stravaActivity type $activityType")
        return getCumulativeDataPerYear(activityType) { activitiesByDay ->
            cumulativeElevation(activitiesByDay)
        }
    }

    /**
     * Get the Eddington number for a specific stravaActivity type.
     * @param activityType the stravaActivity type
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

    override fun getDashboardData(activityType: ActivityType): DashboardData {
        logger.info("Get dashboard data for activity type $activityType")

        val activitiesByYear = activityProvider.getActivitiesByActivityTypeAndYear(activityType)
            .groupBy { activity -> activity.startDateLocal.subSequence(0, 4).toString() }

        // compute nb of activities for all years
        val nbActivitiesByYear = activitiesByYear
            .mapValues { (_, activities) -> activities.size }
            .filter { it.value > 0 }

        // compute total distance for all years
        val totalDistanceByYear = activitiesByYear
            .mapValues { (_, activities) ->
                activities.sumOf { activity -> activity.distance / 1000 }.toFloat()
            }
            .filter { it.value > 0 }

        // compute average distance for all years
        val averageDistanceByYear = activitiesByYear
            .mapValues { (_, activities) ->
                (activities.sumOf { activity -> activity.distance / 1000 } / activities.size).toFloat()
            }
            .filter { it.value > 0 }

        // compute max distance for all years
        val maxDistanceByYear = activitiesByYear
            .mapValues { (_, activities) ->
                activities.maxOf { activity -> activity.distance / 1000 }.toFloat()
            }
            .filter { it.value > 0 }

        // compute total elevation for all years
        val totalElevationByYear = activitiesByYear
            .mapValues { (_, activities) -> activities.sumOf { activity -> activity.totalElevationGain.toInt() } }
            .filter { it.value > 0 }

        // compute average elevation for all years
        val averageElevationByYear = activitiesByYear
            .mapValues { (_, activities) ->
                (activities.sumOf { activity -> activity.totalElevationGain } / activities.size).toInt()
            }
            .filter { it.value > 0 }

        // compute max elevation for all years
        val maxElevationByYear = activitiesByYear
            .mapValues { (_, activities) ->
                activities.maxOf { activity -> activity.totalElevationGain.toInt() }
            }
            .filter { it.value > 0 }

        // compute average speed for all years
        val averageSpeedByYear = activitiesByYear
            .mapValues { (_, activities) ->
                val count = activities.count { it.averageSpeed > 0.0 }
                if (count == 0) return@mapValues 0F
                (activities.filter { it.averageSpeed > 0.0 }
                    .sumOf { activity -> activity.averageSpeed } / count).toFloat()
            }
            .filter { it.value > 0 }

        // compute max speed for all years
        val maxSpeedByYear = activitiesByYear
            .mapValues { (_, activities) ->
                activities.maxOf { activity -> activity.maxSpeed }
            }
            .filter { it.value > 0 }

        // compute average heart rate for all years
        val averageHeartRateByYear = activitiesByYear
            .mapValues { (_, activities) ->
                val count = activities.count { it.averageHeartrate > 0.0 }
                if (count == 0) return@mapValues 0
                (activities.filter { it.averageHeartrate > 0.0 }
                    .sumOf { activity -> activity.averageHeartrate } / count)
                    .toInt()
            }
            .filter { it.value > 0 }

        // compute max heart rate for all years
        val maxHeartRateByYear = activitiesByYear
            .mapValues { (_, activities) ->
                activities.maxOf { activity -> activity.maxHeartrate }
            }
            .filter { it.value > 0 }

        // compute average watts for all years
        val averageWattsByYear = activitiesByYear
            .mapValues { (_, activities) ->
                val count = activities.count { it.averageWatts > 0 }
                if (count == 0) return@mapValues 0
                (activities.filter { it.averageWatts > 0 }
                    .sumOf { activity -> activity.averageWatts } / count)
            }
            .filter { it.value > 0 }

        // compute max watts for all years
        val maxWattsByYear = activitiesByYear
            .mapValues { entry: Map.Entry<String, List<StravaActivity>> ->
                entry.value.maxOf { activity -> activity.averageWatts }
            }
            .filter { it.value > 0 }

        val filteredActivities = activityProvider.getActivitiesByActivityTypeAndYear(activityType)

        val averageCadence = filteredActivities
            .filter { activity -> activity.averageCadence > 0 }
            .groupBy { activity -> activity.startDateLocal.substringBefore('T') }
            .map { (day, activities) ->
                val milliseconds = LocalDate.parse(day).atStartOfDay().toInstant(ZoneOffset.UTC).toEpochMilli()
                val averageCadence =
                    (activities.sumOf { activity -> activity.averageCadence * 2 } / activities.size).toLong()
                listOf(milliseconds, averageCadence)
            }

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
            maxWattsByYear,
            averageCadence
        )
    }

    private fun getCumulativeDataPerYear(activityType: ActivityType, calculate: (Map<String, List<StravaActivity>>) -> Map<String, Number>): Map<String, Map<String, Number>> {
        val activitiesByYear = activityProvider.getActivitiesByActivityTypeGroupByYear(activityType)
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
}