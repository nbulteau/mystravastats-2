package me.nicolas.stravastats.domain.services

import io.mockk.every
import io.mockk.mockk
import me.nicolas.stravastats.domain.business.ActivityType
import me.nicolas.stravastats.domain.business.ActivityShort
import me.nicolas.stravastats.domain.business.Coordinates
import me.nicolas.stravastats.domain.business.RouteExplorerRequest
import me.nicolas.stravastats.domain.business.RouteRecommendation
import me.nicolas.stravastats.domain.business.RouteVariantType
import me.nicolas.stravastats.domain.business.strava.AthleteRef
import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.business.strava.stream.DistanceStream
import me.nicolas.stravastats.domain.business.strava.stream.LatLngStream
import me.nicolas.stravastats.domain.business.strava.stream.Stream
import me.nicolas.stravastats.domain.business.strava.stream.TimeStream
import me.nicolas.stravastats.domain.services.activityproviders.IActivityProvider
import me.nicolas.stravastats.domain.services.routing.RoutingEnginePort
import me.nicolas.stravastats.domain.services.routing.RoutingEngineRequest
import org.junit.jupiter.api.BeforeEach
import org.junit.jupiter.api.Test
import kotlin.test.assertEquals
import kotlin.test.assertNotNull
import kotlin.test.assertTrue
import kotlin.test.fail

class RouteExplorerServiceTest {

    private val activityProvider = mockk<IActivityProvider>()
    private val routingEngine = object : RoutingEnginePort {
        override fun generateTargetLoops(request: RoutingEngineRequest) = emptyList<me.nicolas.stravastats.domain.business.RouteRecommendation>()
        override fun generateShapeLoops(request: RoutingEngineRequest) = emptyList<me.nicolas.stravastats.domain.business.RouteRecommendation>()
        override fun healthDetails(): Map<String, Any?> = mapOf("status" to "disabled")
    }
    private lateinit var routeExplorerService: IRouteExplorerService

    @BeforeEach
    fun setUp() {
        routeExplorerService = RouteExplorerService(activityProvider, routingEngine)
    }

    @Test
    fun `route explorer returns closest loops variants seasonal and shape matches`() {
        // GIVEN
        val activityTypes = setOf(ActivityType.Ride)
        val activities = listOf(
            buildActivity(
                id = 1L,
                name = "Loop Base",
                startDateLocal = "2025-04-01T08:00:00+02:00",
                distanceKm = 43.5,
                elevationM = 620.0,
                durationSec = 8100,
                start = listOf(45.0, 6.0),
                track = listOf(
                    listOf(45.0, 6.0), listOf(45.01, 6.03), listOf(45.03, 6.02), listOf(45.0, 6.0),
                ),
            ),
            buildActivity(
                id = 2L,
                name = "Short Tempo",
                startDateLocal = "2025-04-10T08:00:00+02:00",
                distanceKm = 31.2,
                elevationM = 420.0,
                durationSec = 5600,
                start = listOf(45.1, 6.1),
                track = listOf(
                    listOf(45.1, 6.1), listOf(45.12, 6.11), listOf(45.1, 6.1),
                ),
            ),
            buildActivity(
                id = 3L,
                name = "Long Endurance",
                startDateLocal = "2025-04-20T08:00:00+02:00",
                distanceKm = 72.4,
                elevationM = 760.0,
                durationSec = 13200,
                start = listOf(45.2, 6.2),
                track = listOf(
                    listOf(45.2, 6.2), listOf(45.25, 6.24), listOf(45.29, 6.22), listOf(45.2, 6.2),
                ),
            ),
            buildActivity(
                id = 4L,
                name = "Hill Repeats",
                startDateLocal = "2025-04-23T08:00:00+02:00",
                distanceKm = 44.0,
                elevationM = 1310.0,
                durationSec = 9200,
                start = listOf(45.3, 6.3),
                track = listOf(
                    listOf(45.3, 6.3), listOf(45.32, 6.31), listOf(45.35, 6.33), listOf(45.3, 6.3),
                ),
            ),
        )
        every { activityProvider.getActivitiesByActivityTypeAndYear(activityTypes, null) } returns activities
        val request = RouteExplorerRequest(
            distanceTargetKm = 45.0,
            elevationTargetM = 650.0,
            durationTargetMin = 140,
            season = "SPRING",
            limit = 6,
            shape = "LOOP",
            includeRemix = false,
        )

        // WHEN
        val result = routeExplorerService.getRouteExplorer(activityTypes, null, request)

        // THEN
        assertTrue(result.closestLoops.isNotEmpty(), "closest loops should not be empty")
        assertTrue(result.variants.size >= 3, "smart variants should include shorter/longer/hillier")
        assertTrue(result.seasonal.isNotEmpty(), "seasonal recommendations should not be empty")
        assertTrue(result.shapeMatches.isNotEmpty(), "shape matches should not be empty")
        assertTrue(result.closestLoops.first().routeId.isNotBlank(), "closest loops should expose a stable route id")
        assertTrue(result.variants.first().routeId.isNotBlank(), "smart variants should expose a stable route id")
    }

