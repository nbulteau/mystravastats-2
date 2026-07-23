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
        // GIVEN
        val activity = StatisticsFixtures.syntheticRideActivity(id = 11)

        // WHEN
        val effort = activity.calculateBestDistanceForTime(seconds = 20)

        // THEN
        assertNotNull(effort)
        val actualEffort = effort!!
        assertEquals(200.0, actualEffort.distance, 1e-6)
        assertEquals(20, actualEffort.seconds)
        assertEquals(15.0, actualEffort.deltaAltitude, 1e-6)
        assertEquals(15.0, actualEffort.elevationGain ?: -1.0, 1e-6)
        assertEquals(0.0, actualEffort.elevationLoss ?: -1.0, 1e-6)
    }

    @Test
    fun `calculateBestDistanceForTime keeps cumulative elevation when net delta is zero`() {
        // GIVEN
        val activity = StatisticsFixtures.syntheticRideActivity(
            id = 16,
            stream = StatisticsFixtures.defaultStream(
                distances = listOf(0.0, 100.0, 200.0, 300.0, 400.0),
                times = listOf(0, 10, 20, 30, 40),
                altitudes = listOf(100.0, 120.0, 100.0, 125.0, 100.0),
            )
        )

        // WHEN
        val effort = activity.calculateBestDistanceForTime(seconds = 40)

        // THEN
        assertNotNull(effort)
        val actualEffort = effort!!
        assertEquals(0.0, actualEffort.deltaAltitude, 1e-6)
        assertEquals(45.0, actualEffort.elevationGain ?: -1.0, 1e-6)
        assertEquals(45.0, actualEffort.elevationLoss ?: -1.0, 1e-6)
    }

    @Test
    fun `calculateBestDistanceForTime returns null when altitude stream is missing`() {
        // GIVEN
        val activity = StatisticsFixtures.syntheticRideActivity(
            id = 12,
            stream = StatisticsFixtures.defaultStream(altitudes = null)
        )

        // WHEN
        val effort = activity.calculateBestDistanceForTime(seconds = 20)

        // THEN
        assertNull(effort)
    }

    @Test
    fun `calculateBestDistanceForTime returns null when target duration is longer than stream`() {
        // GIVEN
        val activity = StatisticsFixtures.syntheticRideActivity(id = 13)

        // WHEN
        val effort = activity.calculateBestDistanceForTime(seconds = 4_000)

        // THEN
        assertNull(effort)
    }

    @Test
    fun `calculateBestDistanceForTime returns null when altitude stream is truncated`() {
        // GIVEN
        val activity = StatisticsFixtures.syntheticRideActivity(
            id = 17,
            stream = StatisticsFixtures.defaultStream(
                distances = listOf(0.0, 100.0, 200.0, 300.0),
                times = listOf(0, 10, 20, 30),
                altitudes = listOf(100.0),
            )
        )

        // WHEN
        val effort = activity.calculateBestDistanceForTime(seconds = 20)

        // THEN
        assertNull(effort)
    }

    @Test
    fun `statistic returns Not available when no effort exists`() {
        // GIVEN
        val activity = StatisticsFixtures.syntheticRideActivity(
            id = 14,
            stream = StatisticsFixtures.defaultStream(
                distances = emptyList(),
                times = emptyList(),
                altitudes = emptyList(),
            )
        )

        // WHEN
        val statistic = BestEffortTimeStatistic(
            name = "Best 1 h",
            activities = listOf(activity),
            seconds = 3600
        )

        // THEN
        assertEquals("Not available", statistic.value)
    }

    @Test
    fun `statistic requires duration greater than 10 seconds`() {
        // GIVEN
        val activity = StatisticsFixtures.syntheticRideActivity(id = 15)

        // WHEN / THEN
        assertThrows<IllegalArgumentException> {
            BestEffortTimeStatistic(
                name = "Invalid",
                activities = listOf(activity),
                seconds = 10
            )
        }
    }

    @Test
    fun `calculateBestDistanceForTime ignores impossible speed segments`() {
        val activity = StatisticsFixtures.syntheticRideActivity(
            id = 18,
            stream = StatisticsFixtures.defaultStream(
                distances = listOf(0.0, 100.0, 200.0, 1_200.0, 1_300.0, 1_400.0, 1_500.0),
                times = listOf(0, 10, 20, 21, 31, 41, 51),
                altitudes = listOf(100.0, 101.0, 102.0, 103.0, 104.0, 105.0, 106.0),
            )
        )

        val effort = activity.calculateBestDistanceForTime(seconds = 20)

        assertNotNull(effort)
        assertEquals(200.0, effort?.distance ?: -1.0, 1e-6)
    }
}
