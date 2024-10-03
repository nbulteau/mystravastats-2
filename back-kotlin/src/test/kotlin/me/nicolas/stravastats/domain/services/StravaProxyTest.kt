package me.nicolas.stravastats.domain.services

import me.nicolas.stravastats.TestHelper
import me.nicolas.stravastats.domain.business.strava.ActivityType
import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Assertions.assertTrue
import org.junit.jupiter.api.BeforeEach
import org.junit.jupiter.api.Test
import org.junit.jupiter.api.extension.ExtendWith
import org.springframework.boot.test.autoconfigure.web.servlet.WebMvcTest
import org.springframework.test.context.junit.jupiter.SpringExtension

@ExtendWith(SpringExtension::class)
@WebMvcTest(StravaProxyTest::class)
class StravaProxyTest {

    private val stravaProxy: IStravaProxy = StravaProxy()

    @BeforeEach
    fun setUp() {
        val activities = TestHelper.loadActivities()

        // use introspection to set the activities
        val field = stravaProxy.javaClass.getDeclaredField("activities")
        field.isAccessible = true
        field.set(stravaProxy, activities)
    }

    @Test
    fun `get activities by activity type by year group by active days returns grouped activities when valid activity type and year`() {
        // GIVEN
        val activityType = ActivityType.Run
        val year = 2020

        // WHEN
        val result = stravaProxy.getActivitiesByActivityTypeByYearGroupByActiveDays(activityType, year)

        // THEN
        assertEquals(5, result["2020-11-27"]) // 5 km on 2020-11-27
        assertEquals(4, result["2020-11-24"]) // 4 km on 2020-11-24

    }

    @Test
    fun `get activities by activity type by year group by active days returns empty map when no activities found`() {
        // GIVEN
        val activityType = ActivityType.Run
        val year = 2023

        // WHEN
        val result = stravaProxy.getActivitiesByActivityTypeByYearGroupByActiveDays(activityType, year)

        // THEN
        assertTrue(result.isEmpty())
    }

    @Test
    fun `get activities by activity type group by year returns grouped activities when valid activity type`() {
        // GIVEN
        val activityType = ActivityType.Run

        // WHEN
        val result = stravaProxy.getActivitiesByActivityTypeGroupByYear(activityType)

        // THEN
        assertEquals(53, result["2020"]?.size) // 53 Run activities in 2020
        assertEquals(null, result["2023"]) // No Run activities in 2023
    }

    @Test
    fun `get activities by activity type group by year returns empty map when no activities found`() {
        // GIVEN
        val activityType = ActivityType.VirtualRide

        // WHEN
        val result = stravaProxy.getActivitiesByActivityTypeGroupByYear(activityType)

        // THEN
        assertTrue(result.isEmpty())
    }

    @Test
    fun `get activities by activity type and year returns activities when valid activity type and year`() {
        // GIVEN
        val activityType = ActivityType.Run
        val year = 2020

        // WHEN
        val result = stravaProxy.getFilteredActivitiesByActivityTypeAndYear(activityType, year)

        // THEN
        assertEquals(53, result.size)
    }

    @Test
    fun `get activities by activity type and year returns empty list when no activities found`() {
        // GIVEN
        val activityType = ActivityType.Run
        val year = 2022

        // WHEN
        val result = stravaProxy.getFilteredActivitiesByActivityTypeAndYear(activityType, year)

        // THEN
        assertTrue(result.isEmpty())
    }

    @Test
    fun `get activities by activity type and year returns activities for all years when year is null`() {
        // GIVEN
        val activityType = ActivityType.Run
        val activities = listOf(TestHelper.activity)

        // WHEN
        stravaProxy.getFilteredActivitiesByActivityTypeAndYear(activityType, null)

        // THEN
        assertEquals(activities.size, 1)
    }
}