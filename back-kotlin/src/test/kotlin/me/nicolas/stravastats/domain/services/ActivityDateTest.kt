package me.nicolas.stravastats.domain.services

import me.nicolas.stravastats.domain.business.strava.AthleteRef
import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.services.ActivityHelper.activityYearOrNull
import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Assertions.assertNull
import org.junit.jupiter.api.Test

class ActivityDateTest {
    @Test
    fun `activity year falls back to UTC date`() {
        assertEquals(2025, activity(localDate = "", utcDate = "2025-04-03T08:00:00Z").activityYearOrNull())
    }

    @Test
    fun `activity year rejects truncated and invalid dates`() {
        assertNull(activity(localDate = "202", utcDate = "invalid").activityYearOrNull())
        assertNull(activity(localDate = "9999-01-01", utcDate = "").activityYearOrNull())
    }

    private fun activity(localDate: String, utcDate: String) = StravaActivity(
        athlete = AthleteRef(1),
        averageSpeed = 0.0,
        commute = false,
        distance = 0.0,
        elapsedTime = 0,
        id = 1,
        maxSpeed = 0f,
        movingTime = 0,
        name = "Invalid date fixture",
        startDate = utcDate,
        startDateLocal = localDate,
        startLatlng = null,
        totalElevationGain = 0.0,
        type = "Ride",
        uploadId = 1,
    )
}
