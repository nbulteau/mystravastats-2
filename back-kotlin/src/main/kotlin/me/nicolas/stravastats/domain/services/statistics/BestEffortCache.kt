package me.nicolas.stravastats.domain.services.statistics

import me.nicolas.stravastats.domain.business.ActivityEffort
import me.nicolas.stravastats.domain.business.strava.stream.Stream
import java.util.Optional
import java.util.concurrent.ConcurrentHashMap

private data class EffortCacheKey(
    val activityId: Long,
    val metric: String,
    val target: String,
    val streamSize: Int,
)

internal object BestEffortCache {
    private val cache = ConcurrentHashMap<EffortCacheKey, Optional<ActivityEffort>>()

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
}
