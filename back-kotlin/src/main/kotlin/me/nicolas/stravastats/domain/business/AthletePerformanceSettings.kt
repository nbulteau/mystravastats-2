package me.nicolas.stravastats.domain.business

import java.time.LocalDate

data class AthleteFtpSetting(
    val effectiveFrom: String,
    val ftp: Int,
)

data class AthletePerformanceSettings(
    val ftpHistory: List<AthleteFtpSetting> = emptyList(),
    val weightKg: Double? = null,
)

fun AthletePerformanceSettings.normalize(): AthletePerformanceSettings {
    val normalizedHistory = ftpHistory
        .filter { setting -> setting.ftp > 0 && setting.effectiveFrom.isIsoLocalDate() }
        .associateBy { setting -> setting.effectiveFrom }
        .values
        .sortedBy { setting -> setting.effectiveFrom }

    return AthletePerformanceSettings(
        ftpHistory = normalizedHistory,
        weightKg = weightKg?.takeIf { weight -> weight > 0.0 },
    )
}

private fun String.isIsoLocalDate(): Boolean =
    runCatching { LocalDate.parse(this) }.isSuccess
