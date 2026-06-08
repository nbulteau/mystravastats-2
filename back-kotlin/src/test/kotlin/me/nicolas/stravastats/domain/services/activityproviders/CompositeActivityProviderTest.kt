package me.nicolas.stravastats.domain.services.activityproviders

import me.nicolas.stravastats.domain.business.ActivityType
import me.nicolas.stravastats.domain.business.strava.AthleteRef
import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.business.strava.StravaAthlete
import me.nicolas.stravastats.domain.business.strava.StravaDetailedActivity
import me.nicolas.stravastats.domain.business.strava.stream.DistanceStream
import me.nicolas.stravastats.domain.business.strava.stream.LatLngStream
import me.nicolas.stravastats.domain.business.strava.stream.Stream
import me.nicolas.stravastats.domain.business.strava.stream.TimeStream
import me.nicolas.stravastats.domain.services.toStravaDetailedActivity
import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Assertions.assertNotNull
import org.junit.jupiter.api.Test
import org.springframework.data.domain.PageRequest

class CompositeActivityProviderTest {
    @Test
    fun `keeps Strava id and enriches matched activity with local stream`() {
        val stravaActivity = testActivity(
            id = 123,
            name = "strava ride",
            sport = "Ride",
            start = "2026-05-01T08:00:00Z",
            distance = 10_000.0,
            movingTime = 3_600,
            stream = null,
        )
        val gpxActivity = testActivity(
            id = 9001,
            name = "local ride",
            sport = "Ride",
            start = "2026-05-01T08:04:00Z",
            distance = 10_100.0,
            movingTime = 3_620,
            stream = testStream(120),
        )

        val provider = CompositeActivityProvider(
            listOf(
                CompositeActivitySource("strava", StubProvider("strava", listOf(stravaActivity))),
                CompositeActivitySource("gpx", StubProvider("gpx", listOf(gpxActivity))),
            )
        )

        val activities = provider.getActivitiesByActivityTypeAndYear(setOf(ActivityType.Ride))

        assertEquals(1, activities.size)
        assertEquals(123, activities.first().id)
        assertEquals(120, activities.first().stream?.latlng?.data?.size)
        assertEquals(1, (provider.getCacheDiagnostics()["composite"] as Map<*, *>)["matchedActivities"])
        assertNotNull(provider.getDetailedActivity(123)?.stream)
    }

    @Test
    fun `matches activities with one hour timezone offset`() {
        val stravaActivity = testActivity(
            id = 8395020437,
            name = "strava ride",
            sport = "Ride",
            start = "2023-01-15T10:52:42Z",
            distance = 40_000.0,
            movingTime = 7_200,
            stream = null,
        )
        val gpxActivity = testActivity(
            id = 853484847,
            name = "local ride",
            sport = "Ride",
            start = "2023-01-15T11:52:42Z",
            distance = 40_150.0,
            movingTime = 7_210,
            stream = testStream(180),
        )

        val provider = CompositeActivityProvider(
            listOf(
                CompositeActivitySource("strava", StubProvider("strava", listOf(stravaActivity))),
                CompositeActivitySource("gpx", StubProvider("gpx", listOf(gpxActivity))),
            )
        )

        val activities = provider.getActivitiesByActivityTypeAndYear(setOf(ActivityType.Ride))

        assertEquals(1, activities.size)
        assertEquals(8395020437, activities.first().id)
        assertEquals(1, (provider.getCacheDiagnostics()["composite"] as Map<*, *>)["matchedActivities"])
    }

    @Test
    fun `matches activities with summer timezone offset`() {
        val stravaActivity = testActivity(
            id = 8813720582,
            name = "strava ride",
            sport = "Ride",
            start = "2023-04-01T13:45:00Z",
            distance = 40_000.0,
            movingTime = 7_200,
            stream = null,
        )
        val gpxActivity = testActivity(
            id = 3004085239,
            name = "local ride",
            sport = "Ride",
            start = "2023-04-01T15:45:00Z",
            distance = 40_150.0,
            movingTime = 7_210,
            stream = testStream(180),
        )

        val provider = CompositeActivityProvider(
            listOf(
                CompositeActivitySource("strava", StubProvider("strava", listOf(stravaActivity))),
                CompositeActivitySource("gpx", StubProvider("gpx", listOf(gpxActivity))),
            )
        )

        val activities = provider.getActivitiesByActivityTypeAndYear(setOf(ActivityType.Ride))

        assertEquals(1, activities.size)
        assertEquals(8813720582, activities.first().id)
        assertEquals(1, (provider.getCacheDiagnostics()["composite"] as Map<*, *>)["matchedActivities"])
    }

