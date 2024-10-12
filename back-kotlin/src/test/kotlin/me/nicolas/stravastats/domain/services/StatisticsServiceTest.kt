package me.nicolas.stravastats.domain.services

import io.mockk.every
import io.mockk.mockk
import me.nicolas.stravastats.TestHelper
import me.nicolas.stravastats.domain.business.strava.ActivityType
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
        val activityType = ActivityType.Run
        val year = 2020

        every { activityProvider.getActivitiesByActivityTypeAndYear(ActivityType.Run, 2020) } returns run2020Activities

        // WHEN
        val result = statisticsService.getStatistics(activityType, year)

        // THEN
        assertEquals(29, result.size)
        assertEquals(53, result.find { statistic -> statistic.name == "Nb activities" }?.value?.toInt())
    }

    @Test
    fun `compute statistics for Ride activity type`() {
        // GIVEN
        val activityType = ActivityType.Ride
        val year = 2020

        every { activityProvider.getActivitiesByActivityTypeAndYear(ActivityType.Ride, 2020) } returns ride2020Activities

        // WHEN
        val result = statisticsService.getStatistics(activityType, year)

        // THEN
        assertEquals(36, result.size)
        assertEquals(44, result.find { statistic -> statistic.name == "Nb activities" }?.value?.toInt())
    }

    @Test
    fun `compute statistics for Hike activity type`() {
        // GIVEN
        val activityType = ActivityType.Hike
        val year = 2020

        every { activityProvider.getActivitiesByActivityTypeAndYear(ActivityType.Hike, 2020) } returns hike2020Activities

        // WHEN
        val result = statisticsService.getStatistics(activityType, year)

        // THEN
        assertEquals(16, result.size)
        assertEquals(8, result.find { statistic -> statistic.name == "Nb activities" }?.value?.toInt())
    }
}
