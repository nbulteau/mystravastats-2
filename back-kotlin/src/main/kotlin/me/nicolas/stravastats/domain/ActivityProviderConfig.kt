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
import java.io.File


@Configuration
class ActivityProviderConfig {
    private val logger = LoggerFactory.getLogger(ActivityProviderConfig::class.java)

    private fun readConfigValue(key: String): String? {
        val fromEnv = System.getenv(key)?.trim()
        if (!fromEnv.isNullOrEmpty()) {
            return fromEnv
        }

        val dotEnv = File(".env")
        if (!dotEnv.exists() || !dotEnv.isFile) {
            return null
        }

        return dotEnv.useLines { lines ->
            lines.asSequence()
                .map { it.trim() }
                .filter { it.isNotEmpty() && !it.startsWith("#") && it.contains("=") }
                .map { line ->
                    val separator = line.indexOf('=')
                    val envKey = line.substring(0, separator).trim()
                    val envValue = line.substring(separator + 1).trim().trim('"', '\'')
                    envKey to envValue
                }
                .firstOrNull { (envKey, _) -> envKey == key }
                ?.second
                ?.takeIf { it.isNotEmpty() }
        }
    }

    @Bean
    fun activityProvider(): IActivityProvider {
        val stravaCache: String? = readConfigValue("STRAVA_CACHE_PATH")
        val fitCache: String? = readConfigValue("FIT_FILES_PATH")
        val gpxCache: String? = readConfigValue("GPX_FILES_PATH")

        logger.info("Resolved STRAVA_CACHE_PATH={}", stravaCache ?: "strava-cache (default)")

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
        println("To access MyStravaStats: copy paste this url http://localhost:8080 in a browser")
        println()

        return activityProvider
    }
}
