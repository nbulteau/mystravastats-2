package me.nicolas.stravastats.domain.services.routing

import me.nicolas.stravastats.domain.business.Coordinates
import org.junit.jupiter.api.Test
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
}
