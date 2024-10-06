package me.nicolas.stravastats.domain.services

import me.nicolas.stravastats.TestHelper
import me.nicolas.stravastats.domain.business.strava.Activity
import org.junit.jupiter.api.Assertions
import org.junit.jupiter.api.Test
import kotlin.test.assertNotNull


internal class ActivityHelperTest {

    @Test
    fun `groupActivitiesByYear with empty list returns empty map`() {
        // GIVEN
        val activities = emptyList<Activity>()

        // WHEN
        val result = ActivityHelper.groupActivitiesByYear(activities)

        // THEN
        Assertions.assertEquals(0, result.size)
    }

    @Test
    fun `groupActivitiesByYear with activities returns correct map`() {
        // GIVEN
        val activities = TestHelper.loadActivities()

        // WHEN
        val result = ActivityHelper.groupActivitiesByYear(activities)

        // THEN
        Assertions.assertEquals(2, result.size)
    }

    @Test
    fun `groupActivitiesByMonth with empty list returns map with all months`() {
        // GIVEN
        val activities = emptyList<Activity>()

        // WHEN
        val result = ActivityHelper.groupActivitiesByMonth(activities)

        // THEN
        Assertions.assertEquals(12, result.size)
    }

    @Test
    fun `groupActivitiesByMonth with activities returns map with all months`() {
        // GIVEN
        val activities = TestHelper.loadActivities()

        // WHEN
        val result = ActivityHelper.groupActivitiesByMonth(activities)

        // THEN
        Assertions.assertEquals(12, result.size)
    }

    @Test
    fun `groupActivitiesByDay with empty list returns map with all days`() {
        // GIVEN
        val activities = emptyList<Activity>()

        // WHEN
        val result = ActivityHelper.groupActivitiesByDay(activities, 2021)

        // THEN
        Assertions.assertEquals(365, result.size)
    }

    @Test
    fun `groupActivitiesByDay with activities returns correct map`() {
        // GIVEN
        val activities = TestHelper.loadActivities()

        // WHEN
        val result = ActivityHelper.groupActivitiesByDay(activities, 2021)

        // THEN
        Assertions.assertEquals(365, result.size)
    }

    @Test
    fun `groupActivitiesByWeek with empty list returns empty map`() {
        // GIVEN
        val activities = emptyList<Activity>()

        // WHEN
        val result = ActivityHelper.groupActivitiesByWeek(activities)

        // THEN
        Assertions.assertEquals(52, result.size)
    }

    @Test
    fun `groupActivitiesByWeek with activities returns correct map`() {
        // GIVEN
        val activities = TestHelper.loadActivities()

        // WHEN
        val result = ActivityHelper.groupActivitiesByWeek(activities)

        // THEN
        Assertions.assertEquals(54, result.size) // 54 because the first week of the year is not complete
    }

    @Test
    fun `cumulativeDistance with empty activities returns empty map`() {
        // GIVEN
        val activities = emptyMap<String, List<Activity>>()

        // WHEN
        val result = ActivityHelper.cumulativeDistance(activities)

        // THEN
        Assertions.assertEquals(emptyMap<String, Double>(), result)
    }

    @Test
    fun `cumulativeDistance with activities returns correct map`() {
        // GIVEN
        val activities = TestHelper.loadActivities()
        val activitiesByYear = ActivityHelper.groupActivitiesByYear(activities)

        // WHEN
        val result = ActivityHelper.cumulativeDistance(activitiesByYear)

        // THEN
        Assertions.assertEquals(2, result.size)
        assertNotNull(result["2020"])
        assertNotNull(result["2021"])
        val delta = 0.1
        Assertions.assertEquals(2574.8, result["2020"]!!, delta)
        Assertions.assertEquals(5191.7, result["2021"]!!, delta)
    }
}