package me.nicolas.stravastats.domain.services

import io.mockk.every
import io.mockk.mockk
import me.nicolas.stravastats.domain.business.ActivityType
import me.nicolas.stravastats.domain.business.GearKind
import me.nicolas.stravastats.domain.business.GearMaintenanceRecordRequest
import me.nicolas.stravastats.domain.business.strava.AthleteRef
import me.nicolas.stravastats.domain.business.strava.Bike
import me.nicolas.stravastats.domain.business.strava.Shoe
import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.business.strava.StravaAthlete
import me.nicolas.stravastats.domain.services.activityproviders.ActivityProviderCacheIdentity
import me.nicolas.stravastats.domain.services.activityproviders.IActivityProvider
import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Assertions.assertFalse
import org.junit.jupiter.api.Assertions.assertTrue
import org.junit.jupiter.api.BeforeEach
import org.junit.jupiter.api.Test
import org.junit.jupiter.api.io.TempDir
import java.nio.file.Path

class GearAnalysisServiceTest {

    private lateinit var gearAnalysisService: IGearAnalysisService

    private val activityProvider = mockk<IActivityProvider>()

    @TempDir
    lateinit var tempDir: Path

    @BeforeEach
    fun setUp() {
        gearAnalysisService = GearAnalysisService(activityProvider)
    }

    @Test
    fun `getGearAnalysis aggregates gear and unassigned activities`() {
        val activityTypes = setOf(ActivityType.Ride)
        val activities = listOf(
            gearAnalysisActivity(1L, "Morning ride", "Ride", "2026-01-03T08:00:00Z", "b123", 10000.0, 1800, 100.0),
            gearAnalysisActivity(2L, "Long ride", "Ride", "2026-02-05T08:00:00Z", "b123", 20000.0, 3000, 300.0),
            gearAnalysisActivity(3L, "Run shoes", "Run", "2026-03-07T08:00:00Z", "g456", 5000.0, 1500, 20.0),
            gearAnalysisActivity(4L, "No gear", "Hike", "2026-03-08T08:00:00Z", null, 7000.0, 2400, 200.0),
        )
        val athlete = StravaAthlete(
            id = 42L,
            bikes = listOf(
                Bike(
                    id = "b123",
                    name = "Road Bike",
                    nickname = "Fast bike",
                    retired = false,
                    convertedDistance = 0.0,
                    distance = 0,
                    primary = true,
                    resourceState = 2,
                )
            ),
            shoes = listOf(
                Shoe(
                    id = "g456",
                    name = "Trail Shoes",
                    nickname = null,
                    retired = null,
                    convertedDistance = 0.0,
                    distance = 0,
                    primary = true,
                    resourceState = 2,
                )
            )
        )

        every { activityProvider.getActivitiesByActivityTypeAndYear(activityTypes, 2026) } returns activities
        every { activityProvider.getActivitiesByActivityTypeAndYear(ActivityType.values().toSet(), null) } returns activities
        every { activityProvider.athlete() } returns athlete

        val result = gearAnalysisService.getGearAnalysis(activityTypes, 2026)

        assertEquals(4, result.coverage.totalActivities)
        assertEquals(3, result.coverage.assignedActivities)
        assertEquals(1, result.coverage.unassignedActivities)
        assertEquals(2, result.items.size)

        val bike = result.items.first()
        assertEquals("b123", bike.id)
        assertEquals("Fast bike", bike.name)
        assertEquals(GearKind.BIKE, bike.kind)
        assertTrue(bike.primary)
        assertFalse(bike.retired)
        assertEquals(30000.0, bike.distance)
        assertEquals(30000.0, bike.totalDistance)
        assertEquals(4800, bike.movingTime)
        assertEquals(400.0, bike.elevationGain)
        assertEquals(2, bike.activities)
        assertEquals(6.3, bike.averageSpeed)
        assertEquals("2026-01-03T08:00:00Z", bike.firstUsed)
        assertEquals("2026-02-05T08:00:00Z", bike.lastUsed)
        assertEquals(2L, bike.longestActivity?.id)
        assertEquals(2L, bike.fastestActivity?.id)
        assertEquals("2026-01", bike.monthlyDistance[0].periodKey)
        assertEquals(10000.0, bike.monthlyDistance[0].value)

        assertEquals(1, result.unassigned.activities)
        assertEquals(7000.0, result.unassigned.distance)
        assertEquals(200.0, result.unassigned.elevationGain)
    }

