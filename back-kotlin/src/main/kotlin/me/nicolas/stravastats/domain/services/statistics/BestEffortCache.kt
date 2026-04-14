package me.nicolas.stravastats.domain.services.statistics

import me.nicolas.stravastats.domain.business.ActivityEffort
import me.nicolas.stravastats.domain.business.strava.stream.Stream
import tools.jackson.databind.DeserializationFeature
import tools.jackson.databind.SerializationFeature
import tools.jackson.databind.json.JsonMapper
import tools.jackson.module.kotlin.KotlinModule
import java.nio.file.Files
import java.nio.file.Path
import java.nio.file.StandardCopyOption
import java.util.Optional
import java.util.concurrent.ConcurrentHashMap

data class EffortCacheKey(
    val activityId: Long,
    val metric: String,
    val target: String,
    val streamSize: Int,
)

private data class PersistedEffortEntry(
    val key: EffortCacheKey,
    val hasValue: Boolean,
    val effort: ActivityEffort? = null,
)

internal object BestEffortCache {
    private val cache = ConcurrentHashMap<EffortCacheKey, Optional<ActivityEffort>>()
    private val objectMapper = JsonMapper.builder()
        .addModule(KotlinModule.Builder()
            .withReflectionCacheSize(512)
            .build())
        .disable(DeserializationFeature.FAIL_ON_NULL_FOR_PRIMITIVES)
        .disable(SerializationFeature.FAIL_ON_EMPTY_BEANS)
        .build()

    fun getOrCompute(
        activityId: Long,
        metric: String,
        target: String,
        stream: Stream,
        supplier: () -> ActivityEffort?
    ): ActivityEffort? {
        val key = EffortCacheKey(
            activityId = activityId,
            metric = metric,
            target = target,
            streamSize = stream.distance.originalSize,
        )

        return cache.computeIfAbsent(key) {
            Optional.ofNullable(supplier())
        }.orElse(null)
    }

    fun loadFromDisk(path: Path): Int {
        if (!Files.exists(path)) {
            return 0
        }

        val entries = runCatching {
            objectMapper.readValue(path.toFile(), Array<PersistedEffortEntry>::class.java).toList()
        }.getOrElse {
            clear()
            return 0
        }

        cache.clear()
        entries.forEach { entry ->
            val optional = if (entry.hasValue) {
                Optional.ofNullable(entry.effort)
            } else {
                Optional.empty()
            }
            cache[entry.key] = optional
        }

        return cache.size
    }

    fun saveToDisk(path: Path): Int {
        val entries = cache.entries
            .map { (key, value) ->
                PersistedEffortEntry(
                    key = key,
                    hasValue = value.isPresent,
                    effort = value.orElse(null),
                )
            }
            .sortedWith(
                compareBy<PersistedEffortEntry> { it.key.activityId }
                    .thenBy { it.key.metric }
                    .thenBy { it.key.target }
                    .thenBy { it.key.streamSize }
            )

        return runCatching {
            Files.createDirectories(path.parent)
            val payload = objectMapper.writerWithDefaultPrettyPrinter().writeValueAsBytes(entries)
            val tmpPath = path.resolveSibling("${path.fileName}.tmp")
            Files.write(tmpPath, payload)
            runCatching {
                Files.move(
                    tmpPath,
                    path,
                    StandardCopyOption.REPLACE_EXISTING,
                    StandardCopyOption.ATOMIC_MOVE,
                )
            }.onFailure {
                Files.move(tmpPath, path, StandardCopyOption.REPLACE_EXISTING)
            }
            entries.size
        }.getOrElse { e ->
            // Log the error but don't crash - this is not critical
            System.err.println("Warning: Failed to save best effort cache: ${e.message}")
            e.printStackTrace()
            0
        }
    }

    fun invalidateActivities(activityIds: Set<Long>): Int {
        if (activityIds.isEmpty()) {
            return 0
        }

        var removed = 0
        cache.keys.removeIf { key ->
            val shouldRemove = activityIds.contains(key.activityId)
            if (shouldRemove) {
                removed++
            }
            shouldRemove
        }
        return removed
    }

    fun size(): Int = cache.size

    fun clear() {
        cache.clear()
    }
}
