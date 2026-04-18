package me.nicolas.stravastats.domain.services.routing

import me.nicolas.stravastats.domain.business.Coordinates
import me.nicolas.stravastats.domain.business.RouteRecommendation

data class RoutingEngineRequest(
    val startPoint: Coordinates,
    val distanceTargetKm: Double,
    val elevationTargetM: Double?,
    val startDirection: String?,
    val directionStrict: Boolean = false,
    val strictBacktracking: Boolean = false,
    val backtrackingProfile: String? = null,
    val targetMode: String? = null,
    val waypoints: List<Coordinates> = emptyList(),
    val shapePolyline: String? = null,
    val routeType: String?,
    val limit: Int,
)

interface RoutingEnginePort {
    fun generateTargetLoops(request: RoutingEngineRequest): List<RouteRecommendation>
    fun generateShapeLoops(request: RoutingEngineRequest): List<RouteRecommendation>
    fun healthDetails(): Map<String, Any?>
}
