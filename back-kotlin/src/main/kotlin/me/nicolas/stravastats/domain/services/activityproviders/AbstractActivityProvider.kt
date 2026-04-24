package me.nicolas.stravastats.domain.services.activityproviders

import me.nicolas.stravastats.domain.business.ActivityType
import me.nicolas.stravastats.domain.business.SportType
import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.business.strava.StravaAthlete
import org.slf4j.LoggerFactory
import org.springframework.data.domain.Page
import org.springframework.data.domain.PageImpl
import org.springframework.data.domain.Pageable
import java.time.Instant
import kotlin.math.roundToInt

abstract class AbstractActivityProvider : IActivityProvider {

    private val logger = LoggerFactory.getLogger(AbstractActivityProvider::class.java)

    protected lateinit var stravaAthlete: StravaAthlete

    /** Index for O(1) activity lookups by id. Kept in sync with [activities]. */
    @Volatile
    private var activitiesIndex: Map<Long, StravaActivity> = emptyMap()

    @Volatile
    protected var activities: List<StravaActivity> = emptyList()
        set(value) {
            field = value
            activitiesIndex = value.associateBy { stravaActivity -> stravaActivity.id }
        }

    override fun athlete(): StravaAthlete {
        check(::stravaAthlete.isInitialized) {
            "Athlete has not been loaded yet. Ensure initializeAndLoadActivities() completed successfully before calling athlete()."
        }
        return stravaAthlete
    }

    /**
     * List activities paginated. It returns a page of activities.
     * @param pageable the pageable
     * @return a page of activities
     */
    override fun listActivitiesPaginated(pageable: Pageable): Page<StravaActivity> {
        logger.info("List activities paginated")

        val activitiesSnapshot = activities
        if (activitiesSnapshot.isEmpty()) {
            return PageImpl(emptyList(), pageable, 0)
        }

        val from = pageable.offset.toInt()
        if (from >= activitiesSnapshot.size) {
            return PageImpl(emptyList(), pageable, activitiesSnapshot.size.toLong())
        }
        val to = (pageable.offset + pageable.pageSize).toInt().coerceAtMost(activitiesSnapshot.size)

        // Apply sort from Pageable, respecting the requested property and direction
        val sortedActivities = if (pageable.sort.isSorted) {
            var result = activitiesSnapshot
            for (order in pageable.sort) {
                val comparator: Comparator<StravaActivity> = when (order.property) {
                    "averageSpeed" -> compareBy { it.averageSpeed }
                    "averageCadence" -> compareBy { it.averageCadence }
                    "averageHeartrate" -> compareBy { it.averageHeartrate }
                    "maxHeartrate" -> compareBy { it.maxHeartrate }
                    "averageWatts" -> compareBy { it.averageWatts }
                    "distance" -> compareBy { it.distance }
                    "elapsedTime" -> compareBy { it.elapsedTime }
                    "elevHigh" -> compareBy { it.elevHigh }
                    "maxSpeed" -> compareBy { it.maxSpeed }
                    "movingTime" -> compareBy { it.movingTime }
                    "startDate" -> compareBy { it.startDateLocal }
                    "totalElevationGain" -> compareBy { it.totalElevationGain }
                    "weightedAverageWatts" -> compareBy { it.weightedAverageWatts }
                    else -> compareBy { it.startDateLocal }
                }
                result =
                    if (order.isDescending) result.sortedWith(comparator.reversed()) else result.sortedWith(comparator)
            }
            result
        } else {
            activitiesSnapshot
        }

        val subList = sortedActivities.subList(from, to)

        return PageImpl(subList, pageable, activitiesSnapshot.size.toLong())
    }

    override fun getActivity(activityId: Long): StravaActivity? {
        logger.info("Get stravaActivity for stravaActivity id $activityId")
        return activitiesIndex[activityId]
    }

    override fun getCacheDiagnostics(): Map<String, Any?> {
        return basicCacheDiagnostics(
            provider = this::class.simpleName?.removeSuffix("ActivityProvider")?.lowercase() ?: "local",
        )
    }

