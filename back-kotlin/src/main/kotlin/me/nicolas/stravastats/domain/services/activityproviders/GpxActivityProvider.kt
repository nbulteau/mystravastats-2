package me.nicolas.stravastats.domain.services.activityproviders

import kotlinx.coroutines.runBlocking
import me.nicolas.stravastats.adapters.localrepositories.gpx.GPXRepository
import me.nicolas.stravastats.adapters.srtm.SRTMProvider
import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.business.strava.StravaAthlete
import me.nicolas.stravastats.domain.business.strava.StravaDetailedActivity
import me.nicolas.stravastats.domain.services.toStravaDetailedActivity
import org.slf4j.LoggerFactory
import java.time.LocalDate
import kotlin.system.measureTimeMillis

class GpxActivityProvider(private val gpxCache: String, private val srtmProvider: SRTMProvider) : AbstractActivityProvider() {
    private val logger = LoggerFactory.getLogger(GpxActivityProvider::class.java)

    private val localStorageProvider = GPXRepository(gpxCache)

    init {
        logger.info("Initialize GPX stravaActivity provider ...")
        val firstname = gpxCache.substringAfterLast("-")
        stravaAthlete = StravaAthlete(id = 0, firstname = firstname, lastname = "")
        activities = loadFromLocalCache()
    }

    override fun getDetailedActivity(activityId: Long): StravaDetailedActivity? {
        return getActivity(activityId)?.toStravaDetailedActivity()
    }

    override fun getCacheDiagnostics(): Map<String, Any?> {
        return basicCacheDiagnostics(
            provider = "gpx",
            sourcePathKey = "gpxDirectory",
            sourcePath = gpxCache,
        )
    }

    private fun loadFromLocalCache(): List<StravaActivity> {
        logger.info("Load GPX activities from local cache ...")

        val loadedActivities = mutableListOf<StravaActivity>()
        val elapsed = measureTimeMillis {
            runBlocking {
                for (currentYear in LocalDate.now().year downTo StravaActivityProvider.STRAVA_FIRST_YEAR) {
                    logger.info("Load $currentYear activities ...")
                    loadedActivities.addAll(localStorageProvider.loadActivitiesFromCache(currentYear))
                }
            }
        }

        logger.info("${loadedActivities.size} activities loaded in ${elapsed / 1000} s.")

        val sortedActivities = loadedActivities.sortedBy { it.startDateLocal }.reversed()

        return if (srtmProvider.isAvailable()) {
            sortedActivities.processAltitudeStreamIfMissing(srtmProvider)
        } else {
            sortedActivities
        }
    }
}
