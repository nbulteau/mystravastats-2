package me.nicolas.stravastats.api.dto

import me.nicolas.stravastats.domain.business.ChartPeriodPoint

data class ChartPeriodPointDto(
    val periodKey: String,
    val value: Double,
    val activityCount: Int,
)

fun ChartPeriodPoint.toDto() = ChartPeriodPointDto(
    periodKey = this.periodKey,
    value = this.value,
    activityCount = this.activityCount,
)
