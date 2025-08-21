package me.nicolas.stravastats.domain.services

import me.nicolas.stravastats.domain.business.ActivityType
import me.nicolas.stravastats.domain.business.rideActivities
import me.nicolas.stravastats.domain.business.runActivities
import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.business.strava.StravaDetailedActivity
import me.nicolas.stravastats.domain.services.activityproviders.IActivityProvider
import me.nicolas.stravastats.domain.services.csv.*
import org.slf4j.LoggerFactory
import org.springframework.data.domain.Page
import org.springframework.data.domain.Pageable
import org.springframework.stereotype.Service
import java.util.*
import kotlin.jvm.optionals.getOrElse


interface IActivityService {

    fun getDetailedActivity(activityId: Long): Optional<StravaDetailedActivity>

    fun getActivitiesByActivityTypeAndYear(activityTypes: Set<ActivityType>, year: Int?): List<StravaActivity>

    fun getActivitiesByActivityTypeGroupByActiveDays(activityTypes: Set<ActivityType>): Map<String, Int>

    fun listActivitiesPaginated(pageable: Pageable): Page<StravaActivity>

    fun exportCSV(activityTypes: Set<ActivityType>, year: Int): String
}

@Service
internal class ActivityService(
    activityProvider: IActivityProvider,
) : IActivityService, AbstractStravaService(activityProvider) {

    private val logger = LoggerFactory.getLogger(ActivityService::class.java)

    override fun getActivitiesByActivityTypeGroupByActiveDays(activityTypes: Set<ActivityType>): Map<String, Int> {
        logger.info("Get activities by activity type ($activityTypes) group by active days")

        return activityProvider.getActivitiesByActivityTypeGroupByActiveDays(activityTypes)
    }

    override fun listActivitiesPaginated(pageable: Pageable): Page<StravaActivity> {
        logger.info("List activities paginated")

        return activityProvider.listActivitiesPaginated(pageable)
    }

    override fun getActivitiesByActivityTypeAndYear(activityTypes: Set<ActivityType>, year: Int?): List<StravaActivity> {
        logger.info("Get activities by activity type ($activityTypes) for ${year ?: "all years"}")

        return activityProvider.getActivitiesByActivityTypeAndYear(activityTypes, year)
    }

    override fun exportCSV(activityTypes: Set<ActivityType>, year: Int): String {
        logger.info("Export CSV for activity type $activityTypes and year $year")

        val clientId = activityProvider.athlete().id.toString()

        val activities = activityProvider.getActivitiesByActivityTypeAndYear(activityTypes, year)

        // activityTypes must not be empty, otherwise we cannot determine the activity type
        if (activityTypes.isEmpty()) {
            logger.warn("No activity types provided, defaulting to Ride")
            return RideCSVExporter(clientId, activities, year).export()
        }

        // Determine the activity type based on the first activity type in the set
        // This is a simplification, as we assume all activities in the set are of the same type
        val activityType = when {
            rideActivities.contains(activityTypes.first()) -> ActivityType.Ride
            runActivities.contains(activityTypes.first()) -> ActivityType.Run
            else -> activityTypes.firstOrNull()
        }

        val exporter = when (activityType) {
            ActivityType.Ride -> RideCSVExporter(clientId = clientId, activities = activities, year = year)
            ActivityType.Run -> RunCSVExporter(clientId = clientId, activities = activities, year = year)
            ActivityType.InlineSkate -> InlineSkateCSVExporter(
                clientId = clientId,
                activities = activities,
                year = year
            )

            ActivityType.Hike -> HikeCSVExporter(clientId = clientId, activities = activities, year = year)
            ActivityType.AlpineSki -> AlpineSkiCSVExporter(clientId = clientId, activities = activities, year = year)
            else -> RideCSVExporter(clientId = clientId, activities = activities, year = year)
        }
        return exporter.export()
    }

    override fun getDetailedActivity(activityId: Long): Optional<StravaDetailedActivity> {
        logger.info("Get detailed activity $activityId")

        val detailedActivity = activityProvider.getDetailedActivity(activityId).getOrElse {
            logger.error("Activity $activityId not found")
            return Optional.empty()
        }

        return Optional.of(detailedActivity)
    }
}
