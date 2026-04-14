package me.nicolas.stravastats.domain.services.statistics

import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Assertions.assertNotNull
import org.junit.jupiter.api.Assertions.assertNull
import org.junit.jupiter.api.BeforeEach
import org.junit.jupiter.api.Test
import org.junit.jupiter.api.assertThrows

class BestEffortDistanceStatisticTest {

    @BeforeEach
    fun clearCache() {
        BestEffortCache.clear()
    }

    @Test
    fun `calculateBestTimeForDistance returns expected best effort on synthetic stream`() {
        // GIVEN
        val activity = StatisticsFixtures.syntheticRideActivity(id = 1)

        // WHEN
        val effort = activity.calculateBestTimeForDistance(distance = 200.0)

        // THEN
        assertNotNull(effort)
        val actualEffort = effort!!
        assertEquals(200.0, actualEffort.distance, 1e-6)
        assertEquals(20, actualEffort.seconds)
        assertEquals(15.0, actualEffort.deltaAltitude, 1e-6)
    }

    @Test
    fun `calculateBestTimeForDistance returns null when stream is missing`() {
        // GIVEN
        val activity = StatisticsFixtures.syntheticRideActivity(id = 2, stream = null)

        // WHEN
        val effort = activity.calculateBestTimeForDistance(distance = 200.0)

        // THEN
        assertNull(effort)
    }

    @Test
    fun `calculateBestTimeForDistance returns null when target is longer than stream`() {
        // GIVEN
        val activity = StatisticsFixtures.syntheticRideActivity(id = 3)

        // WHEN
        val effort = activity.calculateBestTimeForDistance(distance = 2_000.0)

        // THEN
        assertNull(effort)
    }

    @Test
    fun `statistic returns Not available when no effort exists`() {
        // GIVEN
        val activity = StatisticsFixtures.syntheticRideActivity(
            id = 4,
            stream = StatisticsFixtures.defaultStream(
                distances = emptyList(),
                times = emptyList(),
                altitudes = emptyList(),
            )
        )

        // WHEN
        val statistic = BestEffortDistanceStatistic(
            name = "Best 1 km",
            activities = listOf(activity),
            distance = 1_000.0
        )

        // THEN
        assertEquals("Not available", statistic.value)
    }

    @Test
    fun `statistic requires distance of at least 100 meters`() {
        // GIVEN
        val activity = StatisticsFixtures.syntheticRideActivity(id = 5)

        // WHEN / THEN
        assertThrows<IllegalArgumentException> {
            BestEffortDistanceStatistic(
                name = "Invalid",
                activities = listOf(activity),
                distance = 99.0
            )
        }
    }
}
