package me.nicolas.stravastats.domain.services

import me.nicolas.stravastats.TestHelper
import me.nicolas.stravastats.domain.business.Period
import me.nicolas.stravastats.domain.business.strava.ActivityType
import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Assertions.assertTrue
import org.junit.jupiter.api.BeforeEach
import org.junit.jupiter.api.extension.ExtendWith
import org.springframework.test.context.junit.jupiter.SpringExtension
import kotlin.test.Test

@ExtendWith(SpringExtension::class)
class ChartsServiceTest {

    private lateinit var chartsStravaService: IChartsService

    @BeforeEach
    fun setUp() {
        val activities = TestHelper.loadActivities()
        val stravaProxy = StravaProxy()

        // use introspection to set the activities
        val field = stravaProxy.javaClass.getDeclaredField("activities")
        field.isAccessible = true
        field.set(stravaProxy, activities)

        chartsStravaService  = ChartsService(stravaProxy)
    }

    @Test
    fun `get distance by period by activity type by year returns distances when valid activity type, year, and period`() {
        // GIVEN
        val activityType = ActivityType.Run
        val year = 2020
        val period = Period.MONTHS

        // WHEN
        val result = chartsStravaService.getDistanceByPeriodByActivityTypeByYear(activityType, year, period)

        // THEN
        val delta = 0.01
        assertEquals(84.54, result[0].second, delta)
        assertEquals(10.32, result[2].second, delta)
    }

    @Test
    fun `get distance by period by activity type by year returns empty list when no activities found`() {
        // GIVEN
        val activityType = ActivityType.Run
        val year = 2023
        val period = Period.MONTHS

        // WHEN
        val result = chartsStravaService.getDistanceByPeriodByActivityTypeByYear(activityType, year, period)

        // THEN
        assertTrue(result[0].second == 0.0)
        assertTrue(result[11].second == 0.0)
    }

}