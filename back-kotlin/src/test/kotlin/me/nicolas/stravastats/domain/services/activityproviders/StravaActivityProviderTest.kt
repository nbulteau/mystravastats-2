package me.nicolas.stravastats.domain.services.activityproviders

import io.mockk.coEvery
import io.mockk.coVerify
import io.mockk.mockk
import kotlinx.coroutines.runBlocking
import me.nicolas.stravastats.domain.interfaces.ILocalStorageProvider
import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.business.strava.AthleteRef
import me.nicolas.stravastats.domain.business.strava.stream.DistanceStream
import me.nicolas.stravastats.domain.business.strava.stream.Stream
import me.nicolas.stravastats.domain.business.strava.stream.TimeStream
import me.nicolas.stravastats.domain.interfaces.IStravaApi
import org.junit.jupiter.api.Assertions.assertNotNull
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
            coEvery { api.getActivityStream(activity) } returns Stream(
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
        coVerify(exactly = activities.size) { api.getActivityStream(any()) }
    }
}
