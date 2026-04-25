package me.nicolas.stravastats.adapters.localrepositories.fit

import me.nicolas.stravastats.domain.business.strava.stream.Stream
import kotlin.math.pow
import kotlin.math.roundToInt

internal data class FitPowerMetrics(
    val averageWatts: Int,
    val weightedAverageWatts: Int,
    val kilojoules: Double,
    val hasDeviceWatts: Boolean,
)

internal fun computeFitPowerMetrics(
    sessionAveragePower: Int?,
    stream: Stream,
    elapsedTime: Int,
): FitPowerMetrics {
    val samples = fitPowerSamples(stream)
    val streamAverageWatts = averageFitPower(samples)
    val averageWatts = sessionAveragePower?.takeIf { it > 0 } ?: streamAverageWatts
    val weightedAverageWatts = sessionAveragePower?.takeIf { it > 0 } ?: normalizedFitPower(samples)

    return FitPowerMetrics(
        averageWatts = averageWatts,
        weightedAverageWatts = weightedAverageWatts,
        kilojoules = 0.8604 * averageWatts * maxOf(elapsedTime, 0) / 1000,
        hasDeviceWatts = sessionAveragePower?.let { it > 0 } == true || samples.isNotEmpty(),
    )
}

internal fun fitPowerSamples(stream: Stream): List<Int> {
    val samples = stream.watts?.data.orEmpty().mapNotNull { watts ->
        watts?.takeIf { it >= 0 }
    }
    return if (samples.any { it > 0 }) samples else emptyList()
}

internal fun averageFitPower(samples: List<Int>): Int {
    if (samples.isEmpty()) {
        return 0
    }
    return samples.average().roundToInt()
}

internal fun normalizedFitPower(samples: List<Int>): Int {
    if (samples.isEmpty()) {
        return 0
    }

    val rollingWindowSeconds = 30
    if (samples.size < rollingWindowSeconds) {
        return averageFitPower(samples)
    }

    val fourthPowerAverage = samples
        .windowed(rollingWindowSeconds)
        .map { window -> window.average().pow(4.0) }
        .average()

    return fourthPowerAverage.pow(0.25).roundToInt()
}
