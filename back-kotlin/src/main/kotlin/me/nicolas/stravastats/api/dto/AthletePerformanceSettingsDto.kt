package me.nicolas.stravastats.api.dto

import io.swagger.v3.oas.annotations.media.Schema
import me.nicolas.stravastats.domain.business.AthleteFtpSetting
import me.nicolas.stravastats.domain.business.AthletePerformanceSettings
import me.nicolas.stravastats.domain.business.FtpEstimate

@Schema(description = "Manual FTP entry", name = "AthleteFtpSetting")
data class AthleteFtpSettingDto(
    val effectiveFrom: String,
    val ftp: Int,
)

@Schema(description = "Athlete performance settings", name = "AthletePerformanceSettings")
data class AthletePerformanceSettingsDto(
    val ftpHistory: List<AthleteFtpSettingDto> = emptyList(),
    val weightKg: Double? = null,
)

fun AthletePerformanceSettings.toDto() = AthletePerformanceSettingsDto(
    ftpHistory = ftpHistory.map { setting ->
        AthleteFtpSettingDto(
            effectiveFrom = setting.effectiveFrom,
            ftp = setting.ftp,
        )
    },
    weightKg = weightKg,
)

fun AthletePerformanceSettingsDto.toDomain() = AthletePerformanceSettings(
    ftpHistory = ftpHistory.map { setting ->
        AthleteFtpSetting(
            effectiveFrom = setting.effectiveFrom,
            ftp = setting.ftp,
        )
    },
    weightKg = weightKg,
)

@Schema(description = "FTP estimate derived from recorded power data", name = "FtpEstimate")
data class FtpEstimateDto(
    val available: Boolean = false,
    val ftp: Int = 0,
    val method: String = "",
    val methodLabel: String = "",
    val bestPower: Int = 0,
    val multiplier: Double = 0.0,
    val basedOnSeconds: Int = 0,
    val confidence: String = "unavailable",
    val source: String = "",
    val sourceKind: String = "none",
    val activityId: Long = 0,
    val activityName: String = "",
    val activityType: String = "",
    val activityDate: String = "",
    val windowDays: Int = 180,
    val activityCount: Int = 0,
)

fun FtpEstimate.toDto() = FtpEstimateDto(
    available = available,
    ftp = ftp,
    method = method,
    methodLabel = methodLabel,
    bestPower = bestPower,
    multiplier = multiplier,
    basedOnSeconds = basedOnSeconds,
    confidence = confidence,
    source = source,
    sourceKind = sourceKind,
    activityId = activityId,
    activityName = activityName,
    activityType = activityType,
    activityDate = activityDate,
    windowDays = windowDays,
    activityCount = activityCount,
)
