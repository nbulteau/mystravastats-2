package me.nicolas.stravastats.adapters.localrepositories.fit

import me.nicolas.stravastats.domain.business.strava.stream.DistanceStream
import me.nicolas.stravastats.domain.business.strava.stream.PowerStream
import me.nicolas.stravastats.domain.business.strava.stream.Stream
import me.nicolas.stravastats.domain.business.strava.stream.TimeStream
import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Assertions.assertFalse
import org.junit.jupiter.api.Assertions.assertTrue
import org.junit.jupiter.api.Test

class FITPowerMetricsTest {
    @Test
    fun `computeFitPowerMetrics uses power stream when session power is missing`() {
        // GIVEN
        val stream = powerStreamOf(0, 100, 200, 300)

        // WHEN
        val metrics = computeFitPowerMetrics(null, stream, 100)

        // THEN
        assertEquals(150, metrics.averageWatts)
        assertEquals(150, metrics.weightedAverageWatts)
        assertEquals(12.906, metrics.kilojoules, 0.0001)
        assertTrue(metrics.hasDeviceWatts)
    }

    @Test
    fun `computeFitPowerMetrics keeps session average power when present`() {
        // GIVEN
        val stream = powerStreamOf(0, 100, 200)

        // WHEN
        val metrics = computeFitPowerMetrics(250, stream, 120)

        // THEN
        assertEquals(250, metrics.averageWatts)
        assertEquals(250, metrics.weightedAverageWatts)
        assertEquals(25.812, metrics.kilojoules, 0.0001)
        assertTrue(metrics.hasDeviceWatts)
    }

    @Test
    fun `computeFitPowerMetrics ignores empty power stream`() {
        // GIVEN
        val stream = powerStreamOf(0, 0, null, -20)

        // WHEN
        val metrics = computeFitPowerMetrics(null, stream, 100)

        // THEN
        assertEquals(0, metrics.averageWatts)
        assertEquals(0, metrics.weightedAverageWatts)
        assertEquals(0.0, metrics.kilojoules, 0.0001)
        assertFalse(metrics.hasDeviceWatts)
    }

    private fun powerStreamOf(vararg watts: Int?): Stream {
        val size = watts.size
        return Stream(
            distance = DistanceStream(
                data = List(size) { it.toDouble() },
                originalSize = size,
                resolution = "high",
                seriesType = "distance",
            ),
            time = TimeStream(
                data = List(size) { it },
                originalSize = size,
                resolution = "high",
                seriesType = "distance",
            ),
            watts = PowerStream(
                data = watts.toList(),
                originalSize = size,
                resolution = "high",
                seriesType = "distance",
            ),
        )
    }
}
