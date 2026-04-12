package me.nicolas.stravastats.api.dto

import io.swagger.v3.oas.annotations.media.Schema
import me.nicolas.stravastats.domain.business.HeartRateZoneActivitySummary
import me.nicolas.stravastats.domain.business.HeartRateZoneAnalysis
import me.nicolas.stravastats.domain.business.HeartRateZoneDistribution
import me.nicolas.stravastats.domain.business.HeartRateZonePeriodSummary
import me.nicolas.stravastats.domain.business.HeartRateZoneSettings
import me.nicolas.stravastats.domain.business.ResolvedHeartRateZoneSettings

@Schema(description = "Heart rate zone setup", name = "HeartRateZoneSettings")
data class HeartRateZoneSettingsDto(
    val maxHr: Int? = null,
    val thresholdHr: Int? = null,
    val reserveHr: Int? = null,
)

@Schema(description = "Resolved heart rate zone setup", name = "ResolvedHeartRateZoneSettings")
data class ResolvedHeartRateZoneSettingsDto(
    val maxHr: Int,
    val thresholdHr: Int? = null,
    val reserveHr: Int? = null,
    val method: String,
    val source: String,
)

@Schema(description = "Heart rate zone distribution", name = "HeartRateZoneDistribution")
data class HeartRateZoneDistributionDto(
    val zone: String,
    val label: String,
    val seconds: Int,
    val percentage: Double,
)

@Schema(description = "Heart rate zone summary for one activity", name = "HeartRateZoneActivitySummary")
data class HeartRateZoneActivitySummaryDto(
    val activity: ActivityShortDto,
    val activityDate: String,
    val totalTrackedSeconds: Int,
    val easySeconds: Int,
    val hardSeconds: Int,
    val easyHardRatio: Double?,
    val zones: List<HeartRateZoneDistributionDto>,
)

@Schema(description = "Heart rate zone summary for one period", name = "HeartRateZonePeriodSummary")
data class HeartRateZonePeriodSummaryDto(
    val period: String,
    val totalTrackedSeconds: Int,
    val easySeconds: Int,
    val hardSeconds: Int,
    val easyHardRatio: Double?,
    val zones: List<HeartRateZoneDistributionDto>,
)

@Schema(description = "Heart rate zone analysis payload", name = "HeartRateZoneAnalysis")
data class HeartRateZoneAnalysisDto(
    val settings: HeartRateZoneSettingsDto,
    val resolvedSettings: ResolvedHeartRateZoneSettingsDto? = null,
    val hasHeartRateData: Boolean,
    val totalTrackedSeconds: Int,
    val easyHardRatio: Double?,
    val zones: List<HeartRateZoneDistributionDto>,
    val activities: List<HeartRateZoneActivitySummaryDto>,
    val byMonth: List<HeartRateZonePeriodSummaryDto>,
    val byYear: List<HeartRateZonePeriodSummaryDto>,
)

fun HeartRateZoneSettings.toDto() = HeartRateZoneSettingsDto(
    maxHr = maxHr,
    thresholdHr = thresholdHr,
    reserveHr = reserveHr,
)

fun HeartRateZoneSettingsDto.toDomain() = HeartRateZoneSettings(
    maxHr = maxHr,
    thresholdHr = thresholdHr,
    reserveHr = reserveHr,
)

fun ResolvedHeartRateZoneSettings.toDto() = ResolvedHeartRateZoneSettingsDto(
    maxHr = maxHr,
    thresholdHr = thresholdHr,
    reserveHr = reserveHr,
    method = method.name,
    source = source.name,
)

fun HeartRateZoneDistribution.toDto() = HeartRateZoneDistributionDto(
    zone = zone,
    label = label,
    seconds = seconds,
    percentage = percentage,
)

fun HeartRateZoneActivitySummary.toDto() = HeartRateZoneActivitySummaryDto(
    activity = activity.toDto(),
    activityDate = activityDate,
    totalTrackedSeconds = totalTrackedSeconds,
    easySeconds = easySeconds,
    hardSeconds = hardSeconds,
    easyHardRatio = easyHardRatio,
    zones = zones.map { distribution -> distribution.toDto() },
)

fun HeartRateZonePeriodSummary.toDto() = HeartRateZonePeriodSummaryDto(
    period = period,
    totalTrackedSeconds = totalTrackedSeconds,
    easySeconds = easySeconds,
    hardSeconds = hardSeconds,
    easyHardRatio = easyHardRatio,
    zones = zones.map { distribution -> distribution.toDto() },
)

fun HeartRateZoneAnalysis.toDto() = HeartRateZoneAnalysisDto(
    settings = settings.toDto(),
    resolvedSettings = resolvedSettings?.toDto(),
    hasHeartRateData = hasHeartRateData,
    totalTrackedSeconds = totalTrackedSeconds,
    easyHardRatio = easyHardRatio,
    zones = zones.map { distribution -> distribution.toDto() },
    activities = activities.map { summary -> summary.toDto() },
    byMonth = byMonth.map { summary -> summary.toDto() },
    byYear = byYear.map { summary -> summary.toDto() },
)
