package me.nicolas.stravastats.domain.business.strava

import me.nicolas.stravastats.domain.business.ActivityEffort
import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Test

internal class StravaActivityEffortTest {

    @Test
    fun getSpeed() {

        // Given
        val colAgnelActivity = loadColAgnelActivity()

        // When
        val colAgnelActivityEffort = ActivityEffort(
            colAgnelActivity,
            colAgnelActivity.distance,
            colAgnelActivity.elapsedTime,
            colAgnelActivity.totalElevationGain,
            0,
            10,
            null,
            "Desctiption"
        )

        // Then
        assertEquals("15.48 km/h", colAgnelActivityEffort.getFormattedSpeed())
        assertEquals("15.48", colAgnelActivityEffort.getSpeed())

        assertEquals("2.33 %", colAgnelActivityEffort.getFormattedGradient())
        assertEquals("2.33", colAgnelActivityEffort.getGradient())
    }
}