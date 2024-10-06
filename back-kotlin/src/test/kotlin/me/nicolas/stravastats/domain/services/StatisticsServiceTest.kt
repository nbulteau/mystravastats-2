package me.nicolas.stravastats.domain.services

import me.nicolas.stravastats.TestHelper
import me.nicolas.stravastats.domain.business.strava.ActivityType
import org.junit.jupiter.api.BeforeEach
import org.junit.jupiter.api.Test
import org.junit.jupiter.api.extension.ExtendWith
import org.springframework.test.context.junit.jupiter.SpringExtension
import kotlin.test.assertEquals

@ExtendWith(SpringExtension::class)
class StatisticsServiceTest {


    private lateinit var statisticsService: IStatisticsService

    @BeforeEach
    fun setUp() {
        val activities = TestHelper.loadActivities()
        val stravaProxy = StravaProxy()

        // use introspection to set the activities
        val field = stravaProxy.javaClass.getDeclaredField("activities")
        field.isAccessible = true
        field.set(stravaProxy, activities)

        statisticsService = StatisticsService(stravaProxy)
    }

    @Test
    fun `compute statistics for Run activity type`() {
        // GIVEN
        val activityType = ActivityType.Run
        val year = 2020

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

        // WHEN
        val result = statisticsService.getStatistics(activityType, year)

        // THEN
        assertEquals(16, result.size)
        assertEquals(8, result.find { statistic -> statistic.name == "Nb activities" }?.value?.toInt())
    }
}
