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

    @Test
    fun `statistic selects the fastest effort across multiple activities`() {
        // GIVEN
        val slowRide = StatisticsFixtures.syntheticRideActivity(
            id = 6,
            stream = StatisticsFixtures.defaultStream(
                distances = listOf(0.0, 100.0, 200.0, 300.0),
                times = listOf(0, 20, 40, 60),
                altitudes = listOf(100.0, 102.0, 104.0, 106.0),
            )
        )
        val fastRide = StatisticsFixtures.syntheticRideActivity(
            id = 7,
            stream = StatisticsFixtures.defaultStream(
                distances = listOf(0.0, 100.0, 200.0, 300.0),
                times = listOf(0, 10, 20, 30),
                altitudes = listOf(100.0, 103.0, 106.0, 109.0),
            )
        )

        // WHEN
        val statistic = BestEffortDistanceStatistic(
            name = "Best 200 m",
            activities = listOf(slowRide, fastRide),
            distance = 200.0
        )

        // THEN
        assertEquals("20s => 36.00 km/h", statistic.value)
        assertEquals(7L, statistic.activity?.id)
    }
}
