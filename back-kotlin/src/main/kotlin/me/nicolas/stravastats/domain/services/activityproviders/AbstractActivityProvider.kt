package me.nicolas.stravastats.domain.services.activityproviders

import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.business.ActivityType

import me.nicolas.stravastats.domain.business.strava.StravaAthlete
import me.nicolas.stravastats.domain.utils.GenericCache
import me.nicolas.stravastats.domain.utils.SoftCache
import org.slf4j.LoggerFactory
import org.springframework.data.domain.Page
import org.springframework.data.domain.PageImpl
import org.springframework.data.domain.Pageable
import java.util.*

abstract class AbstractActivityProvider : IActivityProvider {

    private val logger = LoggerFactory.getLogger(AbstractActivityProvider::class.java)

    protected lateinit var clientId: String

    protected lateinit var stravaAthlete: StravaAthlete

    protected lateinit var activities: List<StravaActivity>

    private val filteredActivitiesCache: GenericCache<String, List<StravaActivity>> = SoftCache()

    override fun athlete(): StravaAthlete {
        return stravaAthlete
    }

    /**
     * List activities paginated. It returns a page of activities.
     * @param pageable the pageable
     * @return a page of activities
     */
    override fun listActivitiesPaginated(pageable: Pageable): Page<StravaActivity> {
        logger.info("List activities paginated")

        val from = pageable.offset.toInt()
        val to = (pageable.offset + pageable.pageSize).toInt().coerceAtMost(activities.size)

        val sortedActivities = pageable.sort.let { sort ->
            if (sort.isSorted) {
                activities.sortedWith(compareBy { it.startDateLocal }).toList()
            } else {
                activities
            }
        }

        val subList = sortedActivities.subList(from, to)

        return PageImpl(subList, pageable, activities.size.toLong())
    }

    override fun getActivity(activityId: Long): Optional<StravaActivity> {
        logger.info("Get stravaActivity for stravaActivity id $activityId")

        return activities.find { activity -> activity.id == activityId }.let {
            if (it != null) {
                Optional.of(it)
            } else {
                Optional.empty()
            }
        }
    }

    override fun getActivitiesByActivityTypeGroupByActiveDays(activityType: ActivityType): Map<String, Int> {
        logger.info("Get activities by stravaActivity type ($activityType) group by active days")

        val filteredActivities = activities
            .filterActivitiesByType(activityType)

        return filteredActivities
            .groupBy { activity -> activity.startDateLocal.substringBefore('T') }
            .mapValues { (_, activities) -> activities.sumOf { activity -> activity.distance / 1000 } }
            .mapValues { entry -> entry.value.toInt() }
            .toMap()
    }

    override fun getActivitiesByActivityTypeByYearGroupByActiveDays(
        activityType: ActivityType,
        year: Int,
    ): Map<String, Int> {
        logger.info("Get activities by stravaActivity type ($activityType) group by active days for year $year")

        val filteredActivities = activities
            .filterActivitiesByYear(year)
            .filterActivitiesByType(activityType)

        return filteredActivities
            .groupBy { activity -> activity.startDateLocal.substringBefore('T') }
            .mapValues { (_, activities) -> activities.sumOf { activity -> activity.distance / 1000 } }
            .mapValues { entry -> entry.value.toInt() }
            .toMap()
    }

    override fun getActivitiesByActivityTypeAndYear(activityType: ActivityType, year: Int?): List<StravaActivity> {

        val key: String = year?.let { "${activityType.name}-$it" } ?: activityType.name
        val filteredActivities = filteredActivitiesCache[key] ?: activities
            .filterActivitiesByYear(year)
            .filterActivitiesByType(activityType)
        filteredActivitiesCache[key] = filteredActivities

        return filteredActivities
    }

    override fun getActivitiesByActivityTypeGroupByYear(activityType: ActivityType): Map<String, List<StravaActivity>> {
        logger.info("Get activities by stravaActivity type ($activityType) group by year")

        val filteredActivities = activities.filterActivitiesByType(activityType)

        return groupActivitiesByYear(filteredActivities)
    }

    /**
     * Group activities by year
     * @param activities list of activities
     * @return a map with the year as key and the list of activities as value
     * @see StravaActivity
     */
    private fun groupActivitiesByYear(activities: List<StravaActivity>): Map<String, List<StravaActivity>> {
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

    private fun List<StravaActivity>.filterActivitiesByType(activityType: ActivityType): List<StravaActivity> {
        return if (activityType == ActivityType.Commute) {
            this.filter { activity -> activity.type == ActivityType.Ride.name && activity.commute }
        } else {
            this.filter { activity -> (activity.type == activityType.name) && !activity.commute }
        }
    }

    private fun List<StravaActivity>.filterActivitiesByYear(year: Int?): List<StravaActivity> {
        return if (year == null) {
            this
        } else {
            this.filter { activity -> activity.startDateLocal.subSequence(0, 4).toString().toInt() == year }
        }
    }
}