package me.nicolas.stravastats.domain.utils

import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Assertions.assertFalse
import org.junit.jupiter.api.Assertions.assertTrue
import org.junit.jupiter.api.Test
import org.junit.jupiter.api.io.TempDir
import java.nio.file.Files
import java.nio.file.Path
import java.util.concurrent.Executors
import java.util.concurrent.TimeUnit

class SafeLocalFileTest {
    @TempDir
    lateinit var tempDir: Path

    @Test
    fun `write replaces content and backs up the previous complete version`() {
        val file = tempDir.resolve("settings.json").toFile()

        SafeLocalFile.write(file, "first".toByteArray())
        assertFalse(tempDir.resolve("settings.json.bak").toFile().exists())
        SafeLocalFile.write(file, "second".toByteArray())

        assertEquals("second", file.readText())
        assertEquals("first", tempDir.resolve("settings.json.bak").toFile().readText())
        Files.list(tempDir).use { files ->
            assertFalse(files.anyMatch { it.fileName.toString().startsWith(".settings.json.tmp-") })
        }
    }

    @Test
    fun `withLock serializes read modify write operations for one file`() {
        val file = tempDir.resolve("counter.txt").toFile()
        val executor = Executors.newFixedThreadPool(8)
        repeat(50) {
            executor.submit {
                SafeLocalFile.withLock(file) {
                    val current = if (file.exists()) file.readText().toInt() else 0
                    SafeLocalFile.write(file, (current + 1).toString().toByteArray())
                }
            }
        }
        executor.shutdown()

        assertTrue(executor.awaitTermination(10, TimeUnit.SECONDS))
        assertEquals("50", file.readText())
    }
}
