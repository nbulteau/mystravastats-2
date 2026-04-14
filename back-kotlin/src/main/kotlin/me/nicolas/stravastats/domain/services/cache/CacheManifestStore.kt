package me.nicolas.stravastats.domain.services.cache

import tools.jackson.databind.DeserializationFeature
import tools.jackson.databind.json.JsonMapper
import tools.jackson.module.kotlin.KotlinModule
import java.nio.file.Files
import java.nio.file.Path
import java.nio.file.StandardCopyOption
import java.time.Instant

private const val SCHEMA_VERSION = 1
private const val DEFAULT_BEST_EFFORT_FILE = "best-effort-cache.json"
private const val DEFAULT_WARMUP_FILE = "warmup-summaries.json"

data class CacheManifest(
    val schemaVersion: Int = SCHEMA_VERSION,
    val athleteId: String,
    val updatedAt: String = Instant.now().toString(),
    val bestEffortCache: BestEffortCacheManifest = BestEffortCacheManifest(),
    val warmup: WarmupManifest = WarmupManifest(),
)

data class BestEffortCacheManifest(
    val algoVersion: String = "best-effort-v1",
    val file: String = DEFAULT_BEST_EFFORT_FILE,
    val entries: Int = 0,
    val lastPersistedAt: String? = null,
)

data class WarmupManifest(
    val algoVersion: String = "warmup-v1",
    val file: String = DEFAULT_WARMUP_FILE,
    val priority1: String = "pending",
    val priority2: String = "pending",
    val priority3: String = "pending",
    val preparedYears: List<Int> = emptyList(),
    val lastRunAt: String? = null,
)

data class WarmupSummariesFile(
    val schemaVersion: Int = SCHEMA_VERSION,
    val athleteId: String,
    val generatedAt: String = Instant.now().toString(),
    val yearSummaries: List<WarmupYearSummary>,
    val majorBestEfforts: List<WarmupMetricSummary> = emptyList(),
    val advancedMetrics: List<WarmupMetricSummary> = emptyList(),
)

data class WarmupYearSummary(
    val year: Int,
    val activityCount: Int,
    val totalDistanceKm: Double,
    val totalElevationM: Double,
    val elapsedSeconds: Int,
)

data class WarmupMetricSummary(
    val activityGroup: String,
    val metric: String,
    val target: String,
    val value: String,
    val activityId: Long? = null,
)

object CacheManifestStore {
    private val objectMapper = JsonMapper.builder()
        .addModule(KotlinModule.Builder().build())
        .disable(DeserializationFeature.FAIL_ON_NULL_FOR_PRIMITIVES)
        .build()

    fun defaultManifest(athleteId: String): CacheManifest = CacheManifest(athleteId = athleteId)

    fun load(cacheRoot: String, athleteId: String): CacheManifest? {
        val path = manifestPath(cacheRoot, athleteId)
        if (!Files.exists(path)) {
            return null
        }
        return runCatching {
            objectMapper.readValue(path.toFile(), CacheManifest::class.java)
        }.getOrNull()?.normalize(athleteId)
    }

    fun save(cacheRoot: String, manifest: CacheManifest) {
        writeAtomically(manifestPath(cacheRoot, manifest.athleteId), objectMapper.writerWithDefaultPrettyPrinter().writeValueAsBytes(manifest.normalize(manifest.athleteId)))
    }

    fun saveWarmupSummaries(cacheRoot: String, athleteId: String, summaries: WarmupSummariesFile, manifest: CacheManifest) {
        val filePath = warmupSummariesPath(cacheRoot, athleteId, manifest)
        val normalized = summaries.copy(
            schemaVersion = SCHEMA_VERSION,
            athleteId = athleteId,
            generatedAt = Instant.now().toString(),
            yearSummaries = summaries.yearSummaries.sortedByDescending { it.year }
        )
        writeAtomically(filePath, objectMapper.writerWithDefaultPrettyPrinter().writeValueAsBytes(normalized))
    }

    fun manifestPath(cacheRoot: String, athleteId: String): Path =
        athleteDirectory(cacheRoot, athleteId).resolve("cache-manifest.json")

    fun bestEffortCachePath(cacheRoot: String, athleteId: String, manifest: CacheManifest): Path =
        athleteDirectory(cacheRoot, athleteId).resolve(manifest.bestEffortCache.file.ifBlank { DEFAULT_BEST_EFFORT_FILE })

    fun warmupSummariesPath(cacheRoot: String, athleteId: String, manifest: CacheManifest): Path =
        athleteDirectory(cacheRoot, athleteId).resolve(manifest.warmup.file.ifBlank { DEFAULT_WARMUP_FILE })

    private fun athleteDirectory(cacheRoot: String, athleteId: String): Path =
        Path.of(cacheRoot).resolve("strava-$athleteId")

    private fun CacheManifest.normalize(athleteId: String): CacheManifest = copy(
        schemaVersion = SCHEMA_VERSION,
        athleteId = athleteId,
        updatedAt = updatedAt.ifBlank { Instant.now().toString() },
        bestEffortCache = bestEffortCache.copy(
            algoVersion = bestEffortCache.algoVersion.ifBlank { "best-effort-v1" },
            file = bestEffortCache.file.ifBlank { DEFAULT_BEST_EFFORT_FILE },
        ),
        warmup = warmup.copy(
            algoVersion = warmup.algoVersion.ifBlank { "warmup-v1" },
            file = warmup.file.ifBlank { DEFAULT_WARMUP_FILE },
            priority1 = warmup.priority1.ifBlank { "pending" },
            priority2 = warmup.priority2.ifBlank { "pending" },
            priority3 = warmup.priority3.ifBlank { "pending" },
        ),
    )

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