    @Test
    fun `route explorer returns experimental shape remix when requested`() {
        // GIVEN
        val activityTypes = setOf(ActivityType.Ride)
        val activities = listOf(
            buildActivity(
                id = 11L,
                name = "Outbound Segment",
                startDateLocal = "2025-06-05T08:00:00+02:00",
                distanceKm = 18.0,
                elevationM = 340.0,
                durationSec = 3600,
                start = listOf(45.0, 6.0),
                track = listOf(
                    listOf(45.0, 6.0), listOf(45.02, 6.02), listOf(45.05, 6.05),
                ),
            ),
            buildActivity(
                id = 12L,
                name = "Return Segment",
                startDateLocal = "2025-06-06T08:00:00+02:00",
                distanceKm = 17.5,
                elevationM = 320.0,
                durationSec = 3500,
                start = listOf(45.05, 6.05),
                track = listOf(
                    listOf(45.05, 6.05), listOf(45.02, 6.02), listOf(45.0, 6.0),
                ),
            ),
        )
        every { activityProvider.getActivitiesByActivityTypeAndYear(activityTypes, null) } returns activities

        val request = RouteExplorerRequest(
            distanceTargetKm = 35.0,
            elevationTargetM = null,
            durationTargetMin = null,
            season = null,
            limit = 4,
            shape = null,
            includeRemix = true,
        )

        // WHEN
        val result = routeExplorerService.getRouteExplorer(activityTypes, null, request)

        // THEN
        assertTrue(result.shapeRemixes.isNotEmpty(), "shape remixes should not be empty")
        assertTrue(result.shapeRemixes.first().experimental, "shape remix should be experimental")
        assertEquals(2, result.shapeRemixes.first().components.size, "shape remix should contain 2 activities")
    }

    @Test
    fun `route explorer prioritizes requested departure direction`() {
        // GIVEN
        val activityTypes = setOf(ActivityType.Ride)
        val activities = listOf(
            buildActivity(
                id = 31L,
                name = "North Aligned",
                startDateLocal = "2025-08-01T08:00:00+02:00",
                distanceKm = 46.0,
                elevationM = 670.0,
                durationSec = 8500,
                start = listOf(45.0, 6.0),
                track = listOf(
                    listOf(45.0, 6.0), listOf(45.03, 6.0), listOf(45.04, 6.03), listOf(45.0, 6.0),
                ),
            ),
            buildActivity(
                id = 32L,
                name = "South Aligned",
                startDateLocal = "2025-08-02T08:00:00+02:00",
                distanceKm = 45.0,
                elevationM = 650.0,
                durationSec = 8400,
                start = listOf(45.0, 6.0),
                track = listOf(
                    listOf(45.0, 6.0), listOf(44.97, 6.0), listOf(44.96, 5.97), listOf(45.0, 6.0),
                ),
            ),
        )
        every { activityProvider.getActivitiesByActivityTypeAndYear(activityTypes, null) } returns activities
        val request = RouteExplorerRequest(
            distanceTargetKm = 45.0,
            elevationTargetM = 650.0,
            durationTargetMin = 140,
            startDirection = "N",
            routeType = "RIDE",
            season = null,
            limit = 6,
            shape = null,
            includeRemix = false,
        )

        // WHEN
        val result = routeExplorerService.getRouteExplorer(activityTypes, null, request)

        // THEN
        assertTrue(result.closestLoops.isNotEmpty(), "closest loops should not be empty")
        assertEquals("North Aligned", result.closestLoops.first().activity.name)
    }

