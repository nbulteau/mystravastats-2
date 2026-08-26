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
        val goals = directory.resolve("annual-goals-athlete-1.json")
        Files.writeString(goals, """{"goals":{"2026:Ride":{"distance":1000}}}""")
        Files.writeString(directory.resolve("activities-athlete-1-2026.json"), "[]")

        val backup = service.export()
        assertEquals(setOf("annualGoals"), backup.files.keys)

        Files.writeString(goals, """{"goals":{}}""")
        val result = service.restore(backup)

        assertEquals(listOf("annualGoals"), result.restored)
        assertEquals("""{"goals":{}}""", Files.readString(directory.resolve("annual-goals-athlete-1.json.bak")))
        assertEquals(1000, ((backup.files.getValue("annualGoals") as Map<*, *>)["goals"] as Map<*, *>).values
            .map { (it as Map<*, *>)["distance"] }
            .single())
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
