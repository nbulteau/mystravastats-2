package me.nicolas.stravastats.api.dto

import io.swagger.v3.oas.annotations.media.Schema
import me.nicolas.stravastats.domain.business.AthleteFtpSetting
import me.nicolas.stravastats.domain.business.AthletePerformanceSettings

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
