package me.nicolas.stravastats.domain.business

enum class AnnualGoalMetric {
    DISTANCE_KM,
    ELEVATION_METERS,
    MOVING_TIME_SECONDS,
    ACTIVITIES,
    ACTIVE_DAYS,
    EDDINGTON,
}

enum class AnnualGoalStatus {
    NOT_SET,
    AHEAD,
    ON_TRACK,
    BEHIND,
}

data class AnnualGoalTargets(
    val distanceKm: Double? = null,
    val elevationMeters: Int? = null,
    val movingTimeSeconds: Int? = null,
    val activities: Int? = null,
    val activeDays: Int? = null,
    val eddington: Int? = null,
)

data class AnnualGoalProgress(
    val metric: AnnualGoalMetric,
    val label: String,
    val unit: String,
    val current: Double,
    val target: Double,
    val progressPercent: Double,
    val expectedProgressPercent: Double,
    val projectedEndOfYear: Double,
    val requiredPace: Double,
    val requiredPaceUnit: String,
    val status: AnnualGoalStatus,
)

data class AnnualGoals(
    val year: Int,
    val activityTypeKey: String,
    val targets: AnnualGoalTargets,
    val progress: List<AnnualGoalProgress>,
)