    @Test
    fun `route explorer calibrates scoring profile by route type`() {
        // GIVEN
        val activityTypes = setOf(ActivityType.Ride)
        val activities = listOf(
            buildActivity(
                id = 41L,
                name = "Distance Focused",
                startDateLocal = "2025-09-01T08:00:00+02:00",
                distanceKm = 45.0,
                elevationM = 520.0,
                durationSec = 8400,
                start = listOf(45.1, 6.1),
                track = listOf(
                    listOf(45.1, 6.1), listOf(45.13, 6.12), listOf(45.1, 6.1),
                ),
            ),
            buildActivity(
                id = 42L,
                name = "Climb Focused",
                startDateLocal = "2025-09-02T08:00:00+02:00",
                distanceKm = 56.0,
                elevationM = 650.0,
                durationSec = 8400,
                start = listOf(45.1, 6.1),
                track = listOf(
                    listOf(45.1, 6.1), listOf(45.14, 6.15), listOf(45.1, 6.1),
                ),
            ),
        )
        every { activityProvider.getActivitiesByActivityTypeAndYear(activityTypes, null) } returns activities
        val baseRequest = RouteExplorerRequest(
            distanceTargetKm = 45.0,
            elevationTargetM = 650.0,
            durationTargetMin = 140,
            season = null,
            limit = 6,
            shape = null,
            includeRemix = false,
        )

        // WHEN
        val rideResult = routeExplorerService.getRouteExplorer(activityTypes, null, baseRequest.copy(routeType = "RIDE"))
        val hikeResult = routeExplorerService.getRouteExplorer(activityTypes, null, baseRequest.copy(routeType = "HIKE"))

        // THEN
        assertTrue(rideResult.closestLoops.isNotEmpty(), "ride closest loops should not be empty")
        assertTrue(hikeResult.closestLoops.isNotEmpty(), "hike closest loops should not be empty")
        assertEquals("Distance Focused", rideResult.closestLoops.first().activity.name)
        assertEquals("Climb Focused", hikeResult.closestLoops.first().activity.name)
    }

    @Test
    fun `route explorer prioritizes preferred start point when provided`() {
        // GIVEN
        val activityTypes = setOf(ActivityType.Ride)
        val activities = listOf(
            buildActivity(
                id = 51L,
                name = "Near Start Loop",
                startDateLocal = "2025-07-01T08:00:00+02:00",
                distanceKm = 42.0,
                elevationM = 600.0,
                durationSec = 7800,
                start = listOf(45.0008, 6.0006),
                track = listOf(
                    listOf(45.0008, 6.0006), listOf(45.012, 6.018), listOf(45.0008, 6.0006),
                ),
            ),
            buildActivity(
                id = 52L,
                name = "Far Start Loop",
                startDateLocal = "2025-07-02T08:00:00+02:00",
                distanceKm = 42.0,
                elevationM = 600.0,
                durationSec = 7800,
                start = listOf(45.55, 6.55),
                track = listOf(
                    listOf(45.55, 6.55), listOf(45.57, 6.57), listOf(45.55, 6.55),
                ),
            ),
        )
        every { activityProvider.getActivitiesByActivityTypeAndYear(activityTypes, null) } returns activities
        val request = RouteExplorerRequest(
            distanceTargetKm = 42.0,
            elevationTargetM = 600.0,
            durationTargetMin = 130,
            startPoint = Coordinates(lat = 45.0, lng = 6.0),
            routeType = "RIDE",
            season = null,
            limit = 4,
            shape = null,
            includeRemix = false,
        )

        // WHEN
        val result = routeExplorerService.getRouteExplorer(activityTypes, null, request)

        // THEN
        assertTrue(result.closestLoops.isNotEmpty(), "closest loops should not be empty")
        assertEquals("Near Start Loop", result.closestLoops.first().activity.name)
    }

