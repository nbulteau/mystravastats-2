package me.nicolas.stravastats.domain.services

import me.nicolas.stravastats.domain.business.strava.Activity
import me.nicolas.stravastats.domain.business.strava.ActivityType
import me.nicolas.stravastats.domain.services.csv.*
import org.slf4j.LoggerFactory
import org.springframework.data.domain.Page
import org.springframework.data.domain.Pageable
import org.springframework.stereotype.Service
import java.util.*


interface IActivityService {

    fun getActivitiesByActivityTypeAndYear(activityType: ActivityType, year: Int?): List<Activity>

    fun getActivitiesByActivityTypeGroupByActiveDays(activityType: ActivityType): Map<String, Int>

    fun listActivitiesPaginated(pageable: Pageable): Page<Activity>

    fun getFilteredActivitiesByActivityTypeAndYear(activityType: ActivityType, year: Int?): List<Activity>

    fun exportCSV(activityType: ActivityType, year: Int): String

    fun getActivity(activityId: Long): Optional<Activity>
}

@Service
internal class ActivityService(
    stravaProxy: StravaProxy,
) : IActivityService, AbstractStravaService(stravaProxy) {

    private val logger = LoggerFactory.getLogger(ActivityService::class.java)

    /**
     * Get filtered activities by activity type and year.
     * @param activityType the activity type
     * @param year the year
     * @return a list of activities
     */
    override fun getActivitiesByActivityTypeAndYear(activityType: ActivityType, year: Int?): List<Activity> {
        logger.info("Get activities by activity type ($activityType) for ${year ?: "all years"}")

        return stravaProxy.getFilteredActivitiesByActivityTypeAndYear(activityType, year)
    }

    override fun getActivitiesByActivityTypeGroupByActiveDays(activityType: ActivityType): Map<String, Int> {
        logger.info("Get activities by activity type ($activityType) group by active days")

        return stravaProxy.getActivitiesByActivityTypeGroupByActiveDays(activityType)
    }

    override fun listActivitiesPaginated(pageable: Pageable): Page<Activity> {
        logger.info("List activities paginated")

        return stravaProxy.listActivitiesPaginated(pageable)
    }

    override fun getFilteredActivitiesByActivityTypeAndYear(activityType: ActivityType, year: Int?): List<Activity> {
        logger.info("Get filtered activities by activity type ($activityType) for ${year ?: "all years"}")

        return stravaProxy.getFilteredActivitiesByActivityTypeAndYear(activityType, year)
    }

    override fun exportCSV(activityType: ActivityType, year: Int): String {
        logger.info("Export CSV for activity type $activityType and year $year")

        val clientId = stravaProxy.getAthlete()?.id?.toString() ?: ""

        val activities = stravaProxy.getFilteredActivitiesByActivityTypeAndYear(activityType, year)
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

        return stravaProxy.getActivity(activityId)
    }
}