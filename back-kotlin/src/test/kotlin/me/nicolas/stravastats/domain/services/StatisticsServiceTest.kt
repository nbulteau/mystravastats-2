package me.nicolas.stravastats.domain.services

import io.mockk.every
import io.mockk.mockk
import me.nicolas.stravastats.TestHelper
import me.nicolas.stravastats.domain.business.ActivityType
import me.nicolas.stravastats.domain.business.strava.AthleteRef
import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.business.strava.stream.AltitudeStream
import me.nicolas.stravastats.domain.business.strava.stream.DistanceStream
import me.nicolas.stravastats.domain.business.strava.stream.Stream
import me.nicolas.stravastats.domain.business.strava.stream.TimeStream
import me.nicolas.stravastats.domain.services.activityproviders.IActivityProvider
import org.junit.jupiter.api.BeforeEach
import org.junit.jupiter.api.Test
import org.junit.jupiter.api.extension.ExtendWith
import org.springframework.test.context.junit.jupiter.SpringExtension
import kotlin.test.assertEquals
import kotlin.test.assertNull
import kotlin.test.assertTrue

@ExtendWith(SpringExtension::class)
class StatisticsServiceTest {

    private lateinit var statisticsService: IStatisticsService

    private val activityProvider = mockk<IActivityProvider>()

    private val run2020Activities = TestHelper.run2020Activities()

    private val ride2020Activities = TestHelper.ride2020Activities()

    private val hike2020Activities = TestHelper.hike2020Activities()

    @BeforeEach
    fun setUp() {
        statisticsService = StatisticsService(activityProvider)
    }

    @Test
    fun `compute statistics for Run activity type`() {
        // GIVEN
        val activityTypes = setOf(ActivityType.Run)
        val year = 2020

        every { activityProvider.getActivitiesByActivityTypeAndYear(activityTypes, 2020) } returns run2020Activities

        // WHEN
        val result = statisticsService.getStatistics(activityTypes, year)

        // THEN
        assertEquals(31, result.size)
        assertEquals(53, result.find { statistic -> statistic.name == "Nb activities" }?.value?.toInt())
    }

    @Test
    fun `compute statistics for Ride activity type`() {
        // GIVEN
        val activityTypes = setOf(ActivityType.Ride)
        val year = 2020

        every { activityProvider.getActivitiesByActivityTypeAndYear(activityTypes, 2020) } returns ride2020Activities

        // WHEN
        val result = statisticsService.getStatistics(activityTypes, year)

        // THEN
        assertEquals(40, result.size)
        assertEquals(44, result.find { statistic -> statistic.name == "Nb activities" }?.value?.toInt())
    }

    @Test
    fun `compute statistics for Hike activity type`() {
        // GIVEN
        val activityTypes = setOf(ActivityType.Hike)
        val year = 2020

        every { activityProvider.getActivitiesByActivityTypeAndYear(activityTypes, 2020) } returns hike2020Activities

        // WHEN
        val result = statisticsService.getStatistics(activityTypes, year)

        // THEN
        assertEquals(18, result.size)
        assertEquals(8, result.find { statistic -> statistic.name == "Nb activities" }?.value?.toInt())
    }

    @Test
    fun `personal records timeline keeps chronological order for best 1 h with mixed date formats`() {
        // GIVEN
        val activityTypes = setOf(ActivityType.Ride)
        val activities = listOf(
            rideActivityForTimeline(
                id = 1001L,
                startDate = "2026-04-05T06:00:00Z",
                startDateLocal = "2026-04-05T08:00:00+02:00",
                bestDistanceFor1hMeters = 25540.0,
            ),
            rideActivityForTimeline(
                id = 1002L,
                startDate = "2025-08-04T06:00:00Z",
                startDateLocal = "2025-08-04T08:00:00+02:00",
                bestDistanceFor1hMeters = 29770.0,
            ),
            rideActivityForTimeline(
                id = 1003L,
                startDate = "2024-08-16T06:00:00Z",
                startDateLocal = "2024-08-16T08:00:00+0200", // non-ISO offset format
                bestDistanceFor1hMeters = 34510.0,
            ),
        )

        every { activityProvider.getActivitiesByActivityTypeAndYear(activityTypes, null) } returns activities

        // WHEN
        val timeline = statisticsService.getPersonalRecordsTimeline(activityTypes, null, "best-distance-1h")

        // THEN
        assertEquals(1, timeline.size)
        assertEquals("Best 1 h", timeline[0].metricLabel)
        assertEquals("2024-08-16T08:00:00+0200", timeline[0].activityDate)
        assertNull(timeline[0].previousValue)
        assertNull(timeline[0].improvement)
        assertTrue(timeline[0].value.contains("34.51 km"))
    }

