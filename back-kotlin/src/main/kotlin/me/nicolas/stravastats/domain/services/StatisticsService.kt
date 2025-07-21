package me.nicolas.stravastats.domain.services

import me.nicolas.stravastats.api.controllers.AthleteController
import me.nicolas.stravastats.domain.business.ActivityType
import me.nicolas.stravastats.domain.business.strava.*
import me.nicolas.stravastats.domain.services.activityproviders.IActivityProvider
import me.nicolas.stravastats.domain.services.statistics.*
import org.slf4j.LoggerFactory
import org.springframework.stereotype.Service


interface IStatisticsService {
    fun getStatistics(activityTypes: Set<ActivityType>, year: Int?): List<Statistic>
}

@Service
internal class StatisticsService(
    activityProvider: IActivityProvider,
) : IStatisticsService, AbstractStravaService(activityProvider) {

    private val logger = LoggerFactory.getLogger(AthleteController::class.java)

    override fun getStatistics(activityTypes: Set<ActivityType>, year: Int?): List<Statistic> {
        logger.info("Compute $activityTypes statistics for ${year ?: "all years"}")

        val filteredActivities = activityProvider.getActivitiesByActivityTypeAndYear(activityTypes, year)

        // TODO: handle case multiple activity types
        return when (activityTypes.first()) {
            ActivityType.Run -> computeRunStatistics(filteredActivities)
            ActivityType.InlineSkate -> computeInlineSkateStatistics(filteredActivities)
            ActivityType.Hike -> computeHikeStatistics(filteredActivities)
            ActivityType.AlpineSki -> computeAlpineSkiStatistics(filteredActivities)
            else -> computeRideStatistics(filteredActivities)
        }
    }

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
                BestDayStatistic("Max distance in a day", hikeActivities, formatString = "%s => %.02f km")
                {
                    hikeActivities
                        .groupBy { stravaActivity: StravaActivity -> stravaActivity.startDateLocal.substringBefore('T') }
                        .mapValues { it.value.sumOf { activity -> activity.distance / 1000 } }
                        .maxByOrNull { entry: Map.Entry<String, Double> -> entry.value }
                        ?.toPair()
                },
                BestDayStatistic("Max elevation in a day", hikeActivities, formatString = "%s => %.02f m")
                {
                    hikeActivities
                        .groupBy { stravaActivity: StravaActivity -> stravaActivity.startDateLocal.substringBefore('T') }
                        .mapValues { it.value.sumOf { activity -> activity.totalElevationGain } }
                        .maxByOrNull { entry: Map.Entry<String, Double> -> entry.value }
                        ?.toPair()
                }
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


    private fun computeCommonStats(activities: List<StravaActivity>): List<Statistic> {

        return listOf(
            GlobalStatistic("Nb activities", activities, "%d", List<StravaActivity>::size),

            GlobalStatistic("Nb actives days", activities, "%d") {
                activities
                    .groupBy { stravaActivity: StravaActivity -> stravaActivity.startDateLocal.substringBefore('T') }
                    .count()
            },
            MaxStreakStatistic(activities),
            GlobalStatistic("Total distance", activities, "%.2f km") {
                activities.sumOf { stravaActivity: StravaActivity -> stravaActivity.distance } / 1000
            },
            GlobalStatistic("Total elevation", activities, "%.2f m") {
                activities.sumOf { stravaActivity: StravaActivity -> stravaActivity.totalElevationGain }
            },
            GlobalStatistic("Km by activity", activities, "%.2f km") {
                activities.sumOf { stravaActivity: StravaActivity -> stravaActivity.distance }
                    .div(activities.size) / 1000
            },

            MaxDistanceStatistic(activities),
            MaxDistanceInADayStatistic(activities),

            MaxElevationStatistic(activities),
            MaxElevationInADayStatistic(activities),

            HighestPointStatistic(activities),
            MaxMovingTimeStatistic(activities),
            MostActiveMonthStatistic(activities),
            EddingtonStatistic(activities),
        )
    }
}