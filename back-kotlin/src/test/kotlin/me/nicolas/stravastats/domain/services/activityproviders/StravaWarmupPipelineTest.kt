package me.nicolas.stravastats.domain.services.activityproviders

import me.nicolas.stravastats.domain.services.cache.WarmupYearSummary
import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Test

class StravaWarmupPipelineTest {
    @Test
    fun `prepared years exclude all-years bucket`() {
        val summaries = listOf(
            WarmupYearSummary(year = 0, activityCount = 3, totalDistanceKm = 42.0, totalElevationM = 300.0, elapsedSeconds = 3600),
            WarmupYearSummary(year = 2024, activityCount = 1, totalDistanceKm = 12.0, totalElevationM = 100.0, elapsedSeconds = 1200),
            WarmupYearSummary(year = 2026, activityCount = 2, totalDistanceKm = 30.0, totalElevationM = 200.0, elapsedSeconds = 2400),
            WarmupYearSummary(year = 2024, activityCount = 1, totalDistanceKm = 12.0, totalElevationM = 100.0, elapsedSeconds = 1200),
        )

        assertEquals(listOf(2026, 2024), extractPreparedYears(summaries))
    }

    @Test
    fun `prepared years normalize old manifests`() {
        assertEquals(listOf(2026, 2024), normalizePreparedYears(listOf(0, 2024, 2026, 2024)))
    }
}
