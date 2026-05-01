package me.nicolas.stravastats.domain.services.routing

import me.nicolas.stravastats.domain.business.Coordinates
import org.junit.jupiter.api.Test
import kotlin.math.PI
import kotlin.math.cos
import kotlin.math.sin
import kotlin.test.assertEquals
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
