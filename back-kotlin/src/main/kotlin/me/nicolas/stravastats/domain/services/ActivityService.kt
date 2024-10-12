package me.nicolas.stravastats.domain.services

import me.nicolas.stravastats.domain.business.strava.Activity
import me.nicolas.stravastats.domain.business.strava.ActivityType
import me.nicolas.stravastats.domain.services.activityproviders.IActivityProvider
import me.nicolas.stravastats.domain.services.csv.*
import org.slf4j.LoggerFactory
import org.springframework.data.domain.Page
import org.springframework.data.domain.Pageable
import org.springframework.stereotype.Service
import java.util.*


interface IActivityService {

    fun getActivity(activityId: Long): Optional<Activity>

    fun getActivitiesByActivityTypeAndYear(activityType: ActivityType, year: Int?): List<Activity>

    fun getActivitiesByActivityTypeGroupByActiveDays(activityType: ActivityType): Map<String, Int>

    fun listActivitiesPaginated(pageable: Pageable): Page<Activity>

    fun exportCSV(activityType: ActivityType, year: Int): String
}

@Service
internal class ActivityService(
    activityProvider: IActivityProvider,
) : IActivityService, AbstractStravaService(activityProvider) {

    private val logger = LoggerFactory.getLogger(ActivityService::class.java)


    override fun getActivitiesByActivityTypeGroupByActiveDays(activityType: ActivityType): Map<String, Int> {
        logger.info("Get activities by activity type ($activityType) group by active days")

        return activityProvider.getActivitiesByActivityTypeGroupByActiveDays(activityType)
    }

    override fun listActivitiesPaginated(pageable: Pageable): Page<Activity> {
        logger.info("List activities paginated")

        return activityProvider.listActivitiesPaginated(pageable)
    }

    override fun getActivitiesByActivityTypeAndYear(activityType: ActivityType, year: Int?): List<Activity> {
        logger.info("Get activities by activity type ($activityType) for ${year ?: "all years"}")

        return activityProvider.getActivitiesByActivityTypeAndYear(activityType, year)
    }

    override fun exportCSV(activityType: ActivityType, year: Int): String {
        logger.info("Export CSV for activity type $activityType and year $year")

        val clientId = activityProvider.athlete().id.toString()

        val activities = activityProvider.getActivitiesByActivityTypeAndYear(activityType, year)
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
            else -> throw IllegalArgumentException("Unknown activity type: $activityType")
        }
        return exporter.export()
    }

    override fun getActivity(activityId: Long): Optional<Activity> {
        logger.info("Get detailed activity $activityId")

        return activityProvider.getActivity(activityId)
    }
}