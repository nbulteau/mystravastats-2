package me.nicolas.stravastats.domain.services

import me.nicolas.stravastats.domain.business.ActivityType
import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.business.strava.StravaAthlete
import me.nicolas.stravastats.domain.business.strava.StravaDetailedActivity
import me.nicolas.stravastats.domain.services.activityproviders.IActivityProvider
import org.junit.jupiter.api.AfterEach
import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Assertions.assertNotNull
import org.junit.jupiter.api.Assertions.assertTrue
import org.junit.jupiter.api.Test
import org.springframework.data.domain.Page
import org.springframework.data.domain.PageImpl
import org.springframework.data.domain.Pageable
import java.nio.file.Files
import kotlin.io.path.copyTo

class SourceSyncServiceTest {
    private val runtimeKeys = listOf("FIT_FILES_PATH", "FIT_INBOX_PATH", "GARMIN_FIT_SOURCE_PATH")

    @AfterEach
    fun tearDown() {
        runtimeKeys.forEach(System::clearProperty)
    }

    @Test
    fun `synchronize copies Garmin FIT files to inbox then imports into year directory`() {
        val temp = Files.createTempDirectory("source-sync-kotlin")
        val sourceRoot = temp.resolve("FENIX")
        val activityDirectory = sourceRoot.resolve("GARMIN").resolve("ACTIVITY")
        val destinationDirectory = temp.resolve("fit")
        val inboxDirectory = destinationDirectory.resolve("_inbox")
        Files.createDirectories(activityDirectory)

        val fixture = java.io.File("fit-colin/2022/C5CC1616.FIT").toPath()
        fixture.copyTo(activityDirectory.resolve("ride.fit"))

        System.setProperty("FIT_FILES_PATH", destinationDirectory.toString())
        System.setProperty("FIT_INBOX_PATH", inboxDirectory.toString())
        System.setProperty("GARMIN_FIT_SOURCE_PATH", sourceRoot.toString())

        val provider = ReloadOnlyActivityProvider()
        val result = SourceSyncService(provider).synchronize("test")

        assertEquals("completed", result.status)
        assertEquals("imported", result.fit.status)
        assertEquals(1, result.fit.deviceSync?.copiedFiles)
        assertEquals(1, result.fit.importedFiles)
        assertTrue(result.reloaded)
        assertTrue(provider.reloadCalled)
        assertTrue(inboxDirectory.resolve("ride.fit").toFile().exists())
        assertNotNull(result.fit.imported.firstOrNull())
        assertTrue(java.io.File(result.fit.imported.first().destination).exists())
    }

    private class ReloadOnlyActivityProvider : IActivityProvider {
        var reloadCalled = false

        override fun athlete(): StravaAthlete = StravaAthlete(id = 0)

        override fun listActivitiesPaginated(pageable: Pageable): Page<StravaActivity> = PageImpl(emptyList(), pageable, 0)

        override fun getActivity(activityId: Long): StravaActivity? = null

        override fun getDetailedActivity(activityId: Long): StravaDetailedActivity? = null

        override fun getActivitiesByActivityTypeGroupByActiveDays(activityTypes: Set<ActivityType>): Map<String, Int> = emptyMap()

        override fun getActivitiesByActivityTypeByYearGroupByActiveDays(
            activityTypes: Set<ActivityType>,
            year: Int,
        ): Map<String, Int> = emptyMap()

        override fun getActivitiesByActivityTypeAndYear(
            activityTypes: Set<ActivityType>,
            year: Int?,
        ): List<StravaActivity> = emptyList()

        override fun getActivitiesByActivityTypeGroupByYear(
            activityTypes: Set<ActivityType>,
        ): Map<String, List<StravaActivity>> = emptyMap()

        override fun reload(): Boolean {
            reloadCalled = true
            return true
        }
    }
}
