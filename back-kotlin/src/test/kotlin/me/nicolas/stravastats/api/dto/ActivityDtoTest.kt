package me.nicolas.stravastats.api.dto

import me.nicolas.stravastats.TestHelper
import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Assertions.assertTrue
import org.junit.jupiter.api.Test

class ActivityDtoTest {

    @Test
    fun `toDto maps commute and heart rate fields used by Activities filters`() {
        // GIVEN
        val activity = TestHelper.stravaActivity.copy(
            commute = true,
            averageHeartrate = 154.8,
            averageWatts = 213,
            movingTime = 3210
        )

        // WHEN
        val dto = activity.toDto()

        // THEN
        assertTrue(dto.commute)
        assertEquals(154, dto.averageHeartrate)
        assertEquals(213, dto.averageWatts)
        assertEquals(3210, dto.movingTime)
    }
}

