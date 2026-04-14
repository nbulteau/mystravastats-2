package me.nicolas.stravastats.domain.services.activityproviders

import io.mockk.coEvery
import io.mockk.every
import io.mockk.mockk
import io.mockk.verify
import kotlinx.coroutines.runBlocking
import me.nicolas.stravastats.adapters.strava.StravaRateLimitException
import me.nicolas.stravastats.domain.interfaces.ILocalStorageProvider
import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.business.strava.AthleteRef
import me.nicolas.stravastats.domain.business.strava.stream.DistanceStream
import me.nicolas.stravastats.domain.business.strava.stream.Stream
import me.nicolas.stravastats.domain.business.strava.stream.TimeStream
import me.nicolas.stravastats.domain.interfaces.IStravaApi
import org.junit.jupiter.api.Assertions.assertNotNull
import org.junit.jupiter.api.Assertions.assertTrue
import org.junit.jupiter.api.Test

class StravaActivityProviderTest {

    @Test
    fun `loadMissingStreamsFromApi parallelizes and updates activities streams`() = runBlocking {
        // Mocks
        val repository = mockk<ILocalStorageProvider>(relaxed = true)
        val api = mockk<IStravaApi>(relaxed = true)

        // Stub authentication to provide a clientId
        coEvery { repository.readStravaAuthentication(any()) } returns Triple("12345", "secret", false)

        // Prepare provider with mocks
        val provider = StravaActivityProvider(localStorageProvider = repository, stravaApi = api)

        // Prepare activities without stream
        val activities = (1L..3L).map { id ->
            StravaActivity(
                id = id,
                name = "test-$id",
                startDate = "2020-01-01T00:00:00Z",
                athlete = AthleteRef(1),
                averageSpeed = 0.0,
                commute = false,
                distance = 0.0,
                elapsedTime = 0,
                elevHigh = 0.0,
                maxSpeed = 0.0f,
                movingTime = 0,
                startDateLocal = "2020-01-01T00:00:00Z",
                startLatlng = null,
                totalElevationGain = 0.0,
                type = "Run",
                uploadId = 12345L
            )
        }.toMutableList()

        val year = 2020

        // Mock API to return a Stream for each activity
        activities.forEach { activity ->
            every { api.getActivityStreamFailFastOnRateLimit(activity) } returns Stream(
                distance = DistanceStream(data = emptyList(), originalSize = 0, resolution = "", seriesType = ""),
                time = TimeStream(data = emptyList(), originalSize = 0, resolution = "", seriesType = "")
            )
        }

        // Call the suspend function
        provider.loadMissingStreamsFromApi(year, activities)

        // Assertions: each activity should have a stream
        activities.forEach { activity ->
            assertNotNull(activity.stream)
        }

        // Verify api was called for each activity
        verify(exactly = activities.size) { api.getActivityStreamFailFastOnRateLimit(any()) }
    }

    @Test
    fun `getDetailedActivity switches to cache-only mode after rate limit`() {
        val repository = mockk<ILocalStorageProvider>(relaxed = true)
        val api = mockk<IStravaApi>(relaxed = true)

        coEvery { repository.readStravaAuthentication(any()) } returns Triple("12345", "secret", false)
        every { repository.loadDetailedActivityFromCache(any(), any(), any()) } returns null
        every { repository.loadActivitiesStreamsFromCache(any(), any(), any()) } returns null
        every { api.getDetailedActivityFailFastOnRateLimit(any()) } throws StravaRateLimitException("429")

        val provider = StravaActivityProvider(localStorageProvider = repository, stravaApi = api)
        val activity = StravaActivity(
            id = 42L,
            name = "test-42",
            startDate = "2020-01-01T00:00:00Z",
            athlete = AthleteRef(1),
            averageSpeed = 0.0,
            commute = false,
            distance = 1000.0,
            elapsedTime = 300,
            elevHigh = 0.0,
            maxSpeed = 0.0f,
            movingTime = 290,
            startDateLocal = "2020-01-01T00:00:00Z",
            startLatlng = null,
            totalElevationGain = 10.0,
            type = "Ride",
            uploadId = 12345L
        )
        val activitiesField = AbstractActivityProvider::class.java.getDeclaredField("activities")
        activitiesField.isAccessible = true
        activitiesField.set(provider, listOf(activity))

        val firstCall = provider.getDetailedActivity(42L)
        assertTrue(firstCall.isPresent, "Expected cache fallback detailed activity after rate limit")

        val secondCall = provider.getDetailedActivity(42L)
        assertTrue(secondCall.isPresent, "Expected cache-only detailed activity while rate limit is active")

        verify(exactly = 1) { api.getDetailedActivityFailFastOnRateLimit(42L) }
    }
}
