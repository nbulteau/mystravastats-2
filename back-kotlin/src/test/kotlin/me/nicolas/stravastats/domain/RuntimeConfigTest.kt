package me.nicolas.stravastats.domain

import org.junit.jupiter.api.AfterEach
import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Assertions.assertFalse
import org.junit.jupiter.api.Assertions.assertTrue
import org.junit.jupiter.api.Test

class RuntimeConfigTest {
    private val runtimeKeys = listOf(
        "STRAVA_CACHE_PATH",
        "FIT_FILES_PATH",
        "GPX_FILES_PATH",
        "SERVER_ADDRESS",
        "SERVER_PORT",
        "PORT",
        "OPEN_BROWSER",
        "CORS_ALLOWED_ORIGINS",
        "OSM_ROUTING_ENABLED",
        "OSM_ROUTING_BASE_URL",
        "OSM_ROUTING_TIMEOUT_MS",
        "OSM_ROUTING_PROFILE",
        "OSM_ROUTING_EXTRACT_PROFILE",
        "OSM_ROUTING_EXTRACT_PROFILE_FILE",
        "OSM_ROUTING_V3_ENABLED",
        "OSM_ROUTING_DEBUG",
        "OSM_ROUTING_HISTORY_BIAS_ENABLED",
        "OSM_ROUTING_HISTORY_HALF_LIFE_DAYS",
        "OSRM_CONTROL_ENABLED",
        "OSRM_CONTROL_TIMEOUT_MS",
        "OSRM_CONTROL_PROJECT_DIR",
        "OSRM_CONTROL_COMPOSE_FILE",
        "OSRM_CONTROL_DOCKER_BIN",
    )

    @AfterEach
    fun tearDown() {
        runtimeKeys.forEach(System::clearProperty)
    }

    @Test
    fun `details defaults to strava runtime config`() {
        clearRuntimeConfigForTest()

        val details = RuntimeConfig.details()
        val data = details["data"] as Map<*, *>
        val cors = details["cors"] as Map<*, *>
        val routing = details["routing"] as Map<*, *>

        assertEquals("kotlin", details["backend"])
        assertEquals("strava", data["provider"])
        assertEquals("strava-cache", data["stravaCachePath"])
        assertFalse(data["fitFilesConfigured"] as Boolean)
        assertTrue(data["gpxFilesSupported"] as Boolean)
        assertEquals(listOf("http://localhost", "http://localhost:5173"), cors["allowedOrigins"])
        assertEquals(listOf("Content-Type", "Authorization", "X-Request-Id"), cors["allowedHeaders"])
        assertEquals(true, cors["allowCredentials"])
        assertEquals("http://localhost:5000", routing["baseUrl"])
        assertEquals(3000, routing["timeoutMs"])
        assertEquals(75, routing["historyHalfLifeDays"])
        assertEquals(true, routing["controlEnabled"])
        assertEquals("docker-compose-routing-osrm.yml", routing["controlComposeFile"])
    }

    @Test
    fun `details exposes configured runtime values`() {
        clearRuntimeConfigForTest()
        System.setProperty("FIT_FILES_PATH", "/data/fit")
        System.setProperty("GPX_FILES_PATH", "/data/gpx")
        System.setProperty("STRAVA_CACHE_PATH", "/data/strava")
        System.setProperty("CORS_ALLOWED_ORIGINS", "http://localhost:5173, https://app.example")
        System.setProperty("OSM_ROUTING_ENABLED", "false")
        System.setProperty("OSM_ROUTING_BASE_URL", "http://osrm:5000/")
        System.setProperty("OSM_ROUTING_TIMEOUT_MS", "250")
        System.setProperty("OSM_ROUTING_HISTORY_HALF_LIFE_DAYS", "90")
        System.setProperty("OSRM_CONTROL_ENABLED", "false")
        System.setProperty("OSRM_CONTROL_TIMEOUT_MS", "12000")

        val details = RuntimeConfig.details()
        val data = details["data"] as Map<*, *>
        val cors = details["cors"] as Map<*, *>
        val routing = details["routing"] as Map<*, *>

        assertEquals("fit", data["provider"])
        assertEquals("/data/fit", data["fitFilesPath"])
        assertTrue(data["fitFilesConfigured"] as Boolean)
        assertEquals("/data/gpx", data["gpxFilesPath"])
        assertEquals(listOf("http://localhost:5173", "https://app.example"), cors["allowedOrigins"])
        assertEquals(false, routing["enabled"])
        assertEquals("http://osrm:5000", routing["baseUrl"])
        assertEquals(300, routing["timeoutMs"])
        assertEquals(90, routing["historyHalfLifeDays"])
        assertEquals(false, routing["controlEnabled"])
        assertEquals(12000, routing["controlTimeoutMs"])
    }

    private fun clearRuntimeConfigForTest() {
        runtimeKeys.forEach { key -> System.setProperty(key, "") }
    }
}
