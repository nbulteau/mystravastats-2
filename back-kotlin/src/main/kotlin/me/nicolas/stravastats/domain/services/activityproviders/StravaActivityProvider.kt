package me.nicolas.stravastats.domain.services.activityproviders

import kotlinx.coroutines.async
import kotlinx.coroutines.runBlocking
import me.nicolas.stravastats.adapters.localrepositories.strava.StravaRepository
import me.nicolas.stravastats.adapters.strava.StravaApi
import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.business.strava.StravaAthlete
import me.nicolas.stravastats.domain.business.strava.StravaDetailedActivity
import me.nicolas.stravastats.domain.business.strava.stream.Stream
import me.nicolas.stravastats.domain.interfaces.IStravaApi
import me.nicolas.stravastats.domain.services.ActivityHelper.filterByActivityTypes
import org.slf4j.LoggerFactory
import java.time.LocalDate
import java.util.*
import kotlin.system.exitProcess
import kotlin.system.measureTimeMillis


class StravaActivityProvider(
    stravaCache: String = "strava-cache",
) : AbstractActivityProvider() {

    private val logger = LoggerFactory.getLogger(StravaActivityProvider::class.java)

    private val localStorageProvider = StravaRepository(stravaCache = stravaCache)

    private lateinit var stravaApi: IStravaApi

    init {
        localStorageProvider.readStravaAuthentication(stravaCache).let { (id, secret, useCache) ->
            if (id == null) {
                logger.error("Strava authentication not found")
                exitProcess(-1)
            }

            clientId = id
            if (useCache == true) {
                stravaAthlete = localStorageProvider.loadAthleteFromCache(clientId)
                activities = loadFromLocalCache(clientId)
            } else {
                if (secret != null) {
                    localStorageProvider.initLocalStorageForClientId(clientId)
                    stravaApi = StravaApi(clientId, secret)
                    stravaAthlete = retrieveLoggedInAthlete(clientId)
                    activities = loadCurrentYearFromStrava(clientId)
                } else {
                    logger.error("Strava authentication not found")
                    exitProcess(-1)
                }
            }
        }

        logger.info("ActivityService initialized with clientId=$clientId and ${activities.size} activities")
    }

    override fun getDetailedActivity(activityId: Long): Optional<StravaDetailedActivity> {
        logger.info("Get detailed stravaActivity for stravaActivity id $activityId")

        // find detailed activity in cache or retrieve from Strava
        val activity = activities.find { it.id == activityId } ?: return Optional.empty()
        val year = activity.startDate.take(4).toInt()

        // load detailed activity from cache or retrieve from Strava
        localStorageProvider.loadDetailedActivityFromCache(clientId, year, activityId)?.let {
            return Optional.of(it)
        }

        // It's not in local cache, retrieve from Strava
        val detailedActivity = stravaApi.getDetailedActivity(activityId)
        if (detailedActivity.isPresent) {
            localStorageProvider.saveDetailedActivityToCache(clientId, year, detailedActivity.get())
            var stream = localStorageProvider.loadActivitiesStreamsFromCache(clientId, year, activity)
            if (stream == null) {
                val optionalStream = stravaApi.getActivityStream(activity)
                if (optionalStream.isPresent) {
                    stream = optionalStream.get()
                    localStorageProvider.saveActivitiesStreamsToCache(clientId, year, activity, stream)
                } else {
                    stream = null
                }
            }
            val updatedActivity = detailedActivity.get()
            updatedActivity.stream = stream

            return Optional.of(updatedActivity)
        }

        return Optional.empty()
    }

    private fun loadFromLocalCache(clientId: String): List<StravaActivity> {
        logger.info("Load Strava activities from local cache ...")

        val loadedActivities = mutableListOf<StravaActivity>()
        val elapsed = measureTimeMillis {
            runBlocking {
                for (currentYear in LocalDate.now().year downTo 2010) {
                    logger.info("Load $currentYear activities ...")
                    loadedActivities.addAll(localStorageProvider.loadActivitiesFromCache(clientId, currentYear))
                }
            }
        }
        logger.info("${loadedActivities.size} activities loaded in ${elapsed / 1000} s.")

        return loadedActivities
    }

    private fun loadCurrentYearFromStrava(clientId: String): List<StravaActivity> {
        logger.info("Load Strava activities from Strava ...")

        val loadedActivities = mutableListOf<StravaActivity>()
        val currentYear = LocalDate.now().year
        val elapsed = measureTimeMillis {
            runBlocking<Unit> {
                val currentYearActivities = async {
                    retrieveActivities(clientId, currentYear)
                }

                val previousYearsActivities = async {
                    val activities = mutableListOf<StravaActivity>()
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
        logger.info("${loadedActivities.size} activities loaded in ${elapsed / 1000} s.")

        return loadedActivities
    }

    private fun loadActivitiesStreams(clientId: String, year: Int, activities: List<StravaActivity>) {

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

    private fun retrieveLoggedInAthlete(clientId: String): StravaAthlete {
        logger.info("Load stravaAthlete with id $clientId description from Strava")

        val athlete = stravaApi.retrieveLoggedInAthlete()

        if (athlete.isPresent) {
            localStorageProvider.saveAthleteToCache(clientId, athlete.get())
        }

        return athlete.get()
    }

    private fun retrieveActivities(clientId: String, year: Int): List<StravaActivity> {
        logger.info("Load activities from Strava for year $year")

        val activities = stravaApi.getActivities(year).filterByActivityTypes()

        localStorageProvider.saveActivitiesToCache(clientId, year, activities)

        this.loadActivitiesStreams(clientId, year, activities)

        logger.info("${activities.size} activities loaded")

        return activities
    }
}