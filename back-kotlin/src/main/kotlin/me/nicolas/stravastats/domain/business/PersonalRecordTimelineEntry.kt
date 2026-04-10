package me.nicolas.stravastats.domain.business

data class PersonalRecordTimelineEntry(
    val metricKey: String,
    val metricLabel: String,
    val activityDate: String,
    val value: String,
    val previousValue: String?,
    val improvement: String?,
    val activity: ActivityShort,
)
