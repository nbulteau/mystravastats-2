package me.nicolas.stravastats.domain.business.strava.stream

import me.nicolas.stravastats.domain.business.SlopeType
import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Assertions.assertTrue
import org.junit.jupiter.api.Test

class StreamClimbDetectionTest {

    @Test
    fun `listSlopes keeps irregular climb together`() {
        val stream = irregularClimbStream()

        val slopes = stream.listSlopes()

        assertEquals(1, slopes.size)
        val climb = slopes[0]
        assertEquals(SlopeType.ASCENT, climb.type)
        assertTrue(climb.startIndex <= 1, "expected climb to start near the first ramp, got ${climb.startIndex}")
        assertTrue(climb.endIndex >= 17, "expected climb to include the resumed ramp, got ${climb.endIndex}")
        assertTrue(climb.distance >= 1600.0, "expected climb distance to cover the irregular ascent, got ${climb.distance}")
        assertTrue(climb.endAltitude - climb.startAltitude >= 80.0)
    }

    @Test
    fun `listSlopes accepts ratio grade smooth`() {
        val stream = Stream(
            distance = DistanceStream(
                data = listOf(0.0, 100.0, 200.0, 300.0, 400.0, 500.0, 600.0, 700.0, 800.0, 900.0, 1000.0, 1100.0, 1200.0),
                originalSize = 13,
                resolution = "high",
                seriesType = "distance",
            ),
            time = TimeStream(
                data = listOf(0, 40, 80, 120, 160, 200, 240, 280, 320, 360, 400, 440, 480),
                originalSize = 13,
                resolution = "high",
                seriesType = "time",
            ),
            altitude = AltitudeStream(
                data = listOf(100.0, 106.0, 112.0, 118.0, 124.0, 130.0, 136.0, 142.0, 148.0, 154.0, 160.0, 166.0, 172.0),
                originalSize = 13,
                resolution = "high",
                seriesType = "distance",
            ),
            gradeSmooth = SmoothGradeStream(
                data = listOf(0.0f, 0.06f, 0.06f, 0.06f, 0.06f, 0.06f, 0.06f, 0.06f, 0.06f, 0.06f, 0.06f, 0.06f, 0.06f),
                originalSize = 13,
                resolution = "high",
                seriesType = "distance",
            ),
        )

        val slopes = stream.listSlopes()

        assertEquals(1, slopes.size)
        assertTrue(slopes[0].maxGrade >= 5.0, "expected grade_smooth ratio to be normalized to percent")
    }

    @Test
    fun `listSlopes falls back to altitude when grade smooth has no signal`() {
        val stream = Stream(
            distance = DistanceStream(
                data = listOf(0.0, 100.0, 200.0, 300.0, 400.0, 500.0, 600.0, 700.0, 800.0, 900.0, 1000.0, 1100.0, 1200.0),
                originalSize = 13,
                resolution = "high",
                seriesType = "distance",
            ),
            time = TimeStream(
                data = listOf(0, 40, 80, 120, 160, 200, 240, 280, 320, 360, 400, 440, 480),
                originalSize = 13,
                resolution = "high",
                seriesType = "time",
            ),
            altitude = AltitudeStream(
                data = listOf(100.0, 106.0, 112.0, 118.0, 124.0, 130.0, 136.0, 142.0, 148.0, 154.0, 160.0, 166.0, 172.0),
                originalSize = 13,
                resolution = "high",
                seriesType = "distance",
            ),
            gradeSmooth = SmoothGradeStream(
                data = listOf(0.0f, 0.0f, 0.0f, 0.0f, 0.0f, 0.0f, 0.0f, 0.0f, 0.0f, 0.0f, 0.0f, 0.0f, 0.0f),
                originalSize = 13,
                resolution = "high",
                seriesType = "distance",
            ),
        )

        val slopes = stream.listSlopes()

        assertEquals(1, slopes.size)
    }

    private fun irregularClimbStream(): Stream {
        val distances = listOf(
            0.0, 100.0, 200.0, 300.0, 400.0, 500.0, 600.0, 700.0, 800.0, 900.0,
            1000.0, 1100.0, 1200.0, 1300.0, 1400.0, 1500.0, 1600.0, 1700.0, 1800.0,
        )
        val times = listOf(
            0, 40, 80, 120, 160, 200, 240, 280, 320, 360,
            400, 440, 480, 520, 560, 600, 640, 680, 720,
        )
        val altitudes = listOf(
            100.0, 106.0, 112.0, 118.0, 124.0, 126.0, 125.0, 127.0, 133.0, 139.0,
            145.0, 151.0, 157.0, 163.0, 169.0, 175.0, 181.0, 187.0, 193.0,
        )
        return Stream(
            distance = DistanceStream(distances, distances.size, "high", "distance"),
            time = TimeStream(times, times.size, "high", "time"),
            altitude = AltitudeStream(altitudes, altitudes.size, "high", "distance"),
        )
    }
}
