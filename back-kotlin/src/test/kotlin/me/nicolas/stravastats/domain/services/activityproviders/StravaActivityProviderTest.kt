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
import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Assertions.assertTrue
import org.junit.jupiter.api.Test
import me.nicolas.stravastats.domain.services.toStravaDetailedActivity
import java.util.Optional

class StravaActivityProviderTest {

    @Test
    fun `loadMissingStreamsFromApi parallelizes and updates activities streams`() = runBlocking {
        // GIVEN
        val repository = mockk<ILocalStorageProvider>(relaxed = true)
        val api = mockk<IStravaApi>(relaxed = true)

        coEvery { repository.readStravaAuthentication(any()) } returns Triple("12345", "secret", false)

        val provider = StravaActivityProvider(localStorageProvider = repository, stravaApi = api)

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

        activities.forEach { activity ->
            every { api.getActivityStreamFailFastOnRateLimit(activity) } returns Stream(
                distance = DistanceStream(data = emptyList(), originalSize = 0, resolution = "", seriesType = ""),
                time = TimeStream(data = emptyList(), originalSize = 0, resolution = "", seriesType = "")
            )
        }

        // WHEN
        provider.loadMissingStreamsFromApi(year, activities)

        // THEN
        activities.forEach { activity ->
            assertNotNull(activity.stream)
        }
        verify(exactly = activities.size) { api.getActivityStreamFailFastOnRateLimit(any()) }
    }

    @Test
    fun `getDetailedActivity switches to cache-only mode after rate limit`() {
        // GIVEN
        val repository = mockk<ILocalStorageProvider>(relaxed = true)
        val api = mockk<IStravaApi>(relaxed = true)

        coEvery { repository.readStravaAuthentication(any()) } returns Triple("12345", "secret", false)
        var detailedCache: me.nicolas.stravastats.domain.business.strava.StravaDetailedActivity? = null
        every { repository.loadDetailedActivityFromCache(any(), any(), any()) } answers { detailedCache }
        every { repository.saveDetailedActivityToCache(any(), any(), any()) } answers {
            detailedCache = thirdArg()
            Unit
        }
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

        // WHEN
        val firstCall = provider.getDetailedActivity(42L)

        // THEN
        assertTrue(firstCall.isPresent, "Expected cache fallback detailed activity after rate limit")

        // WHEN - second call while rate limit is active
        val secondCall = provider.getDetailedActivity(42L)

        // THEN
        assertTrue(secondCall.isPresent, "Expected cache-only detailed activity while rate limit is active")
        verify(exactly = 1) { api.getDetailedActivityFailFastOnRateLimit(42L) }
        verify(exactly = 1) { repository.saveDetailedActivityToCache(any(), 2020, any()) }
    }

    @Test
    fun `getDetailedActivity returns cached detailed activity when base activity is missing`() {
        // GIVEN
        val repository = mockk<ILocalStorageProvider>(relaxed = true)
        val api = mockk<IStravaApi>(relaxed = true)
        coEvery { repository.readStravaAuthentication(any()) } returns Triple("12345", "secret", false)

        val cachedActivityId = 77L
        val cachedDetailed = StravaActivity(
            id = cachedActivityId,
            name = "cached-$cachedActivityId",
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
        ).toStravaDetailedActivity()

        every { repository.loadDetailedActivityFromCache(any(), any(), cachedActivityId) } returns cachedDetailed

        val provider = StravaActivityProvider(localStorageProvider = repository, stravaApi = api)

        // WHEN
        val detailed = provider.getDetailedActivity(cachedActivityId)

        // THEN
        assertTrue(detailed.isPresent, "Expected cached detailed activity")
        assertEquals(cachedActivityId, detailed.get().id)
        verify(exactly = 0) { api.getDetailedActivityFailFastOnRateLimit(any()) }
    }

    @Test
    fun `getDetailedActivity fetches and persists detailed activity when base activity is missing`() {
        // GIVEN
        val repository = mockk<ILocalStorageProvider>(relaxed = true)
        val api = mockk<IStravaApi>(relaxed = true)
        coEvery { repository.readStravaAuthentication(any()) } returns Triple("12345", "secret", false)

        var detailedCache: me.nicolas.stravastats.domain.business.strava.StravaDetailedActivity? = null
        every { repository.loadDetailedActivityFromCache(any(), any(), any()) } answers { detailedCache }
        every { repository.saveDetailedActivityToCache(any(), any(), any()) } answers {
            detailedCache = thirdArg()
            Unit
        }

        val activityId = 901L
        val fromApi = StravaActivity(
            id = activityId,
            name = "api-detailed-$activityId",
            startDate = "2022-01-01T00:00:00Z",
            athlete = AthleteRef(1),
            averageSpeed = 0.0,
            commute = false,
            distance = 1000.0,
            elapsedTime = 300,
            elevHigh = 0.0,
            maxSpeed = 0.0f,
            movingTime = 290,
            startDateLocal = "2022-01-01T00:00:00Z",
            startLatlng = null,
            totalElevationGain = 10.0,
            type = "Ride",
            uploadId = 12345L
        ).toStravaDetailedActivity()
        every { api.getDetailedActivityFailFastOnRateLimit(activityId) } returns Optional.of(fromApi)

        val provider = StravaActivityProvider(localStorageProvider = repository, stravaApi = api)

        // WHEN
        val firstCall = provider.getDetailedActivity(activityId)
        val secondCall = provider.getDetailedActivity(activityId)

        // THEN
        assertTrue(firstCall.isPresent, "Expected detailed activity fetched from Strava API")
        assertTrue(secondCall.isPresent, "Expected detailed activity loaded from cache on second call")
        assertEquals(activityId, secondCall.get().id)
        verify(exactly = 1) { api.getDetailedActivityFailFastOnRateLimit(activityId) }
        verify(exactly = 1) { repository.saveDetailedActivityToCache(any(), 2022, any()) }
    }
}
