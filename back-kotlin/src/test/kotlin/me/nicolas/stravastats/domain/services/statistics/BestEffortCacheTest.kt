package me.nicolas.stravastats.domain.services.statistics

import me.nicolas.stravastats.domain.business.ActivityEffort
import me.nicolas.stravastats.domain.business.ActivityShort
import me.nicolas.stravastats.domain.business.ActivityType
import me.nicolas.stravastats.domain.business.strava.stream.AltitudeStream
import me.nicolas.stravastats.domain.business.strava.stream.DistanceStream
import me.nicolas.stravastats.domain.business.strava.stream.Stream
import me.nicolas.stravastats.domain.business.strava.stream.TimeStream
import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Assertions.assertNotNull
import org.junit.jupiter.api.Assertions.assertNull
import org.junit.jupiter.api.Test
import org.junit.jupiter.api.io.TempDir
import java.nio.file.Path

class BestEffortCacheTest {

    @TempDir
    lateinit var tempDir: Path

    @Test
    fun `best effort cache is persisted and reloaded from disk`() {
        // GIVEN
        BestEffortCache.clear()
        val stream = testStream()
        var calls = 0
        val effort = ActivityEffort(
            distance = 1000.0,
            seconds = 180,
            deltaAltitude = 20.0,
            idxStart = 0,
            idxEnd = 2,
            averagePower = 250,
            label = "Best speed for 1000m",
            activityShort = ActivityShort(42, "Warmup effort", ActivityType.Ride.name),
        )

        // WHEN
        val first = BestEffortCache.getOrCompute(
            activityId = 42,
            metric = "best-time-distance",
            target = "1000.0",
            stream = stream,
        ) {
            calls += 1
            effort
        }

        // THEN
        assertEquals(1, calls)
        assertNotNull(first)

        // WHEN - persisted and reloaded
        val cacheFile = tempDir.resolve("best-effort-cache.json")
        val persisted = BestEffortCache.saveToDisk(cacheFile)
        assertEquals(1, persisted)

        BestEffortCache.clear()
        val loaded = BestEffortCache.loadFromDisk(cacheFile)
        assertEquals(persisted, loaded)

        calls = 0
        val second = BestEffortCache.getOrCompute(
            activityId = 42,
            metric = "best-time-distance",
            target = "1000.0",
            stream = stream,
        ) {
            calls += 1
            null
        }

        // THEN - loaded from disk, no recomputation
        assertEquals(0, calls)
        assertEquals(180, second?.seconds)
    }

    @Test
    fun `invalidateActivities removes only targeted entries`() {
        // GIVEN
        BestEffortCache.clear()
        val stream = testStream()

        BestEffortCache.getOrCompute(1, "best-time-distance", "1000.0", stream) {
            ActivityEffort(
                distance = 1000.0,
                seconds = 200,
                deltaAltitude = 10.0,
                idxStart = 0,
                idxEnd = 2,
                averagePower = null,
                label = "A",
                activityShort = ActivityShort(1, "A", ActivityType.Ride.name),
            )
        }
        BestEffortCache.getOrCompute(2, "best-time-distance", "1000.0", stream) {
            ActivityEffort(
                distance = 1000.0,
                seconds = 210,
                deltaAltitude = 11.0,
                idxStart = 0,
                idxEnd = 2,
                averagePower = null,
                label = "B",
                activityShort = ActivityShort(2, "B", ActivityType.Ride.name),
            )
        }

        // WHEN
        val removed = BestEffortCache.invalidateActivities(setOf(1L))

        // THEN
        assertEquals(1, removed)
        assertEquals(1, BestEffortCache.size())

        val missing = BestEffortCache.getOrCompute(1, "best-time-distance", "1000.0", stream) { null }
        assertNull(missing)
    }

    private fun testStream(): Stream = Stream(
        distance = DistanceStream(
            data = listOf(0.0, 500.0, 1000.0),
            originalSize = 3,
            resolution = "high",
            seriesType = "distance",
        ),
        time = TimeStream(
            data = listOf(0, 90, 180),
            originalSize = 3,
            resolution = "high",
            seriesType = "time",
        ),
        altitude = AltitudeStream(
            data = listOf(100.0, 110.0, 120.0),
            originalSize = 3,
            resolution = "high",
            seriesType = "distance",
        ),
    )
}
