package me.nicolas.stravastats.domain.business.training

data class LoadActivity(
    var activityId: Long = 0,
    var name: String = "",
    var date: String = "",
    var sport: String = "",
    var source: String = "",
    var reason: String = "",
    var load: Double? = null,
    var ftp: Int? = null,
    var ftpEffectiveFrom: String? = null,
    var normalizedPower: Double? = null,
    var intensityFactor: Double? = null,
    var best5MinutePower: Double? = null,
    var movingSeconds: Int = 0,
    var elapsedSeconds: Int = 0,
    var recordedSeconds: Double = 0.0,
    var coveredSeconds: Double = 0.0,
    var distanceMeters: Double = 0.0,
    var elevationMeters: Double = 0.0,
    var aerobicSeconds: Double = 0.0,
    var thresholdSeconds: Double = 0.0,
    var highIntensitySeconds: Double = 0.0,
)

data class LoadPeriod(
    var startDate: String = "",
    var endDate: String = "",
    var activityCount: Int = 0,
    var scoredCount: Int = 0,
    var measuredLoad: Double? = null,
    var estimatedLoad: Double? = null,
    var movingSeconds: Int = 0,
    var distanceMeters: Double = 0.0,
    var elevationMeters: Double = 0.0,
    var coveredSeconds: Double = 0.0,
    var recordedSeconds: Double = 0.0,
    var aerobicSeconds: Double = 0.0,
    var thresholdSeconds: Double = 0.0,
    var highIntensitySeconds: Double = 0.0,
    var best5MinutePower: Double? = null,
    var best5MinuteActivityId: Long? = null,
)

data class LoadWeek(
    var summary: LoadPeriod = LoadPeriod(),
    var days: List<LoadPeriod> = emptyList(),
)

data class Report(
    var method: String = "",
    var maxGapSeconds: Int = 0,
    var undatedActivities: Int = 0,
    var weeks: List<LoadWeek> = emptyList(),
    var activities: List<LoadActivity> = emptyList(),
)

