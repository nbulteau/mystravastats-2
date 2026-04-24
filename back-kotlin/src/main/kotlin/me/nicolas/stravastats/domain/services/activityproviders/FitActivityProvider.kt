package me.nicolas.stravastats.domain.services.activityproviders

import me.nicolas.stravastats.adapters.localrepositories.fit.FITRepository
import me.nicolas.stravastats.adapters.srtm.SRTMProvider
import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.business.strava.StravaAthlete
import me.nicolas.stravastats.domain.business.strava.StravaDetailedActivity
import me.nicolas.stravastats.domain.services.toStravaDetailedActivity
import org.slf4j.LoggerFactory
import java.time.LocalDate
import kotlin.system.measureTimeMillis

class FitActivityProvider(private val fitCache: String, private val srtmProvider: SRTMProvider) : AbstractActivityProvider() {
    private val logger = LoggerFactory.getLogger(FitActivityProvider::class.java)

    private val localStorageProvider = FITRepository(fitCache)

    init {
        logger.info("Initialize FIT stravaActivity provider ...")
        val firstname = fitCache.substringAfterLast("-")
        stravaAthlete = StravaAthlete(id = 0, firstname = firstname, lastname = "")
        activities = loadFromLocalCache()
    }

    override fun getDetailedActivity(activityId: Long): StravaDetailedActivity? {
        return getActivity(activityId)?.toStravaDetailedActivity()
    }

    override fun getCacheDiagnostics(): Map<String, Any?> {
        return basicCacheDiagnostics(
            provider = "fit",
            sourcePathKey = "fitDirectory",
            sourcePath = fitCache,
        )
    }

    private fun loadFromLocalCache(): List<StravaActivity> {
        logger.info("Load FIT activities from local cache ...")

        val loadedActivities = mutableListOf<StravaActivity>()
        val elapsed = measureTimeMillis {
            for (currentYear in LocalDate.now().year downTo StravaActivityProvider.STRAVA_FIRST_YEAR) {
                logger.info("Load $currentYear activities ...")
                loadedActivities.addAll(localStorageProvider.loadActivitiesFromCache(currentYear))
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
