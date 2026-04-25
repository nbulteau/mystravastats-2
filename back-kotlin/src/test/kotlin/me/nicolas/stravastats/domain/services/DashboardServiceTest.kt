package me.nicolas.stravastats.domain.services

import io.mockk.every
import io.mockk.mockk
import me.nicolas.stravastats.domain.business.AnnualGoalMetric
import me.nicolas.stravastats.domain.business.AnnualGoalStatus
import me.nicolas.stravastats.domain.business.AnnualGoalTargets
import me.nicolas.stravastats.domain.business.ActivityType
import me.nicolas.stravastats.domain.business.strava.AthleteRef
import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.business.strava.stream.AltitudeStream
import me.nicolas.stravastats.domain.business.strava.stream.DistanceStream
import me.nicolas.stravastats.domain.business.strava.stream.Stream
import me.nicolas.stravastats.domain.business.strava.stream.TimeStream
import me.nicolas.stravastats.domain.services.activityproviders.ActivityProviderCacheIdentity
import me.nicolas.stravastats.domain.services.activityproviders.IActivityProvider
import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Assertions.assertTrue
import org.junit.jupiter.api.BeforeEach
import org.junit.jupiter.api.Test
import org.junit.jupiter.api.io.TempDir
import java.nio.file.Path
import java.time.LocalDate

class DashboardServiceTest {

    private lateinit var dashboardService: IDashboardService

    private val activityProvider = mockk<IActivityProvider>()

    @TempDir
    private lateinit var tempDir: Path

    @BeforeEach
    fun setUp() {
        dashboardService = DashboardService(activityProvider)
    }

    @Test
    fun `getEddingtonNumber returns zero when no active day is available`() {
        // GIVEN
        every {
            activityProvider.getActivitiesByActivityTypeGroupByActiveDays(setOf(ActivityType.Ride))
        } returns emptyMap()

        // WHEN
        val result = dashboardService.getEddingtonNumber(setOf(ActivityType.Ride))

        // THEN
        assertEquals(0, result.eddingtonNumber)
        assertEquals(emptyList<Int>(), result.eddingtonList)
    }

    @Test
    fun `getEddingtonNumber does not round up when equality threshold is not met`() {
        // GIVEN
        val dailyTotals = (1..49).associate { day -> "2024-01-${day.toString().padStart(2, '0')}" to 51 }
        every {
            activityProvider.getActivitiesByActivityTypeGroupByActiveDays(setOf(ActivityType.Ride))
        } returns dailyTotals

        // WHEN
        val result = dashboardService.getEddingtonNumber(setOf(ActivityType.Ride))

        // THEN
        assertEquals(49, result.eddingtonNumber)
        assertEquals(51, result.eddingtonList.size)
        assertEquals(49, result.eddingtonList[48]) // >= 49km
        assertEquals(49, result.eddingtonList[49]) // >= 50km
    }

    @Test
    fun `getEddingtonNumber ignores non positive daily totals`() {
        // GIVEN
        every {
            activityProvider.getActivitiesByActivityTypeGroupByActiveDays(setOf(ActivityType.Ride))
        } returns mapOf(
            "2025-01-01" to 4,
            "2025-01-02" to 4,
            "2025-01-03" to 4,
            "2025-01-04" to 4,
            "2025-01-05" to 0,
            "2025-01-06" to -2,
        )

        // WHEN
        val result = dashboardService.getEddingtonNumber(setOf(ActivityType.Ride))

        // THEN
        assertEquals(4, result.eddingtonNumber)
        assertEquals(listOf(4, 4, 4, 4), result.eddingtonList)
    }

    @Test
    fun `getDashboardData computes active days moving time and elevation efficiency for a past year`() {
        // GIVEN
        val activityTypes = setOf(ActivityType.Ride)
        val activities = listOf(
            createActivity(
                id = 1L,
                startDateLocal = "2025-01-01T08:00:00Z",
                distanceMeters = 20000.0,
                elevationGainMeters = 300.0,
                movingTimeSeconds = 3600,
                elapsedTimeSeconds = 3800,
            ),
            createActivity(
                id = 2L,
                startDateLocal = "2025-01-01T18:00:00Z",
                distanceMeters = 10000.0,
                elevationGainMeters = 100.0,
                movingTimeSeconds = 0,
                elapsedTimeSeconds = 1800,
            ),
            createActivity(
                id = 3L,
                startDateLocal = "2025-01-03T07:30:00Z",
                distanceMeters = 30000.0,
                elevationGainMeters = 600.0,
                movingTimeSeconds = 5400,
                elapsedTimeSeconds = 5600,
            ),
        )
        every {
            activityProvider.getActivitiesByActivityTypeAndYear(activityTypes)
        } returns activities

        // WHEN
        val result = dashboardService.getDashboardData(activityTypes)

        // THEN
        assertEquals(3, result.nbActivitiesByYear["2025"])
        assertEquals(2, result.activeDaysByYear["2025"])
        assertEquals(10800, result.movingTimeByYear["2025"])
        assertEquals(60.0f, result.totalDistanceByYear["2025"])
        assertEquals(1000, result.totalElevationByYear["2025"])
        val expectedEfficiency = (1000.0f / 60.0f) * 10.0f
        assertEquals(expectedEfficiency, result.elevationEfficiencyByYear["2025"]!!, 0.001f)
    }

