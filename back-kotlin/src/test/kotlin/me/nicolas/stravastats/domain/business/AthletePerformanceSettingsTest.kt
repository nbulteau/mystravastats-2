package me.nicolas.stravastats.domain.business

import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Assertions.assertNull
import org.junit.jupiter.api.Test

class AthletePerformanceSettingsTest {

    @Test
    fun `normalize keeps positive dated FTP entries sorted and deduplicated`() {
        val settings = AthletePerformanceSettings(
            ftpHistory = listOf(
                AthleteFtpSetting(effectiveFrom = "2026-02-01", ftp = 170),
                AthleteFtpSetting(effectiveFrom = "bad-date", ftp = 999),
                AthleteFtpSetting(effectiveFrom = "2026-01-01", ftp = 160),
                AthleteFtpSetting(effectiveFrom = "2026-02-01", ftp = 175),
                AthleteFtpSetting(effectiveFrom = "2026-03-01", ftp = -1),
            ),
            weightKg = -72.5,
        )

        val normalized = settings.normalize()

        assertNull(normalized.weightKg)
        assertEquals(
            listOf(
                AthleteFtpSetting(effectiveFrom = "2026-01-01", ftp = 160),
                AthleteFtpSetting(effectiveFrom = "2026-02-01", ftp = 175),
            ),
            normalized.ftpHistory,
        )
    }
}
