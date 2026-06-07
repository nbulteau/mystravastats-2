package me.nicolas.stravastats.domain.services.statistics

import org.junit.jupiter.api.Assertions.assertNull
import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Assertions.assertNotNull
import org.junit.jupiter.api.BeforeEach
import org.junit.jupiter.api.Test

class BestEffortPowerStatisticTest {

    @BeforeEach
    fun clearCache() {
        BestEffortCache.clear()
    }

    @Test
    fun `calculateBestPowerForTime returns null when watts stream is truncated`() {
        // GIVEN
        val activity = StatisticsFixtures.syntheticRideActivity(
            id = 31,
            stream = StatisticsFixtures.defaultStream(
                distances = listOf(0.0, 100.0, 200.0, 300.0),
                times = listOf(0, 10, 20, 30),
                altitudes = listOf(100.0, 102.0, 104.0, 106.0),
                watts = listOf(180),
            )
        )

        // WHEN
        val effort = activity.calculateBestPowerForTime(seconds = 20)

        // THEN
        assertNull(effort)
    }

    @Test
    fun `calculateBestPowerForDistance selects highest average power window`() {
        // GIVEN
        val activity = StatisticsFixtures.syntheticRideActivity(
            id = 32,
            stream = StatisticsFixtures.defaultStream(
                distances = listOf(0.0, 500.0, 1000.0, 1500.0),
                times = listOf(0, 30, 60, 90),
                altitudes = listOf(100.0, 105.0, 110.0, 120.0),
                watts = listOf(100, 150, 200, 400),
            )
        )

        // WHEN
        val effort = activity.calculateBestPowerForDistance(distance = 1000.0)

        // THEN
        assertNotNull(effort)
        assertEquals(250, effort!!.averagePower)
        assertEquals("Best Power for 1000 m", effort.label)
        assertEquals(1, effort.idxStart)
        assertEquals(3, effort.idxEnd)
    }
}