    @Test
    fun `getDashboardData uses year to date consistency for current year`() {
        // GIVEN
        val activityTypes = setOf(ActivityType.Ride)
        val currentYear = LocalDate.now().year
        val activities = listOf(
            createActivity(
                id = 10L,
                startDateLocal = "$currentYear-01-01T08:00:00Z",
                distanceMeters = 12000.0,
                elevationGainMeters = 120.0,
                movingTimeSeconds = 2400,
                elapsedTimeSeconds = 2500,
            )
        )
        every {
            activityProvider.getActivitiesByActivityTypeAndYear(activityTypes)
        } returns activities

        // WHEN
        val result = dashboardService.getDashboardData(activityTypes)

        // THEN
        val expected = kotlin.math.round((1.0 / LocalDate.now().dayOfYear.toDouble()) * 1000.0) / 10.0
        assertEquals(expected.toFloat(), result.consistencyByYear[currentYear.toString()]!!, 0.001f)
        assertTrue(result.consistencyByYear[currentYear.toString()]!! > 0.0f)
    }

    @Test
    fun `saveAnnualGoals persists targets locally and returns projections`() {
        // GIVEN
        val activityTypes = setOf(ActivityType.Ride)
        val activities = listOf(
            createActivity(
                id = 20L,
                startDateLocal = "2025-01-01T08:00:00Z",
                distanceMeters = 20000.0,
                elevationGainMeters = 200.0,
                movingTimeSeconds = 3600,
                elapsedTimeSeconds = 3600,
            ),
            createActivity(
                id = 21L,
                startDateLocal = "2025-01-02T08:00:00Z",
                distanceMeters = 40000.0,
                elevationGainMeters = 500.0,
                movingTimeSeconds = 7200,
                elapsedTimeSeconds = 7200,
            ),
        )
        every { activityProvider.cacheIdentity() } returns ActivityProviderCacheIdentity(
            cacheRoot = tempDir.toString(),
            athleteId = "athlete-1",
        )
        every { activityProvider.getActivitiesByActivityTypeAndYear(activityTypes, 2025) } returns activities

        // WHEN
        val saved = dashboardService.saveAnnualGoals(
            year = 2025,
            activityTypes = activityTypes,
            targets = AnnualGoalTargets(distanceKm = 60.0, eddington = 2),
        )
        val loaded = dashboardService.getAnnualGoals(2025, activityTypes)

        // THEN
        assertEquals(60.0, saved.targets.distanceKm)
        assertEquals(60.0, loaded.targets.distanceKm)
        val distance = saved.progress.first { item -> item.metric == AnnualGoalMetric.DISTANCE_KM }
        assertEquals(60.0, distance.current)
        assertEquals(AnnualGoalStatus.ON_TRACK, distance.status)
        assertTrue(tempDir.resolve("strava-athlete-1").resolve("annual-goals-athlete-1.json").toFile().exists())
    }

    @Test
    fun `getAnnualGoals returns all goal rows and computes annual Eddington for selected year`() {
        // GIVEN
        val activityTypes = setOf(ActivityType.Ride)
        val activities = listOf(
            createActivity(
                id = 30L,
                startDateLocal = "2025-01-01T08:00:00Z",
                distanceMeters = 3000.0,
                elevationGainMeters = 0.0,
                movingTimeSeconds = 900,
                elapsedTimeSeconds = 900,
            ),
            createActivity(
                id = 31L,
                startDateLocal = "2025-01-02T08:00:00Z",
                distanceMeters = 2000.0,
                elevationGainMeters = 0.0,
                movingTimeSeconds = 900,
                elapsedTimeSeconds = 900,
            ),
        )
        every { activityProvider.cacheIdentity() } returns ActivityProviderCacheIdentity(
            cacheRoot = tempDir.toString(),
            athleteId = "athlete-2",
        )
        every { activityProvider.getActivitiesByActivityTypeAndYear(activityTypes, 2025) } returns activities

        // WHEN
        val result = dashboardService.getAnnualGoals(2025, activityTypes)

        // THEN
        assertEquals(6, result.progress.size)
        val eddington = result.progress.first { item -> item.metric == AnnualGoalMetric.EDDINGTON }
        assertEquals(2.0, eddington.current)
        assertEquals(AnnualGoalStatus.NOT_SET, eddington.status)
    }

    private fun createActivity(
        id: Long,
        startDateLocal: String,
        distanceMeters: Double,
        elevationGainMeters: Double,
        movingTimeSeconds: Int,
        elapsedTimeSeconds: Int,
    ): StravaActivity {
        return StravaActivity(
            athlete = AthleteRef(1),
            averageSpeed = if (movingTimeSeconds > 0) distanceMeters / movingTimeSeconds else 0.0,
            commute = false,
            distance = distanceMeters,
            elapsedTime = elapsedTimeSeconds,
            id = id,
            maxSpeed = 0.0f,
            movingTime = movingTimeSeconds,
            name = "Activity $id",
            startDate = startDateLocal,
            startDateLocal = startDateLocal,
            startLatlng = listOf(48.1, -1.6),
            totalElevationGain = elevationGainMeters,
            type = "Ride",
            uploadId = 1000L + id,
            stream = simpleStream(),
        )
    }

    private fun simpleStream(): Stream {
        val distance = listOf(0.0, 100.0, 200.0, 300.0, 400.0)
        val time = listOf(0, 20, 40, 60, 80)
        val altitude = listOf(10.0, 12.0, 15.0, 18.0, 20.0)
        return Stream(
            distance = DistanceStream(distance, distance.size, "high", "distance"),
            time = TimeStream(time, time.size, "high", "distance"),
            altitude = AltitudeStream(altitude),
        )
    }
}
