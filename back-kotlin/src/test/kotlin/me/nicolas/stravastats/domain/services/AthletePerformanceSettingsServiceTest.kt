package me.nicolas.stravastats.domain.services

import io.mockk.every
import io.mockk.mockk
import me.nicolas.stravastats.domain.business.ActivityType
import me.nicolas.stravastats.domain.business.strava.AthleteRef
import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.business.strava.stream.AltitudeStream
import me.nicolas.stravastats.domain.business.strava.stream.DistanceStream
import me.nicolas.stravastats.domain.business.strava.stream.PowerStream
import me.nicolas.stravastats.domain.business.strava.stream.Stream
import me.nicolas.stravastats.domain.business.strava.stream.TimeStream
import me.nicolas.stravastats.domain.services.activityproviders.IActivityProvider
import me.nicolas.stravastats.domain.services.statistics.BestEffortCache
import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.BeforeEach
import org.junit.jupiter.api.Test

class AthletePerformanceSettingsServiceTest {

    private val activityProvider = mockk<IActivityProvider>()
    private lateinit var service: IAthletePerformanceSettingsService

    @BeforeEach
    fun setUp() {
        BestEffortCache.clear()
        service = AthletePerformanceSettingsService(activityProvider)
        every { activityProvider.cacheIdentity() } returns null
    }

    @Test
    fun `estimateFtp uses recent device best 60 minute power`() {
        // GIVEN
        val activityTypes = setOf(ActivityType.Ride)
        every { activityProvider.getActivitiesByActivityTypeAndYear(activityTypes) } returns listOf(
            powerActivity(201, "Estimated power", "2026-06-18T09:00:00Z", deviceWatts = false, watts = 300, durationSeconds = 3600),
            powerActivity(202, "Current power meter", "2026-06-20T09:00:00Z", deviceWatts = true, watts = 210, durationSeconds = 3600),
            powerActivity(203, "Old power meter", "2025-01-01T09:00:00Z", deviceWatts = true, watts = 260, durationSeconds = 3600),
        )

        // WHEN
        val result = service.estimateFtp(activityTypes, windowDays = 180)

        // THEN
        assertEquals(true, result.available)
        assertEquals(210, result.ftp)
        assertEquals(202, result.activityId)
        assertEquals("best-60min", result.method)
        assertEquals("high", result.confidence)
        assertEquals("2026-06-20", result.activityDate)
    }

    @Test
    fun `estimateFtp falls back to 95 percent of 20 minute power`() {
        // GIVEN
        val activityTypes = setOf(ActivityType.Ride)
        every { activityProvider.getActivitiesByActivityTypeAndYear(activityTypes) } returns listOf(
            powerActivity(204, "Twenty minute test", "2026-06-20T09:00:00Z", deviceWatts = true, watts = 200, durationSeconds = 1200),
        )

        // WHEN
        val result = service.estimateFtp(activityTypes, windowDays = 180)

        // THEN
        assertEquals(true, result.available)
        assertEquals(190, result.ftp)
        assertEquals(200, result.bestPower)
        assertEquals(1200, result.basedOnSeconds)
        assertEquals("95-percent-20min", result.method)
    }

    @Test
    fun `estimateFtp selects highest average power rather than longest distance`() {
        // GIVEN
        val activityTypes = setOf(ActivityType.Ride)
        every { activityProvider.getActivitiesByActivityTypeAndYear(activityTypes) } returns listOf(
            powerActivity(205, "Long steady ride", "2026-06-19T09:00:00Z", deviceWatts = true, watts = 180, durationSeconds = 3600, totalDistance = 6000.0),
            powerActivity(206, "Short hard ride", "2026-06-20T09:00:00Z", deviceWatts = true, watts = 230, durationSeconds = 3600, totalDistance = 1500.0),
        )

        // WHEN
        val result = service.estimateFtp(activityTypes, windowDays = 180)

        // THEN
        assertEquals(206, result.activityId)
        assertEquals(230, result.ftp)
    }

    private fun powerActivity(
        id: Long,
        name: String,
        startDateLocal: String,
        deviceWatts: Boolean,
        watts: Int,
        durationSeconds: Int,
        totalDistance: Double = durationSeconds / 600.0 * 500.0,
    ): StravaActivity {
        val points = durationSeconds / 600 + 1
        val distances = List(points) { index -> totalDistance * index / (points - 1).coerceAtLeast(1) }
        val times = List(points) { index -> index * 600 }
        val altitudes = List(points) { index -> 100.0 + index }
        val powers = List(points) { watts }
        return StravaActivity(
            athlete = AthleteRef(id = 1),
            averageSpeed = 0.0,
            averageCadence = 0.0,
            averageHeartrate = 0.0,
            maxHeartrate = 0,
            averageWatts = watts,
            commute = false,
            distance = totalDistance,
            deviceWatts = deviceWatts,
            elapsedTime = durationSeconds,
            elevHigh = altitudes.maxOrNull() ?: 0.0,
            id = id,
            kilojoules = 0.0,
            maxSpeed = 0f,
            movingTime = durationSeconds,
            name = name,
            startDate = startDateLocal,
            startDateLocal = startDateLocal,
            startLatlng = null,
            totalElevationGain = 0.0,
            type = "Ride",
            uploadId = id + 1000,
            weightedAverageWatts = watts,
            stream = Stream(
                distance = DistanceStream(distances, distances.size, "high", "distance"),
                time = TimeStream(times, times.size, "high", "time"),
                altitude = AltitudeStream(altitudes, altitudes.size, "high", "distance"),
                watts = PowerStream(powers, powers.size, "high", "time"),
            ),
        )
    }
}
