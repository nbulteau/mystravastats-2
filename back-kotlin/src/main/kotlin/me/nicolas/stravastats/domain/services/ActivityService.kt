package me.nicolas.stravastats.domain.services

import me.nicolas.stravastats.domain.business.ActivityType
import me.nicolas.stravastats.domain.business.rideActivities
import me.nicolas.stravastats.domain.business.runActivities
import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.business.strava.StravaDetailedActivity
import me.nicolas.stravastats.domain.services.activityproviders.IActivityProvider
import me.nicolas.stravastats.domain.services.csv.*
import org.slf4j.LoggerFactory
import org.springframework.stereotype.Service


interface IActivityService {

    fun getDetailedActivity(activityId: Long, corrected: Boolean = true): StravaDetailedActivity?

    fun getActivitiesByActivityTypeAndYear(activityTypes: Set<ActivityType>, year: Int?): List<StravaActivity>

    fun exportCSV(activityTypes: Set<ActivityType>, year: Int?): String
}

@Service
internal class ActivityService(
    activityProvider: IActivityProvider,
    private val exporters: List<ICSVExporter> = emptyList()
) : IActivityService, AbstractStravaService(activityProvider) {

    private val logger = LoggerFactory.getLogger(ActivityService::class.java)

    override fun getActivitiesByActivityTypeAndYear(activityTypes: Set<ActivityType>, year: Int?): List<StravaActivity> {
        logger.info("Get activities by activity type ($activityTypes) for ${year ?: "all years"}")

        return activityProvider.getActivitiesByActivityTypeAndYear(activityTypes, year)
            .withDataQualityCorrections(activityProvider)
    }

    override fun exportCSV(activityTypes: Set<ActivityType>, year: Int?): String {
        logger.info("Export CSV for activity type $activityTypes and year ${year ?: "all years"}")

        val clientId = activityProvider.athlete().id.toString()

        val activities = activityProvider.getActivitiesByActivityTypeAndYear(activityTypes, year)
            .withDataQualityCorrections(activityProvider)

        val targetActivityType = if (activityTypes.isEmpty()) {
            logger.warn("No activity types provided, defaulting to Ride")
            ActivityType.Ride
        } else {
            when {
                rideActivities.contains(activityTypes.first()) -> ActivityType.Ride
                runActivities.contains(activityTypes.first()) -> ActivityType.Run
                else -> activityTypes.first()
            }
        }

        // Find supported exporter or default to Ride if not found
        val exporter = exporters.firstOrNull { it.supports(targetActivityType) }
            ?: exporters.first { it.supports(ActivityType.Ride) }

        return exporter.export(clientId, activities, year)
    }

    override fun getDetailedActivity(activityId: Long, corrected: Boolean): StravaDetailedActivity? {
        logger.info("Get detailed activity $activityId")
        val activity = activityProvider.getDetailedActivity(activityId)
        return if (corrected) {
            activity?.withDataQualityCorrections(activityProvider)
        } else {
            activity
        }
    }
}
