package me.nicolas.stravastats.domain.services.routing

import me.nicolas.stravastats.domain.business.ActivityShort
import me.nicolas.stravastats.domain.business.ActivityType
import me.nicolas.stravastats.domain.business.Coordinates
import me.nicolas.stravastats.domain.business.RouteRecommendation
import me.nicolas.stravastats.domain.business.RouteVariantType
import com.sun.net.httpserver.HttpServer
import org.junit.jupiter.api.Test
import java.net.InetSocketAddress
import java.net.URLDecoder
import java.nio.charset.StandardCharsets
import java.util.Locale
import java.util.concurrent.atomic.AtomicInteger
import kotlin.math.PI
import kotlin.math.cos
import kotlin.math.sin
import kotlin.test.assertEquals
import kotlin.test.assertNotNull
import kotlin.test.assertNull
import kotlin.test.assertTrue

class OsmRoutingEngineAdapterShapeTest {

    @Test
    fun `parse shape polyline coordinates decodes encoded polyline`() {
        // GIVEN
        val adapter = OsmRoutingEngineAdapter()
        val encoded = "_p~iF~ps|U_ulLnnqC_mqNvxq`@"

        // WHEN
        val points = invokeParseShapePolylineCoordinates(adapter, encoded)

        // THEN
        assertEquals(3, points.size)
        assertTrue(points.first().lat > 38.49 && points.first().lat < 38.51)
    }

    @Test
    fun `parse shape polyline coordinates extracts gpx track points`() {
        // GIVEN
        val adapter = OsmRoutingEngineAdapter()
        val gpx = """
            <gpx version="1.1" creator="test">
              <trk><trkseg>
                <trkpt lat="48.1000" lon="-1.6000"></trkpt>
                <trkpt lat="48.1200" lon="-1.6200"></trkpt>
                <trkpt lat="48.1300" lon="-1.6300"></trkpt>
              </trkseg></trk>
            </gpx>
        """.trimIndent()

        // WHEN
        val points = invokeParseShapePolylineCoordinates(adapter, gpx)

        // THEN
        assertEquals(3, points.size)
        assertTrue(points.last().lat > 48.129 && points.last().lat < 48.131)
    }

    @Test
    fun `build shape road first waypoints returns anchored loop`() {
        // GIVEN
        val adapter = OsmRoutingEngineAdapter()
        val start = Coordinates(lat = 48.13000, lng = -1.63000)
        val shape = listOf(
            Coordinates(lat = 48.13000, lng = -1.63000),
            Coordinates(lat = 48.14200, lng = -1.62000),
            Coordinates(lat = 48.14800, lng = -1.60000),
            Coordinates(lat = 48.13700, lng = -1.59000),
            Coordinates(lat = 48.13000, lng = -1.63000),
        )

        // WHEN
        val waypoints = invokeBuildShapeRoadFirstWaypoints(adapter, start, shape)
        val shapeFirstWaypoints = invokeBuildShapeLoopWaypoints(adapter, start, shape)

        // THEN
        assertTrue(waypoints.size >= 3)
        assertEquals(start, waypoints.first())
        assertEquals(start, waypoints.last())
        assertTrue(waypoints.size <= shapeFirstWaypoints.size + 1)
    }

    @Test
    fun `build shape simplified waypoints keeps simple shape anchors`() {
        // GIVEN
        val adapter = OsmRoutingEngineAdapter()
        val start = Coordinates(lat = 48.1300, lng = -1.6300)
        val circle = invokePrepareShapeForRouting(
            adapter,
            coordinatesFromLatLng(testCircleLatLng(start.lat, start.lng, 1000.0, 72)),
            start,
        )
        val square = listOf(
            invokeDestinationFromBearing(adapter, start, 1.0, 315.0),
            invokeDestinationFromBearing(adapter, start, 1.0, 45.0),
            invokeDestinationFromBearing(adapter, start, 1.0, 135.0),
            invokeDestinationFromBearing(adapter, start, 1.0, 225.0),
            invokeDestinationFromBearing(adapter, start, 1.0, 315.0),
        )
        val star = invokePrepareShapeForRouting(
            adapter,
            coordinatesFromLatLng(testStarLatLng(start.lat, start.lng, 1000.0, 420.0)),
            start,
        )

        // WHEN
        val circleWaypoints = invokeBuildShapeSimplifiedWaypoints(adapter, circle.first(), circle)
        val circleShapeFirstWaypoints = invokeBuildShapeLoopWaypoints(adapter, circle.first(), circle)
        val squareWaypoints = invokeBuildShapeSimplifiedWaypoints(adapter, square.first(), square)
        val starWaypoints = invokeBuildShapeSimplifiedWaypoints(adapter, star.first(), star)

        // THEN
        assertTrue(circleWaypoints.size >= 7, "circle should keep enough anchors")
        assertTrue(
            circleWaypoints.size < circleShapeFirstWaypoints.size,
            "circle anchors should be simpler than shape-first"
        )
        assertEquals(square.size, squareWaypoints.size, "square corners should be preserved")
        assertTrue(starWaypoints.size >= 10, "star points should be preserved")
        mapOf(
            "circle" to circleWaypoints,
            "square" to squareWaypoints,
            "star" to starWaypoints,
        ).forEach { (label, waypoints) ->
            val closureDistance = haversineDistanceMeters(waypoints.first(), waypoints.last())
            assertTrue(closureDistance < 120.0, "$label waypoints should close the loop")
        }
    }

