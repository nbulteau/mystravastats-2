package me.nicolas.stravastats.domain

import java.io.File
import java.util.Locale

object RuntimeConfig {
    private const val DEFAULT_STRAVA_CACHE_PATH = "strava-cache"
    private const val DEFAULT_SERVER_ADDRESS = "0.0.0.0"
    private const val DEFAULT_SERVER_PORT = "8080"
    private const val DEFAULT_OSM_ROUTING_BASE_URL = "http://localhost:5000"
    private const val DEFAULT_OSM_ROUTING_TIMEOUT_MS = 3000
    private const val DEFAULT_OSM_ROUTING_V3_ENABLED = true
    private const val DEFAULT_OSM_ROUTING_EXTRACT_PROFILE_FILE = "./osm/region.osrm.profile"
    private const val DEFAULT_OSM_ROUTING_HISTORY_HALF_LIFE_DAYS = 75

    private val defaultCorsAllowedOrigins = listOf("http://localhost", "http://localhost:5173")
    private val defaultCorsAllowedMethods = listOf("GET", "POST", "PUT", "DELETE", "OPTIONS")
    private val defaultCorsAllowedHeaders = listOf("Content-Type", "Authorization", "X-Request-Id")

    fun details(): Map<String, Any?> {
        val stravaCachePath = readStringConfig("STRAVA_CACHE_PATH", DEFAULT_STRAVA_CACHE_PATH)
        val fitFilesPath = readConfigValue("FIT_FILES_PATH")
        val gpxFilesPath = readConfigValue("GPX_FILES_PATH")
        val provider = when {
            fitFilesPath != null -> "fit"
            gpxFilesPath != null -> "gpx"
            else -> "strava"
        }
        val historyHalfLifeDays = readIntConfig(
            "OSM_ROUTING_HISTORY_HALF_LIFE_DAYS",
            DEFAULT_OSM_ROUTING_HISTORY_HALF_LIFE_DAYS,
        ).coerceAtLeast(1)

        return mapOf(
            "backend" to "kotlin",
            "data" to mapOf(
                "provider" to provider,
                "stravaCachePath" to stravaCachePath,
                "stravaCacheConfigured" to isConfigured("STRAVA_CACHE_PATH"),
                "fitFilesPath" to (fitFilesPath ?: ""),
                "fitFilesConfigured" to (fitFilesPath != null),
                "gpxFilesPath" to (gpxFilesPath ?: ""),
                "gpxFilesConfigured" to (gpxFilesPath != null),
                "gpxFilesSupported" to true,
                "providerSelectionOrder" to listOf("FIT_FILES_PATH", "GPX_FILES_PATH", "STRAVA_CACHE_PATH"),
            ),
            "server" to mapOf(
                "address" to readStringConfig("SERVER_ADDRESS", DEFAULT_SERVER_ADDRESS),
                "port" to readFirstStringConfig(DEFAULT_SERVER_PORT, "SERVER_PORT", "PORT"),
                "openBrowser" to readBoolConfig("OPEN_BROWSER", true),
                "openBrowserSource" to sourceFor("OPEN_BROWSER"),
            ),
            "cors" to mapOf(
                "allowedOrigins" to corsAllowedOrigins(),
                "allowedMethods" to corsAllowedMethods(),
                "allowedHeaders" to corsAllowedHeaders(),
                "allowCredentials" to true,
                "source" to if (isConfigured("CORS_ALLOWED_ORIGINS")) "CORS_ALLOWED_ORIGINS" else "default",
            ),
            "routing" to mapOf(
                "enabled" to readBoolConfig("OSM_ROUTING_ENABLED", true),
                "v3Enabled" to readBoolConfig("OSM_ROUTING_V3_ENABLED", DEFAULT_OSM_ROUTING_V3_ENABLED),
                "debug" to readBoolConfig("OSM_ROUTING_DEBUG", false),
                "baseUrl" to readStringConfig("OSM_ROUTING_BASE_URL", DEFAULT_OSM_ROUTING_BASE_URL).trimEnd('/'),
                "timeoutMs" to normalizedRoutingTimeoutMs(),
                "profile" to readStringConfig("OSM_ROUTING_PROFILE", ""),
                "extractProfile" to readStringConfig("OSM_ROUTING_EXTRACT_PROFILE", ""),
                "extractProfileFile" to readStringConfig(
                    "OSM_ROUTING_EXTRACT_PROFILE_FILE",
                    DEFAULT_OSM_ROUTING_EXTRACT_PROFILE_FILE,
                ),
                "historyBiasEnabled" to readBoolConfig("OSM_ROUTING_HISTORY_BIAS_ENABLED", false),
                "historyHalfLifeDays" to historyHalfLifeDays,
                "controlEnabled" to readBoolConfig("OSRM_CONTROL_ENABLED", true),
                "controlTimeoutMs" to readIntConfig("OSRM_CONTROL_TIMEOUT_MS", 30_000),
                "controlProjectDir" to readStringConfig("OSRM_CONTROL_PROJECT_DIR", ""),
                "controlComposeFile" to readStringConfig("OSRM_CONTROL_COMPOSE_FILE", "docker-compose-routing-osrm.yml"),
                "controlDockerBin" to readStringConfig("OSRM_CONTROL_DOCKER_BIN", ""),
            ),
        )
    }

    fun corsAllowedOrigins(): List<String> {
        val configured = readConfigValue("CORS_ALLOWED_ORIGINS") ?: return defaultCorsAllowedOrigins
        val origins = configured
            .split(',')
            .map { origin -> origin.trim() }
            .filter { origin -> origin.isNotEmpty() }
        return origins.ifEmpty { defaultCorsAllowedOrigins }
    }

    fun corsAllowedMethods(): List<String> = defaultCorsAllowedMethods

    fun corsAllowedHeaders(): List<String> = defaultCorsAllowedHeaders

    fun readConfigValue(key: String): String? {
        val property = System.getProperty(key)
        if (property != null) {
            return property.trim().takeIf { it.isNotEmpty() }
        }

        val fromEnv = System.getenv(key)?.trim()
        if (!fromEnv.isNullOrEmpty()) {
            return fromEnv
        }

        val dotEnv = File(".env")
        if (!dotEnv.exists() || !dotEnv.isFile) {
            return null
        }

        return dotEnv.useLines { lines ->
            lines
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

    private fun readFirstStringConfig(fallback: String, vararg keys: String): String {
        return keys.firstNotNullOfOrNull { key -> readConfigValue(key) } ?: fallback
    }

    private fun readStringConfig(key: String, fallback: String): String {
        return readConfigValue(key) ?: fallback
    }

    private fun readBoolConfig(key: String, fallback: Boolean): Boolean {
        val normalized = readConfigValue(key)?.lowercase(Locale.ROOT) ?: return fallback
        return when (normalized) {
            "1", "true", "yes", "y", "on" -> true
            "0", "false", "no", "n", "off" -> false
            else -> fallback
        }
    }

    private fun readIntConfig(key: String, fallback: Int): Int {
        return readConfigValue(key)?.toIntOrNull() ?: fallback
    }

    private fun normalizedRoutingTimeoutMs(): Int {
        val timeoutMs = readIntConfig("OSM_ROUTING_TIMEOUT_MS", DEFAULT_OSM_ROUTING_TIMEOUT_MS)
        return timeoutMs.coerceAtLeast(300)
    }

    private fun isConfigured(key: String): Boolean {
        return readConfigValue(key) != null
    }

    private fun sourceFor(key: String): String {
        return if (isConfigured(key)) key else "default"
    }
}
