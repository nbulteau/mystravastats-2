package me.nicolas.stravastats.domain.services.cache

import tools.jackson.databind.DeserializationFeature
import tools.jackson.databind.json.JsonMapper
import tools.jackson.module.kotlin.KotlinModule
import me.nicolas.stravastats.domain.business.ActivityShort
import java.nio.file.Files
import java.nio.file.Path
import java.nio.file.StandardCopyOption
import java.time.Instant

private const val SEGMENT_CACHE_SCHEMA_VERSION = 1
private const val DEFAULT_SEGMENT_ANALYSIS_FILE = "segment-analysis-cache-v1.json"

data class SegmentAnalysisCacheFile(
    val schemaVersion: Int = SEGMENT_CACHE_SCHEMA_VERSION,
    val athleteId: String,
    val generatedAt: String = Instant.now().toString(),
    val entries: List<SegmentAnalysisCacheEntryFile>,
)

data class SegmentAnalysisCacheEntryFile(
    val key: String,
    val createdAt: String,
    val expiresAt: String,
    val fallbackUsed: Boolean = false,
    val attempts: List<SegmentAttemptRawSnapshot>,
)

data class SegmentAttemptRawSnapshot(
    val effortId: Long,
    val targetId: Long,
    val targetName: String,
    val targetType: String,
    val climbCategory: Int,
    val distance: Double,
    val averageGrade: Double,
    val elapsedTimeSeconds: Int,
    val movingTimeSeconds: Int,
    val speedKph: Double,
    val averagePowerWatts: Double,
    val averageHeartRate: Double,
    val activityDate: String,
    val prRank: Int?,
    val activity: ActivityShort,
)

object SegmentAnalysisCacheStore {
    private val objectMapper = JsonMapper.builder()
        .addModule(KotlinModule.Builder().build())
        .disable(DeserializationFeature.FAIL_ON_NULL_FOR_PRIMITIVES)
        .build()

    fun load(cacheRoot: String, athleteId: String): SegmentAnalysisCacheFile? {
        val cachePath = path(cacheRoot, athleteId)
        if (!Files.exists(cachePath)) {
            return null
        }

        return runCatching {
            objectMapper.readValue(cachePath.toFile(), SegmentAnalysisCacheFile::class.java)
        }.getOrNull()
    }

    fun save(cacheRoot: String, athleteId: String, payload: SegmentAnalysisCacheFile) {
        val normalized = payload.copy(
            schemaVersion = SEGMENT_CACHE_SCHEMA_VERSION,
            athleteId = athleteId,
            generatedAt = Instant.now().toString(),
            entries = payload.entries.sortedByDescending { entry -> entry.createdAt },
        )
        writeAtomically(
            path(cacheRoot, athleteId),
            objectMapper.writerWithDefaultPrettyPrinter().writeValueAsBytes(normalized)
        )
    }

    fun path(cacheRoot: String, athleteId: String): Path =
        athleteDirectory(cacheRoot, athleteId).resolve(DEFAULT_SEGMENT_ANALYSIS_FILE)

    private fun athleteDirectory(cacheRoot: String, athleteId: String): Path =
        Path.of(cacheRoot).resolve("strava-$athleteId")

    private fun writeAtomically(path: Path, payload: ByteArray) {
        Files.createDirectories(path.parent)
        val tempPath = path.resolveSibling("${path.fileName}.tmp")
        Files.write(tempPath, payload)
        try {
            Files.move(
                tempPath,
                path,
                StandardCopyOption.REPLACE_EXISTING,
                StandardCopyOption.ATOMIC_MOVE
            )
        } catch (_: Exception) {
            Files.move(tempPath, path, StandardCopyOption.REPLACE_EXISTING)
        }
    }
}
