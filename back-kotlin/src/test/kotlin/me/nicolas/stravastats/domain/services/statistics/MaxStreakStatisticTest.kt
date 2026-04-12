package me.nicolas.stravastats.domain.services.statistics

import me.nicolas.stravastats.TestHelper
import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Test

class MaxStreakStatisticTest {

    @Test
    fun `should compute max streak with unsorted activities`() {
        // GIVEN
        val activities = listOf(
            TestHelper.stravaActivity.copy(id = 1, startDateLocal = "2024-01-03T10:00:00+01:00"),
            TestHelper.stravaActivity.copy(id = 2, startDateLocal = "2024-01-01T10:00:00+01:00"),
            TestHelper.stravaActivity.copy(id = 3, startDateLocal = "2024-01-02T10:00:00+01:00"),
        )

        // WHEN
        val value = MaxStreakStatistic(activities).value

        // THEN
        assertEquals("3", value)
    }

    @Test
    fun `should ignore duplicate day attempts and malformed dates`() {
        // GIVEN
        val activities = listOf(
            TestHelper.stravaActivity.copy(id = 10, startDateLocal = "2024-02-01T10:00:00+01:00"),
            TestHelper.stravaActivity.copy(id = 11, startDateLocal = "2024-02-01T18:00:00+01:00"),
            TestHelper.stravaActivity.copy(id = 12, startDateLocal = "2024-02-02T10:00:00+01:00"),
            TestHelper.stravaActivity.copy(id = 13, startDateLocal = ""),
        )

        // WHEN
        val value = MaxStreakStatistic(activities).value

        // THEN
        assertEquals("2", value)
    }
}
