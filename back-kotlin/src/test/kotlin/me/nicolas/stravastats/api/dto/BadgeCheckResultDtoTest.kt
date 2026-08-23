package me.nicolas.stravastats.api.dto

import me.nicolas.stravastats.TestHelper
import me.nicolas.stravastats.domain.business.ActivityType
import me.nicolas.stravastats.domain.business.badges.BadgeCheckResult
import me.nicolas.stravastats.domain.business.badges.FamousClimbBadge
import me.nicolas.stravastats.domain.business.strava.GeoCoordinate
import me.nicolas.stravastats.domain.business.strava.stream.AltitudeStream
import me.nicolas.stravastats.domain.business.strava.stream.DistanceStream
import me.nicolas.stravastats.domain.business.strava.stream.LatLngStream
import me.nicolas.stravastats.domain.business.strava.stream.Stream
import me.nicolas.stravastats.domain.business.strava.stream.TimeStream
import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Assertions.assertNotNull
import org.junit.jupiter.api.Test

class BadgeCheckResultDtoTest {
    @Test
    fun `famous climb result exposes print-ready poster details`() {
        val badge = FamousClimbBadge(
            label = "Test col from valley",
            name = "Test col",
            country = "FR",
            massif = "Alpes",
            sourceUrl = "https://example.test/test-col",
            topOfTheAscent = 1850,
            start = GeoCoordinate(latitude = 45.1000, longitude = 6.1000),
            end = GeoCoordinate(latitude = 45.2000, longitude = 6.2000),
            length = 12.4,
            totalAscent = 980,
            minimumAltitude = 870,
            maximumGradient = 12.5,
            averageGradient = 7.9,
            difficulty = 800,
            category = "1",
        )
        val distance = listOf(0.0, 1000.0, 7000.0, 13400.0)
        val time = listOf(0, 100, 650, 1200)
        val coordinates = listOf(
            listOf(45.0000, 6.0000),
            listOf(45.1000, 6.1000),
            listOf(45.1500, 6.1500),
            listOf(45.2000, 6.2000),
        )
        val stream = Stream(
            distance = DistanceStream(distance, distance.size, "high", "distance"),
            time = TimeStream(time, time.size, "high", "time"),
            latlng = LatLngStream(coordinates, coordinates.size, "high", "distance"),
            altitude = AltitudeStream(listOf(90.0, 100.0, 120.0, 135.0)),
        )
        val activity = TestHelper.stravaActivity.copy(
            id = 42,
            type = "Ride",
            movingTime = 3600,
            startDateLocal = "2026-07-14T08:00:00Z",
            stream = stream,
        )
        val slowerActivity = activity.copy(
            id = 43,
            startDateLocal = "2025-06-12T08:00:00Z",
            stream = stream.copy(time = TimeStream(listOf(0, 100, 750, 1400), 4, "high", "time")),
        )

        val dto = BadgeCheckResult(
            badge = badge,
            activities = listOf(slowerActivity, activity),
            isCompleted = true,
        ).toDto(setOf(ActivityType.Ride))

        assertNotNull(dto.climbDetails)
        val details = dto.climbDetails!!
        assertEquals("Test col", details.name)
        assertEquals("FR", details.country)
        assertEquals("Alpes", details.massif)
        assertEquals("https://example.test/test-col", details.sourceUrl)
        assertEquals(ClimbCoordinateDto(45.2, 6.2), details.summitCoordinate)
        assertEquals(ClimbCoordinateDto(45.1, 6.1), details.startCoordinate)
        assertEquals(1850, details.summitAltitude)
        assertEquals(870, details.minimumAltitude)
        assertEquals(12.4, details.lengthKm)
        assertEquals(800, details.difficulty)
        assertEquals(2, details.ascentCount)
        assertEquals(1100, details.bestAscent?.durationSeconds)
        assertEquals(42, details.bestAscent?.activityId)
        assertEquals("2026-07-14T08:00:00Z", details.bestAscent?.date)
        assertEquals(3, details.profile.size)
        assertEquals(0.0, details.profile.first().distanceKm)
        assertEquals(12.4, details.profile.last().distanceKm)
        assertEquals(12.5, details.maximumGradient)
    }

