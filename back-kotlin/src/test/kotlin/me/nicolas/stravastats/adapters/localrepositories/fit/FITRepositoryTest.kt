package me.nicolas.stravastats.adapters.localrepositories.fit

import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Assertions.assertNotEquals
import org.junit.jupiter.api.Assertions.assertNotNull
import org.junit.jupiter.api.Assertions.assertTrue
import org.junit.jupiter.api.Test
import org.junit.jupiter.api.io.TempDir
import java.io.File
import java.nio.file.Path
import java.time.Instant
import java.time.OffsetDateTime

class FITRepositoryTest {
    @Test
    fun `decodeActivity keeps FIT activity local and emits parseable dates`() {
        val fixture = File(sharedSourceModeFixtureRoot(), "fit/2026/smoke-ride.fit")
        val repository = FITRepository(fixture.parentFile.parentFile.path)

        val activity = repository.decodeActivity(fixture)

        assertNotNull(activity)
        activity!!
        assertEquals(0L, activity.uploadId)
        assertTrue(activity.id in 1L..MAX_SAFE_JS_INTEGER, "expected FIT activity id to be safe for JavaScript clients, got ${activity.id}")
        assertEquals(activity.id, repository.decodeActivity(fixture)?.id)
        assertNotEquals(listOf(0.0, 0.0), activity.startLatlng)
        val stream = activity.stream
        assertNotNull(stream)
        val expectedSize = stream!!.time.data.size
        assertEquals(expectedSize, stream.distance.data.size)
        stream.latlng?.let { assertEquals(expectedSize, it.data.size) }
        stream.altitude?.let { assertEquals(expectedSize, it.data.size) }
        stream.heartrate?.let { assertTrue(it.data.any { value -> value > 0 }) }
        stream.cadence?.let { assertTrue(it.data.any { value -> value > 0 }) }
        stream.watts?.let {
            assertEquals(expectedSize, it.data.size)
            assertTrue(it.data.none { value -> value == null })
        }
        Instant.parse(activity.startDate)
        OffsetDateTime.parse(activity.startDateLocal)
    }

    @Test
    fun `decodeActivity id does not depend on file path`(@TempDir tempDir: Path) {
        val fixture = File(sharedSourceModeFixtureRoot(), "fit/2026/smoke-ride.fit")
        val copiedFixture = fixture.copyTo(tempDir.resolve("renamed.fit").toFile())
        val repository = FITRepository(tempDir.toString())

        assertEquals(
            repository.decodeActivity(fixture)?.id,
            repository.decodeActivity(copiedFixture)?.id,
        )
    }

    private fun sharedSourceModeFixtureRoot(): File {
        val workingDirectory = File(System.getProperty("user.dir")).canonicalFile
        val repoRoot = if (workingDirectory.name == "back-kotlin") workingDirectory.parentFile else workingDirectory
        return File(repoRoot, "test-fixtures/source-modes").canonicalFile
    }

    private companion object {
        const val MAX_SAFE_JS_INTEGER: Long = 9_007_199_254_740_991L
    }
}
