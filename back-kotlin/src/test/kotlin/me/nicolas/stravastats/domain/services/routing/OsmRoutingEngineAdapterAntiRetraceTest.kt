package me.nicolas.stravastats.domain.services.routing

import me.nicolas.stravastats.domain.business.Coordinates
import org.junit.jupiter.api.Test
import kotlin.test.assertEquals
import kotlin.test.assertFalse
import kotlin.test.assertTrue

class OsmRoutingEngineAdapterAntiRetraceTest {

    @Test
    fun `evaluate axis reuse outside start zone detects opposite traversal away from start`() {
        // GIVEN
        val adapter = OsmRoutingEngineAdapter()
        val start = Coordinates(lat = 48.13000, lng = -1.63000)
        val points = listOf(
            listOf(48.13000, -1.63000), // start
            listOf(48.15000, -1.63000), // far north
            listOf(48.15000, -1.62000), // far east
            listOf(48.15000, -1.63000), // reverse traversal on same far axis
            listOf(48.13000, -1.63000), // return start
        )

        // WHEN
        val (hasOpposite, maxReuse, oppositeRatio) = invokeEvaluateAxisReuseOutsideStartZone(
            adapter = adapter,
            points = points,
            start = start,
            startZoneMeters = 2000.0,
            minOppositeMeters = 120.0,
        )

        // THEN
        assertTrue(hasOpposite)
        assertTrue(maxReuse >= 2)
        assertTrue(oppositeRatio > 0.0)
    }

    @Test
    fun `evaluate axis reuse outside start zone detects same direction reuse and strict policy`() {
        // GIVEN
        val adapter = OsmRoutingEngineAdapter()
        val start = Coordinates(lat = 48.13000, lng = -1.63000)
        val points = listOf(
            listOf(48.13000, -1.63000), // start
            listOf(48.15600, -1.63000), // far north
            listOf(48.15600, -1.61800), // far east
            listOf(48.16000, -1.61200), // farther east
            listOf(48.16400, -1.62000), // turn south-west
            listOf(48.15600, -1.61800), // back near prior axis
            listOf(48.16000, -1.61200), // same axis reused in same direction
            listOf(48.13000, -1.63000), // return start
        )

        // WHEN
        val (hasOpposite, maxReuse, oppositeRatio) = invokeEvaluateAxisReuseOutsideStartZone(
            adapter = adapter,
            points = points,
            start = start,
            startZoneMeters = 2000.0,
            minOppositeMeters = 120.0,
        )
        val sameDirectionLimit = invokeOutsideStartAxisReuseLimit(adapter, routeType = "RIDE", strict = false)
        val oppositeLimit = invokeAllowedOppositeOutsideStartRatio(adapter, routeType = "RIDE", strict = false)

        // THEN
        assertFalse(hasOpposite)
        assertTrue(maxReuse >= 2)
        assertEquals(0.0, oppositeRatio, 1e-9)
        assertEquals(1, sameDirectionLimit)
        assertEquals(0.0, oppositeLimit, 1e-9)
        assertTrue(maxReuse > sameDirectionLimit)
    }

    @Suppress("UNCHECKED_CAST")
    private fun invokeEvaluateAxisReuseOutsideStartZone(
        adapter: OsmRoutingEngineAdapter,
        points: List<List<Double>>,
        start: Coordinates,
        startZoneMeters: Double,
        minOppositeMeters: Double,
    ): Triple<Boolean, Int, Double> {
        val method = adapter.javaClass.getDeclaredMethod(
            "evaluateAxisReuseOutsideStartZone",
            List::class.java,
            Coordinates::class.java,
            Double::class.javaPrimitiveType,
            Double::class.javaPrimitiveType,
        )
        method.isAccessible = true
        return method.invoke(adapter, points, start, startZoneMeters, minOppositeMeters) as Triple<Boolean, Int, Double>
    }

    private fun invokeOutsideStartAxisReuseLimit(
        adapter: OsmRoutingEngineAdapter,
        routeType: String,
        strict: Boolean,
    ): Int {
        val method = adapter.javaClass.getDeclaredMethod(
            "outsideStartAxisReuseLimit",
            String::class.java,
            Boolean::class.javaPrimitiveType,
        )
        method.isAccessible = true
        return method.invoke(adapter, routeType, strict) as Int
    }

    private fun invokeAllowedOppositeOutsideStartRatio(
        adapter: OsmRoutingEngineAdapter,
        routeType: String,
        strict: Boolean,
    ): Double {
        val method = adapter.javaClass.getDeclaredMethod(
            "allowedOppositeOutsideStartRatio",
            String::class.java,
            Boolean::class.javaPrimitiveType,
        )
        method.isAccessible = true
        return method.invoke(adapter, routeType, strict) as Double
    }
}