    @Test
    fun `getGearAnalysis uses lifetime bike distance for maintenance odometer`() {
        val activityTypes = setOf(ActivityType.Ride)
        val activities2026 = listOf(
            gearAnalysisActivity(1L, "Current year ride", "Ride", "2026-01-03T08:00:00Z", "b123", 1_000_000.0, 1800, 100.0),
        )
        val lifetimeActivities = activities2026 + listOf(
            gearAnalysisActivity(2L, "Previous year ride", "Ride", "2025-01-03T08:00:00Z", "b123", 9_000_000.0, 1800, 100.0),
        )
        val athlete = StravaAthlete(
            id = 42L,
            bikes = listOf(
                Bike(
                    id = "b123",
                    name = "Road Bike",
                    nickname = null,
                    retired = false,
                    convertedDistance = 0.0,
                    distance = 0,
                    primary = true,
                    resourceState = 2,
                )
            ),
        )
        every { activityProvider.athlete() } returns athlete
        every { activityProvider.cacheIdentity() } returns ActivityProviderCacheIdentity(
            cacheRoot = tempDir.toString(),
            athleteId = "athlete-1",
        )
        every { activityProvider.getActivitiesByActivityTypeAndYear(activityTypes, 2026) } returns activities2026
        every { activityProvider.getActivitiesByActivityTypeAndYear(ActivityType.values().toSet(), null) } returns lifetimeActivities

        gearAnalysisService.saveMaintenanceRecord(
            GearMaintenanceRecordRequest(
                gearId = "b123",
                component = "CHAIN",
                operation = "Chain changed",
                date = "2026-01-01",
                distance = 0.0,
            )
        )
        gearAnalysisService.saveMaintenanceRecord(
            GearMaintenanceRecordRequest(
                gearId = "b123",
                component = "Tires",
                operation = "Tires changed",
                date = "2026-01-01",
                distance = 0.0,
            )
        )

        val result = gearAnalysisService.getGearAnalysis(activityTypes, 2026)
        val bike = result.items.first()
        val chain = bike.maintenanceTasks.first { it.component == "CHAIN" }
        val frontTire = bike.maintenanceTasks.first { it.component == "TIRE_FRONT" }
        val rearTire = bike.maintenanceTasks.first { it.component == "TIRE_REAR" }

        assertEquals(1_000_000.0, bike.distance)
        assertEquals(10_000_000.0, bike.totalDistance)
        assertEquals(10_000_000.0, chain.distanceSince)
        assertEquals(10_000_000.0, frontTire.distanceSince)
        assertEquals(10_000_000.0, rearTire.distanceSince)
    }

    @Test
    fun `saveMaintenanceRecord accepts free-form components`() {
        val athlete = StravaAthlete(
            id = 42L,
            bikes = listOf(
                Bike(
                    id = "b123",
                    name = "Gravel Bike",
                    nickname = null,
                    retired = false,
                    convertedDistance = 0.0,
                    distance = 0,
                    primary = true,
                    resourceState = 2,
                )
            ),
        )
        every { activityProvider.athlete() } returns athlete
        every { activityProvider.cacheIdentity() } returns ActivityProviderCacheIdentity(
            cacheRoot = tempDir.toString(),
            athleteId = "athlete-1",
        )

        val record = gearAnalysisService.saveMaintenanceRecord(
            GearMaintenanceRecordRequest(
                gearId = " b123 ",
                component = "Rear valve core",
                operation = "",
                date = "2026-04-27T12:00:00Z",
                distance = 3603000.0,
                note = " slow leak ",
            )
        )

        assertEquals("b123", record.gearId)
        assertEquals("REAR_VALVE_CORE", record.component)
        assertEquals("Rear Valve Core", record.componentLabel)
        assertEquals("Rear Valve Core serviced", record.operation)
        assertEquals("2026-04-27", record.date)
        assertEquals("slow leak", record.note)
    }

    private fun gearAnalysisActivity(
        id: Long,
        name: String,
        type: String,
        startDateLocal: String,
        gearId: String?,
        distance: Double,
        movingTime: Int,
        elevationGain: Double,
    ): StravaActivity {
        return StravaActivity(
            athlete = AthleteRef(id = 1),
            averageSpeed = 0.0,
            averageCadence = 0.0,
            averageHeartrate = 0.0,
            maxHeartrate = 0,
            averageWatts = 0,
            commute = false,
            distance = distance,
            deviceWatts = false,
            elapsedTime = movingTime,
            elevHigh = 0.0,
            id = id,
            kilojoules = 0.0,
            maxSpeed = 0.0F,
            movingTime = movingTime,
            name = name,
            startDate = startDateLocal,
            startDateLocal = startDateLocal,
            startLatlng = null,
            totalElevationGain = elevationGain,
            type = type,
            uploadId = 0L,
            weightedAverageWatts = 0,
            gearId = gearId,
        )
    }
}
