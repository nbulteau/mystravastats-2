package me.nicolas.stravastats.domain.services

import me.nicolas.stravastats.domain.business.ActivityType
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

    fun getActivitiesByActivityTypeAndYear(activityType: ActivityType, year: Int?): List<StravaActivity>

    fun getActivitiesByActivityTypeGroupByActiveDays(activityType: ActivityType): Map<String, Int>

    fun listActivitiesPaginated(pageable: Pageable): Page<StravaActivity>

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

    override fun listActivitiesPaginated(pageable: Pageable): Page<StravaActivity> {
        logger.info("List activities paginated")

        return activityProvider.listActivitiesPaginated(pageable)
    }

    override fun getActivitiesByActivityTypeAndYear(activityType: ActivityType, year: Int?): List<StravaActivity> {
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

    override fun getDetailedActivity(activityId: Long): Optional<StravaDetailedActivity> {
        logger.info("Get detailed activity $activityId")

        val detailedActivity = activityProvider.getDetailedActivity(activityId).getOrElse {
            logger.error("Activity $activityId not found")
            return Optional.empty()
        }

        return Optional.of(detailedActivity)
    }
}
