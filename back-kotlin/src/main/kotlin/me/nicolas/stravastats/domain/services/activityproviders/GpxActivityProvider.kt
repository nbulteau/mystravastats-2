package me.nicolas.stravastats.domain.services.activityproviders

import me.nicolas.stravastats.adapters.localrepositories.gpx.GPXRepository
import me.nicolas.stravastats.domain.business.strava.Activity
import me.nicolas.stravastats.domain.business.strava.Athlete
import me.nicolas.stravastats.domain.business.strava.DetailedActivity
import org.slf4j.LoggerFactory
import java.time.LocalDate
import java.util.*

class GpxActivityProvider (gpxCache: String) : AbstractActivityProvider() {
    private val logger = LoggerFactory.getLogger(GpxActivityProvider::class.java)

    private val localStorageProvider = GPXRepository(gpxCache)

    init {
        logger.info("Initialize GPX activity provider ...")
        val firstname = gpxCache.substringAfterLast("-")
        athlete = Athlete(id = 0, firstname = firstname, lastname = "")
        activities = loadFromLocalCache()
    }

    override fun getDetailedActivity(activityId: Long): Optional<DetailedActivity> {
        TODO("Not yet implemented")
    }

    private fun loadFromLocalCache(): List<Activity> {
        logger.info("Load GPX activities from local cache ...")

        val loadedActivities = mutableListOf<Activity>()
        for (currentYear in LocalDate.now().year downTo 2010) {
            logger.info("Load $currentYear activities ...")
            loadedActivities.addAll(localStorageProvider.loadActivitiesFromCache(currentYear))
        }

        logger.info("All activities are loaded.")

        return loadedActivities.sortedBy { it.startDateLocal }.reversed()
    }
}