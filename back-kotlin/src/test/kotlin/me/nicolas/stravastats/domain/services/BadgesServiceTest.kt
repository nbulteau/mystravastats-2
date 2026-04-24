package me.nicolas.stravastats.domain.services

import io.mockk.every
import io.mockk.mockk
import me.nicolas.stravastats.api.dto.toDto
import me.nicolas.stravastats.domain.business.ActivityType
import me.nicolas.stravastats.domain.business.badges.DistanceBadge
import me.nicolas.stravastats.domain.business.badges.ElevationBadge
import me.nicolas.stravastats.domain.business.badges.MovingTimeBadge
import me.nicolas.stravastats.domain.business.strava.AthleteRef
import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.services.activityproviders.IActivityProvider
import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Assertions.assertTrue
import org.junit.jupiter.api.BeforeEach
import org.junit.jupiter.api.Test

class BadgesServiceTest {
    private lateinit var badgesService: IBadgesService

    private val activityProvider = mockk<IActivityProvider>()

    @BeforeEach
    fun setUp() {
        badgesService = BadgesService(activityProvider)
    }

    @Test
    fun `getGeneralBadges uses cycling badge family for gravel mountain bike and ride selections`() {
        // GIVEN
        val activityTypes = setOf(ActivityType.GravelRide, ActivityType.MountainBikeRide, ActivityType.Ride)
        every {
            activityProvider.getActivitiesByActivityTypeAndYear(activityTypes, 2026)
        } returns listOf(
            activity(type = ActivityType.GravelRide, distance = 60_000.0, totalElevationGain = 1_200.0, movingTime = 7_200),
        )

        // WHEN
        val results = badgesService.getGeneralBadges(activityTypes, 2026)

        // THEN
        assertTrue(results.any { it.badge is DistanceBadge && it.isCompleted })
        assertTrue(results.any { it.badge is ElevationBadge && it.isCompleted })
        assertTrue(results.any { it.badge is MovingTimeBadge && it.isCompleted })
        assertEquals(
            setOf("RideDistanceBadge", "RideElevationBadge", "RideMovingTimeBadge"),
            results.map { it.toDto(activityTypes).badge.type }.toSet(),
        )
    }

    @Test
    fun `getGeneralBadges uses running badge family for trail run selections`() {
        // GIVEN
        val activityTypes = setOf(ActivityType.TrailRun)
        every {
            activityProvider.getActivitiesByActivityTypeAndYear(activityTypes, 2026)
        } returns listOf(
            activity(type = ActivityType.TrailRun, distance = 12_000.0, totalElevationGain = 300.0, movingTime = 4_200),
        )

        // WHEN
        val results = badgesService.getGeneralBadges(activityTypes, 2026)

        // THEN
        assertTrue(results.any { it.badge is DistanceBadge && it.isCompleted })
        assertEquals(
            setOf("RunDistanceBadge", "RunElevationBadge", "RunMovingTimeBadge"),
            results.map { it.toDto(activityTypes).badge.type }.toSet(),
        )
    }

    @Test
    fun `getGeneralBadges uses hiking badge family for walk selections`() {
        // GIVEN
        val activityTypes = setOf(ActivityType.Walk)
        every {
            activityProvider.getActivitiesByActivityTypeAndYear(activityTypes, 2026)
        } returns listOf(
            activity(type = ActivityType.Walk, distance = 11_000.0, totalElevationGain = 1_100.0, movingTime = 4_000),
        )

        // WHEN
        val results = badgesService.getGeneralBadges(activityTypes, 2026)

        // THEN
        assertTrue(results.any { it.badge is DistanceBadge && it.isCompleted })
        assertEquals(
            setOf("HikeDistanceBadge", "HikeElevationBadge", "HikeMovingTimeBadge"),
            results.map { it.toDto(activityTypes).badge.type }.toSet(),
        )
    }

    @Test
    fun `getGeneralBadges returns no badges for unsupported activity family`() {
        // GIVEN
        val activityTypes = setOf(ActivityType.AlpineSki)
        every {
            activityProvider.getActivitiesByActivityTypeAndYear(activityTypes, 2026)
        } returns listOf(
            activity(type = ActivityType.AlpineSki, distance = 20_000.0, totalElevationGain = 1_000.0, movingTime = 3_600),
        )

        // WHEN
        val results = badgesService.getGeneralBadges(activityTypes, 2026)

        // THEN
        assertTrue(results.isEmpty())
    }

    private fun activity(
        type: ActivityType,
        distance: Double,
        totalElevationGain: Double,
        movingTime: Int,
    ): StravaActivity {
        return StravaActivity(
            athlete = AthleteRef(1),
            averageSpeed = 5.0,
            commute = false,
            distance = distance,
            elapsedTime = movingTime,
            id = type.ordinal.toLong(),
            maxSpeed = 8.0f,
            movingTime = movingTime,
            name = "${type.name} test activity",
            startDate = "2026-04-24T07:00:00Z",
            startDateLocal = "2026-04-24T09:00:00Z",
            startLatlng = null,
            totalElevationGain = totalElevationGain,
            type = type.name,
            uploadId = 1,
        )
    }
}
