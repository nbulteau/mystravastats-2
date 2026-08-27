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
        val start = Coordinates(lat = 48.13000, lng = -1.63000)
        val points = listOf(
            listOf(48.13000, -1.63000), // start
            listOf(48.15000, -1.63000), // far north
            listOf(48.15000, -1.62000), // far east
            listOf(48.15000, -1.63000), // reverse traversal on same far axis
            listOf(48.13000, -1.63000), // return start
        )

        // WHEN
        val (hasOpposite, maxReuse, oppositeRatio) = evaluateAxisReuseOutsideStartZone(
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
        val (hasOpposite, maxReuse, oppositeRatio) = evaluateAxisReuseOutsideStartZone(
            points = points,
            start = start,
            startZoneMeters = 2000.0,
            minOppositeMeters = 120.0,
        )
        val sameDirectionLimit = outsideStartAxisReuseLimit(routeType = "RIDE", strict = false)
        val oppositeLimit = allowedOppositeOutsideStartRatio(routeType = "RIDE", strict = false)

        // THEN
        assertFalse(hasOpposite)
        assertTrue(maxReuse >= 2)
        assertEquals(0.0, oppositeRatio, 1e-9)
        assertEquals(1, sameDirectionLimit)
        assertEquals(0.0, oppositeLimit, 1e-9)
        assertTrue(maxReuse > sameDirectionLimit)
    }

    @Test
    fun `evaluate axis reuse outside start zone counts long segment crossing hub boundary`() {
        // GIVEN
        val start = Coordinates(lat = 48.13000, lng = -1.63000)
        val points = listOf(
            listOf(48.13000, -1.63000), // start
            listOf(48.17000, -1.63000), // far north (~4.4km)
            listOf(48.13000, -1.63000), // retrace same axis back to start
        )

        // WHEN
        val (hasOpposite, maxReuse, oppositeRatio) = evaluateAxisReuseOutsideStartZone(
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
    fun `evaluate axis reuse outside start zone keeps local hub reuse allowed`() {
        // GIVEN
        val start = Coordinates(lat = 48.13000, lng = -1.63000)
        val points = listOf(
            listOf(48.13000, -1.63000), // start
            listOf(48.13600, -1.63000), // ~660m north (inside 2km hub)
            listOf(48.13000, -1.63000), // back
            listOf(48.13600, -1.63000), // same local axis again
            listOf(48.13000, -1.63000), // back
        )

        // WHEN
        val (hasOpposite, maxReuse, oppositeRatio) = evaluateAxisReuseOutsideStartZone(
            points = points,
            start = start,
            startZoneMeters = 2000.0,
            minOppositeMeters = 120.0,
        )

        // THEN
        assertFalse(hasOpposite)
        assertEquals(0, maxReuse)
        assertEquals(0.0, oppositeRatio, 1e-9)
    }

}
