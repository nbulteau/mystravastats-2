package me.nicolas.stravastats.domain.services

import jakarta.annotation.PostConstruct
import kotlinx.coroutines.async
import kotlinx.coroutines.runBlocking
import me.nicolas.stravastats.adapter.localstorage.LocalStorageProvider
import me.nicolas.stravastats.adapter.strava.StravaApi
import me.nicolas.stravastats.adapter.strava.StravaProperties
import me.nicolas.stravastats.domain.business.strava.*
import me.nicolas.stravastats.domain.interfaces.ILocalStorageProvider
import me.nicolas.stravastats.domain.interfaces.IStravaApi
import me.nicolas.stravastats.domain.services.ActivityHelper.filterActivities
import me.nicolas.stravastats.domain.utils.GenericCache
import me.nicolas.stravastats.domain.utils.SoftCache
import org.slf4j.LoggerFactory
import org.springframework.boot.ApplicationArguments
import org.springframework.boot.DefaultApplicationArguments
import org.springframework.data.domain.Page
import org.springframework.data.domain.PageImpl
import org.springframework.data.domain.Pageable
import org.springframework.stereotype.Component
import java.io.File
import java.io.FileInputStream
import java.time.LocalDate
import java.util.*
import kotlin.system.exitProcess
import kotlin.system.measureTimeMillis

interface IStravaProxy {

    fun listActivitiesPaginated(pageable: Pageable): Page<Activity>

    fun getDetailedActivity(year: Int, activityId: Long): Optional<DetailledActivity>
    fun getActivitiesByActivityTypeGroupByActiveDays(activityType: ActivityType): Map<String, Int>
    fun getActivitiesByActivityTypeByYearGroupByActiveDays(activityType: ActivityType, year: Int): Map<String, Int>
    fun getFilteredActivitiesByActivityTypeAndYear(activityType: ActivityType, year: Int?): List<Activity>
    fun getActivitiesByActivityTypeGroupByYear(activityType: ActivityType): Map<String, List<Activity>>
    fun getAthlete(): Athlete?
}

