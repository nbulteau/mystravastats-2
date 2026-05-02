package me.nicolas.stravastats.domain.services.activityproviders

import io.mockk.every
import io.mockk.mockk
import io.mockk.verify
import me.nicolas.stravastats.domain.business.strava.AthleteRef
import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.business.strava.stream.DistanceStream
import me.nicolas.stravastats.domain.business.strava.stream.LatLngStream
import me.nicolas.stravastats.domain.business.strava.stream.Stream
import me.nicolas.stravastats.domain.business.strava.stream.TimeStream
import me.nicolas.stravastats.domain.interfaces.ISRTMProvider
import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Assertions.assertNotNull
import org.junit.jupiter.api.Test

class ActivityProviderUtilsTest {

    @Test
    fun `processAltitudeStreamIfMissing enriches activities without altitude`() {
        val srtmProvider = mockk<ISRTMProvider>()
        every { srtmProvider.getElevation(any()) } returns listOf(100.0, 110.0)

        val activity = baseActivity(
            stream = Stream(
                distance = DistanceStream(data = listOf(0.0, 1000.0), originalSize = 2, resolution = "high", seriesType = "distance"),
                time = TimeStream(data = listOf(0, 300), originalSize = 2, resolution = "high", seriesType = "distance"),
                latlng = LatLngStream(
                    data = listOf(listOf(48.0, -1.0), listOf(48.1, -1.1)),
                    originalSize = 2,
                    resolution = "high",
                    seriesType = "distance",
                ),
                altitude = null,
            ),
        )

        val enriched = listOf(activity).processAltitudeStreamIfMissing(srtmProvider)

        assertNotNull(enriched.first().stream?.altitude)
        assertEquals(listOf(100.0, 110.0), enriched.first().stream?.altitude?.data)
        verify(exactly = 1) { srtmProvider.getElevation(any()) }
    }

    @Test
    fun `resolveYearFromDateString falls back when prefix is invalid`() {
        val year = resolveYearFromDateString("xx24-01-01", fallback = 2022)
        assertEquals(2022, year)
    }

    private fun baseActivity(stream: Stream?): StravaActivity {
        return StravaActivity(
            athlete = AthleteRef(1),
            averageSpeed = 10.0,
            commute = false,
            distance = 1000.0,
            elapsedTime = 300,
            id = 1L,
            maxSpeed = 12.0f,
            movingTime = 290,
            name = "test",
            startDate = "2024-01-01T00:00:00Z",
            startDateLocal = "2024-01-01T00:00:00Z",
            startLatlng = listOf(48.0, -1.0),
            totalElevationGain = 0.0,
            type = "Ride",
            uploadId = 1L,
            stream = stream,
        )
    }
}