    @Test
    fun `route explorer uses routing engine output when available for road graph loops`() {
        // GIVEN
        val activityTypes = setOf(ActivityType.Ride)
        val activities = listOf(
            buildActivity(
                id = 91L,
                name = "Fallback Loop",
                startDateLocal = "2025-10-01T08:00:00+02:00",
                distanceKm = 30.0,
                elevationM = 300.0,
                durationSec = 5400,
                start = listOf(45.0, 6.0),
                track = listOf(
                    listOf(45.0, 6.0), listOf(45.01, 6.01), listOf(45.0, 6.0),
                ),
            ),
        )
        every { activityProvider.getActivitiesByActivityTypeAndYear(activityTypes, null) } returns activities

        val generatedByEngine = RouteRecommendation(
            routeId = "generated-osm-test",
            activity = ActivityShort(id = 0L, name = "Generated OSM Loop", type = ActivityType.Ride),
            activityDate = "2025-10-01T08:00:00Z",
            distanceKm = 42.0,
            elevationGainM = 600.0,
            durationSec = 7200,
            isLoop = true,
            start = Coordinates(45.0, 6.0),
            end = Coordinates(45.0, 6.0),
            startArea = "45.0000, 6.0000",
            season = "AUTUMN",
            variantType = RouteVariantType.ROAD_GRAPH,
            matchScore = 95.0,
            reasons = listOf("Generated with OSM road graph (OSRM)"),
            previewLatLng = listOf(
                listOf(45.0, 6.0),
                listOf(45.02, 6.02),
                listOf(45.0, 6.0),
            ),
            shape = null,
            shapeScore = null,
            experimental = false,
        )
        val engine = object : RoutingEnginePort {
            override fun generateTargetLoops(request: RoutingEngineRequest): List<RouteRecommendation> = listOf(generatedByEngine)
            override fun generateShapeLoops(request: RoutingEngineRequest): List<RouteRecommendation> = emptyList()
            override fun healthDetails(): Map<String, Any?> = mapOf("status" to "up")
        }
        val serviceWithEngine = RouteExplorerService(activityProvider, engine)
        val request = RouteExplorerRequest(
            distanceTargetKm = 42.0,
            elevationTargetM = 600.0,
            durationTargetMin = 120,
            startPoint = Coordinates(lat = 45.0, lng = 6.0),
            routeType = "RIDE",
            season = null,
            limit = 4,
            shape = null,
            includeRemix = false,
        )

        // WHEN
        val result = serviceWithEngine.getRouteExplorer(activityTypes, null, request)

        // THEN
        assertTrue(result.roadGraphLoops.isNotEmpty(), "road graph loops should not be empty")
        assertEquals("generated-osm-test", result.roadGraphLoops.first().routeId)
        assertEquals("Generated OSM Loop", result.roadGraphLoops.first().activity.name)
    }

    @Test
    fun `route explorer falls back to cache road graph when routing engine returns empty`() {
        // GIVEN
        val activityTypes = setOf(ActivityType.Ride)
        val activities = listOf(
            buildActivity(
                id = 101L,
                name = "Cached Loop",
                startDateLocal = "2025-10-10T08:00:00+02:00",
                distanceKm = 25.0,
                elevationM = 200.0,
                durationSec = 3600,
                start = listOf(45.0, 6.0),
                track = listOf(
                    listOf(45.0, 6.0),
                    listOf(45.01, 6.01),
                    listOf(45.0, 6.0),
                ),
            ),
        )
        every { activityProvider.getActivitiesByActivityTypeAndYear(activityTypes, null) } returns activities

        val emptyEngine = object : RoutingEnginePort {
            override fun generateTargetLoops(request: RoutingEngineRequest): List<RouteRecommendation> = emptyList()
            override fun generateShapeLoops(request: RoutingEngineRequest): List<RouteRecommendation> = emptyList()
            override fun healthDetails(): Map<String, Any?> = mapOf("status" to "up")
        }
        val serviceWithEmptyEngine = RouteExplorerService(activityProvider, emptyEngine)
        val request = RouteExplorerRequest(
            distanceTargetKm = 40.0,
            elevationTargetM = 500.0,
            durationTargetMin = 120,
            startPoint = Coordinates(lat = 45.0, lng = 6.0),
            routeType = "RIDE",
            season = null,
            limit = 5,
            shape = null,
            includeRemix = false,
        )

        // WHEN
        val result = serviceWithEmptyEngine.getRouteExplorer(activityTypes, null, request)

        // THEN
        assertTrue(result.roadGraphLoops.isNotEmpty(), "road graph loops should fallback to cache when OSRM returns no route")
    }

