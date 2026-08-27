package me.nicolas.stravastats.domain.services.routing

import org.junit.jupiter.api.Test
import kotlin.test.assertEquals
import kotlin.test.assertTrue

class OsmRoutingEngineAdapterSurfaceTest {

    @Test
    fun `classify surface bucket uses surface and tracktype tags when available`() {
        // GIVEN
        val stepFromSurfaceClass = OsrmStep(
            distance = 1000.0,
            mode = "cycling",
            classes = listOf("surface=asphalt"),
            surface = null,
            tracktype = null,
        )
        val stepFromSurfaceTag = OsrmStep(
            distance = 1000.0,
            mode = "cycling",
            classes = emptyList(),
            surface = "surface:fine_gravel",
            tracktype = null,
        )
        val stepFromTrackTypeClass = OsrmStep(
            distance = 1000.0,
            mode = "cycling",
            classes = listOf("tracktype=grade4"),
            surface = null,
            tracktype = null,
        )
        val stepFromTrackTypeTag = OsrmStep(
            distance = 1000.0,
            mode = "cycling",
            classes = emptyList(),
            surface = null,
            tracktype = "tracktype=grade3",
        )

        // WHEN
        val pavedBucket = classifySurfaceBucket(stepFromSurfaceClass)
        val gravelBucket = classifySurfaceBucket(stepFromSurfaceTag)
        val trailBucket = classifySurfaceBucket(stepFromTrackTypeClass)
        val gravelFromTrackTypeBucket = classifySurfaceBucket(stepFromTrackTypeTag)

        // THEN
        assertEquals("paved", pavedBucket)
        assertEquals("gravel", gravelBucket)
        assertEquals("trail", trailBucket)
        assertEquals("gravel", gravelFromTrackTypeBucket)
    }

    @Test
    fun `surface match score adapts to requested route type`() {
        // GIVEN
        val mixedBreakdown = RouteSurfaceBreakdown(
            pavedM = 3500.0,
            gravelM = 5500.0,
            trailM = 1000.0,
            unknownM = 0.0,
        )
        val trailBreakdown = RouteSurfaceBreakdown(
            pavedM = 800.0,
            gravelM = 2900.0,
            trailM = 6300.0,
            unknownM = 0.0,
        )

        // WHEN
        val gravelScore = surfaceMatchScore("GRAVEL", mixedBreakdown)
        val rideScoreOnMixed = surfaceMatchScore("RIDE", mixedBreakdown)
        val mtbScoreOnTrail = surfaceMatchScore("MTB", trailBreakdown)
        val rideScoreOnTrail = surfaceMatchScore("RIDE", trailBreakdown)

        // THEN
        assertTrue(gravelScore > rideScoreOnMixed)
        assertTrue(mtbScoreOnTrail > rideScoreOnTrail)
    }

    @Test
    fun `required path ratio keeps gravel minimum and ride at zero`() {
        // GIVEN
        // WHEN
        val gravelRequiredPathRatio = requiredPathRatioForRequest("GRAVEL", strict = false)
        val rideRequiredPathRatio = requiredPathRatioForRequest("RIDE", strict = false)

        // THEN
        assertEquals(0.25, gravelRequiredPathRatio, 1e-9)
        assertEquals(0.0, rideRequiredPathRatio, 1e-9)
    }

}
