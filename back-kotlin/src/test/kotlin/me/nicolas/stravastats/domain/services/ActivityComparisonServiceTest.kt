package me.nicolas.stravastats.domain.services

import io.mockk.every
import io.mockk.mockk
import me.nicolas.stravastats.TestHelper
import me.nicolas.stravastats.domain.business.ActivityType
import me.nicolas.stravastats.domain.business.strava.Achievement
import me.nicolas.stravastats.domain.business.strava.MetaActivity
import me.nicolas.stravastats.domain.business.strava.MetaAthlete
import me.nicolas.stravastats.domain.business.strava.Segment
import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.business.strava.StravaDetailedActivity
import me.nicolas.stravastats.domain.business.strava.StravaSegmentEffort
import me.nicolas.stravastats.domain.services.activityproviders.IActivityProvider
import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Assertions.assertNotNull
import org.junit.jupiter.api.Test

class ActivityComparisonServiceTest {

    @Test
    fun `activity comparison compares similar activities and cached common segments`() {
        // GIVEN
        val target = comparisonDetailedActivity(100, "Target", "2025-05-10T09:00:00Z", 50_000, 500, 6.9)
            .copy(segmentEfforts = listOf(comparisonSegmentEffort(10, "Shared climb"), comparisonSegmentEffort(20, "Only target")))
        val activityProvider = mockk<IActivityProvider>(relaxed = true)
        every { activityProvider.cacheIdentity() } returns null
        every { activityProvider.getActivitiesByActivityTypeAndYear(setOf(ActivityType.Ride), 2025) } returns listOf(
            comparisonActivity(1, "Close A", "2025-05-01T09:00:00Z", 51_000.0, 520.0, 6.3),
            comparisonActivity(2, "Close B", "2025-06-01T09:00:00Z", 48_000.0, 460.0, 6.2),
            comparisonActivity(3, "Too far", "2025-07-01T09:00:00Z", 110_000.0, 1_800.0, 5.0),
            comparisonActivity(100, "Target", "2025-05-10T09:00:00Z", 50_000.0, 500.0, 6.9),
        )
        every { activityProvider.getCachedDetailedActivity(1) } returns comparisonDetailedActivity(1, "Close A", "2025-05-01T09:00:00Z", 51_000, 520, 6.3)
            .copy(segmentEfforts = listOf(comparisonSegmentEffort(10, "Shared climb")))
        every { activityProvider.getCachedDetailedActivity(2) } returns comparisonDetailedActivity(2, "Close B", "2025-06-01T09:00:00Z", 48_000, 460, 6.2)
            .copy(segmentEfforts = listOf(comparisonSegmentEffort(30, "Other segment")))
        val activityService = ActivityService(activityProvider)

        // WHEN
        val comparison = activityService.getActivityComparison(target)

        // THEN
        assertNotNull(comparison)
        assertEquals(2, comparison?.criteria?.sampleSize)
        assertEquals("faster", comparison?.status)
        assertEquals(1, comparison?.similarActivities?.first()?.id)
        assertEquals(1, comparison?.commonSegments?.size)
        assertEquals(10, comparison?.commonSegments?.first()?.id)
        assertEquals(1, comparison?.commonSegments?.first()?.matchCount)
    }

    @Test
    fun `activity comparison keeps flat activities with small elevation delta`() {
        // GIVEN
        val target = comparisonDetailedActivity(100, "Flat target", "2025-05-10T09:00:00Z", 40_000, 0, 6.8)
        val activityProvider = mockk<IActivityProvider>(relaxed = true)
        every { activityProvider.cacheIdentity() } returns null
        every { activityProvider.getActivitiesByActivityTypeAndYear(setOf(ActivityType.Ride), 2025) } returns listOf(
            comparisonActivity(4, "Flat close", "2025-05-03T09:00:00Z", 40_800.0, 35.0, 6.7),
        )
        val activityService = ActivityService(activityProvider)

        // WHEN
        val comparison = activityService.getActivityComparison(target)

        // THEN
        assertNotNull(comparison)
        assertEquals(1, comparison?.criteria?.sampleSize)
        assertEquals(4, comparison?.similarActivities?.first()?.id)
    }

    private fun comparisonDetailedActivity(
        id: Long,
        name: String,
        date: String,
        distance: Int,
        elevation: Int,
        speed: Double,
    ): StravaDetailedActivity =
        TestHelper.stravaActivity.toStravaDetailedActivity().copy(
            id = id,
            name = name,
            type = "Ride",
            sportType = "Ride",
            startDate = date,
            startDateLocal = date,
            distance = distance,
            totalElevationGain = elevation,
            movingTime = (distance / speed).toInt(),
            averageSpeed = speed,
            averageHeartrate = 140.0,
            averageWatts = 210.0,
            averageCadence = 82.0,
        )

    private fun comparisonActivity(
        id: Long,
        name: String,
        date: String,
        distance: Double,
        elevation: Double,
        speed: Double,
    ): StravaActivity =
        TestHelper.stravaActivity.copy(
            id = id,
            name = name,
            type = "Ride",
            startDate = date,
            startDateLocal = date,
            distance = distance,
            totalElevationGain = elevation,
            movingTime = (distance / speed).toInt(),
            averageSpeed = speed,
            averageHeartrate = 138.0,
            averageWatts = 190,
            averageCadence = 80.0,
        )

    private fun comparisonSegmentEffort(segmentId: Long, name: String): StravaSegmentEffort =
        StravaSegmentEffort(
            achievements = emptyList<Achievement>(),
            activity = MetaActivity(1),
            athlete = MetaAthlete(1),
            averageCadence = 0.0,
            averageHeartRate = 0.0,
            averageWatts = 0.0,
            deviceWatts = false,
            distance = 0.0,
            elapsedTime = 0,
            endIndex = 0,
            hidden = false,
            id = segmentId,
            komRank = null,
            maxHeartRate = 0.0,
            movingTime = 0,
            name = name,
            prRank = null,
            resourceState = 2,
            segment = Segment(
                activityType = "Ride",
                averageGrade = 0.0,
                city = null,
                climbCategory = 0,
                country = null,
                distance = 0.0,
                elevationHigh = 0.0,
                elevationLow = 0.0,
                endLatLng = emptyList(),
                hazardous = false,
                id = segmentId,
                maximumGrade = 0.0,
                name = name,
                isPrivate = false,
                resourceState = 2,
                starred = false,
                startLatLng = emptyList(),
                state = null,
            ),
            startDate = "",
            startDateLocal = "",
            startIndex = 0,
            visibility = null,
        )
}
