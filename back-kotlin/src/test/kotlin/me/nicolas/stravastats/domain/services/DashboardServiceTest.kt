package me.nicolas.stravastats.domain.services

import io.mockk.every
import io.mockk.mockk
import me.nicolas.stravastats.domain.business.ActivityType
import me.nicolas.stravastats.domain.services.activityproviders.IActivityProvider
import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.BeforeEach
import org.junit.jupiter.api.Test

class DashboardServiceTest {

    private lateinit var dashboardService: IDashboardService

    private val activityProvider = mockk<IActivityProvider>()

    @BeforeEach
    fun setUp() {
        dashboardService = DashboardService(activityProvider)
    }

    @Test
    fun `getEddingtonNumber returns zero when no active day is available`() {
        // GIVEN
        every {
            activityProvider.getActivitiesByActivityTypeGroupByActiveDays(setOf(ActivityType.Ride))
        } returns emptyMap()

        // WHEN
        val result = dashboardService.getEddingtonNumber(setOf(ActivityType.Ride))

        // THEN
        assertEquals(0, result.eddingtonNumber)
        assertEquals(emptyList<Int>(), result.eddingtonList)
    }

    @Test
    fun `getEddingtonNumber does not round up when equality threshold is not met`() {
        // GIVEN
        val dailyTotals = (1..49).associate { day -> "2024-01-${day.toString().padStart(2, '0')}" to 51 }
        every {
            activityProvider.getActivitiesByActivityTypeGroupByActiveDays(setOf(ActivityType.Ride))
        } returns dailyTotals

        // WHEN
        val result = dashboardService.getEddingtonNumber(setOf(ActivityType.Ride))

        // THEN
        assertEquals(49, result.eddingtonNumber)
        assertEquals(51, result.eddingtonList.size)
        assertEquals(49, result.eddingtonList[48]) // >= 49km
        assertEquals(49, result.eddingtonList[49]) // >= 50km
    }

    @Test
    fun `getEddingtonNumber ignores non positive daily totals`() {
        // GIVEN
        every {
            activityProvider.getActivitiesByActivityTypeGroupByActiveDays(setOf(ActivityType.Ride))
        } returns mapOf(
            "2025-01-01" to 4,
            "2025-01-02" to 4,
            "2025-01-03" to 4,
            "2025-01-04" to 4,
            "2025-01-05" to 0,
            "2025-01-06" to -2,
        )

        // WHEN
        val result = dashboardService.getEddingtonNumber(setOf(ActivityType.Ride))

        // THEN
        assertEquals(4, result.eddingtonNumber)
        assertEquals(listOf(4, 4, 4, 4), result.eddingtonList)
    }
}