    @Test
    fun `build shape stitched waypoints returns compact anchored loop`() {
        // GIVEN
        val adapter = OsmRoutingEngineAdapter()
        val start = Coordinates(lat = 48.1300, lng = -1.6300)
        val circle = invokePrepareShapeForRouting(
            adapter,
            coordinatesFromLatLng(testCircleLatLng(start.lat, start.lng, 1000.0, 72)),
            start,
        )
        val shapeFirstWaypoints = invokeBuildShapeLoopWaypoints(adapter, circle.first(), circle)

        // WHEN
        val stitchedWaypoints = invokeBuildShapeStitchedWaypoints(adapter, circle.first(), circle)

        // THEN
        assertTrue(stitchedWaypoints.size >= 8, "stitched waypoints should keep enough contour anchors")
        assertTrue(
            stitchedWaypoints.size < shapeFirstWaypoints.size,
            "stitched waypoints should stay more compact than dense shape-first"
        )
        val closureDistance = haversineDistanceMeters(stitchedWaypoints.first(), stitchedWaypoints.last())
        assertTrue(closureDistance < 120.0, "stitched waypoints should close the loop")
    }

    @Test
    fun `nearest road trace routes between snapped anchors`() {
        // GIVEN
        val routeCalls = AtomicInteger(0)
        val server = HttpServer.create(InetSocketAddress(0), 0)
        server.createContext("/") { exchange ->
            val path = exchange.requestURI.rawPath
            when {
                path.startsWith("/nearest/v1/cycling/") -> {
                    val rawCoordinate = path.removePrefix("/nearest/v1/cycling/").urlDecoded()
                    val (lng, lat) = parseOsrmTestCoordinate(rawCoordinate)
                    exchange.writeJson(
                        200,
                        """{"code":"Ok","waypoints":[{"location":[${lng.osrmTestFormat()},${lat.osrmTestFormat()}],"distance":5.0}]}"""
                    )
                }
                path.startsWith("/route/v1/cycling/") -> {
                    routeCalls.incrementAndGet()
                    val rawCoordinates = path.removePrefix("/route/v1/cycling/").urlDecoded()
                    val parts = rawCoordinates.split(";")
                    assertEquals(2, parts.size, "expected two route coordinates")
                    val (startLng, startLat) = parseOsrmTestCoordinate(parts[0])
                    val (endLng, endLat) = parseOsrmTestCoordinate(parts[1])
                    val midLng = (startLng + endLng) / 2.0 + 0.0002
                    val midLat = (startLat + endLat) / 2.0 + 0.0002
                    exchange.writeJson(
                        200,
                        """{"code":"Ok","routes":[{"distance":100.0,"duration":20.0,"geometry":{"type":"LineString","coordinates":[[${startLng.osrmTestFormat()},${startLat.osrmTestFormat()}],[${midLng.osrmTestFormat()},${midLat.osrmTestFormat()}],[${endLng.osrmTestFormat()},${endLat.osrmTestFormat()}]]},"legs":[{"steps":[{"distance":100.0,"mode":"cycling"}]}]}]}"""
                    )
                }
                else -> exchange.writeJson(404, """{"code":"NotFound"}""")
            }
        }
        server.start()
        val previousBaseUrl = System.getProperty("OSM_ROUTING_BASE_URL")
        System.setProperty("OSM_ROUTING_BASE_URL", "http://127.0.0.1:${server.address.port}")

        try {
            val adapter = OsmRoutingEngineAdapter()
            val shape = listOf(
                Coordinates(lat = 48.1300, lng = -1.6300),
                Coordinates(lat = 48.1310, lng = -1.6200),
                Coordinates(lat = 48.1300, lng = -1.6100),
            )

            // WHEN
            val route = invokeFetchNearestRoadTraceRoute(adapter, "cycling", shape)

            // THEN
            assertNotNull(route)
            val coordinates = invokeOsrmRouteCoordinates(route)
            assertEquals(3, routeCalls.get(), "expected one OSRM route call per snapped segment")
            assertTrue(coordinates.size > shape.size + 1, "expected routed geometry points beyond snapped anchors")
            assertEquals(shape.first().lng, coordinates.first()[0], 0.000001)
            assertEquals(shape.first().lat, coordinates.first()[1], 0.000001)
        } finally {
            if (previousBaseUrl == null) {
                System.clearProperty("OSM_ROUTING_BASE_URL")
            } else {
                System.setProperty("OSM_ROUTING_BASE_URL", previousBaseUrl)
            }
            server.stop(0)
        }
    }

