package me.nicolas.stravastats.domain.services.routing

import me.nicolas.stravastats.domain.business.strava.AthleteRef
import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.business.strava.stream.DistanceStream
import me.nicolas.stravastats.domain.business.strava.stream.LatLngStream
import me.nicolas.stravastats.domain.business.strava.stream.Stream
import me.nicolas.stravastats.domain.business.strava.stream.TimeStream
import org.junit.jupiter.api.Test
import java.time.Instant
import java.util.Locale
import kotlin.test.assertEquals
import kotlin.test.assertNotNull
import kotlin.test.assertNull
import kotlin.test.assertTrue

class RouteHistoryProfileBuilderTest {

    @Test
    fun `build routing history profile filters by route type`() {
        // GIVEN
        val now = Instant.parse("2026-04-20T10:00:00Z")
        val activities = listOf(
            buildHistoryActivity(
                id = 1L,
                startDate = "2026-04-15T08:00:00Z",
                sportType = "GravelRide",
                legacyType = "Ride",
                track = listOf(listOf(45.0, 6.0), listOf(45.01, 6.02), listOf(45.02, 6.03)),
            ),
            buildHistoryActivity(
                id = 2L,
                startDate = "2026-04-16T08:00:00Z",
                sportType = "Ride",
                legacyType = "Ride",
                track = listOf(listOf(45.1, 6.1), listOf(45.11, 6.12), listOf(45.12, 6.13)),
            ),
        )

        // WHEN
        val profile = buildRoutingHistoryProfile(activities, "GRAVEL", now, halfLifeDays = 75.0)

        // THEN
        assertNotNull(profile)
        assertEquals("GRAVEL", profile.routeType)
        assertEquals(1, profile.activityCount)
        assertTrue(profile.axisScores.isNotEmpty())
        assertTrue(profile.zoneScores.isNotEmpty())
    }

    @Test
    fun `build routing history profile applies recency decay`() {
        // GIVEN
        val now = Instant.parse("2026-04-20T10:00:00Z")
        val recentTrack = listOf(listOf(45.0, 6.0), listOf(45.02, 6.0))
        val oldTrack = listOf(listOf(46.0, 7.0), listOf(46.02, 7.0))
        val activities = listOf(
            buildHistoryActivity(11L, "2026-04-10T08:00:00Z", "Ride", "Ride", recentTrack),
            buildHistoryActivity(12L, "2025-04-10T08:00:00Z", "Ride", "Ride", oldTrack),
        )
        val recentAxis = axisKey(recentTrack[0], recentTrack[1])
        val oldAxis = axisKey(oldTrack[0], oldTrack[1])

        // WHEN
        val profile = buildRoutingHistoryProfile(activities, "RIDE", now, halfLifeDays = 75.0)

        // THEN
        assertNotNull(profile)
        val recentScore = profile.axisScores[recentAxis]
        val oldScore = profile.axisScores[oldAxis]
        assertNotNull(recentScore)
        assertNotNull(oldScore)
        assertTrue(recentScore > oldScore, "recent score should be greater than old score")
    }

    @Test
    fun `build routing history profile returns null when no matching track`() {
        // GIVEN
        val now = Instant.parse("2026-04-20T10:00:00Z")
        val activities = listOf(
            buildHistoryActivity(
                id = 21L,
                startDate = "2026-04-15T08:00:00Z",
                sportType = "Ride",
                legacyType = "Ride",
                track = listOf(listOf(45.0, 6.0), listOf(45.02, 6.0)),
            )
        )

        // WHEN
        val profile = buildRoutingHistoryProfile(activities, "HIKE", now, halfLifeDays = 75.0)

        // THEN
        assertNull(profile)
    }

    private fun buildHistoryActivity(
        id: Long,
        startDate: String,
        sportType: String,
        legacyType: String,
        track: List<List<Double>>,
    ): StravaActivity {
        return StravaActivity(
            athlete = AthleteRef(id = 1),
            averageSpeed = 2.8,
            averageCadence = 80.0,
            averageHeartrate = 145.0,
            maxHeartrate = 175,
            averageWatts = 210,
            commute = false,
            distance = 10_000.0,
            deviceWatts = true,
            elapsedTime = 3600,
            elevHigh = 1900.0,
            id = id,
            kilojoules = 500.0,
            maxSpeed = 15.0f,
            movingTime = 3600,
            name = "History-$id",
            _sportType = sportType,
            startDate = startDate,
            startDateLocal = startDate,
            startLatlng = track.firstOrNull(),
            totalElevationGain = 450.0,
            type = legacyType,
            uploadId = id + 1000,
            weightedAverageWatts = 220,
            stream = Stream(
                distance = DistanceStream(
                    data = listOf(0.0, 10_000.0),
                    originalSize = 2,
                    resolution = "high",
                    seriesType = "distance",
                ),
                time = TimeStream(
                    data = listOf(0, 3600),
                    originalSize = 2,
                    resolution = "high",
                    seriesType = "time",
                ),
                latlng = LatLngStream(
                    data = track,
                    originalSize = track.size,
                    resolution = "high",
                    seriesType = "distance",
                ),
            ),
        )
    }

    private fun axisKey(from: List<Double>, to: List<Double>): String {
        val first = nodeKey(from[0], from[1], 4)
        val second = nodeKey(to[0], to[1], 4)
        return if (first <= second) "$first|$second" else "$second|$first"
    }

    private fun nodeKey(lat: Double, lng: Double, precision: Int): String {
        return "%.${precision}f:%.${precision}f".format(Locale.US, lat, lng)
    }
}
