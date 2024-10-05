package me.nicolas.stravastats.domain.services

import me.nicolas.stravastats.TestHelper
import me.nicolas.stravastats.domain.business.strava.ActivityType
import org.junit.jupiter.api.BeforeEach
import org.junit.jupiter.api.Test

class BadgesServiceTest {
    private lateinit var badgesService: IBadgesService


    @BeforeEach
    fun setUp() {
        val activities = TestHelper.loadActivities()

        // use introspection to set the activities
        val stravaProxy = StravaProxy()

        // use introspection to set the activities
        val field = stravaProxy.javaClass.getDeclaredField("activities")
        field.isAccessible = true
        field.set(stravaProxy, activities)

        badgesService = BadgesService(stravaProxy)
    }

    @Test
    fun `getGeneralBadges should return the right badges for a ride`() {
        // GIVEN

        // WHEN
        val badges = badgesService.getGeneralBadges(ActivityType.Ride, 2021)

        // THEN
        // Check the badges
        // ...
    }
}