    @Test
    fun `shape selection prioritizes art fit before retrace practicality`() {
        // GIVEN
        val adapter = OsmRoutingEngineAdapter()
        val request = RoutingEngineRequest(
            startPoint = Coordinates(lat = 48.1300, lng = -1.6300),
            distanceTargetKm = 12.0,
            elevationTargetM = null,
            startDirection = null,
            routeType = "RIDE",
            shapePolyline = "[[48.13,-1.63],[48.14,-1.62],[48.13,-1.63]]",
            limit = 1,
        )
        val candidates = listOf(
            buildOsrmRouteCandidate(
                recommendation = testShapeRecommendation("strict-low-art-fit", shapeScore = 0.44, matchScore = 92.0),
                backtrackingRatio = 0.0005,
                effectiveMatchScore = 91.0,
            ),
            buildOsrmRouteCandidate(
                recommendation = testShapeRecommendation("retraced-high-art-fit", shapeScore = 0.82, matchScore = 78.0),
                backtrackingRatio = 0.61,
                corridorOverlap = 0.66,
                edgeReuseRatio = 0.50,
                maxAxisReuseCount = 12,
                segmentDiversity = 0.03,
                effectiveMatchScore = 70.0,
            ),
        )

        // WHEN
        val recommendations = invokeSelectCandidatesWithRelaxation(adapter, request, candidates)

        // THEN
        assertEquals(1, recommendations.size)
        assertEquals("retraced-high-art-fit", recommendations.first().routeId)
        assertTrue(
            recommendations.first().reasons.contains("Selection priority: art-fit first"),
            "expected art-fit selection reason, got ${recommendations.first().reasons}"
        )
        assertTrue(
            recommendations.first().reasons.contains("Selection profile: art-fit-diagnostic (retrace allowed)"),
            "expected diagnostic selection profile, got ${recommendations.first().reasons}"
        )
    }

    @Test
    fun `shape selection keeps retrace as diagnostic`() {
        // GIVEN
        val adapter = OsmRoutingEngineAdapter()
        val request = RoutingEngineRequest(
            startPoint = Coordinates(lat = 48.1300, lng = -1.6300),
            distanceTargetKm = 12.0,
            elevationTargetM = null,
            startDirection = null,
            routeType = "RIDE",
            shapePolyline = "[[48.13,-1.63],[48.14,-1.62],[48.13,-1.63]]",
            limit = 1,
        )
        val candidates = listOf(
            buildOsrmRouteCandidate(
                recommendation = testShapeRecommendation("shape-retrace-art-fit", shapeScore = 0.57, matchScore = 70.0),
                backtrackingRatio = 0.74,
                corridorOverlap = 0.83,
                edgeReuseRatio = 0.80,
                maxAxisReuseCount = 18,
                segmentDiversity = 0.01,
                effectiveMatchScore = 20.0,
            ),
        )
        val rejectCounts = mutableMapOf<String, Int>()

        // WHEN
        val recommendations = invokeSelectCandidatesWithRelaxation(adapter, request, candidates, rejectCounts)

        // THEN
        assertEquals(1, recommendations.size)
        assertEquals("shape-retrace-art-fit", recommendations.first().routeId)
        assertTrue(
            recommendations.first().reasons.contains("Selection priority: art-fit first"),
            "expected art-fit selection reason, got ${recommendations.first().reasons}"
        )
        assertTrue(
            recommendations.first().reasons.contains("Selection profile: art-fit-diagnostic (retrace allowed)"),
            "expected art-fit diagnostic profile, got ${recommendations.first().reasons}"
        )
        assertTrue(rejectCounts.isEmpty(), "expected Strava Art retrace to remain diagnostic-only, got $rejectCounts")
    }

