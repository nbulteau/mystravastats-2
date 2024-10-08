package me.nicolas.stravastats.domain.services

import me.nicolas.stravastats.TestHelper
import org.junit.jupiter.api.BeforeEach


class ActivityServiceTest {

    private lateinit var activityService: IActivityService


    @BeforeEach
    fun setUp() {
        val activities = TestHelper.loadActivities()

        // use introspection to set the activities
        val stravaProxy = StravaProxy()

        // use introspection to set the activities
        val field = stravaProxy.javaClass.getDeclaredField("activities")
        field.isAccessible = true
        field.set(stravaProxy, activities)

        activityService = ActivityService(stravaProxy)
    }
}