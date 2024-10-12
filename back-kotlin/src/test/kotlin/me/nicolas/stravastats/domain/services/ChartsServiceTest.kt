package me.nicolas.stravastats.domain.services

import io.mockk.every
import io.mockk.mockk
import me.nicolas.stravastats.TestHelper
import me.nicolas.stravastats.domain.business.Period
import me.nicolas.stravastats.domain.business.strava.ActivityType
import me.nicolas.stravastats.domain.services.activityproviders.IActivityProvider
import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Assertions.assertTrue
import org.junit.jupiter.api.BeforeEach
import org.junit.jupiter.api.extension.ExtendWith
import org.springframework.test.context.junit.jupiter.SpringExtension
import kotlin.test.Test

@ExtendWith(SpringExtension::class)
class ChartsServiceTest {

    private lateinit var chartsStravaService: IChartsService

    private val activityProvider = mockk<IActivityProvider>()

    private val run2020Activities = TestHelper.run2020Activities()

    private val run2023Activities = TestHelper.run2023Activities()


    @BeforeEach
    fun setUp() {


        chartsStravaService  = ChartsService(activityProvider)
    }

    @Test
    fun `get distance by period by activity type by year returns distances when valid activity type, year, and period`() {
        // GIVEN
        val activityType = ActivityType.Run
        val year = 2020
        val period = Period.MONTHS

        every { activityProvider.getActivitiesByActivityTypeAndYear(ActivityType.Run, 2020) } returns run2020Activities

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

        every { activityProvider.getActivitiesByActivityTypeAndYear(ActivityType.Run, 2023) } returns run2023Activities

        // WHEN
        val result = chartsStravaService.getDistanceByPeriodByActivityTypeByYear(activityType, year, period)

        // THEN
        assertTrue(result[0].second == 0.0)
        assertTrue(result[11].second == 0.0)
    }

}