    @Test
    fun `shape candidate conversion keeps highly retraced route but classic loop rejects it`() {
        // GIVEN
        val adapter = OsmRoutingEngineAdapter()
        val start = Coordinates(lat = 48.1300, lng = -1.6300)
        val preview = listOf(
            listOf(start.lat, start.lng),
            listOf(start.lat + 0.035, start.lng),
            listOf(start.lat, start.lng),
            listOf(start.lat + 0.035, start.lng),
            listOf(start.lat, start.lng),
        )
        val shapeRequest = RoutingEngineRequest(
            startPoint = start,
            distanceTargetKm = 4.0,
            elevationTargetM = null,
            startDirection = null,
            routeType = "RIDE",
            shapePolyline = "[[48.13,-1.63],[48.15,-1.63],[48.13,-1.63]]",
            limit = 1,
        )
        val classicRequest = shapeRequest.copy(shapePolyline = null)

        // WHEN / THEN
        assertNull(
            invokeToRouteCandidateFromPreview(adapter, classicRequest, preview),
            "classic loop generation should keep rejecting excessive retrace",
        )
        assertNotNull(
            invokeToRouteCandidateFromPreview(adapter, shapeRequest, preview),
            "Strava Art should keep retraced candidates so Art fit can rank them",
        )
    }

    @Test
    fun `project shape keeps map placed sketch around start`() {
        // GIVEN
        val adapter = OsmRoutingEngineAdapter()
        val start = Coordinates(lat = 48.1300, lng = -1.6300)
        val shape = coordinatesFromLatLng(testCircleLatLng(start.lat, start.lng, 1000.0, 72))
        val targetDistanceKm = polylineDistanceKm(shape)

        // WHEN
        val projected = invokeProjectShapePolylineToStart(adapter, shape, start, targetDistanceKm)
        val projectedCenter = boundingCenter(projected)

        // THEN
        assertEquals(shape.size, projected.size)
        val firstPointDistance = haversineDistanceMeters(start, projected.first())
        assertTrue(firstPointDistance > 900.0, "first sketch point should remain on contour, got ${firstPointDistance}m")
        val centerDrift = haversineDistanceMeters(start, projectedCenter)
        assertTrue(centerDrift < 30.0, "map-placed shape center should stay near start, drift=${centerDrift}m")
    }

    @Test
    fun `project shape recenters remote sketch by center`() {
        // GIVEN
        val adapter = OsmRoutingEngineAdapter()
        val start = Coordinates(lat = 48.1300, lng = -1.6300)
        val remoteShape = coordinatesFromLatLng(testCircleLatLng(45.1885, 5.7245, 1000.0, 72))
        val targetDistanceKm = polylineDistanceKm(remoteShape)

        // WHEN
        val projected = invokeProjectShapePolylineToStart(adapter, remoteShape, start, targetDistanceKm)
        val projectedCenter = boundingCenter(projected)

        // THEN
        val centerDrift = haversineDistanceMeters(start, projectedCenter)
        assertTrue(centerDrift < 30.0, "remote shape should be recentered around start, drift=${centerDrift}m")
        val firstPointDistance = haversineDistanceMeters(start, projected.first())
        assertTrue(firstPointDistance > 900.0, "recentered shape should preserve contour radius, got ${firstPointDistance}m")
    }

    @Test
    fun `prepare shape for routing rotates closed sketch to nearest contour point`() {
        // GIVEN
        val adapter = OsmRoutingEngineAdapter()
        val start = Coordinates(lat = 48.1300, lng = -1.6300)
        val shape = coordinatesFromLatLng(testCircleLatLng(start.lat, start.lng, 1000.0, 72))

        // WHEN
        val routed = invokePrepareShapeForRouting(adapter, shape, start)

        // THEN
        assertEquals(shape.size, routed.size)
        val firstPointDistance = haversineDistanceMeters(start, routed.first())
        assertTrue(firstPointDistance > 900.0, "routing should start on the drawn contour, got ${firstPointDistance}m")
        val closureDistance = haversineDistanceMeters(routed.first(), routed.last())
        assertTrue(closureDistance < 120.0, "routing shape should stay closed, closure=${closureDistance}m")
    }

    @Test
    fun `best effort shape routing strategies keep fallback waypoint sets`() {
        // GIVEN
        val adapter = OsmRoutingEngineAdapter()
        val start = Coordinates(lat = 48.1300, lng = -1.6300)
        val shape = invokePrepareShapeForRouting(
            adapter,
            coordinatesFromLatLng(testCircleLatLng(start.lat, start.lng, 1000.0, 72)),
            start,
        )

        // WHEN
        val strategies = invokeBuildShapeBestEffortRoutingStrategies(adapter, shape.first(), shape)

        // THEN
        assertTrue(strategies.size >= 2, "expected simplified and envelope fallback strategies")
        strategies.forEach { strategy ->
            val waypoints = strategy.javaClass.getDeclaredField("waypoints")
                .apply { isAccessible = true }
                .get(strategy) as List<*>
            val bestEffort = strategy.javaClass.getDeclaredField("bestEffort")
                .apply { isAccessible = true }
                .get(strategy) as Boolean
            assertTrue(bestEffort, "fallback strategy should be marked best effort")
            assertTrue(waypoints.size >= 3, "fallback strategy should keep at least 3 waypoints")
        }
    }