    @Test
    fun `rejects same start when distances disagree`() {
        val stravaActivity = testActivity(
            id = 9101,
            name = "strava ride",
            sport = "Ride",
            start = "2023-07-08T08:00:00Z",
            distance = 72_519.0,
            movingTime = 10_248,
            stream = null,
        )
        val fitActivity = testActivity(
            id = 9102,
            name = "fit ride",
            sport = "Ride",
            start = "2023-07-08T08:03:00Z",
            distance = 51_585.0,
            movingTime = 11_312,
            stream = testStream(180),
        )

        val provider = CompositeActivityProvider(
            listOf(
                CompositeActivitySource("strava", StubProvider("strava", listOf(stravaActivity))),
                CompositeActivitySource("fit", StubProvider("fit", listOf(fitActivity))),
            )
        )

        val activities = provider.getActivitiesByActivityTypeAndYear(setOf(ActivityType.Ride))

        assertEquals(2, activities.size)
        assertEquals(0, (provider.getCacheDiagnostics()["composite"] as Map<*, *>)["matchedActivities"])
    }

    @Test
    fun `matches same distance when moving time differs`() {
        val stravaActivity = testActivity(
            id = 9201,
            name = "strava ride",
            sport = "Ride",
            start = "2023-07-08T08:00:00Z",
            distance = 50_000.0,
            movingTime = 10_757,
            stream = null,
        )
        val fitActivity = testActivity(
            id = 9202,
            name = "fit ride",
            sport = "Ride",
            start = "2023-07-08T08:03:00Z",
            distance = 50_100.0,
            movingTime = 13_197,
            stream = testStream(180),
        )

        val provider = CompositeActivityProvider(
            listOf(
                CompositeActivitySource("strava", StubProvider("strava", listOf(stravaActivity))),
                CompositeActivitySource("fit", StubProvider("fit", listOf(fitActivity))),
            )
        )

        val activities = provider.getActivitiesByActivityTypeAndYear(setOf(ActivityType.Ride))

        assertEquals(1, activities.size)
        assertEquals(9201, activities.first().id)
        assertEquals(1, (provider.getCacheDiagnostics()["composite"] as Map<*, *>)["matchedActivities"])
    }

    @Test
    fun `keeps unmatched local activities in union mode`() {
        val fitActivity = testActivity(
            id = 7001,
            name = "fit run",
            sport = "Run",
            start = "2026-05-01T08:00:00Z",
            distance = 5_000.0,
            movingTime = 1_800,
            stream = testStream(20),
        )
        val gpxActivity = testActivity(
            id = 8001,
            name = "gpx run",
            sport = "Run",
            start = "2026-05-02T08:00:00Z",
            distance = 6_000.0,
            movingTime = 2_100,
            stream = testStream(25),
        )

        val provider = CompositeActivityProvider(
            listOf(
                CompositeActivitySource("fit", StubProvider("fit", listOf(fitActivity))),
                CompositeActivitySource("gpx", StubProvider("gpx", listOf(gpxActivity))),
            )
        )

        val ids = provider.getActivitiesByActivityTypeAndYear(setOf(ActivityType.Run)).map { activity -> activity.id }.toSet()

        assertEquals(setOf(7001L, 8001L), ids)
    }

