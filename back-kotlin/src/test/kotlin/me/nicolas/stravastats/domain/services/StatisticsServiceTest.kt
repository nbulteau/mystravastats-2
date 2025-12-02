package me.nicolas.stravastats.domain.services

import io.mockk.every
import io.mockk.mockk
import me.nicolas.stravastats.TestHelper
import me.nicolas.stravastats.domain.business.ActivityType

import me.nicolas.stravastats.domain.services.activityproviders.IActivityProvider
import org.junit.jupiter.api.BeforeEach
import org.junit.jupiter.api.Test
import org.junit.jupiter.api.extension.ExtendWith
import org.springframework.test.context.junit.jupiter.SpringExtension
import kotlin.test.assertEquals

@ExtendWith(SpringExtension::class)
class StatisticsServiceTest {

    private lateinit var statisticsService: IStatisticsService

    private val activityProvider = mockk<IActivityProvider>()

    private val run2020Activities = TestHelper.run2020Activities()

    private val ride2020Activities = TestHelper.ride2020Activities()

    private val hike2020Activities = TestHelper.hike2020Activities()

    @BeforeEach
    fun setUp() {
        statisticsService = StatisticsService(activityProvider)
    }

    @Test
    fun `compute statistics for Run activity type`() {
        // GIVEN
        val activityTypes = setOf(ActivityType.Run)
        val year = 2020

        every { activityProvider.getActivitiesByActivityTypeAndYear(activityTypes, 2020) } returns run2020Activities

        // WHEN
        val result = statisticsService.getStatistics(activityTypes, year)

        // THEN
        assertEquals(31, result.size)
        assertEquals(53, result.find { statistic -> statistic.name == "Nb activities" }?.value?.toInt())
    }

    @Test
    fun `compute statistics for Ride activity type`() {
        // GIVEN
        val activityTypes = setOf(ActivityType.Ride)
        val year = 2020

        every { activityProvider.getActivitiesByActivityTypeAndYear(activityTypes, 2020) } returns ride2020Activities

        // WHEN
        val result = statisticsService.getStatistics(activityTypes, year)

        // THEN
        assertEquals(40, result.size)
        assertEquals(44, result.find { statistic -> statistic.name == "Nb activities" }?.value?.toInt())
    }

    @Test
    fun `compute statistics for Hike activity type`() {
        // GIVEN
        val activityTypes = setOf(ActivityType.Hike)
        val year = 2020

        every { activityProvider.getActivitiesByActivityTypeAndYear(activityTypes, 2020) } returns hike2020Activities

        // WHEN
        val result = statisticsService.getStatistics(activityTypes, year)

        // THEN
        assertEquals(18, result.size)
        assertEquals(8, result.find { statistic -> statistic.name == "Nb activities" }?.value?.toInt())
    }
}
