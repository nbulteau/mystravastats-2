package me.nicolas.stravastats.domain.services.routing

import me.nicolas.stravastats.domain.business.ActivityShort
import me.nicolas.stravastats.domain.business.ActivityType
import me.nicolas.stravastats.domain.business.Coordinates
import me.nicolas.stravastats.domain.business.RouteRecommendation
import me.nicolas.stravastats.domain.business.RouteVariantType
import org.junit.jupiter.api.Test
import kotlin.test.assertTrue

class OsmRoutingEngineAdapterStrictFallbackTest {

    @Test
    fun `strict backtracking rejects direction best effort candidate`() {
        val recommendation = recommendation("directional-safety-net")
        val candidate = OsrmRouteCandidate(
            recommendation = recommendation,
            directionPenalty = 0.34,
            backtrackingRatio = 0.17,
            corridorOverlap = 0.13,
            edgeReuseRatio = 0.12,
            maxAxisReuseCount = 1,
            maxAxisReuseRatio = 0.0,
            segmentDiversity = 0.18,
            distanceDeltaRatio = 0.23,
            pathRatio = 0.0,
            effectiveMatchScore = 68.0,
        )
        val request = request(startDirection = "N")

        val selected = selectCandidates(request, candidate)

        assertTrue(selected.isEmpty(), "direction fallback must not relax strict anti-retrace")
    }

    @Test
    fun `strict backtracking rejects emergency fallback candidate`() {
        val candidate = OsrmRouteCandidate(
            recommendation = recommendation("retrace-emergency"),
            directionPenalty = 0.0,
            backtrackingRatio = 0.50,
            corridorOverlap = 0.50,
            edgeReuseRatio = 0.50,
            maxAxisReuseCount = 20,
            maxAxisReuseRatio = 0.0,
            segmentDiversity = 0.05,
            distanceDeltaRatio = 0.10,
            pathRatio = 0.0,
            effectiveMatchScore = 90.0,
        )

        val selected = selectCandidates(request(startDirection = null), candidate)

        assertTrue(selected.isEmpty(), "emergency fallback must not relax strict anti-retrace")
    }

    private fun selectCandidates(
        request: RoutingEngineRequest,
        candidate: OsrmRouteCandidate,
    ): List<RouteRecommendation> {
        val adapter = OsmRoutingEngineAdapter()
        val method = adapter.javaClass.getDeclaredMethod(
            "selectCandidatesWithRelaxation",
            RoutingEngineRequest::class.java,
            List::class.java,
            MutableMap::class.java,
        )
        method.isAccessible = true
        @Suppress("UNCHECKED_CAST")
        return method.invoke(adapter, request, listOf(candidate), mutableMapOf<String, Int>()) as List<RouteRecommendation>
    }

    private fun request(startDirection: String?) = RoutingEngineRequest(
        startPoint = Coordinates(48.13, -1.63),
        distanceTargetKm = 40.0,
        elevationTargetM = null,
        startDirection = startDirection,
        strictBacktracking = true,
        routeType = "RIDE",
        limit = 1,
    )

    private fun recommendation(routeId: String) = RouteRecommendation(
        routeId = routeId,
        activity = ActivityShort(id = 0L, name = routeId, type = ActivityType.Ride),
        activityDate = "2026-01-01",
        distanceKm = 40.0,
        elevationGainM = 0.0,
        durationSec = 7200,
        isLoop = true,
        start = null,
        end = null,
        startArea = "Rennes",
        season = "SPRING",
        variantType = RouteVariantType.CLOSE_MATCH,
        matchScore = 80.0,
        reasons = emptyList(),
        previewLatLng = emptyList(),
        shape = null,
        shapeScore = null,
        experimental = false,
    )
}