    @Test
    fun `poster profile selects repeated waypoint occurrence closest to catalogue length`() {
        val badge = FamousClimbBadge(
            label = "Test col from valley",
            name = "Test col",
            topOfTheAscent = 1500,
            start = GeoCoordinate(latitude = 45.1, longitude = 6.1),
            end = GeoCoordinate(latitude = 45.2, longitude = 6.2),
            length = 10.0,
            totalAscent = 700,
            averageGradient = 7.0,
            difficulty = 500,
            category = "1",
        )
        val distances = listOf(0.0, 42000.0, 50000.0, 55000.0, 60000.0)
        val coordinates = listOf(
            listOf(45.1, 6.1),
            listOf(45.0, 6.0),
            listOf(45.1, 6.1),
            listOf(45.15, 6.15),
            listOf(45.2, 6.2),
        )
        val activity = TestHelper.stravaActivity.copy(
            id = 44,
            type = "Ride",
            stream = Stream(
                distance = DistanceStream(distances, distances.size, "high", "distance"),
                time = TimeStream(listOf(0, 2000, 3000, 3400, 4000), 5, "high", "time"),
                latlng = LatLngStream(coordinates, coordinates.size, "high", "distance"),
                altitude = AltitudeStream(listOf(800.0, 900.0, 800.0, 1100.0, 1500.0)),
            ),
        )

        val details = BadgeCheckResult(
            badge = badge,
            activities = listOf(activity),
            isCompleted = true,
        ).toDto(setOf(ActivityType.Ride)).climbDetails!!

        assertEquals(10.0, details.profile.last().distanceKm)
        assertEquals(1000, details.bestAscent?.durationSeconds)
    }

    @Test
    fun `poster profile rejects implausibly long route between climb waypoints`() {
        val badge = FamousClimbBadge(
            label = "Test col from valley",
            name = "Test col",
            topOfTheAscent = 1500,
            start = GeoCoordinate(latitude = 45.1, longitude = 6.1),
            end = GeoCoordinate(latitude = 45.2, longitude = 6.2),
            length = 10.0,
            totalAscent = 700,
            averageGradient = 7.0,
            difficulty = 500,
            category = "1",
        )
        val coordinates = listOf(listOf(45.1, 6.1), listOf(45.2, 6.2))
        val activity = TestHelper.stravaActivity.copy(
            id = 45,
            type = "Ride",
            stream = Stream(
                distance = DistanceStream(listOf(0.0, 52000.0), 2, "high", "distance"),
                time = TimeStream(listOf(0, 5000), 2, "high", "time"),
                latlng = LatLngStream(coordinates, coordinates.size, "high", "distance"),
                altitude = AltitudeStream(listOf(800.0, 1500.0)),
            ),
        )

        val details = BadgeCheckResult(
            badge = badge,
            activities = listOf(activity),
            isCompleted = true,
        ).toDto(setOf(ActivityType.Ride)).climbDetails!!

        assertEquals(0, details.ascentCount)
        assertEquals(0, details.profile.size)
    }

    @Test
    fun `published maximum gradient may exceed computed GPS ceiling`() {
        val badge = FamousClimbBadge(
            label = "Steep col from valley",
            name = "Steep col",
            topOfTheAscent = 1500,
            start = GeoCoordinate(latitude = 45.1, longitude = 6.1),
            end = GeoCoordinate(latitude = 45.2, longitude = 6.2),
            length = 10.0,
            totalAscent = 1000,
            maximumGradient = 21.0,
            averageGradient = 10.0,
            difficulty = 1000,
            category = "HC",
        )

        val details = BadgeCheckResult(
            badge = badge,
            activities = emptyList(),
            isCompleted = false,
        ).toDto(setOf(ActivityType.Ride)).climbDetails!!

        assertEquals(21.0, details.maximumGradient)
    }

    @Test
    fun `computed maximum gradient ignores a short altitude spike`() {
        val badge = FamousClimbBadge(
            label = "Noisy col from valley",
            name = "Noisy col",
            topOfTheAscent = 140,
            start = GeoCoordinate(latitude = 45.1000, longitude = 6.1000),
            end = GeoCoordinate(latitude = 45.2000, longitude = 6.2000),
            length = 0.5,
            totalAscent = 40,
            averageGradient = 8.0,
            difficulty = 100,
            category = "4",
        )
        val distance = listOf(0.0, 100.0, 500.0)
        val coordinates = listOf(
            listOf(45.1000, 6.1000),
            listOf(45.1500, 6.1500),
            listOf(45.2000, 6.2000),
        )
        val activity = TestHelper.stravaActivity.copy(
            id = 43,
            type = "Ride",
            stream = Stream(
                distance = DistanceStream(distance, distance.size, "high", "distance"),
                time = TimeStream(listOf(0, 60, 300), 3, "high", "time"),
                latlng = LatLngStream(coordinates, coordinates.size, "high", "distance"),
                altitude = AltitudeStream(listOf(100.0, 160.0, 140.0)),
            ),
        )

        val details = BadgeCheckResult(
            badge = badge,
            activities = listOf(activity),
            isCompleted = true,
        ).toDto(setOf(ActivityType.Ride)).climbDetails!!

        assertEquals(8.0, details.maximumGradient)
    }
}
