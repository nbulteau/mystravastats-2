package me.nicolas.stravastats.domain.business.badges

import me.nicolas.stravastats.domain.business.strava.AthleteRef
import me.nicolas.stravastats.domain.business.strava.GeoCoordinate
import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.business.strava.stream.DistanceStream
import me.nicolas.stravastats.domain.business.strava.stream.LatLngStream
import me.nicolas.stravastats.domain.business.strava.stream.Stream
import me.nicolas.stravastats.domain.business.strava.stream.TimeStream
import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Assertions.assertTrue
import org.junit.jupiter.api.Test

class FamousClimbBadgeTest {

    @Test
    fun `check matches Télégraphe when activity starts far away but contains both climb points`() {
        // GIVEN
        val badge = FamousClimbBadge(
            label = "Col du Télégraphe from Saint Michel de Maurienne",
            name = "Col du Télégraphe",
            topOfTheAscent = 1566,
            start = GeoCoordinate(latitude = 45.2178751, longitude = 6.4750846),
            end = GeoCoordinate(latitude = 45.2026999, longitude = 6.4446143),
            length = 11.8,
            totalAscent = 837,
            averageGradient = 7.1,
            difficulty = 628,
            category = "1",
        )
        val activity = buildRideActivity(
            startLatLng = listOf(45.1885, 5.7245), // Grenoble area
            streamPoints = listOf(
                listOf(45.2178751, 6.4750846),
                listOf(45.2026999, 6.4446143),
            ),
        )

        // WHEN
        val (activities, matched) = badge.check(listOf(activity))

        // THEN
        assertTrue(matched, "Télégraphe badge should match when both climb points are present")
        assertEquals(1, activities.size)
    }

    @Test
    fun `check matches Télégraphe with stream point within 500m of summit`() {
        // GIVEN
        val badge = FamousClimbBadge(
            label = "Col du Télégraphe from Saint Michel de Maurienne",
            name = "Col du Télégraphe",
            topOfTheAscent = 1566,
            start = GeoCoordinate(latitude = 45.2178751, longitude = 6.4750846),
            end = GeoCoordinate(latitude = 45.2026999, longitude = 6.4446143),
            length = 11.8,
            totalAscent = 837,
            averageGradient = 7.1,
            difficulty = 628,
            category = "1",
        )
        val activity = buildRideActivity(
            startLatLng = listOf(45.2178751, 6.4750846),
            streamPoints = listOf(
                listOf(45.2178751, 6.4750846),
                listOf(45.2058, 6.4446143), // ~340m from summit
            ),
        )

        // WHEN
        val (_, matched) = badge.check(listOf(activity))

        // THEN
        assertTrue(matched, "Télégraphe badge should match within 500m tolerance")
    }

    @Test
    fun `check does not match Télégraphe descent only`() {
        // GIVEN
        val badge = FamousClimbBadge(
            label = "Col du Télégraphe from Saint Michel de Maurienne",
            name = "Col du Télégraphe",
            topOfTheAscent = 1566,
            start = GeoCoordinate(latitude = 45.2178751, longitude = 6.4750846),
            end = GeoCoordinate(latitude = 45.2026999, longitude = 6.4446143),
            length = 11.8,
            totalAscent = 837,
            averageGradient = 7.1,
            difficulty = 628,
            category = "1",
        )
        val activity = buildRideActivity(
            startLatLng = listOf(45.2026999, 6.4446143),
            streamPoints = listOf(
                listOf(45.2026999, 6.4446143), // summit first
                listOf(45.2178751, 6.4750846), // valley after => descent
            ),
        )

        // WHEN
        val (_, matched) = badge.check(listOf(activity))

        // THEN
        assertTrue(!matched, "Télégraphe descent-only activity should not match")
    }

    private fun buildRideActivity(
        startLatLng: List<Double>,
        streamPoints: List<List<Double>>,
    ): StravaActivity {
        return StravaActivity(
            athlete = AthleteRef(id = 41902),
            averageSpeed = 0.0,
            averageCadence = 0.0,
            averageHeartrate = 0.0,
            maxHeartrate = 0,
            averageWatts = 0,
            commute = false,
            distance = 10000.0,
            deviceWatts = false,
            elapsedTime = 3600,
            elevHigh = 0.0,
            id = 1L,
            kilojoules = 0.0,
            maxSpeed = 0.0F,
            movingTime = 3500,
            name = "Ride test",
            startDate = "2019-08-05T07:00:00Z",
            startDateLocal = "2019-08-05T09:00:00+02:00",
            startLatlng = startLatLng,
            totalElevationGain = 0.0,
            type = "Ride",
            uploadId = 1L,
            weightedAverageWatts = 0,
            stream = Stream(
                distance = DistanceStream(
                    data = listOf(0.0, 1000.0),
                    originalSize = 2,
                    resolution = "high",
                    seriesType = "distance",
                ),
                time = TimeStream(
                    data = listOf(0, 60),
                    originalSize = 2,
                    resolution = "high",
                    seriesType = "time",
                ),
                latlng = LatLngStream(
                    data = streamPoints,
                    originalSize = streamPoints.size,
                    resolution = "high",
                    seriesType = "latlng",
                ),
            ),
        )
    }
}
