package me.nicolas.stravastats.domain.services.activityproviders

import me.nicolas.stravastats.adapters.localrepositories.gpx.GPXRepository
import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.business.strava.StravaAthlete
import me.nicolas.stravastats.domain.business.strava.StravaDetailedActivity
import org.slf4j.LoggerFactory
import java.time.LocalDate
import java.util.*

class GpxActivityProvider (gpxCache: String) : AbstractActivityProvider() {
    private val logger = LoggerFactory.getLogger(GpxActivityProvider::class.java)

    private val localStorageProvider = GPXRepository(gpxCache)

    init {
        logger.info("Initialize GPX stravaActivity provider ...")
        val firstname = gpxCache.substringAfterLast("-")
        stravaAthlete = StravaAthlete(id = 0, firstname = firstname, lastname = "")
        activities = loadFromLocalCache()
    }

    override fun getDetailedActivity(activityId: Long): Optional<StravaDetailedActivity> {
        TODO("Not yet implemented")
    }

    private fun loadFromLocalCache(): List<StravaActivity> {
        logger.info("Load GPX activities from local cache ...")

        val loadedActivities = mutableListOf<StravaActivity>()
        for (currentYear in LocalDate.now().year downTo 2010) {
            logger.info("Load $currentYear activities ...")
            loadedActivities.addAll(localStorageProvider.loadActivitiesFromCache(currentYear))
        }

        logger.info("All activities are loaded: ${loadedActivities.size} activities.")

        return loadedActivities.sortedBy { it.startDateLocal }.reversed()
    }
}