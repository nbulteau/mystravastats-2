package me.nicolas.stravastats.domain.services

import me.nicolas.stravastats.TestHelper
import me.nicolas.stravastats.domain.services.activityproviders.StravaActivityProvider
import org.junit.jupiter.api.BeforeEach


class ActivityServiceTest {

    private lateinit var activityService: IActivityService


    @BeforeEach
    fun setUp() {
        val activities = TestHelper.loadActivities()

        // use introspection to set the activities
        val stravaActivityProvider = StravaActivityProvider()

        // use introspection to set the activities
        val field = stravaActivityProvider.javaClass.getDeclaredField("activities")
        field.isAccessible = true
        field.set(stravaActivityProvider, activities)

        activityService = ActivityService(stravaActivityProvider)
    }
}