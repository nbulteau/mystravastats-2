package me.nicolas.stravastats.domain.services

import io.mockk.every
import io.mockk.mockk
import me.nicolas.stravastats.domain.services.activityproviders.ActivityProviderCacheIdentity
import me.nicolas.stravastats.domain.services.activityproviders.IActivityProvider
import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Assertions.assertThrows
import org.junit.jupiter.api.Test
import org.junit.jupiter.api.io.TempDir
import java.nio.file.Files
import java.nio.file.Path

class LocalDataBackupServiceTest {
    @TempDir
    lateinit var tempDir: Path

    @Test
    fun `export and restore only whitelisted local data with backup`() {
        val service = service("athlete-1")
        val directory = Files.createDirectories(tempDir.resolve("strava-athlete-1"))
        val settings = directory.resolve("performance-settings-athlete-1.json")
        Files.writeString(settings, """{"ftpOverride":250}""")
        Files.writeString(directory.resolve("activities-athlete-1-2026.json"), "[]")

        val backup = service.export()
        assertEquals(setOf("performanceSettings"), backup.files.keys)

        Files.writeString(settings, "{}")
        val result = service.restore(backup)

        assertEquals(listOf("performanceSettings"), result.restored)
        assertEquals("{}", Files.readString(directory.resolve("performance-settings-athlete-1.json.bak")))
        assertEquals(250, (backup.files.getValue("performanceSettings") as Map<*, *>)["ftpOverride"])
    }

    @Test
    fun `restore rejects another athlete and unknown entries`() {
        val service = service("athlete-1")
        assertThrows(IllegalArgumentException::class.java) {
            service.restore(LocalDataBackup(1, "now", "other", emptyMap()))
        }
        assertThrows(IllegalArgumentException::class.java) {
            service.restore(LocalDataBackup(1, "now", "athlete-1", mapOf("activities" to emptyList<Any>())))
        }
    }

    private fun service(athleteId: String): LocalDataBackupService {
        val provider = mockk<IActivityProvider>()
        every { provider.cacheIdentity() } returns ActivityProviderCacheIdentity(tempDir.toString(), athleteId)
        return LocalDataBackupService(provider)
    }
}
