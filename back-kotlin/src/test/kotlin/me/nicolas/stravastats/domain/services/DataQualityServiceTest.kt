package me.nicolas.stravastats.domain.services

import io.mockk.every
import io.mockk.mockk
import me.nicolas.stravastats.domain.business.ActivityType
import me.nicolas.stravastats.domain.business.strava.AthleteRef
import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.business.strava.stream.AltitudeStream
import me.nicolas.stravastats.domain.business.strava.stream.DistanceStream
import me.nicolas.stravastats.domain.business.strava.stream.LatLngStream
import me.nicolas.stravastats.domain.business.strava.stream.Stream
import me.nicolas.stravastats.domain.business.strava.stream.TimeStream
import me.nicolas.stravastats.domain.services.activityproviders.ActivityProviderCacheIdentity
import me.nicolas.stravastats.domain.services.activityproviders.IActivityProvider
import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Assertions.assertTrue
import org.junit.jupiter.api.Test
import org.junit.jupiter.api.io.TempDir
import java.nio.file.Path

class DataQualityServiceTest {

    private val activityProvider = mockk<IActivityProvider>()

    @TempDir
    lateinit var tempDir: Path

    @Test
    fun `getReport detects missing streams and GPS glitches for local providers`() {
        val activity = dataQualityActivity(
            stream = Stream(
                distance = DistanceStream(listOf(0.0, 10.0, 20.0), 3, "high", "distance"),
                time = TimeStream(listOf(0, 1, 2), 3, "high", "time"),
                latlng = LatLngStream(
                    listOf(
                        listOf(48.0, -1.0),
                        listOf(49.0, -1.0),
                        listOf(49.0001, -1.0),
                    ),
                    3,
                    "high",
                    "time",
                ),
            )
        )
        every { activityProvider.getCacheDiagnostics() } returns mapOf("provider" to "fit", "fitDirectory" to "/tmp/fit")
        every { activityProvider.cacheIdentity() } returns null
        every { activityProvider.getActivitiesByActivityTypeAndYear(ActivityType.values().toSet(), null) } returns listOf(activity)

        val report = DataQualityService(activityProvider).getReport()

        assertEquals("warning", report.summary.status)
        assertEquals(1, report.summary.impactedActivities)
        assertEquals(1, report.summary.byCategory["GPS_GLITCH"])
        assertEquals(1, report.summary.byCategory["MISSING_STREAM_FIELD"])
        assertTrue(report.issues.any { issue -> issue.activityId == 123L && issue.category == "GPS_GLITCH" })
    }

    @Test
    fun `getReport classifies stream coverage separately`() {
        every { activityProvider.getCacheDiagnostics() } returns mapOf("provider" to "strava")
        every { activityProvider.cacheIdentity() } returns null
        every { activityProvider.getActivitiesByActivityTypeAndYear(ActivityType.values().toSet(), null) } returns listOf(
            dataQualityActivity(stream = completeStream()).copy(id = 11, averageWatts = 180, deviceWatts = true),
            dataQualityActivity(stream = completeStream()).copy(id = 12, averageWatts = 160, deviceWatts = false),
        )

        val report = DataQualityService(activityProvider).getReport()

        assertEquals(1, report.summary.byCategory["STREAM_DATA_COVERAGE"])
    }

    @Test
    fun `getReport detects downloadable Strava streams missing from cache`() {
        every { activityProvider.getCacheDiagnostics() } returns mapOf("provider" to "strava")
        every { activityProvider.cacheIdentity() } returns null
        every { activityProvider.getActivitiesByActivityTypeAndYear(ActivityType.values().toSet(), null) } returns listOf(
            dataQualityActivity(stream = null).copy(uploadId = 12345)
        )

        val report = DataQualityService(activityProvider).getReport()

        assertEquals("ok", report.summary.status)
        assertEquals(1, report.summary.byCategory["MISSING_STREAM"])
    }

    @Test
    fun `getReport detects Strava summary anomalies without requiring streams`() {
        every { activityProvider.getCacheDiagnostics() } returns mapOf("provider" to "strava")
        every { activityProvider.cacheIdentity() } returns null
        every { activityProvider.getActivitiesByActivityTypeAndYear(ActivityType.values().toSet(), null) } returns listOf(
            dataQualityActivity(stream = null).copy(distance = 100_000.0, elapsedTime = 1_200, movingTime = 1_200)
        )

        val report = DataQualityService(activityProvider).getReport()

        assertEquals("warning", report.summary.status)
        assertEquals(1, report.summary.issueCount)
        assertEquals(1, report.summary.byCategory["INVALID_VALUE"])
    }

    @Test
    fun `excludeActivityFromStats persists and marks report issues`() {
        val activity = dataQualityActivity(stream = null).copy(distance = 100_000.0, elapsedTime = 1_200, movingTime = 1_200)
        every { activityProvider.getCacheDiagnostics() } returns mapOf("provider" to "strava", "cacheRoot" to tempDir.toString())
        every { activityProvider.cacheIdentity() } returns ActivityProviderCacheIdentity(tempDir.toString(), "test")
        every { activityProvider.getActivitiesByActivityTypeAndYear(ActivityType.values().toSet(), null) } returns listOf(activity)

        val service = DataQualityService(activityProvider)
        val excluded = service.excludeActivityFromStats(activity.id, "bad speed")

        assertEquals(1, excluded.summary.excludedActivities)
        assertTrue(excluded.issues.all { issue -> issue.excludedFromStats })

        val included = service.includeActivityInStats(activity.id)
        assertEquals(0, included.summary.excludedActivities)
        assertTrue(included.issues.none { issue -> issue.excludedFromStats })
    }

    private fun dataQualityActivity(stream: Stream?): StravaActivity =
        StravaActivity(
            athlete = AthleteRef(id = 1),
            averageSpeed = 5.0,
            commute = false,
            distance = 10_000.0,
            elapsedTime = 600,
            id = 123,
            maxSpeed = 8.0f,
            movingTime = 600,
            name = "Suspicious ride",
            startDate = "2026-04-26T06:00:00Z",
            startDateLocal = "2026-04-26T08:00:00Z",
            startLatlng = null,
            totalElevationGain = 100.0,
            type = "Ride",
            uploadId = 0,
            stream = stream,
        )

    private fun completeStream(): Stream =
        Stream(
            distance = DistanceStream(listOf(0.0, 5_000.0, 10_000.0), 3, "high", "distance"),
            time = TimeStream(listOf(0, 900, 1_800), 3, "high", "time"),
            latlng = LatLngStream(
                listOf(
                    listOf(48.0, -1.0),
                    listOf(48.01, -1.0),
                    listOf(48.02, -1.0),
                ),
                3,
                "high",
                "time",
            ),
            altitude = AltitudeStream(listOf(50.0, 60.0, 70.0), 3, "high", "altitude"),
        )
}
