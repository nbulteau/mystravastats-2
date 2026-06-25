package me.nicolas.stravastats.domain.business.badges

import me.nicolas.stravastats.domain.business.strava.AthleteRef
import me.nicolas.stravastats.domain.business.strava.StravaActivity
import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Assertions.assertTrue
import org.junit.jupiter.api.Test

class HikingBadgeTest {
    @Test
    fun `hiking adventure badge set checks outdoor specific badges`() {
        // GIVEN
        val activities = listOf(
            activity(
                id = 1,
                name = "Summit loop",
                startDateLocal = "2026-06-20T09:00:00Z",
                distance = 21_000.0,
                elevationGain = 650.0,
                elevHigh = 2350.0,
                startLatlng = listOf(45.1001, 6.1001),
            ),
            activity(
                id = 2,
                name = "Sunday recovery",
                startDateLocal = "2026-06-21T09:00:00Z",
                distance = 8_000.0,
                elevationGain = 120.0,
                elevHigh = 1200.0,
                startLatlng = listOf(45.2001, 6.2001),
            ),
            activity(
                id = 3,
                name = "Highest trail",
                startDateLocal = "2026-07-04T09:00:00Z",
                distance = 9_000.0,
                elevationGain = 450.0,
                elevHigh = 2650.0,
                startLatlng = listOf(45.3001, 6.3001),
            ),
        )

        // WHEN
        val completed = HikingBadge.hikingAdventureBadgeSet.check(activities)
            .filter { it.isCompleted }
            .associate { it.badge.label to it.activities.size }

        // THEN
        assertTrue(completed.containsKey("Summit Day"))
        assertTrue(completed.containsKey("Back-to-back Hiking Weekend"))
        assertTrue(completed.containsKey("High Point PR"))
        assertTrue(completed.containsKey("New Trail"))
        assertEquals(1, completed["High Point PR"])
    }

    @Test
    fun `hiking distance and elevation labels are outdoor specific`() {
        // GIVEN
        val activities = listOf(
            activity(
                id = 1,
                name = "Long mountain hike",
                startDateLocal = "2026-06-20T09:00:00Z",
                distance = 16_000.0,
                elevationGain = 1_100.0,
                elevHigh = 1_900.0,
                startLatlng = listOf(45.1001, 6.1001),
            )
        )

        // WHEN
        val distanceLabels = DistanceBadge.hikeBadgeSet.check(activities)
            .filter { it.isCompleted }
            .map { it.badge.label }
        val elevationLabels = ElevationBadge.hikeBadgeSet.check(activities)
            .filter { it.isCompleted }
            .map { it.badge.label }

        // THEN
        assertTrue(distanceLabels.contains("Long Hike 15 km"))
        assertTrue(elevationLabels.contains("Vertical Kilometer"))
    }

    private fun activity(
        id: Long,
        name: String,
        startDateLocal: String,
        distance: Double,
        elevationGain: Double,
        elevHigh: Double,
        startLatlng: List<Double>,
    ): StravaActivity =
        StravaActivity(
            athlete = AthleteRef(1),
            averageSpeed = 1.0,
            commute = false,
            distance = distance,
            elapsedTime = 7200,
            elevHigh = elevHigh,
            id = id,
            maxSpeed = 2.0f,
            movingTime = 7000,
            name = name,
            startDate = startDateLocal,
            startDateLocal = startDateLocal,
            startLatlng = startLatlng,
            totalElevationGain = elevationGain,
            type = "Hike",
            uploadId = id,
        )
}