    protected fun basicCacheDiagnostics(
        provider: String,
        sourcePathKey: String? = null,
        sourcePath: String? = null,
    ): Map<String, Any?> {
        val activitiesSnapshot = activities
        val details = mutableMapOf<String, Any?>(
            "timestamp" to Instant.now().toString(),
            "provider" to provider,
            "athleteId" to if (::stravaAthlete.isInitialized) stravaAthlete.id else null,
            "activities" to activitiesSnapshot.size,
            "availableYearBins" to activitiesSnapshot
                .mapNotNull { activity ->
                    val year = activity.startDateLocal.extractYear()
                        .ifEmpty { activity.startDate.extractYear() }
                    year.takeIf { it.isNotEmpty() }
                }
                .distinct()
                .sorted(),
        )
        if (sourcePathKey != null && sourcePath != null) {
            details[sourcePathKey] = sourcePath
        }
        return details
    }

    override fun getActivitiesByActivityTypeGroupByActiveDays(activityTypes: Set<ActivityType>): Map<String, Int> {
        logger.info("Get activities by stravaActivity type ($activityTypes) group by active days")

        val filteredActivities = activities
            .filterActivitiesByActivityTypes(activityTypes)

        return filteredActivities
            .groupBy { activity -> activity.startDateLocal.substringBefore('T') }
            .mapValues { (_, activities) -> activities.sumOf { activity -> activity.distance / 1000 } }
            .mapValues { entry -> entry.value.roundToInt() }
            .toMap()
    }

    override fun getActivitiesByActivityTypeByYearGroupByActiveDays(
        activityTypes: Set<ActivityType>,
        year: Int,
    ): Map<String, Int> {
        logger.info("Get activities by stravaActivity type ($activityTypes) group by active days for year $year")

        val filteredActivities = activities
            .filterActivitiesByYear(year)
            .filterActivitiesByActivityTypes(activityTypes)

        return filteredActivities
            .groupBy { activity -> activity.startDateLocal.substringBefore('T') }
            .mapValues { (_, activities) -> activities.sumOf { activity -> activity.distance / 1000 } }
            .mapValues { entry -> entry.value.roundToInt() }
            .toMap()
    }

    override fun getActivitiesByActivityTypeAndYear(
        activityTypes: Set<ActivityType>,
        year: Int?
    ): List<StravaActivity> {
        return activities
            .filterActivitiesByYear(year)
            .filterActivitiesByActivityTypes(activityTypes)
    }

    override fun getActivitiesByActivityTypeGroupByYear(activityTypes: Set<ActivityType>): Map<String, List<StravaActivity>> {
        logger.info("Get activities by stravaActivity type ($activityTypes) group by year")

        val filteredActivities = activities.filterActivitiesByActivityTypes(activityTypes)

        return groupActivitiesByYear(filteredActivities)
    }

    /**
     * Group activities by year
     * @param activities list of activities
     * @return a map with the year as a key and the list of activities as value
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

    private fun List<StravaActivity>.filterActivitiesByActivityTypes(activityTypes: Set<ActivityType>): List<StravaActivity> {
        return this.filter { activity ->
            activityTypes.any { activityType ->
                when (activityType) {
                    ActivityType.Commute ->
                        activity.commute && (activity.sportType == ActivityType.Ride.name || activity.sportType == SportType.MountainBikeRide.name || activity.sportType == SportType.GravelRide.name)

                    else ->
                        !activity.commute && activity.sportType == activityType.name
                }
            }
        }
    }

    private fun List<StravaActivity>.filterActivitiesByYear(year: Int?): List<StravaActivity> {
        return if (year == null) {
            this
        } else {
            this.filter { activity -> activity.startDateLocal.subSequence(0, 4).toString().toInt() == year }
        }
    }

    private fun String?.extractYear(): String {
        val value = this?.trim().orEmpty()
        return if (value.length >= 4) value.substring(0, 4) else ""
    }
}
