package me.nicolas.stravastats.domain.business

data class HeartRateZoneSettings(
    val maxHr: Int? = null,
    val thresholdHr: Int? = null,
    val reserveHr: Int? = null,
)

enum class HeartRateZoneMethod {
    MAX,
    THRESHOLD,
    RESERVE,
}

enum class HeartRateZoneSource {
    ATHLETE_SETTINGS,
    DERIVED_FROM_DATA,
}

data class ResolvedHeartRateZoneSettings(
    val maxHr: Int,
    val thresholdHr: Int? = null,
    val reserveHr: Int? = null,
    val method: HeartRateZoneMethod,
    val source: HeartRateZoneSource,
)

data class HeartRateZoneDistribution(
    val zone: String,
    val label: String,
    val seconds: Int,
    val percentage: Double,
)

data class HeartRateZoneActivitySummary(
    val activity: ActivityShort,
    val activityDate: String,
    val totalTrackedSeconds: Int,
    val easySeconds: Int,
    val hardSeconds: Int,
    val easyHardRatio: Double?,
    val zones: List<HeartRateZoneDistribution>,
)

data class HeartRateZonePeriodSummary(
    val period: String,
    val totalTrackedSeconds: Int,
    val easySeconds: Int,
    val hardSeconds: Int,
    val easyHardRatio: Double?,
    val zones: List<HeartRateZoneDistribution>,
)

data class HeartRateZoneAnalysis(
    val settings: HeartRateZoneSettings,
    val resolvedSettings: ResolvedHeartRateZoneSettings?,
    val hasHeartRateData: Boolean,
    val totalTrackedSeconds: Int,
    val easyHardRatio: Double?,
    val zones: List<HeartRateZoneDistribution>,
    val activities: List<HeartRateZoneActivitySummary>,
    val byMonth: List<HeartRateZonePeriodSummary>,
    val byYear: List<HeartRateZonePeriodSummary>,
)
