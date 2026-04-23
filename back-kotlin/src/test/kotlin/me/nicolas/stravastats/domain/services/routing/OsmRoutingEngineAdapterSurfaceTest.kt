package me.nicolas.stravastats.domain.services.routing

import org.junit.jupiter.api.Test
import kotlin.test.assertEquals
import kotlin.test.assertTrue

class OsmRoutingEngineAdapterSurfaceTest {

    @Test
    fun `classify surface bucket uses surface and tracktype tags when available`() {
        // GIVEN
        val adapter = OsmRoutingEngineAdapter()
        val stepFromSurfaceClass = newOsrmStep(
            distance = 1000.0,
            mode = "cycling",
            classes = listOf("surface=asphalt"),
            surface = null,
            trackType = null,
        )
        val stepFromSurfaceTag = newOsrmStep(
            distance = 1000.0,
            mode = "cycling",
            classes = emptyList(),
            surface = "surface:fine_gravel",
            trackType = null,
        )
        val stepFromTrackTypeClass = newOsrmStep(
            distance = 1000.0,
            mode = "cycling",
            classes = listOf("tracktype=grade4"),
            surface = null,
            trackType = null,
        )
        val stepFromTrackTypeTag = newOsrmStep(
            distance = 1000.0,
            mode = "cycling",
            classes = emptyList(),
            surface = null,
            trackType = "tracktype=grade3",
        )

        // WHEN
        val pavedBucket = invokeClassifySurfaceBucket(adapter, stepFromSurfaceClass)
        val gravelBucket = invokeClassifySurfaceBucket(adapter, stepFromSurfaceTag)
        val trailBucket = invokeClassifySurfaceBucket(adapter, stepFromTrackTypeClass)
        val gravelFromTrackTypeBucket = invokeClassifySurfaceBucket(adapter, stepFromTrackTypeTag)

        // THEN
        assertEquals("paved", pavedBucket)
        assertEquals("gravel", gravelBucket)
        assertEquals("trail", trailBucket)
        assertEquals("gravel", gravelFromTrackTypeBucket)
    }

    @Test
    fun `surface match score adapts to requested route type`() {
        // GIVEN
        val adapter = OsmRoutingEngineAdapter()
        val mixedBreakdown = newRouteSurfaceBreakdown(
            pavedM = 3500.0,
            gravelM = 5500.0,
            trailM = 1000.0,
            unknownM = 0.0,
        )
        val trailBreakdown = newRouteSurfaceBreakdown(
            pavedM = 800.0,
            gravelM = 2900.0,
            trailM = 6300.0,
            unknownM = 0.0,
        )

        // WHEN
        val gravelScore = invokeSurfaceMatchScore(adapter, "GRAVEL", mixedBreakdown)
        val rideScoreOnMixed = invokeSurfaceMatchScore(adapter, "RIDE", mixedBreakdown)
        val mtbScoreOnTrail = invokeSurfaceMatchScore(adapter, "MTB", trailBreakdown)
        val rideScoreOnTrail = invokeSurfaceMatchScore(adapter, "RIDE", trailBreakdown)

        // THEN
        assertTrue(gravelScore > rideScoreOnMixed)
        assertTrue(mtbScoreOnTrail > rideScoreOnTrail)
    }

    @Test
    fun `required path ratio keeps gravel minimum and ride at zero`() {
        // GIVEN
        val adapter = OsmRoutingEngineAdapter()

        // WHEN
        val gravelRequiredPathRatio = invokeRequiredPathRatioForRequest(adapter, "GRAVEL", strict = false)
        val rideRequiredPathRatio = invokeRequiredPathRatioForRequest(adapter, "RIDE", strict = false)

        // THEN
        assertEquals(0.25, gravelRequiredPathRatio, 1e-9)
        assertEquals(0.0, rideRequiredPathRatio, 1e-9)
    }

    private fun invokeClassifySurfaceBucket(adapter: OsmRoutingEngineAdapter, step: Any): String {
        val stepClass = Class.forName("me.nicolas.stravastats.domain.services.routing.OsrmStep")
        val method = adapter.javaClass.getDeclaredMethod("classifySurfaceBucket", stepClass)
        method.isAccessible = true
        return method.invoke(adapter, step) as String
    }

    private fun invokeSurfaceMatchScore(
        adapter: OsmRoutingEngineAdapter,
        routeType: String,
        breakdown: Any,
    ): Double {
        val breakdownClass = Class.forName("me.nicolas.stravastats.domain.services.routing.RouteSurfaceBreakdown")
        val method = adapter.javaClass.getDeclaredMethod("surfaceMatchScore", String::class.java, breakdownClass)
        method.isAccessible = true
        return method.invoke(adapter, routeType, breakdown) as Double
    }

    private fun invokeRequiredPathRatioForRequest(
        adapter: OsmRoutingEngineAdapter,
        routeType: String,
        strict: Boolean,
    ): Double {
        val method = adapter.javaClass.getDeclaredMethod(
            "requiredPathRatioForRequest",
            String::class.java,
            Boolean::class.javaPrimitiveType,
        )
        method.isAccessible = true
        return method.invoke(adapter, routeType, strict) as Double
    }

    private fun newOsrmStep(
        distance: Double,
        mode: String?,
        classes: List<String>,
        surface: String?,
        trackType: String?,
    ): Any {
        val stepClass = Class.forName("me.nicolas.stravastats.domain.services.routing.OsrmStep")
        val constructor = stepClass.getDeclaredConstructor(
            Double::class.javaPrimitiveType,
            String::class.java,
            List::class.java,
            String::class.java,
            String::class.java,
        )
        constructor.isAccessible = true
        return constructor.newInstance(distance, mode, classes, surface, trackType)
    }

    private fun newRouteSurfaceBreakdown(
        pavedM: Double,
        gravelM: Double,
        trailM: Double,
        unknownM: Double,
    ): Any {
        val breakdownClass = Class.forName("me.nicolas.stravastats.domain.services.routing.RouteSurfaceBreakdown")
        val constructor = breakdownClass.getDeclaredConstructor(
            Double::class.javaPrimitiveType,
            Double::class.javaPrimitiveType,
            Double::class.javaPrimitiveType,
            Double::class.javaPrimitiveType,
        )
        constructor.isAccessible = true
        return constructor.newInstance(pavedM, gravelM, trailM, unknownM)
    }
}
