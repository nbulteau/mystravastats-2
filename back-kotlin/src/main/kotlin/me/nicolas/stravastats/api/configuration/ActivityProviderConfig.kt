package me.nicolas.stravastats.api.configuration

import jakarta.annotation.PreDestroy
import kotlinx.coroutines.runBlocking
import me.nicolas.stravastats.adapters.srtm.SRTMProvider
import me.nicolas.stravastats.domain.RuntimeConfig
import me.nicolas.stravastats.domain.services.activityproviders.FitActivityProvider
import me.nicolas.stravastats.domain.services.activityproviders.GpxActivityProvider
import me.nicolas.stravastats.domain.services.activityproviders.IActivityProvider
import me.nicolas.stravastats.domain.services.activityproviders.StravaActivityProvider
import org.slf4j.LoggerFactory
import org.springframework.context.annotation.Bean
import org.springframework.context.annotation.Configuration

/**
 * Spring configuration that selects and initializes the appropriate [IActivityProvider]
 * based on the environment configuration (Strava cache, FIT files, or GPX files).
 *
 * Moved here from the domain package to keep adapter-level wiring out of the domain layer.
 */
@Configuration
class ActivityProviderConfig {
    private val logger = LoggerFactory.getLogger(ActivityProviderConfig::class.java)

    private var createdProvider: AutoCloseable? = null

    @Bean
    fun activityProvider(): IActivityProvider {
        val stravaCache: String? = RuntimeConfig.readConfigValue("STRAVA_CACHE_PATH")
        val fitCache: String? = RuntimeConfig.readConfigValue("FIT_FILES_PATH")
        val gpxCache: String? = RuntimeConfig.readConfigValue("GPX_FILES_PATH")

        logger.info("Resolved STRAVA_CACHE_PATH={}", stravaCache ?: "strava-cache (default)")

        val activityProvider = if (fitCache == null && gpxCache == null) {
            logger.info("Using Strava Activity Provider")

            val provider = if (stravaCache == null) {
                StravaActivityProvider()
            } else {
                StravaActivityProvider(stravaCache = stravaCache)
            }

            // Initialize activity provider (suspend function) in a controlled blocking context
            try {
                runBlocking {
                    provider.initializeAndLoadActivities()
                }
            } catch (e: Exception) {
                logger.error("Failed to initialize StravaActivityProvider", e)
                throw e
            }

            createdProvider = provider
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

        logger.info("")
        logger.info("To access MyStravaStats: copy paste this url http://localhost:8080 in a browser")
        logger.info("")

        return activityProvider
    }

    @PreDestroy
    fun shutdown() {
        createdProvider?.close()
    }
}