    @Test
    fun `shape mode match score penalizes low similarity for road first`() {
        // GIVEN
        val adapter = OsmRoutingEngineAdapter()

        // WHEN
        val (highScore, highPenalty) = invokeShapeModeMatchScore(
            adapter = adapter,
            baseMatchScore = 78.0,
            shapeScore = 0.72,
            backtrackingRatio = 0.0,
            corridorOverlap = 0.0,
            edgeReuseRatio = 0.0,
            maxAxisReuseRatio = 0.0,
            strategyCode = "road-first",
        )
        val (lowScore, lowPenalty) = invokeShapeModeMatchScore(
            adapter = adapter,
            baseMatchScore = 78.0,
            shapeScore = 0.38,
            backtrackingRatio = 0.0,
            corridorOverlap = 0.0,
            edgeReuseRatio = 0.0,
            maxAxisReuseRatio = 0.0,
            strategyCode = "road-first",
        )

        // THEN
        assertTrue(lowPenalty > highPenalty)
        assertTrue(lowScore < highScore)
    }

    @Test
    fun `shape mode low similarity keeps shape first above road first`() {
        // GIVEN
        val adapter = OsmRoutingEngineAdapter()
        val shapeScore = 0.40

        // WHEN
        val (shapeFirstScore, shapeFirstPenalty) = invokeShapeModeMatchScore(
            adapter = adapter,
            baseMatchScore = 82.0,
            shapeScore = shapeScore,
            backtrackingRatio = 0.0,
            corridorOverlap = 0.0,
            edgeReuseRatio = 0.0,
            maxAxisReuseRatio = 0.0,
            strategyCode = "shape-first",
        )
        val (roadFirstScore, roadFirstPenalty) = invokeShapeModeMatchScore(
            adapter = adapter,
            baseMatchScore = 82.0,
            shapeScore = shapeScore,
            backtrackingRatio = 0.0,
            corridorOverlap = 0.0,
            edgeReuseRatio = 0.0,
            maxAxisReuseRatio = 0.0,
            strategyCode = "road-first",
        )

        // THEN
        assertTrue(roadFirstPenalty > shapeFirstPenalty)
        assertTrue(roadFirstScore < shapeFirstScore)
    }

    @Test
    fun `shape similarity score penalizes anchored shape drift`() {
        // GIVEN
        val adapter = OsmRoutingEngineAdapter()
        val shape = testCircleLatLng(48.1300, -1.6300, 1000.0, 96)
        val matchingRoute = testCircleLatLng(48.1300, -1.6300, 1000.0, 96)
        val shiftedRoute = testCircleLatLng(48.1300, -1.6460, 1000.0, 96)

        // WHEN
        val matchingScore = invokeShapeSimilarityScore(adapter, matchingRoute, shape)
        val shiftedScore = invokeShapeSimilarityScore(adapter, shiftedRoute, shape)

        // THEN
        assertTrue(matchingScore > 0.95, "matching circle should keep high shape score, got $matchingScore")
        assertTrue(shiftedScore < 0.62, "shifted circle should be rejected-level similarity, got $shiftedScore")
    }

    @Test
    fun `shape similarity score penalizes ordered path mismatch`() {
        // GIVEN
        val adapter = OsmRoutingEngineAdapter()
        val shape = testCircleLatLng(48.1300, -1.6300, 1000.0, 96)
        val zigzagRoute = listOf(
            listOf(48.1300, -1.6300),
            listOf(48.1390, -1.6400),
            listOf(48.1210, -1.6380),
            listOf(48.1390, -1.6250),
            listOf(48.1210, -1.6220),
            listOf(48.1300, -1.6300),
        )

        // WHEN
        val score = invokeShapeSimilarityScore(adapter, zigzagRoute, shape)

        // THEN
        assertTrue(score < 0.56, "zigzag route should fail shape-first similarity floor, got $score")
    }

    private fun testShapeRecommendation(routeId: String, shapeScore: Double, matchScore: Double): RouteRecommendation {
        return RouteRecommendation(
            routeId = routeId,
            activity = ActivityShort(id = 0L, name = routeId, type = ActivityType.Ride),
            activityDate = "2026-01-01",
            distanceKm = 12.0,
            elevationGainM = 120.0,
            durationSec = 2400,
            isLoop = true,
            start = null,
            end = null,
            startArea = "Rennes",
            season = "SPRING",
            variantType = RouteVariantType.SHAPE_MATCH,
            matchScore = matchScore,
            reasons = emptyList(),
            previewLatLng = emptyList(),
            shape = "CUSTOM_SHAPE",
            shapeScore = shapeScore,
            experimental = true,
        )
    }