    @Test
    fun `route explorer still calls routing engine when cache candidates are empty`() {
        // GIVEN
        val activityTypes = setOf(ActivityType.Ride)
        every { activityProvider.getActivitiesByActivityTypeAndYear(activityTypes, 2026) } returns emptyList()

        val generatedByEngine = RouteRecommendation(
            routeId = "generated-osm-empty-cache",
            activity = ActivityShort(id = 0L, name = "Generated Empty Cache Loop", type = ActivityType.Ride),
            activityDate = "2026-01-01T08:00:00Z",
            distanceKm = 39.5,
            elevationGainM = 540.0,
            durationSec = 7000,
            isLoop = true,
            start = Coordinates(45.0, 6.0),
            end = Coordinates(45.0, 6.0),
            startArea = "45.0000, 6.0000",
            season = "WINTER",
            variantType = RouteVariantType.ROAD_GRAPH,
            matchScore = 90.0,
            reasons = listOf("Generated with OSM road graph (OSRM)"),
            previewLatLng = listOf(
                listOf(45.0, 6.0),
                listOf(45.02, 6.02),
                listOf(45.0, 6.0),
            ),
            shape = null,
            shapeScore = null,
            experimental = false,
        )
        val engine = object : RoutingEnginePort {
            override fun generateTargetLoops(request: RoutingEngineRequest): List<RouteRecommendation> = listOf(generatedByEngine)
            override fun generateShapeLoops(request: RoutingEngineRequest): List<RouteRecommendation> = emptyList()
            override fun healthDetails(): Map<String, Any?> = mapOf("status" to "up")
        }
        val serviceWithEngine = RouteExplorerService(activityProvider, engine)
        val request = RouteExplorerRequest(
            distanceTargetKm = 40.0,
            elevationTargetM = 500.0,
            durationTargetMin = 120,
            startPoint = Coordinates(lat = 45.0, lng = 6.0),
            routeType = "RIDE",
            season = null,
            limit = 5,
            shape = null,
            includeRemix = false,
        )

        // WHEN
        val result = serviceWithEngine.getRouteExplorer(activityTypes, 2026, request)

        // THEN
        assertTrue(result.closestLoops.isEmpty(), "closest loops should be empty when cache has no activities")
        assertTrue(result.roadGraphLoops.isNotEmpty(), "road graph loops should still come from routing engine")
        assertEquals("generated-osm-empty-cache", result.roadGraphLoops.first().routeId)
    }

    @Test
    fun `route explorer still calls shape routing engine when cache candidates are empty`() {
        // GIVEN
        val activityTypes = setOf(ActivityType.Ride)
        every { activityProvider.getActivitiesByActivityTypeAndYear(activityTypes, 2026) } returns emptyList()

        val generatedShape = RouteRecommendation(
            routeId = "generated-shape-empty-cache",
            activity = ActivityShort(id = 0L, name = "Generated Shape", type = ActivityType.Ride),
            activityDate = "2026-01-01T08:00:00Z",
            distanceKm = 12.4,
            elevationGainM = 120.0,
            durationSec = 2400,
            isLoop = true,
            start = Coordinates(45.0, 6.0),
            end = Coordinates(45.0, 6.0),
            startArea = "45.0000, 6.0000",
            season = "WINTER",
            variantType = RouteVariantType.SHAPE_MATCH,
            matchScore = 91.0,
            reasons = listOf(
                "Generated with OSM road graph (OSRM)",
                "Shape similarity: 91%",
                "Shape mode: projected waypoints",
            ),
            previewLatLng = listOf(
                listOf(45.0, 6.0),
                listOf(45.02, 6.02),
                listOf(45.0, 6.0),
            ),
            shape = "CUSTOM_SHAPE",
            shapeScore = 0.91,
            experimental = false,
        )
        var capturedShapeRequest: RoutingEngineRequest? = null
        val engine = object : RoutingEnginePort {
            override fun generateTargetLoops(request: RoutingEngineRequest): List<RouteRecommendation> = emptyList()
            override fun generateShapeLoops(request: RoutingEngineRequest): List<RouteRecommendation> {
                capturedShapeRequest = request
                return listOf(generatedShape)
            }

            override fun healthDetails(): Map<String, Any?> = mapOf("status" to "up")
        }
        val serviceWithEngine = RouteExplorerService(activityProvider, engine)
        val shapePolyline = "[[45.0,6.0],[45.02,6.02],[45.0,6.0]]"
        val request = RouteExplorerRequest(
            distanceTargetKm = null,
            elevationTargetM = null,
            durationTargetMin = null,
            startPoint = Coordinates(lat = 45.0, lng = 6.0),
            routeType = "RIDE",
            season = null,
            limit = 5,
            shape = "LOOP",
            shapePolyline = shapePolyline,
            includeRemix = true,
        )

        // WHEN
        val result = serviceWithEngine.getRouteExplorer(activityTypes, 2026, request)

        // THEN
        assertEquals("generated-shape-empty-cache", result.shapeMatches.firstOrNull()?.routeId)
        assertEquals(shapePolyline, capturedShapeRequest?.shapePolyline)
    }

