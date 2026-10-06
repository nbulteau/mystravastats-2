package me.nicolas.stravastats.domain.services.statistics

/** Complete-window statistics: null readings retain their index and invalidate a window. */
internal class PowerWindowPrefix(samples: List<Int?>?) {
    private val size = samples?.size ?: 0
    private val sums = LongArray(size + 1)
    private val missing = IntArray(size + 1)
    init {
        samples?.forEachIndexed { index, value ->
            sums[index + 1] = sums[index] + (value ?: 0).toLong()
            missing[index + 1] = missing[index] + if (value == null) 1 else 0
        }
    }
    fun sum(start: Int, end: Int): Long? =
        if (start < 0 || end < start || end >= size || missing[end + 1] != missing[start]) null
        else sums[end + 1] - sums[start]

    fun average(start: Int, end: Int): Double? = sum(start, end)?.toDouble()?.div(end - start + 1)
}
