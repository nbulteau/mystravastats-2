package me.nicolas.stravastats.domain.services

import me.nicolas.stravastats.TestHelper
import me.nicolas.stravastats.domain.interfaces.ILocalStorageProvider
import me.nicolas.stravastats.domain.interfaces.IStravaApi
import me.nicolas.stravastats.domain.services.activityproviders.StravaActivityProvider
import io.mockk.every
import io.mockk.mockk
import org.junit.jupiter.api.BeforeEach


class StravaActivityServiceTest {

    private lateinit var activityService: IActivityService


    @BeforeEach
    fun setUp() {
        // GIVEN
        val activities = TestHelper.loadActivities()

        val localStorage = mockk<ILocalStorageProvider>(relaxed = true)
        every { localStorage.readStravaAuthentication(any()) } returns Triple("12345", "secret", true)
        val stravaApi = mockk<IStravaApi>(relaxed = true)

        // use introspection to set the activities
        val stravaActivityProvider = StravaActivityProvider(
            storageProvider = localStorage,
            stravaApiFactory = { _, _ -> stravaApi },
            stravaApi = stravaApi,
        )

        // use introspection to set the activities
        val field = stravaActivityProvider.javaClass.getDeclaredField("activities")
        field.isAccessible = true
        field.set(stravaActivityProvider, activities)

        // WHEN
        activityService = ActivityService(stravaActivityProvider)
        // THEN
        // service is initialized for downstream tests in this class
    }
}
