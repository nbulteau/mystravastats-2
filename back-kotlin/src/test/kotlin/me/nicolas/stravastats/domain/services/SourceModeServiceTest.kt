package me.nicolas.stravastats.domain.services

import me.nicolas.stravastats.domain.business.SourceMode
import me.nicolas.stravastats.domain.business.SourceModePreviewRequest
import me.nicolas.stravastats.domain.interfaces.ILocalStorageProvider
import me.nicolas.stravastats.domain.interfaces.ISourcePreviewRepositoryFactory
import me.nicolas.stravastats.domain.interfaces.IYearActivityStorageProvider
import org.junit.jupiter.api.AfterEach
import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Assertions.assertFalse
import org.junit.jupiter.api.Assertions.assertTrue
import org.junit.jupiter.api.BeforeEach
import org.junit.jupiter.api.Test
import org.junit.jupiter.api.io.TempDir
import java.time.Instant
import java.nio.file.Files
import java.nio.file.Path

class SourceModeServiceTest {
    @TempDir
    private lateinit var tempDir: Path

    private val runtimeKeys = listOf("STRAVA_CACHE_PATH", "FIT_FILES_PATH", "GPX_FILES_PATH")
    private lateinit var repositoryFactory: ISourcePreviewRepositoryFactory

    @BeforeEach
    fun setUp() {
        runtimeKeys.forEach { key -> System.setProperty(key, "") }
        repositoryFactory = object : ISourcePreviewRepositoryFactory {
            override fun createFitRepository(path: String): IYearActivityStorageProvider =
                me.nicolas.stravastats.adapters.localrepositories.fit.FITRepository(path)

            override fun createGpxRepository(path: String): IYearActivityStorageProvider =
                me.nicolas.stravastats.adapters.localrepositories.gpx.GPXRepository(path)

            override fun createStravaRepository(path: String): ILocalStorageProvider =
                me.nicolas.stravastats.adapters.localrepositories.strava.StravaRepository(path)
        }
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
        val service = SourceModeService(repositoryFactory)

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
        val service = SourceModeService(repositoryFactory)

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

    @Test
    fun `preview reports Strava OAuth enrollment status`() {
        // GIVEN
        Files.writeString(tempDir.resolve(".strava"), "clientId=12345\nclientSecret=secret\nuseCache=false\n")
        Files.writeString(
            tempDir.resolve(".strava-token.json"),
            """
            {
              "access_token": "access",
              "refresh_token": "refresh",
              "expires_at": ${Instant.now().plusSeconds(3600).epochSecond},
              "scope": "read_all,activity:read_all,profile:read_all",
              "athlete": { "id": 42, "firstname": "Ada", "lastname": "Lovelace" }
            }
            """.trimIndent(),
        )
        val service = SourceModeService(repositoryFactory)

        // WHEN
        val preview = service.preview(SourceModePreviewRequest(mode = "STRAVA", path = tempDir.toString()))

        // THEN
        val oauth = requireNotNull(preview.stravaOAuth)
        assertEquals("ready", oauth.status)
        assertTrue(oauth.credentialsPresent)
        assertTrue(oauth.tokenPresent)
        assertTrue(oauth.tokenReadable)
        assertEquals("42", oauth.athleteId)
        assertEquals("Ada Lovelace", oauth.athleteName)
        assertTrue(oauth.setupCommand.contains("setup-strava-oauth.mjs"))
    }

    @Test
    fun `starts Strava OAuth enrollment and writes credentials`() {
        // GIVEN
        val service = SourceModeService(repositoryFactory)

        // WHEN
        val result = service.startStravaOAuth(
            me.nicolas.stravastats.domain.business.StravaOAuthStartRequest(
                path = tempDir.toString(),
                clientId = "12345",
                clientSecret = "secret",
                useCache = false,
            ),
        )

        // THEN
        assertEquals("oauth_started", result.status)
        assertTrue(result.authorizeUrl.contains("state="))
        assertTrue(result.authorizeUrl.contains("client_id=12345"))
        val credentials = Files.readString(tempDir.resolve(".strava"))
        assertTrue(credentials.contains("clientId=12345"))
        assertTrue(credentials.contains("useCache=false"))
    }

    @Test
    fun `starts Strava cache-only enrollment without OAuth URL`() {
        // GIVEN
        val service = SourceModeService(repositoryFactory)

        // WHEN
        val result = service.startStravaOAuth(
            me.nicolas.stravastats.domain.business.StravaOAuthStartRequest(
                path = tempDir.toString(),
                clientId = "12345",
                useCache = true,
            ),
        )

        // THEN
        assertEquals("cache_only", result.status)
        assertEquals("", result.authorizeUrl)
        assertTrue(result.cacheOnly)
        assertTrue(Files.readString(tempDir.resolve(".strava")).contains("useCache=true"))
    }

    private fun writeGpx(year: String, name: String, content: String) {
        val yearDirectory = tempDir.resolve(year)
        Files.createDirectories(yearDirectory)
        Files.writeString(yearDirectory.resolve(name), content)
    }
}
