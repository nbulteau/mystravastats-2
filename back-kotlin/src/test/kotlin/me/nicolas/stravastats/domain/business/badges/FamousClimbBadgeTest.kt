package me.nicolas.stravastats.domain.business.badges

import me.nicolas.stravastats.domain.business.strava.AthleteRef
import me.nicolas.stravastats.domain.business.strava.GeoCoordinate
import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.business.strava.stream.DistanceStream
import me.nicolas.stravastats.domain.business.strava.stream.LatLngStream
import me.nicolas.stravastats.domain.business.strava.stream.Stream
import me.nicolas.stravastats.domain.business.strava.stream.TimeStream
import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Assertions.assertFalse
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

    @Test
    fun `check rejects a full ride detour between climb waypoints`() {
        val badge = FamousClimbBadge(
            label = "La Hourquette d'Ancizan from Payolle",
            name = "La Hourquette d'Ancizan",
            topOfTheAscent = 1564,
            start = GeoCoordinate(latitude = 42.943493, longitude = 0.278038),
            end = GeoCoordinate(latitude = 42.899975, longitude = 0.305761),
            length = 10.0,
            totalAscent = 535,
            averageGradient = 5.9,
            difficulty = 348,
            category = "2",
        )
        val activity = buildRideActivity(
            startLatLng = listOf(42.943493, 0.278038),
            streamPoints = listOf(
                listOf(42.943493, 0.278038),
                listOf(42.899975, 0.305761),
            ),
            streamDistances = listOf(0.0, 52700.0),
        )

        val (_, matched) = badge.check(listOf(activity))

        assertTrue(!matched, "A 52.7 km detour must not match the 10 km Payolle ascent")
    }

    @Test
    fun `check requires the route checkpoint of the selected variant`() {
        val badge = FamousClimbBadge(
            label = "Col de la Madeleine from La Chambre, par la D213",
            name = "Col de la Madeleine",
            topOfTheAscent = 1993,
            start = GeoCoordinate(latitude = 45.3597, longitude = 6.29929),
            end = GeoCoordinate(latitude = 45.4352186, longitude = 6.3756008),
            routeCheckpoints = listOf(GeoCoordinate(latitude = 45.386825, longitude = 6.331231)),
            length = 19.6,
            totalAscent = 1520,
            averageGradient = 8.0,
            difficulty = 1305,
            category = "HC",
        )
        val activity = buildRideActivity(
            startLatLng = listOf(45.3597, 6.29929),
            streamPoints = listOf(
                listOf(45.3597, 6.29929),
                listOf(45.391775, 6.319134), // Montgellafrey, not the D213 checkpoint.
                listOf(45.4352186, 6.3756008),
            ),
            streamDistances = listOf(0.0, 10000.0, 19600.0),
        )

        val (_, matched) = badge.check(listOf(activity))

        assertFalse(matched, "The Montgellafrey route must not match the D213 variant")
    }

    @Test
    fun `badge set assigns one activity to only one variant of the same climb`() {
        val start = GeoCoordinate(latitude = 45.3597, longitude = 6.29929)
        val end = GeoCoordinate(latitude = 45.4352186, longitude = 6.3756008)
        val activity = buildRideActivity(
            startLatLng = listOf(start.latitude, start.longitude),
            streamPoints = listOf(
                listOf(start.latitude, start.longitude),
                listOf(end.latitude, end.longitude),
            ),
            streamDistances = listOf(0.0, 19700.0),
        )
        val commonBadgeValues = listOf(
            FamousClimbBadge(
                label = "D213",
                name = "Col de la Madeleine",
                topOfTheAscent = 1993,
                start = start,
                end = end,
                length = 19.6,
                totalAscent = 1520,
                averageGradient = 8.0,
                difficulty = 1305,
                category = "HC",
            ),
            FamousClimbBadge(
                label = "Montgellafrey",
                name = "Col de la Madeleine",
                topOfTheAscent = 1993,
                start = start,
                end = end,
                length = 19.8,
                totalAscent = 1520,
                averageGradient = 7.68,
                difficulty = 1168,
                category = "HC",
            ),
        )

        val results = BadgeSet("france", commonBadgeValues).check(listOf(activity))

        assertEquals(1, results.count { it.isCompleted })
        assertEquals(1, results.sumOf { it.activities.size })
    }

    private fun buildRideActivity(
        startLatLng: List<Double>,
        streamPoints: List<List<Double>>,
        streamDistances: List<Double> = listOf(0.0, 11800.0),
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
                    data = streamDistances,
                    originalSize = streamDistances.size,
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