@Component
class StravaProxy(
    private val args: ApplicationArguments = DefaultApplicationArguments(),
) : IStravaProxy {

    private val logger = LoggerFactory.getLogger(StravaProxy::class.java)

    private val localStorageProvider: ILocalStorageProvider = LocalStorageProvider()

    private lateinit var stravaApi: IStravaApi

    private val properties: StravaProperties = StravaProperties()

    private val filteredActivitiesCache: GenericCache<String, List<Activity>> = SoftCache()

    private var activities: List<Activity> = emptyList()

    private var athlete: Athlete? = null

    private var clientId: String = ""

    @PostConstruct
    private fun init() {
        initStravaService()

        logger.info("ActivityService initialized with clientId=$clientId and ${activities.size} activities")
    }

    override fun getAthlete(): Athlete? {
        logger.info("Get athlete description")

        return athlete
    }

    /**
     * List activities paginated. It returns a page of activities.
     * @param pageable the pageable
     * @return a page of activities
     */
    override fun listActivitiesPaginated(pageable: Pageable): Page<Activity> {
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

    override fun getDetailedActivity(year: Int, activityId: Long): Optional<DetailledActivity> {
        logger.info("Get detailed activity for year $year and activity id $activityId")

        localStorageProvider.loadDetailedActivityFromCache(clientId, year, activityId)?.let {
            return Optional.of(it)
        }

        val optionalDetailedActivity = stravaApi.getActivity(activityId)
        return if (optionalDetailedActivity.isPresent) {
            localStorageProvider.saveDetailedActivityToCache(clientId, year, optionalDetailedActivity.get())
            optionalDetailedActivity
        } else {
            Optional.empty()
        }
    }

    override fun getActivitiesByActivityTypeGroupByActiveDays(activityType: ActivityType): Map<String, Int> {
        logger.info("Get activities by activity type ($activityType) group by active days")

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
        logger.info("Get activities by activity type ($activityType) group by active days for year $year")

        val filteredActivities = activities
            .filterActivitiesByYear(year)
            .filterActivitiesByType(activityType)

        return filteredActivities
            .groupBy { activity -> activity.startDateLocal.substringBefore('T') }
            .mapValues { (_, activities) -> activities.sumOf { activity -> activity.distance / 1000 } }
            .mapValues { entry -> entry.value.toInt() }
            .toMap()
    }

    override fun getFilteredActivitiesByActivityTypeAndYear(activityType: ActivityType, year: Int?): List<Activity> {

        val key: String = year?.let { "${activityType.name}-$it" } ?: activityType.name
        val filteredActivities = filteredActivitiesCache[key] ?: activities
            .filterActivitiesByYear(year)
            .filterActivitiesByType(activityType)
        filteredActivitiesCache[key] = filteredActivities

        return filteredActivities
    }

    override fun getActivitiesByActivityTypeGroupByYear(activityType: ActivityType): Map<String, List<Activity>> {
        logger.info("Get activities by activity type ($activityType) group by year")

        val filteredActivities = activities.filterActivitiesByType(activityType)

        return ActivityHelper.groupActivitiesByYear(filteredActivities)
    }

    fun List<Activity>.filterActivitiesByType(activityType: ActivityType): List<Activity> {
        return if (activityType == ActivityType.Commute) {
            this.filter { activity -> activity.type == Ride && activity.commute }
        } else {
            this.filter { activity -> (activity.type == activityType.name) && !activity.commute }
        }
    }

    /**
     * Initialize the Strava service. It reads the Strava authentication from the ".strava" file.
     * Load the activities from the local cache if useCache is true, otherwise load the activities from Strava.
     */
    private fun initStravaService() {
        val id: String? = args.getOptionValues("clientId")?.get(0)
        val secret: String? = args.getOptionValues("clientSecret")?.get(0)

        if (!id.isNullOrEmpty() && !secret.isNullOrEmpty()) {
            clientId = id
            stravaApi = StravaApi(clientId, secret, properties)
            loadCurrentYearFromStrava(clientId)
        } else {
            readStravaAuthentication().let { (id, secret, useCache) ->
                if (id == null) {
                    logger.error("Strava authentication not found")
                    exitProcess(-1)
                }

                clientId = id
                athlete = localStorageProvider.loadAthleteFromCache(clientId) ?: retrieveLoggedInAthlete(clientId)
                if (useCache == true) {
                    loadFromLocalCache(clientId)
                } else {
                    if (secret != null) {
                        stravaApi = StravaApi(clientId, secret, properties)
                        loadCurrentYearFromStrava(clientId)
                    } else {
                        throw IllegalStateException("Strava authentication not found")
                    }
                }
            }
        }
    }

    private fun loadFromLocalCache(clientId: String) {
        logger.info("Load Strava activities from local cache ...")

        val loadedActivities = mutableListOf<Activity>()
        for (currentYear in LocalDate.now().year downTo 2010) {
            logger.info("Load $currentYear activities ...")
            loadedActivities.addAll(localStorageProvider.loadActivitiesFromCache(clientId, currentYear))
        }

        logger.info("All activities are loaded.")

        activities = loadedActivities
    }

    private fun loadCurrentYearFromStrava(clientId: String) {
        logger.info("Load Strava activities from Strava ...")

        val loadedActivities = mutableListOf<Activity>()
        val currentYear = LocalDate.now().year
        val elapsed = measureTimeMillis {
            runBlocking<Unit> {
                val currentYearActivities = async {
                    retrieveActivities(clientId, currentYear)
                }

                val previousYearsActivities = async {
                    val activities = mutableListOf<Activity>()
                    for (yearToLoad in currentYear - 1 downTo 2010) {
                        if (localStorageProvider.isLocalCacheExistForYear(clientId, yearToLoad)) {
                            activities.addAll(localStorageProvider.loadActivitiesFromCache(clientId, yearToLoad))
                        } else {
                            activities.addAll(retrieveActivities(clientId, yearToLoad))
                        }
                    }

                    activities
                }

                loadedActivities.addAll(currentYearActivities.await())
                loadedActivities.addAll(previousYearsActivities.await())
            }

        }
        logger.info("All activities are loaded in ${elapsed / 1000} s.")

        activities = loadedActivities
    }

    private fun loadActivitiesStreams(clientId: String, year: Int, activities: List<Activity>) {

        // stream id files list
        val streamIdsSet = localStorageProvider.buildStreamIdsSet(clientId, year)

        activities.forEach { activity ->
            val stream: Stream?

            if (streamIdsSet.contains(activity.id)) {
                stream = localStorageProvider.loadActivitiesStreamsFromCache(clientId, year, activity)
            } else {
                val optionalStream = stravaApi.getActivityStream(activity)
                if (optionalStream.isPresent) {
                    stream = optionalStream.get()
                    localStorageProvider.saveActivitiesStreamsToCache(clientId, year, activity, stream)
                } else {
                    stream = null
                }
            }

            activity.stream = stream
        }
    }

    private fun List<Activity>.filterActivitiesByYear(year: Int?): List<Activity> {
        return if (year == null) {
            this
        } else {
            this.filter { activity -> activity.startDateLocal.subSequence(0, 4).toString().toInt() == year }
        }
    }

    private fun retrieveLoggedInAthlete(clientId: String): Athlete {
        logger.info("Load athlete with id $clientId description from Strava")

        val athlete = stravaApi.retrieveLoggedInAthlete()

        if (athlete.isPresent) {
            localStorageProvider.saveAthleteToCache(clientId, athlete.get())
        }

        return athlete.get()
    }

    private fun retrieveActivities(clientId: String, year: Int): List<Activity> {
        logger.info("Load activities from Strava for year $year")

        val activities = stravaApi.getActivities(year).filterActivities()

        localStorageProvider.saveActivitiesToCache(clientId, year, activities)

        this.loadActivitiesStreams(clientId, year, activities)

        logger.info("${activities.size} activities loaded")

        return activities
    }

    /**
     * Read Strava authentication from ".strava" file.
     * The file must contain two properties: clientId and clientSecret.
     * @return a Triple with clientId, clientSecret and useCache
     */
    private fun readStravaAuthentication(): Triple<String?, String?, Boolean?> {
        val cacheDirectory = File("strava-cache")
        val file = File(cacheDirectory, ".strava")
        val properties = Properties()

        if (file.exists()) {
            FileInputStream(file).use { properties.load(it) }
        } else {
            logger.error("File .strava not found")
        }

        return Triple(
            properties["clientId"]?.toString(),
            properties["clientSecret"]?.toString(),
            properties["useCache"]?.toString()?.toBoolean()
        )
    }
}