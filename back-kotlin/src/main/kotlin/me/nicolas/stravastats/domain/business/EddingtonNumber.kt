package me.nicolas.stravastats.domain.business

enum class EddingtonScope(val apiValue: String) {
    LIFETIME("lifetime"),
    YEAR("year"),
    ROLLING_12_MONTHS("rolling-12-months");

    companion object {
        fun fromApiValue(value: String?): EddingtonScope {
            if (value.isNullOrBlank()) {
                return LIFETIME
            }
            return entries.firstOrNull { scope -> scope.apiValue == value }
                ?: throw IllegalArgumentException("invalid Eddington scope: $value")
        }
    }
}

enum class EddingtonMetric(val apiValue: String, val unit: String) {
    DISTANCE("distance", "km"),
    ELEVATION("elevation", "m");

    companion object {
        fun fromApiValue(value: String?): EddingtonMetric {
            if (value.isNullOrBlank()) {
                return DISTANCE
            }
            return entries.firstOrNull { metric -> metric.apiValue == value }
                ?: throw IllegalArgumentException("invalid Eddington metric: $value")
        }
    }
}

enum class EddingtonBasis(val apiValue: String) {
    DAYS("days"),
    ACTIVITIES("activities");

    companion object {
        fun fromApiValue(value: String?): EddingtonBasis {
            if (value.isNullOrBlank()) {
                return DAYS
            }
            return entries.firstOrNull { basis -> basis.apiValue == value }
                ?: throw IllegalArgumentException("invalid Eddington basis: $value")
        }
    }
}

data class EddingtonNumber (
    val eddingtonNumber: Int,
    val eddingtonList: List<Int>,
    val scope: EddingtonScope = EddingtonScope.LIFETIME,
    val metric: EddingtonMetric = EddingtonMetric.DISTANCE,
    val basis: EddingtonBasis = EddingtonBasis.DAYS,
    val unit: String = metric.unit,
    val nextTarget: Int = eddingtonNumber + 1,
    val qualifyingCount: Int = eddingtonList.getOrElse(nextTarget - 1) { 0 },
    val missingCount: Int = maxOf(nextTarget - qualifyingCount, 0),
    val qualifyingDays: Int = qualifyingCount,
    val missingDays: Int = missingCount,
)
