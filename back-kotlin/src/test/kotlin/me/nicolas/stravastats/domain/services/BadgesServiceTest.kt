package me.nicolas.stravastats.domain.services

import io.mockk.every
import io.mockk.mockk
import me.nicolas.stravastats.api.dto.toDto
import me.nicolas.stravastats.domain.business.ActivityType
import me.nicolas.stravastats.domain.business.badges.DistanceBadge
import me.nicolas.stravastats.domain.business.badges.ElevationBadge
import me.nicolas.stravastats.domain.business.badges.HikingBadge
import me.nicolas.stravastats.domain.business.badges.FamousClimbBadge
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
    private val officialClimbSourceDomains = setOf(
        "mycols.app",
        "cols-cyclisme.com",
        "bigcycling.eu",
        "climbfinder.com",
        "cyclinglocations.com",
    )

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
        assertTrue(results.any { it.badge is HikingBadge && it.isCompleted })
        assertEquals(
            setOf("HikeDistanceBadge", "HikeElevationBadge", "HikeHikingBadge", "HikeMovingTimeBadge"),
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

    @Test
    fun `getFamousBadges loads the five national catalogs with geography`() {
        val activityTypes = setOf(ActivityType.Ride)
        every {
            activityProvider.getActivitiesByActivityTypeAndYear(activityTypes, null)
        } returns emptyList()

        val results = badgesService.getFamousBadges(activityTypes, null)
        val climbs = results.map { it.badge as FamousClimbBadge }

        assertEquals(766, climbs.size)
        assertEquals(
            mapOf("FR" to 508, "CH" to 47, "IT" to 78, "ES" to 127, "AD" to 6),
            climbs.groupingBy { it.country }.eachCount(),
        )
        assertTrue(climbs.all { it.massif.isNotBlank() })
		assertTrue(climbs.all { it.summitId.isNotBlank() && it.variantId.startsWith("${it.summitId}--") })
		assertEquals(climbs.size, climbs.map { it.variantId }.distinct().size)
        assertEquals(climbs.size, climbs.map { it.label }.distinct().size)
        climbs.forEach { climb ->
            assertTrue(climb.length > 0, "Invalid length for ${climb.label}")
            assertTrue(climb.totalAscent > 0, "Invalid ascent for ${climb.label}")
            assertTrue(climb.averageGradient > 0, "Invalid average gradient for ${climb.label}")
            assertTrue(climb.difficulty >= 0, "Invalid difficulty for ${climb.label}")
            val estimatedAscent = climb.length * climb.averageGradient * 10
            assertTrue(
                estimatedAscent in (climb.totalAscent * 0.75)..(climb.totalAscent * 1.25),
                "Average gradient is inconsistent with length and ascent for ${climb.label}",
            )
            assertTrue(climb.minimumAltitude >= 0, "Invalid minimum altitude for ${climb.label}")
            assertTrue(climb.maximumGradient in 0.0..30.0, "Invalid maximum gradient for ${climb.label}")
            assertTrue(
                climb.maximumGradient == 0.0 || climb.maximumGradient + 0.1 >= climb.averageGradient,
                "Maximum gradient is below average gradient for ${climb.label}",
            )
            assertTrue(climb.category in setOf("HC", "1", "2", "3", "4"), "Invalid category for ${climb.label}")
            assertTrue(climb.summitToleranceMeters in 0..500, "Invalid summit tolerance for ${climb.label}")
            assertTrue(climb.start.latitude in -90.0..90.0 && climb.start.longitude in -180.0..180.0, "Invalid start for ${climb.label}")
            assertTrue(climb.end.latitude in -90.0..90.0 && climb.end.longitude in -180.0..180.0, "Invalid end for ${climb.label}")
            assertTrue(
                climb.start.haversineInKM(climb.end.latitude, climb.end.longitude) <= climb.length + 0.5,
                "Direct distance exceeds published length for ${climb.label}",
            )
            assertTrue(climb.sourceUrl.startsWith("https://"), "Invalid source for ${climb.label}")
            assertTrue(hasOfficialClimbSource(climb.sourceUrl), "Non-official source for ${climb.label}: ${climb.sourceUrl}")
            if (climb.country == "ES") {
                assertTrue(climb.start.latitude in 27.0..44.5 && climb.start.longitude in -19.0..5.0, "Implausible Spanish start for ${climb.label}")
                assertTrue(climb.end.latitude in 27.0..44.5 && climb.end.longitude in -19.0..5.0, "Implausible Spanish summit for ${climb.label}")
            }
        }
        assertEquals(38, climbs.count { it.massif == "Corse" })
        val alpeDHuez = climbs.single { it.label == "Alpe d'Huez from Le Bourg-d'Oisans" }
        assertEquals("HC", alpeDHuez.category)
        assertEquals(979, alpeDHuez.difficulty)
        assertEquals(
            1,
            climbs.single { it.label == "Col de la Madeleine from La Chambre, via Montgellafrey" }.routeCheckpoints.size,
        )
    }

    private fun hasOfficialClimbSource(sourceUrl: String): Boolean {
        val host = runCatching {
            java.net.URI(sourceUrl).host
                ?.removePrefix("www.")
                ?.lowercase()
        }.getOrNull() ?: return false

        return officialClimbSourceDomains.any { domain ->
            host == domain || host.endsWith(".$domain")
        }
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
