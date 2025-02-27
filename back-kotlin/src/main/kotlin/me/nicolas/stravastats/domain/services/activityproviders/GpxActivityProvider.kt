package me.nicolas.stravastats.domain.services.activityproviders

import kotlinx.coroutines.runBlocking
import me.nicolas.stravastats.adapters.localrepositories.gpx.GPXRepository
import me.nicolas.stravastats.adapters.srtm.SRTMProvider
import me.nicolas.stravastats.domain.business.strava.stream.AltitudeStream
import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.business.strava.StravaAthlete
import me.nicolas.stravastats.domain.business.strava.StravaDetailedActivity
import me.nicolas.stravastats.domain.services.toStravaDetailedActivity
import org.slf4j.LoggerFactory
import java.time.LocalDate
import java.util.*
import kotlin.system.measureTimeMillis

class GpxActivityProvider(gpxCache: String, private val srtmProvider: SRTMProvider) : AbstractActivityProvider() {
    private val logger = LoggerFactory.getLogger(GpxActivityProvider::class.java)

    private val localStorageProvider = GPXRepository(gpxCache)

    init {
        logger.info("Initialize GPX stravaActivity provider ...")
        val firstname = gpxCache.substringAfterLast("-")
        stravaAthlete = StravaAthlete(id = 0, firstname = firstname, lastname = "")
        activities = loadFromLocalCache()
    }

    override fun getDetailedActivity(activityId: Long): Optional<StravaDetailedActivity> {
        val activity = getActivity(activityId)
        return if (activity.isPresent) {
            Optional.of(activity.get().toStravaDetailedActivity())
        } else {
            Optional.empty()
        }
    }

    private fun loadFromLocalCache(): List<StravaActivity> {
        logger.info("Load GPX activities from local cache ...")

        val loadedActivities = mutableListOf<StravaActivity>()
        val elapsed = measureTimeMillis {
            runBlocking {
                for (currentYear in LocalDate.now().year downTo 2010) {
                    logger.info("Load $currentYear activities ...")
                    loadedActivities.addAll(localStorageProvider.loadActivitiesFromCache(currentYear))
                }
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

                val data = srtmProvider.getElevation(activity.stream?.latlng?.data ?: emptyList())
                val altitude = AltitudeStream(data)
                activity.setStreamAltitude(altitude)
            } else {
                activity
            }
        }
    }
}

