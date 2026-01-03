package me.nicolas.stravastats.domain

import me.nicolas.stravastats.adapters.srtm.SRTMProvider
import me.nicolas.stravastats.domain.services.activityproviders.FitActivityProvider
import me.nicolas.stravastats.domain.services.activityproviders.GpxActivityProvider
import me.nicolas.stravastats.domain.services.activityproviders.IActivityProvider
import me.nicolas.stravastats.domain.services.activityproviders.StravaActivityProvider
import kotlinx.coroutines.runBlocking
import org.slf4j.LoggerFactory
import org.springframework.context.annotation.Bean
import org.springframework.context.annotation.Configuration


@Configuration
class ActivityProviderConfig {
    private val logger = LoggerFactory.getLogger(ActivityProviderConfig::class.java)


    @Bean
    fun activityProvider(): IActivityProvider {
        val stravaCache: String? = System.getenv("STRAVA_CACHE_PATH")
        val fitCache: String? = System.getenv("FIT_FILES_PATH")
        val gpxCache: String? = System.getenv("GPX_FILES_PATH")

        val activityProvider = if (fitCache == null && gpxCache == null) {
            logger.info("Using Strava Activity Provider")

            val provider = if (stravaCache == null) {
                StravaActivityProvider()
            } else {
                // use named parameter to match constructor
                StravaActivityProvider(stravaCache = stravaCache)
            }

            // initialize activity provider (suspend function) in a controlled blocking context
            try {
                runBlocking {
                    provider.initializeAndLoadActivities()
                }
            } catch (e: Exception) {
                logger.error("Failed to initialize StravaActivityProvider", e)
                throw e
            }

            provider
        } else {
            // Build a SRTM provider to get elevation data
            val srtmProvider = SRTMProvider()

            if (fitCache != null) {
                logger.info("Using FIT Activity Provider")

                FitActivityProvider(fitCache, srtmProvider)
            } else if (gpxCache != null) {
                logger.info("Using GPX Activity Provider")

                GpxActivityProvider(gpxCache, srtmProvider)
            } else {
                logger.error("No cache provided")

                throw IllegalArgumentException("No cache provided")
            }
        }

        println()
        println("To access MyStravaStats: copy paste this url http://localhost:80 in a browser")
        println()

        return activityProvider
    }
}