    @Test
    fun `route explorer forwards history profile when history bias flag is enabled`() {
        // GIVEN
        val activityTypes = setOf(ActivityType.Ride)
        val activities = listOf(
            buildActivity(
                id = 201L,
                name = "History Source",
                startDateLocal = "2026-04-10T08:00:00Z",
                distanceKm = 35.0,
                elevationM = 450.0,
                durationSec = 6200,
                start = listOf(45.0, 6.0),
                track = listOf(
                    listOf(45.0, 6.0),
                    listOf(45.01, 6.02),
                    listOf(45.02, 6.03),
                ),
            ),
        )
        every { activityProvider.getActivitiesByActivityTypeAndYear(activityTypes, null) } returns activities

        var capturedRequest: RoutingEngineRequest? = null
        val engine = object : RoutingEnginePort {
            override fun generateTargetLoops(request: RoutingEngineRequest): List<RouteRecommendation> {
                capturedRequest = request
                return emptyList()
            }

            override fun generateShapeLoops(request: RoutingEngineRequest): List<RouteRecommendation> = emptyList()

            override fun healthDetails(): Map<String, Any?> = mapOf("status" to "up")
        }
        val serviceWithEngine = RouteExplorerService(activityProvider, engine)
        val request = RouteExplorerRequest(
            distanceTargetKm = 40.0,
            elevationTargetM = 500.0,
            durationTargetMin = 120,
            startPoint = Coordinates(lat = 45.0, lng = 6.0),
            routeType = "RIDE",
            season = null,
            limit = 4,
            shape = null,
            includeRemix = false,
        )

        System.setProperty("OSM_ROUTING_HISTORY_BIAS_ENABLED", "true")
        System.setProperty("OSM_ROUTING_HISTORY_HALF_LIFE_DAYS", "90")

        try {
            // WHEN
            serviceWithEngine.getRouteExplorer(activityTypes, null, request)
        } finally {
            System.clearProperty("OSM_ROUTING_HISTORY_BIAS_ENABLED")
            System.clearProperty("OSM_ROUTING_HISTORY_HALF_LIFE_DAYS")
        }

        // THEN
        val sentRequest = capturedRequest ?: fail("expected routing engine to receive request")
        assertTrue(sentRequest.historyBiasEnabled, "history bias should be enabled in routing request")
        val profile = assertNotNull(sentRequest.historyProfile, "history profile should be forwarded")
        assertEquals("RIDE", profile.routeType)
        assertTrue(profile.activityCount >= 1, "history profile should contain at least one activity")
        assertTrue(profile.axisScores.isNotEmpty(), "history profile should contain axis scores")
    }

    private fun buildActivity(
        id: Long,
        name: String,
        startDateLocal: String,
        distanceKm: Double,
        elevationM: Double,
        durationSec: Int,
        start: List<Double>,
        track: List<List<Double>>,
    ): StravaActivity {
        return StravaActivity(
            athlete = AthleteRef(id = 1),
            averageSpeed = distanceKm * 1000.0 / durationSec.toDouble(),
            averageCadence = 80.0,
            averageHeartrate = 145.0,
            maxHeartrate = 175,
            averageWatts = 210,
            commute = false,
            distance = distanceKm * 1000.0,
            deviceWatts = true,
            elapsedTime = durationSec,
            elevHigh = 1900.0,
            id = id,
            kilojoules = 500.0,
            maxSpeed = 15.0f,
            movingTime = durationSec,
            name = name,
            _sportType = "Ride",
            startDate = startDateLocal,
            startDateLocal = startDateLocal,
            startLatlng = start,
            totalElevationGain = elevationM,
            type = "Ride",
            uploadId = id + 1000,
            weightedAverageWatts = 220,
            stream = Stream(
                distance = DistanceStream(
                    data = listOf(0.0, distanceKm * 1000.0),
                    originalSize = 2,
                    resolution = "high",
                    seriesType = "distance",
                ),
                time = TimeStream(
                    data = listOf(0, durationSec),
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
}
