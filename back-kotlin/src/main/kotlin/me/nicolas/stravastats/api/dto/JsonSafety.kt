package me.nicolas.stravastats.api.dto

internal fun Double.finiteOrZero(): Double {
    if (!java.lang.Double.isFinite(this)) {
        return 0.0
    }
    return this
}

internal fun Float.finiteOrZero(): Float {
    if (!java.lang.Float.isFinite(this)) {
        return 0f
    }
    return this
}

internal fun Double.finiteFloatOrZero(): Float {
    if (!java.lang.Double.isFinite(this) || this > Float.MAX_VALUE || this < -Float.MAX_VALUE) {
        return 0f
    }
    return this.toFloat()
}

internal fun Double.finiteIntOrZero(): Int {
    if (!java.lang.Double.isFinite(this) || this > Int.MAX_VALUE || this < Int.MIN_VALUE) {
        return 0
    }
    return this.toInt()
}

internal fun Double?.finiteOrNull(): Double? {
    return this?.takeIf { java.lang.Double.isFinite(it) }
}

internal fun List<Double>.finiteValues(): List<Double> {
    return this.map { it.finiteOrZero() }
}

internal fun List<List<Double>>.finiteCoordinateValues(): List<List<Double>> {
    return this.map { row -> row.finiteValues() }
}
