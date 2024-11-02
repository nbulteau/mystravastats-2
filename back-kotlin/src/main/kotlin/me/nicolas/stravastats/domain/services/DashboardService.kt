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

interface IDashboardService {
    fun getCumulativeDistancePerYear(activityType: ActivityType): Map<String, Map<String, Double>>

    fun getCumulativeElevationPerYear(activityType: ActivityType): Map<String, Map<String, Int>>

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
    override fun getCumulativeDistancePerYear(activityType: ActivityType): Map<String, Map<String, Double>> {
        logger.info("Get cumulative distance per year for stravaActivity type $activityType")

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

    override fun getCumulativeElevationPerYear(activityType: ActivityType): Map<String, Map<String, Int>> {
        logger.info("Get cumulative elevation per year for stravaActivity type $activityType")

        val activitiesByYear = activityProvider.getActivitiesByActivityTypeGroupByYear(activityType)

        return (2010..LocalDate.now().year).mapNotNull { year ->
            val cumulativeElevation = if (activitiesByYear[year.toString()] != null) {
                val activitiesByDay = groupActivitiesByDay(activitiesByYear[year.toString()]!!, year)
                cumulativeElevation(activitiesByDay)
            } else {
                null
            }
            cumulativeElevation?.let { year.toString() to it }
        }.toMap()
    }

    /**
     * Get the Eddington number for a specific stravaActivity type.
     * @param activityType the stravaActivity type
     * @return the Eddington number structure
     */
    override fun getEddingtonNumber(activityType: ActivityType): EddingtonNumber {
        logger.info("Get Eddington number for stravaActivity type $activityType")

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
        logger.info("Get dashboard data")

        val activitiesByYear = activityProvider.getActivitiesByActivityTypeAndYear(activityType)

        // compute nb of activities for all years
        val nbActivitiesByYear =
            activitiesByYear.groupBy { activity -> activity.startDateLocal.subSequence(0, 4).toString() }
                .mapValues { (_, activities) -> activities.size }

        // compute total distance for all years
        val totalDistanceByYear =
            activitiesByYear.groupBy { activity -> activity.startDateLocal.subSequence(0, 4).toString() }
                .mapValues { (_, activities) ->
                    activities.sumOf { activity -> activity.distance / 1000 }.toFloat()
                }

        // compute average distance for all years
        val averageDistanceByYear =
            activitiesByYear.groupBy { activity -> activity.startDateLocal.subSequence(0, 4).toString() }
                .mapValues { (_, activities) ->
                    (activities.sumOf { activity -> activity.distance / 1000 } / activities.size).toFloat()
                }

        // compute max distance for all years
        val maxDistanceByYear =
            activitiesByYear.groupBy { activity -> activity.startDateLocal.subSequence(0, 4).toString() }
                .mapValues { (_, activities) ->
                    activities.maxOf { activity -> activity.distance / 1000 }.toFloat()
                }

        // compute total elevation for all years
        val totalElevationByYear =
            activitiesByYear.groupBy { activity -> activity.startDateLocal.subSequence(0, 4).toString() }
                .mapValues { (_, activities) -> activities.sumOf { activity -> activity.totalElevationGain.toInt() } }


        // compute average elevation for all years
        val averageElevationByYear =
            activitiesByYear.groupBy { activity -> activity.startDateLocal.subSequence(0, 4).toString() }
                .mapValues { (_, activities) ->
                    (activities.sumOf { activity -> activity.totalElevationGain } / activities.size).toInt()
                }

        // compute max elevation for all years
        val maxElevationByYear =
            activitiesByYear.groupBy { activity -> activity.startDateLocal.subSequence(0, 4).toString() }
                .mapValues { (_, activities) ->
                    activities.maxOf { activity -> activity.totalElevationGain.toInt() }
                }

        // compute average speed for all years
        val averageSpeedByYear =
            activitiesByYear.groupBy { activity -> activity.startDateLocal.subSequence(0, 4).toString() }
                .mapValues { (_, activities) ->
                    (activities.sumOf { activity -> activity.averageSpeed } / activities.size).toFloat()
                }

        // compute max speed for all years
        val maxSpeedByYear =
            activitiesByYear.groupBy { activity -> activity.startDateLocal.subSequence(0, 4).toString() }
                .mapValues { (_, activities) ->
                    activities.maxOf { activity -> activity.maxSpeed }
                }

        // compute average heart rate for all years
        val averageHeartRateByYear =
            activitiesByYear.groupBy { activity -> activity.startDateLocal.subSequence(0, 4).toString() }
                .mapValues { (_, activities) ->
                    (activities.sumOf { activity -> activity.averageHeartrate } / activities.size).toInt()
                }

        // compute max heart rate for all years
        val maxHeartRateByYear =
            activitiesByYear.groupBy { activity -> activity.startDateLocal.subSequence(0, 4).toString() }
                .mapValues { (_, activities) ->
                    activities.maxOf { activity -> activity.maxHeartrate }
                }

        // compute average watts for all years
        val averageWattsByYear =
            activitiesByYear.groupBy { activity -> activity.startDateLocal.subSequence(0, 4).toString() }
                .mapValues { (_, activities) ->
                    (activities.sumOf { activity -> activity.averageWatts } / activities.size).toInt()
                }

        // compute max watts for all years
        val maxWattsByYear =
            activitiesByYear.groupBy { activity -> activity.startDateLocal.subSequence(0, 4).toString() }
                .mapValues { entry: Map.Entry<String, List<StravaActivity>> ->
                    entry.value.maxOf { activity -> activity.maxWatts.toInt() }
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
        )
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
            sum += activities.sumOf { activity -> activity.distance / 1000 }; sum
        }
    }

    private fun cumulativeElevation(activities: Map<String, List<StravaActivity>>): Map<String, Int> {
        var sum = 0
        return activities.mapValues { (_, activities) ->
            sum += activities.sumOf { activity -> activity.totalElevationGain.toInt() }; sum
        }
    }
}