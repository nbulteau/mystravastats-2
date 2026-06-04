package me.nicolas.stravastats.domain.services.statistics

import org.junit.jupiter.api.Assertions.assertNull
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
}