    @Test
    fun `personal records timeline for best 1 h contains initial pr on oldest event then progresses`() {
        // GIVEN
        val activityTypes = setOf(ActivityType.Ride)
        val activities = listOf(
            rideActivityForTimeline(
                id = 2001L,
                startDate = "2025-06-15T06:00:00Z",
                startDateLocal = "2025-06-15T08:00:00+02:00",
                bestDistanceFor1hMeters = 14_000.0,
            ),
            rideActivityForTimeline(
                id = 2002L,
                startDate = "2025-01-10T06:00:00Z",
                startDateLocal = "2025-01-10T08:00:00+02:00",
                bestDistanceFor1hMeters = 9_000.0,
            ),
            rideActivityForTimeline(
                id = 2003L,
                startDate = "2025-02-01T06:00:00Z",
                startDateLocal = "2025-02-01T08:00:00+02:00",
                bestDistanceFor1hMeters = 10_000.0,
            ),
            rideActivityForTimeline(
                id = 2004L,
                startDate = "2025-03-01T06:00:00Z",
                startDateLocal = "2025-03-01T08:00:00+02:00",
                bestDistanceFor1hMeters = 15_000.0,
            ),
        )

        every { activityProvider.getActivitiesByActivityTypeAndYear(activityTypes, null) } returns activities

        // WHEN
        val timeline = statisticsService.getPersonalRecordsTimeline(activityTypes, null, "best-distance-1h")

        // THEN
        assertEquals(3, timeline.size)
        assertEquals("2025-01-10T08:00:00+02:00", timeline[0].activityDate)
        assertNull(timeline[0].previousValue)
        assertNull(timeline[0].improvement)

        assertEquals("2025-02-01T08:00:00+02:00", timeline[1].activityDate)
        assertTrue(timeline[1].previousValue!!.startsWith("9.00 km"))
        assertTrue(timeline[1].improvement!!.contains("1.00 km farther"))

        assertEquals("2025-03-01T08:00:00+02:00", timeline[2].activityDate)
        assertTrue(timeline[2].previousValue!!.startsWith("10.00 km"))
        assertTrue(timeline[2].improvement!!.contains("5.00 km farther"))
    }

    @Test
    fun `personal records timeline returns empty list for unknown metric key`() {
        // GIVEN
        val activityTypes = setOf(ActivityType.Ride)
        val activities = listOf(
            rideActivityForTimeline(
                id = 3001L,
                startDate = "2025-01-10T06:00:00Z",
                startDateLocal = "2025-01-10T08:00:00+02:00",
                bestDistanceFor1hMeters = 9_000.0,
            ),
        )
        every { activityProvider.getActivitiesByActivityTypeAndYear(activityTypes, null) } returns activities

        // WHEN
        val timeline = statisticsService.getPersonalRecordsTimeline(activityTypes, null, "unknown-metric")

        // THEN
        assertEquals(0, timeline.size)
    }

    private fun rideActivityForTimeline(
        id: Long,
        startDate: String,
        startDateLocal: String,
        bestDistanceFor1hMeters: Double,
    ): StravaActivity {
        return StravaActivity(
            athlete = AthleteRef(id = 1),
            averageSpeed = bestDistanceFor1hMeters / 3600.0,
            averageCadence = 80.0,
            averageHeartrate = 150.0,
            maxHeartrate = 180,
            averageWatts = 220,
            commute = false,
            distance = bestDistanceFor1hMeters,
            deviceWatts = true,
            elapsedTime = 3600,
            elevHigh = 1800.0,
            id = id,
            kilojoules = 600.0,
            maxSpeed = 12.0F,
            movingTime = 3600,
            name = "Ride $id",
            startDate = startDate,
            startDateLocal = startDateLocal,
            startLatlng = listOf(45.0, 6.0),
            totalElevationGain = 900.0,
            type = "Ride",
            uploadId = id + 100000,
            weightedAverageWatts = 230,
            stream = Stream(
                distance = DistanceStream(
                    data = listOf(0.0, bestDistanceFor1hMeters),
                    originalSize = 2,
                    resolution = "high",
                    seriesType = "distance",
                ),
                time = TimeStream(
                    data = listOf(0, 3600),
                    originalSize = 2,
                    resolution = "high",
                    seriesType = "time",
                ),
                altitude = AltitudeStream(
                    data = listOf(100.0, 200.0),
                    originalSize = 2,
                    resolution = "high",
                    seriesType = "distance",
                ),
            ),
        )
    }
}
