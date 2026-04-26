package me.nicolas.stravastats.domain.services

import me.nicolas.stravastats.domain.business.SourceMode
import me.nicolas.stravastats.domain.business.SourceModePreviewRequest
import org.junit.jupiter.api.AfterEach
import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Assertions.assertFalse
import org.junit.jupiter.api.Assertions.assertTrue
import org.junit.jupiter.api.BeforeEach
import org.junit.jupiter.api.Test
import org.junit.jupiter.api.io.TempDir
import java.nio.file.Files
import java.nio.file.Path

class SourceModeServiceTest {
    @TempDir
    private lateinit var tempDir: Path

    private val runtimeKeys = listOf("STRAVA_CACHE_PATH", "FIT_FILES_PATH", "GPX_FILES_PATH")

    @BeforeEach
    fun setUp() {
        runtimeKeys.forEach { key -> System.setProperty(key, "") }
    }

    @AfterEach
    fun tearDown() {
        runtimeKeys.forEach(System::clearProperty)
    }

    @Test
    fun `preview validates GPX year folders and activity fields`() {
        // GIVEN
        writeGpx(
            year = "2026",
            name = "ride.gpx",
            content = """<?xml version="1.0" encoding="UTF-8"?>
<gpx version="1.1" creator="test" xmlns:gpxtpx="http://www.garmin.com/xmlschemas/TrackPointExtension/v1">
  <trk>
    <name>Ride</name>
    <type>cycling</type>
    <trkseg>
      <trkpt lat="48.1000" lon="-1.6000">
        <ele>10</ele>
        <time>2026-01-01T08:00:00Z</time>
      </trkpt>
      <trkpt lat="48.1010" lon="-1.6000">
        <ele>15</ele>
        <time>2026-01-01T08:05:00Z</time>
      </trkpt>
    </trkseg>
  </trk>
</gpx>""",
        )
        val service = SourceModeService()

        // WHEN
        val preview = service.preview(SourceModePreviewRequest(mode = "GPX", path = tempDir.toString()))

        // THEN
        assertEquals(SourceMode.GPX, preview.mode)
        assertTrue(preview.supported)
        assertTrue(preview.readable)
        assertTrue(preview.validStructure)
        assertEquals(1, preview.fileCount)
        assertEquals(1, preview.validFileCount)
        assertEquals(1, preview.activityCount)
        assertEquals("2026", preview.years.single().year)
        assertTrue(preview.missingFields.contains("power"))
        assertEquals(SourceMode.STRAVA, preview.activeMode)
        assertFalse(preview.active)
        assertTrue(preview.activationCommand.contains("GPX_FILES_PATH='${tempDir}'"))
        assertEquals("GPX_FILES_PATH", preview.environment.first().key)
        assertEquals(tempDir.toString(), preview.environment.first().value)
    }

    @Test
    fun `preview reports missing local source path`() {
        // GIVEN
        val service = SourceModeService()

        // WHEN
        val preview = service.preview(SourceModePreviewRequest(mode = "FIT", path = ""))

        // THEN
        assertEquals(SourceMode.FIT, preview.mode)
        assertEquals("FIT_FILES_PATH", preview.configKey)
        assertFalse(preview.readable)
        assertFalse(preview.validStructure)
        assertEquals(listOf("activities"), preview.missingFields)
        assertEquals("path is required", preview.errors.single().message)
        assertEquals(SourceMode.STRAVA, preview.activeMode)
        assertEquals("", preview.activationCommand)
        assertEquals("FIT_FILES_PATH", preview.environment.first().key)
    }

    private fun writeGpx(year: String, name: String, content: String) {
        val yearDirectory = tempDir.resolve(year)
        Files.createDirectories(yearDirectory)
        Files.writeString(yearDirectory.resolve(name), content)
    }
}
