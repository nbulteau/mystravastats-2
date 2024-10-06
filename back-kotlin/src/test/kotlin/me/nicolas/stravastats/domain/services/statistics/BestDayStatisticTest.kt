package me.nicolas.stravastats.domain.services.statistics

import me.nicolas.stravastats.TestHelper
import me.nicolas.stravastats.domain.business.strava.Activity
import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Test

class BestDayStatisticTest {

    @Test
    fun `should return the best day`() {
        // GIVEN
        val activities = TestHelper.loadActivities()

        val bestDayStatistic = BestDayStatistic("Best day", activities, formatString = "%s: %.02f")
        { activityList -> activityList.maxByOrNull { it.distance }?.let { it.startDateLocal.substringBefore('T') to it.distance } }

        // WHEN
        val result = bestDayStatistic.value

        // THEN
        assertEquals("sam. 12 juin 2021: 100839,00", result)
    }

    @Test
    fun `should return not available when no activities`() {
        // Given
        val activities = emptyList<Activity>()
        val bestDayStatistic = BestDayStatistic("Best day", activities, formatString = "%s: %.02f")
        { activityList -> activityList.maxByOrNull { it.distance }?.let { it.startDateLocal.substringBefore('T') to it.distance } }

        // When
        val result = bestDayStatistic.value

        // Then
        assertEquals("Not available", result)
    }
}