    private fun buildOsrmRouteCandidate(
        recommendation: RouteRecommendation,
        backtrackingRatio: Double,
        effectiveMatchScore: Double,
        corridorOverlap: Double = 0.0010,
        edgeReuseRatio: Double = 0.005,
        maxAxisReuseCount: Int = 1,
        segmentDiversity: Double = 0.70,
    ): Any {
        val candidateClass = Class.forName("me.nicolas.stravastats.domain.services.routing.OsrmRouteCandidate")
        val constructor = candidateClass.getDeclaredConstructor(
            RouteRecommendation::class.java,
            java.lang.Double.TYPE,
            java.lang.Double.TYPE,
            java.lang.Double.TYPE,
            java.lang.Double.TYPE,
            java.lang.Integer.TYPE,
            java.lang.Double.TYPE,
            java.lang.Double.TYPE,
            java.lang.Double.TYPE,
            java.lang.Double.TYPE,
            java.lang.Double.TYPE,
            java.lang.Double.TYPE,
        )
        constructor.isAccessible = true
        return constructor.newInstance(
            recommendation,
            0.0,
            backtrackingRatio,
            corridorOverlap,
            edgeReuseRatio,
            maxAxisReuseCount,
            0.0,
            segmentDiversity,
            0.02,
            0.0,
            0.0,
            effectiveMatchScore,
        )
    }

    @Suppress("UNCHECKED_CAST")
    private fun invokeSelectCandidatesWithRelaxation(
        adapter: OsmRoutingEngineAdapter,
        request: RoutingEngineRequest,
        candidates: List<Any>,
        rejectCounts: MutableMap<String, Int> = mutableMapOf(),
    ): List<RouteRecommendation> {
        val method = adapter.javaClass.getDeclaredMethod(
            "selectCandidatesWithRelaxation",
            RoutingEngineRequest::class.java,
            List::class.java,
            MutableMap::class.java,
        )
        method.isAccessible = true
        return method.invoke(adapter, request, candidates, rejectCounts) as List<RouteRecommendation>
    }

    private fun invokeToRouteCandidateFromPreview(
        adapter: OsmRoutingEngineAdapter,
        request: RoutingEngineRequest,
        preview: List<List<Double>>,
    ): Any? {
        val surfaceBreakdownClass =
            Class.forName("me.nicolas.stravastats.domain.services.routing.RouteSurfaceBreakdown")
        val constructor = surfaceBreakdownClass.getDeclaredConstructor(
            java.lang.Double.TYPE,
            java.lang.Double.TYPE,
            java.lang.Double.TYPE,
            java.lang.Double.TYPE,
        )
        constructor.isAccessible = true
        val surfaceBreakdown = constructor.newInstance(0.0, 0.0, 0.0, 9000.0)
        val method = adapter.javaClass.getDeclaredMethod(
            "toRouteCandidateFromPreview",
            RoutingEngineRequest::class.java,
            List::class.java,
            surfaceBreakdownClass,
            java.lang.Double.TYPE,
            java.lang.Integer.TYPE,
            java.lang.Integer.TYPE,
            MutableMap::class.java,
            java.lang.Boolean.TYPE,
        )
        method.isAccessible = true
        return method.invoke(
            adapter,
            request,
            preview,
            surfaceBreakdown,
            9.0,
            1800,
            0,
            mutableMapOf<String, Int>(),
            false,
        )
    }

    @Suppress("UNCHECKED_CAST")
    private fun invokeParseShapePolylineCoordinates(
        adapter: OsmRoutingEngineAdapter,
        raw: String,
    ): List<Coordinates> {
        val method = adapter.javaClass.getDeclaredMethod("parseShapePolylineCoordinates", String::class.java)
        method.isAccessible = true
        return method.invoke(adapter, raw) as List<Coordinates>
    }

    @Suppress("UNCHECKED_CAST")
    private fun invokeBuildShapeRoadFirstWaypoints(
        adapter: OsmRoutingEngineAdapter,
        start: Coordinates,
        shape: List<Coordinates>,
    ): List<Coordinates> {
        val method = adapter.javaClass.getDeclaredMethod(
            "buildShapeRoadFirstWaypoints",
            Coordinates::class.java,
            List::class.java,
        )
        method.isAccessible = true
        return method.invoke(adapter, start, shape) as List<Coordinates>
    }

    @Suppress("UNCHECKED_CAST")
    private fun invokeBuildShapeLoopWaypoints(
        adapter: OsmRoutingEngineAdapter,
        start: Coordinates,
        shape: List<Coordinates>,
    ): List<Coordinates> {
        val method = adapter.javaClass.getDeclaredMethod(
            "buildShapeLoopWaypoints",
            Coordinates::class.java,
            List::class.java,
        )
        method.isAccessible = true
        return method.invoke(adapter, start, shape) as List<Coordinates>
    }

