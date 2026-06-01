package me.nicolas.stravastats.domain.services.statistics

internal data class ElevationGainLoss(
    val gain: Double,
    val loss: Double,
)

internal class ElevationGainLossPrefix private constructor(
    private val gains: DoubleArray,
    private val losses: DoubleArray,
) {
    fun between(idxStart: Int, idxEnd: Int): ElevationGainLoss? {
        if (gains.isEmpty() || losses.isEmpty()) {
            return null
        }

        val start = idxStart.coerceIn(0, gains.lastIndex)
        val end = idxEnd.coerceIn(start, gains.lastIndex)
        return ElevationGainLoss(
            gain = gains[end] - gains[start],
            loss = losses[end] - losses[start],
        )
    }

    companion object {
        fun from(altitudes: List<Double>, dataSize: Int): ElevationGainLossPrefix {
            val size = minOf(altitudes.size, dataSize)
            if (size <= 0) {
                return ElevationGainLossPrefix(DoubleArray(0), DoubleArray(0))
            }

            val gains = DoubleArray(size)
            val losses = DoubleArray(size)
            for (index in 1 until size) {
                gains[index] = gains[index - 1]
                losses[index] = losses[index - 1]

                val previous = altitudes[index - 1]
                val current = altitudes[index]
                if (!previous.isFiniteAltitude() || !current.isFiniteAltitude()) {
                    continue
                }

                val delta = current - previous
                if (delta > 0) {
                    gains[index] += delta
                } else if (delta < 0) {
                    losses[index] += -delta
                }
            }

            return ElevationGainLossPrefix(gains, losses)
        }
    }
}

private fun Double.isFiniteAltitude(): Boolean = !isNaN() && !isInfinite()
