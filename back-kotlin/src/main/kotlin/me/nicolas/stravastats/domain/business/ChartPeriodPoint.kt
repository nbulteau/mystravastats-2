package me.nicolas.stravastats.domain.business

data class ChartPeriodPoint(
    val periodKey: String,
    val value: Double,
    val activityCount: Int,
)