    @Suppress("UNCHECKED_CAST")
    private fun invokeBuildShapeSimplifiedWaypoints(
        adapter: OsmRoutingEngineAdapter,
        start: Coordinates,
        shape: List<Coordinates>,
    ): List<Coordinates> {
        val method = adapter.javaClass.getDeclaredMethod(
            "buildShapeSimplifiedWaypoints",
            Coordinates::class.java,
            List::class.java,
        )
        method.isAccessible = true
        return method.invoke(adapter, start, shape) as List<Coordinates>
    }

    @Suppress("UNCHECKED_CAST")
    private fun invokeBuildShapeStitchedWaypoints(
        adapter: OsmRoutingEngineAdapter,
        start: Coordinates,
        shape: List<Coordinates>,
    ): List<Coordinates> {
        val method = adapter.javaClass.getDeclaredMethod(
            "buildShapeStitchedWaypoints",
            Coordinates::class.java,
            List::class.java,
        )
        method.isAccessible = true
        return method.invoke(adapter, start, shape) as List<Coordinates>
    }

    private fun invokeDestinationFromBearing(
        adapter: OsmRoutingEngineAdapter,
        start: Coordinates,
        distanceKm: Double,
        bearingDegrees: Double,
    ): Coordinates {
        val method = adapter.javaClass.getDeclaredMethod(
            "destinationFromBearing",
            Coordinates::class.java,
            java.lang.Double.TYPE,
            java.lang.Double.TYPE,
        )
        method.isAccessible = true
        return method.invoke(adapter, start, distanceKm, bearingDegrees) as Coordinates
    }

    @Suppress("UNCHECKED_CAST")
    private fun invokeProjectShapePolylineToStart(
        adapter: OsmRoutingEngineAdapter,
        shape: List<Coordinates>,
        start: Coordinates,
        targetDistanceKm: Double,
    ): List<Coordinates> {
        val method = adapter.javaClass.getDeclaredMethod(
            "projectShapePolylineToStart",
            List::class.java,
            Coordinates::class.java,
            java.lang.Double.TYPE,
        )
        method.isAccessible = true
        return method.invoke(adapter, shape, start, targetDistanceKm) as List<Coordinates>
    }

    @Suppress("UNCHECKED_CAST")
    private fun invokePrepareShapeForRouting(
        adapter: OsmRoutingEngineAdapter,
        shape: List<Coordinates>,
        start: Coordinates,
    ): List<Coordinates> {
        val method = adapter.javaClass.getDeclaredMethod(
            "prepareShapeForRouting",
            List::class.java,
            Coordinates::class.java,
        )
        method.isAccessible = true
        return method.invoke(adapter, shape, start) as List<Coordinates>
    }

    @Suppress("UNCHECKED_CAST")
    private fun invokeBuildShapeBestEffortRoutingStrategies(
        adapter: OsmRoutingEngineAdapter,
        start: Coordinates,
        shape: List<Coordinates>,
    ): List<Any> {
        val method = adapter.javaClass.getDeclaredMethod(
            "buildShapeBestEffortRoutingStrategies",
            Coordinates::class.java,
            List::class.java,
        )
        method.isAccessible = true
        return method.invoke(adapter, start, shape) as List<Any>
    }

    @Suppress("UNCHECKED_CAST")
    private fun invokeShapeModeMatchScore(
        adapter: OsmRoutingEngineAdapter,
        baseMatchScore: Double,
        shapeScore: Double,
        backtrackingRatio: Double,
        corridorOverlap: Double,
        edgeReuseRatio: Double,
        maxAxisReuseRatio: Double,
        strategyCode: String,
    ): Pair<Double, Double> {
        val method = adapter.javaClass.getDeclaredMethod(
            "shapeModeMatchScore",
            java.lang.Double.TYPE,
            java.lang.Double.TYPE,
            java.lang.Double.TYPE,
            java.lang.Double.TYPE,
            java.lang.Double.TYPE,
            java.lang.Double.TYPE,
            String::class.java,
        )
        method.isAccessible = true
        return method.invoke(
            adapter,
            baseMatchScore,
            shapeScore,
            backtrackingRatio,
            corridorOverlap,
            edgeReuseRatio,
            maxAxisReuseRatio,
            strategyCode,
        ) as Pair<Double, Double>
    }

    @Suppress("UNCHECKED_CAST")
    private fun invokeShapeSimilarityScore(
        adapter: OsmRoutingEngineAdapter,
        routePoints: List<List<Double>>,
        shapePoints: List<List<Double>>,
    ): Double {
        val method = adapter.javaClass.getDeclaredMethod(
            "shapeSimilarityScore",
            List::class.java,
            List::class.java,
        )
        method.isAccessible = true
        return method.invoke(adapter, routePoints, shapePoints) as Double
    }

