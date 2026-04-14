package me.nicolas.stravastats.domain.services.statistics

import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Assertions.assertNotNull
import org.junit.jupiter.api.Assertions.assertNull
import org.junit.jupiter.api.BeforeEach
import org.junit.jupiter.api.Test
import org.junit.jupiter.api.assertThrows

class BestElevationDistanceStatisticTest {

    @BeforeEach
    fun clearCache() {
        BestEffortCache.clear()
    }

    @Test
    fun `calculateBestElevationForDistance returns expected best effort on synthetic stream`() {
        val activity = StatisticsFixtures.syntheticRideActivity(id = 21)

        val effort = activity.calculateBestElevationForDistance(distance = 200.0)

        assertNotNull(effort)
        val actualEffort = effort!!
        assertEquals(200.0, actualEffort.distance, 1e-6)
        assertEquals(20, actualEffort.seconds)
        assertEquals(15.0, actualEffort.deltaAltitude, 1e-6)
    }

    @Test
    fun `calculateBestElevationForDistance returns null when altitude stream is missing`() {
        val activity = StatisticsFixtures.syntheticRideActivity(
            id = 22,
            stream = StatisticsFixtures.defaultStream(altitudes = null)
        )

        val effort = activity.calculateBestElevationForDistance(distance = 200.0)

        assertNull(effort)
    }

    @Test
    fun `calculateBestElevationForDistance returns null when target distance is longer than stream`() {
        val activity = StatisticsFixtures.syntheticRideActivity(id = 23)

        val effort = activity.calculateBestElevationForDistance(distance = 2_000.0)

        assertNull(effort)
    }

    @Test
    fun `statistic requires distance strictly greater than 100 meters`() {
        val activity = StatisticsFixtures.syntheticRideActivity(id = 24)

        assertThrows<IllegalArgumentException> {
            BestElevationDistanceStatistic(
                name = "Invalid",
                activities = listOf(activity),
                distance = 100.0
            )
        }
    }
}
