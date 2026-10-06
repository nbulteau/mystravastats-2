package me.nicolas.stravastats.adapters.localrepositories.fit

import me.nicolas.stravastats.domain.business.strava.stream.Stream
import kotlin.math.roundToInt

internal data class FitPowerMetrics(
    val averageWatts: Int,
    val weightedAverageWatts: Int,
    val kilojoules: Double,
    val hasDeviceWatts: Boolean,
)

internal fun computeFitPowerMetrics(sessionAveragePower: Int?, stream: Stream, elapsedTime: Int): FitPowerMetrics {
    val timeline = me.nicolas.stravastats.domain.business.strava.stream.PowerTimeline(stream.watts?.data.orEmpty(),stream.time.data)
    val session = sessionAveragePower?.takeIf { it > 0 }
    val average = if(timeline.valid) timeline.average(stream.time.data.first().toDouble(),stream.time.data.last().toDouble()) else null
    val duration = if(timeline.valid) stream.time.data.last()-stream.time.data.first() else 0
    return FitPowerMetrics(
        averageWatts = session ?: average?.roundToInt() ?: 0,
        weightedAverageWatts = session ?: timeline.normalized()?.roundToInt() ?: 0,
        kilojoules = if(session!=null) session*maxOf(elapsedTime,0)/1000.0 else (average ?: 0.0)*duration/1000.0,
        hasDeviceWatts = session!=null || stream.watts?.data.orEmpty().any { it != null && it > 0 },
    )
}
