package me.nicolas.stravastats.domain.services.activityproviders

import me.nicolas.stravastats.adapters.localrepositories.fit.FITRepository
import me.nicolas.stravastats.adapters.srtm.SRTMProvider
import me.nicolas.stravastats.domain.business.strava.Altitude
import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.business.strava.StravaAthlete
import org.slf4j.LoggerFactory
import java.time.LocalDate
import kotlin.system.measureTimeMillis

class FitActivityProvider(fitCache: String, private val srtmProvider: SRTMProvider) : AbstractActivityProvider() {
    private val logger = LoggerFactory.getLogger(FitActivityProvider::class.java)

    private val localStorageProvider = FITRepository(fitCache)

    init {
        logger.info("Initialize FIT stravaActivity provider ...")
        val firstname = fitCache.substringAfterLast("-")
        stravaAthlete = StravaAthlete(id = 0, firstname = firstname, lastname = "")
        activities = loadFromLocalCache()
    }

    private fun loadFromLocalCache(): List<StravaActivity> {
        logger.info("Load FIT activities from local cache ...")

        val loadedActivities = mutableListOf<StravaActivity>()
        val elapsed = measureTimeMillis {
            for (currentYear in LocalDate.now().year downTo 2010) {
                logger.info("Load $currentYear activities ...")
                loadedActivities.addAll(localStorageProvider.loadActivitiesFromCache(currentYear))
            }
        }

        logger.info("${loadedActivities.size} activities loaded in ${elapsed / 1000} s.")

        val sortedActivities = loadedActivities.sortedBy { it.startDateLocal }.reversed()

        if (srtmProvider.isAvailable()) {
            return sortedActivities.processAltitudeStreamToActivitiesIfMissing()
        }

        return sortedActivities
    }

    private fun List<StravaActivity>.processAltitudeStreamToActivitiesIfMissing(): List<StravaActivity> {
        logger.debug("Process altitude stream to activities if missing")

        return this.map { activity ->
            if (activity.stream != null && activity.stream?.altitude == null) {
                logger.info("Process altitude stream to activity ${activity.name}")

                val data = srtmProvider.getElevation(activity.stream?.latitudeLongitude?.data ?: emptyList())
                val altitude = Altitude(data)
                activity.setStreamAltitude(altitude)
            } else {
                activity
            }
        }
    }
}