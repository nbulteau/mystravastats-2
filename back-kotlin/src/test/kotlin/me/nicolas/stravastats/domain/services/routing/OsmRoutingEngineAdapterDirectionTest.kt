package me.nicolas.stravastats.domain.services.routing

import me.nicolas.stravastats.domain.business.Coordinates
import org.junit.jupiter.api.Test
import kotlin.test.assertEquals
import kotlin.test.assertTrue

class OsmRoutingEngineAdapterDirectionTest {

    @Test
    fun `far opposite violation ratio ignores local oscillation but catches distant opposite excursion`() {
        // GIVEN
        val start = Coordinates(lat = 48.13000, lng = -1.63000)
        val mostlyNorthWithLocalOscillation = listOf(
            listOf(48.13000, -1.63000),
            listOf(48.13220, -1.62950),
            listOf(48.12995, -1.62980), // local oscillation around start
            listOf(48.13500, -1.62850),
            listOf(48.13800, -1.62700),
            listOf(48.13000, -1.63000),
        )
        val farSouthExcursion = listOf(
            listOf(48.13000, -1.63000),
            listOf(48.13300, -1.62950),
            listOf(48.13600, -1.62800),
            listOf(48.12100, -1.62720), // far opposite
            listOf(48.11850, -1.62680), // far opposite
            listOf(48.13450, -1.62830),
            listOf(48.13000, -1.63000),
        )

        // WHEN
        val cleanPenalty = farOppositeViolationRatio(
            points = mostlyNorthWithLocalOscillation,
            start = start,
            direction = "N",
            toleranceMeters = 120.0,
        )
        val oppositePenalty = farOppositeViolationRatio(
            points = farSouthExcursion,
            start = start,
            direction = "N",
            toleranceMeters = 120.0,
        )

        // THEN
        assertEquals(0.0, cleanPenalty, 1e-9)
        assertTrue(oppositePenalty > 0.0)
    }

    @Test
    fun `combined direction penalty increases when route makes far opposite excursion`() {
        // GIVEN
        val start = Coordinates(lat = 48.13000, lng = -1.63000)
        val northDominant = listOf(
            listOf(48.13000, -1.63000),
            listOf(48.13220, -1.62950),
            listOf(48.13450, -1.62840),
            listOf(48.13680, -1.62710),
            listOf(48.13300, -1.62830),
            listOf(48.13000, -1.63000),
        )
        val northWithFarSouthExcursion = listOf(
            listOf(48.13000, -1.63000),
            listOf(48.13220, -1.62950),
            listOf(48.13600, -1.62800),
            listOf(48.12100, -1.62720), // far opposite
            listOf(48.11850, -1.62680), // far opposite
            listOf(48.13500, -1.62820),
            listOf(48.13000, -1.63000),
        )

        // WHEN
        val cleanPenalty = combinedDirectionPenalty(
            points = northDominant,
            start = start,
            direction = "N",
            toleranceMeters = 120.0,
        )
        val excursionPenalty = combinedDirectionPenalty(
            points = northWithFarSouthExcursion,
            start = start,
            direction = "N",
            toleranceMeters = 120.0,
        )

        // THEN
        assertTrue(excursionPenalty > cleanPenalty)
    }

    @Test
    fun `directional quadrant penalty penalizes opposite-quadrant majority`() {
        // GIVEN
        val start = Coordinates(lat = 48.13000, lng = -1.63000)
        val northMajority = listOf(
            listOf(48.13000, -1.63000),
            listOf(48.13700, -1.62980),
            listOf(48.14100, -1.62830),
            listOf(48.13800, -1.62740),
            listOf(48.13300, -1.62860),
            listOf(48.13000, -1.63000),
        )
        val southMajority = listOf(
            listOf(48.13000, -1.63000),
            listOf(48.12700, -1.62970),
            listOf(48.12100, -1.62820),
            listOf(48.11800, -1.62740),
            listOf(48.12400, -1.62850),
            listOf(48.13000, -1.63000),
        )

        // WHEN
        val northPenalty = directionalQuadrantPenalty(
            points = northMajority,
            start = start,
            direction = "N",
            toleranceMeters = 120.0,
        )
        val southPenalty = directionalQuadrantPenalty(
            points = southMajority,
            start = start,
            direction = "N",
            toleranceMeters = 120.0,
        )

        // THEN
        assertTrue(northPenalty < southPenalty)
        assertTrue(southPenalty > 0.0)
    }

    @Test
    fun `combined direction penalty increases when quadrant majority is opposite`() {
        // GIVEN
        val start = Coordinates(lat = 48.13000, lng = -1.63000)
        val northMajority = listOf(
            listOf(48.13000, -1.63000),
            listOf(48.13700, -1.62980),
            listOf(48.14100, -1.62830),
            listOf(48.13800, -1.62740),
            listOf(48.13300, -1.62860),
            listOf(48.13000, -1.63000),
        )
        val southMajority = listOf(
            listOf(48.13000, -1.63000),
            listOf(48.12700, -1.62970),
            listOf(48.12100, -1.62820),
            listOf(48.11800, -1.62740),
            listOf(48.12400, -1.62850),
            listOf(48.13000, -1.63000),
        )

        // WHEN
        val northPenalty = combinedDirectionPenalty(
            points = northMajority,
            start = start,
            direction = "N",
            toleranceMeters = 120.0,
        )
        val southPenalty = combinedDirectionPenalty(
            points = southMajority,
            start = start,
            direction = "N",
            toleranceMeters = 120.0,
        )

        // THEN
        assertTrue(southPenalty > northPenalty)
    }

}