    @Test
    fun `rebuilds when a source activity count changes`() {
        val firstActivity = testActivity(
            id = 9301,
            name = "morning ride",
            sport = "Ride",
            start = "2026-06-08T07:30:00Z",
            distance = 20_000.0,
            movingTime = 3_600,
            stream = null,
        )
        val nextActivity = testActivity(
            id = 9302,
            name = "lunch ride",
            sport = "Ride",
            start = "2026-06-08T12:00:00Z",
            distance = 15_000.0,
            movingTime = 2_700,
            stream = null,
        )
        val source = StubProvider("strava", listOf(firstActivity))
        val provider = CompositeActivityProvider(
            listOf(CompositeActivitySource("strava", source))
        )

        assertEquals(1, provider.getActivitiesByActivityTypeAndYear(setOf(ActivityType.Ride)).size)

        source.appendActivity(nextActivity)

        assertEquals(2, provider.getActivitiesByActivityTypeAndYear(setOf(ActivityType.Ride)).size)
        assertEquals(2L, provider.listActivitiesPaginated(PageRequest.of(0, 10)).totalElements)
        assertEquals(2, provider.getCacheDiagnostics()["activities"])
    }

    @Test
    fun `aggregates source refresh diagnostics`() {
        val source = StubProvider(
            name = "strava",
            seedActivities = listOf(
                testActivity(
                    id = 9401,
                    name = "morning ride",
                    sport = "Ride",
                    start = "2026-06-08T07:30:00Z",
                    distance = 20_000.0,
                    movingTime = 3_600,
                    stream = null,
                )
            ),
            refresh = mapOf(
                "backgroundInProgress" to true,
                "warmupInProgress" to false,
            ),
        )
        val provider = CompositeActivityProvider(
            listOf(CompositeActivitySource("strava", source))
        )

        val refresh = provider.getCacheDiagnostics()["refresh"] as Map<*, *>

        assertEquals(true, refresh["backgroundInProgress"])
    }

    private class StubProvider(
        private val name: String,
        seedActivities: List<StravaActivity>,
        private val refresh: Map<String, Any?>? = null,
    ) : AbstractActivityProvider() {
        init {
            stravaAthlete = StravaAthlete(id = 42, firstname = name)
            activities = seedActivities
        }

        override fun getDetailedActivity(activityId: Long): StravaDetailedActivity? {
            return getActivity(activityId)?.toStravaDetailedActivity()
        }

        override fun getCachedDetailedActivity(activityId: Long): StravaDetailedActivity? {
            return getDetailedActivity(activityId)
        }

        fun appendActivity(activity: StravaActivity) {
            activities = activities + activity
        }

        override fun getCacheDiagnostics(): Map<String, Any?> {
            return mapOf(
                "provider" to name,
                "athleteId" to "$name-athlete",
                "cacheRoot" to "$name-cache",
                "activities" to activities.size,
                "availableYearBins" to activities
                    .map { activity -> activity.startDateLocal.substring(0, 4) }
                    .distinct()
                    .sorted(),
            ) + if (refresh == null) emptyMap() else mapOf("refresh" to refresh)
        }

        override fun cacheIdentity(): ActivityProviderCacheIdentity {
            return ActivityProviderCacheIdentity(
                cacheRoot = "$name-cache",
                athleteId = "$name-athlete",
            )
        }
    }

    private fun testActivity(
        id: Long,
        name: String,
        sport: String,
        start: String,
        distance: Double,
        movingTime: Int,
        stream: Stream?,
    ): StravaActivity {
        return StravaActivity(
            athlete = AthleteRef(42),
            averageSpeed = distance / movingTime,
            commute = false,
            distance = distance,
            elapsedTime = movingTime,
            id = id,
            maxSpeed = 0f,
            movingTime = movingTime,
            name = name,
            _sportType = sport,
            startDate = start,
            startDateLocal = start,
            startLatlng = listOf(48.8566, 2.3522),
            totalElevationGain = 100.0,
            type = sport,
            uploadId = id,
            stream = stream,
        )
    }

    private fun testStream(points: Int): Stream {
        return Stream(
            distance = DistanceStream(
                data = (0 until points).map { index -> index.toDouble() * 10.0 },
                originalSize = points,
                resolution = "high",
                seriesType = "distance",
            ),
            time = TimeStream(
                data = (0 until points).toList(),
                originalSize = points,
                resolution = "high",
                seriesType = "time",
            ),
            latlng = LatLngStream(
                data = (0 until points).map { index -> listOf(48.8566 + index * 0.0001, 2.3522) },
                originalSize = points,
                resolution = "high",
                seriesType = "distance",
            ),
        )
    }
}