    private fun invokeFetchNearestRoadTraceRoute(
        adapter: OsmRoutingEngineAdapter,
        profile: String,
        shape: List<Coordinates>,
    ): Any? {
        val method = adapter.javaClass.getDeclaredMethod(
            "fetchNearestRoadTraceRoute",
            String::class.java,
            List::class.java,
        )
        method.isAccessible = true
        return method.invoke(adapter, profile, shape)
    }

    @Suppress("UNCHECKED_CAST")
    private fun invokeOsrmRouteCoordinates(route: Any): List<List<Double>> {
        val geometryGetter = route.javaClass.getDeclaredMethod("getGeometry")
        geometryGetter.isAccessible = true
        val geometry = geometryGetter.invoke(route) ?: return emptyList()
        val coordinatesGetter = geometry.javaClass.getDeclaredMethod("getCoordinates")
        coordinatesGetter.isAccessible = true
        return coordinatesGetter.invoke(geometry) as List<List<Double>>
    }

    private fun com.sun.net.httpserver.HttpExchange.writeJson(status: Int, body: String) {
        val bytes = body.toByteArray(StandardCharsets.UTF_8)
        responseHeaders.add("Content-Type", "application/json")
        sendResponseHeaders(status, bytes.size.toLong())
        responseBody.use { stream -> stream.write(bytes) }
    }

    private fun parseOsrmTestCoordinate(raw: String): Pair<Double, Double> {
        val parts = raw.split(",")
        assertEquals(2, parts.size, "expected lon,lat coordinate")
        return parts[0].toDouble() to parts[1].toDouble()
    }

    private fun Double.osrmTestFormat(): String = "%.6f".format(Locale.US, this)

    private fun String.urlDecoded(): String = URLDecoder.decode(this, StandardCharsets.UTF_8)

    private fun testCircleLatLng(
        centerLat: Double,
        centerLng: Double,
        radiusMeters: Double,
        pointCount: Int,
    ): List<List<Double>> {
        val cosLat = cos(Math.toRadians(centerLat))
        return (0..pointCount).map { index ->
            val angle = 2.0 * PI * index.toDouble() / pointCount.toDouble()
            val lat = centerLat + sin(angle) * radiusMeters / 111320.0
            val lng = centerLng + cos(angle) * radiusMeters / (111320.0 * cosLat)
            listOf(lat, lng)
        }
    }

    private fun testStarLatLng(
        centerLat: Double,
        centerLng: Double,
        outerRadiusMeters: Double,
        innerRadiusMeters: Double,
    ): List<List<Double>> {
        val cosLat = cos(Math.toRadians(centerLat))
        return (0..10).map { index ->
            val radius = if (index % 2 == 0) outerRadiusMeters else innerRadiusMeters
            val angle = -PI / 2.0 + index.toDouble() * PI / 5.0
            val lat = centerLat + sin(angle) * radius / 111320.0
            val lng = centerLng + cos(angle) * radius / (111320.0 * cosLat)
            listOf(lat, lng)
        }
    }

    private fun coordinatesFromLatLng(points: List<List<Double>>): List<Coordinates> {
        return points.map { point -> Coordinates(lat = point[0], lng = point[1]) }
    }

    private fun boundingCenter(points: List<Coordinates>): Coordinates {
        val minLat = points.minOf { it.lat }
        val maxLat = points.maxOf { it.lat }
        val minLng = points.minOf { it.lng }
        val maxLng = points.maxOf { it.lng }
        return Coordinates(lat = (minLat + maxLat) / 2.0, lng = (minLng + maxLng) / 2.0)
    }

    private fun polylineDistanceKm(points: List<Coordinates>): Double {
        if (points.size < 2) return 0.0
        return points.zipWithNext().sumOf { (left, right) -> haversineDistanceMeters(left, right) } / 1000.0
    }

    private fun haversineDistanceMeters(left: Coordinates, right: Coordinates): Double {
        val earthRadiusMeters = 6371000.0
        val deltaLat = Math.toRadians(right.lat - left.lat)
        val deltaLng = Math.toRadians(right.lng - left.lng)
        val startLat = Math.toRadians(left.lat)
        val endLat = Math.toRadians(right.lat)
        val a = kotlin.math.sin(deltaLat / 2.0) * kotlin.math.sin(deltaLat / 2.0) +
            kotlin.math.cos(startLat) * kotlin.math.cos(endLat) *
            kotlin.math.sin(deltaLng / 2.0) * kotlin.math.sin(deltaLng / 2.0)
        val c = 2.0 * kotlin.math.atan2(kotlin.math.sqrt(a), kotlin.math.sqrt(1.0 - a))
        return earthRadiusMeters * c
    }
}
