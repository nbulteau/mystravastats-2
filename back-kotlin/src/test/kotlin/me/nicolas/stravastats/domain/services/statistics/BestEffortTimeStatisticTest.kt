package me.nicolas.stravastats.domain.services.statistics

import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Assertions.assertNotNull
import org.junit.jupiter.api.Assertions.assertNull
import org.junit.jupiter.api.BeforeEach
import org.junit.jupiter.api.Test
import org.junit.jupiter.api.assertThrows

class BestEffortTimeStatisticTest {

    @BeforeEach
    fun clearCache() {
        BestEffortCache.clear()
    }

    @Test
    fun `calculateBestDistanceForTime returns expected best effort on synthetic stream`() {
        val activity = StatisticsFixtures.syntheticRideActivity(id = 11)

        val effort = activity.calculateBestDistanceForTime(seconds = 20)

        assertNotNull(effort)
        val actualEffort = effort!!
        assertEquals(200.0, actualEffort.distance, 1e-6)
        assertEquals(20, actualEffort.seconds)
        assertEquals(15.0, actualEffort.deltaAltitude, 1e-6)
    }

    @Test
    fun `calculateBestDistanceForTime returns null when altitude stream is missing`() {
        val activity = StatisticsFixtures.syntheticRideActivity(
            id = 12,
            stream = StatisticsFixtures.defaultStream(altitudes = null)
        )

        val effort = activity.calculateBestDistanceForTime(seconds = 20)

        assertNull(effort)
    }

    @Test
    fun `calculateBestDistanceForTime returns null when target duration is longer than stream`() {
        val activity = StatisticsFixtures.syntheticRideActivity(id = 13)

        val effort = activity.calculateBestDistanceForTime(seconds = 4_000)

        assertNull(effort)
    }

    @Test
    fun `statistic returns Not available when no effort exists`() {
        val activity = StatisticsFixtures.syntheticRideActivity(
            id = 14,
            stream = StatisticsFixtures.defaultStream(
                distances = emptyList(),
                times = emptyList(),
                altitudes = emptyList(),
            )
        )
        val statistic = BestEffortTimeStatistic(
            name = "Best 1 h",
            activities = listOf(activity),
            seconds = 3600
        )

        assertEquals("Not available", statistic.value)
    }

    @Test
    fun `statistic requires duration greater than 10 seconds`() {
        val activity = StatisticsFixtures.syntheticRideActivity(id = 15)

        assertThrows<IllegalArgumentException> {
            BestEffortTimeStatistic(
                name = "Invalid",
                activities = listOf(activity),
                seconds = 10
            )
        }
    }
}
