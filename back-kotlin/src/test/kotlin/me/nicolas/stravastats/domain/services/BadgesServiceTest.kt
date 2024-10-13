package me.nicolas.stravastats.domain.services

import io.mockk.every
import io.mockk.mockk
import me.nicolas.stravastats.TestHelper
import me.nicolas.stravastats.domain.business.ActivityType

import me.nicolas.stravastats.domain.services.activityproviders.IActivityProvider
import org.junit.jupiter.api.BeforeEach
import org.junit.jupiter.api.Test

class BadgesServiceTest {
    private lateinit var badgesService: IBadgesService

    private val activityProvider = mockk<IActivityProvider>()

    @BeforeEach
    fun setUp() {
        val activities = TestHelper.loadActivities()

        every { activityProvider.getActivitiesByActivityTypeAndYear(ActivityType.Ride, 2021) } returns activities

        badgesService = BadgesService(activityProvider)
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
