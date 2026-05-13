package me.nicolas.stravastats.domain.services.routing

import me.nicolas.stravastats.domain.business.Coordinates
import me.nicolas.stravastats.domain.business.RouteGenerationDiagnostic
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
    val historyBiasEnabled: Boolean = false,
    val historyProfile: RoutingHistoryProfile? = null,
)

data class RoutingHistoryProfile(
    val routeType: String,
    val halfLifeDays: Int,
    val activityCount: Int,
    val segmentCount: Int,
    val axisScores: Map<String, Double>,
    val zoneScores: Map<String, Double>,
    val latestActivityEpochMs: Long,
)

class RoutingEngineDiagnosticException(
    val code: String,
    override val message: String,
) : RuntimeException(message)

data class RoutingEngineEditRequest(
    val routeId: String,
    val routeType: String?,
    val controlPoints: List<Coordinates>,
)

data class RoutingEngineEditResult(
    val recommendation: RouteRecommendation? = null,
    val controlPoints: List<Coordinates> = emptyList(),
    val diagnostics: List<RouteGenerationDiagnostic> = emptyList(),
)

interface RoutingEnginePort {
    fun generateTargetLoops(request: RoutingEngineRequest): List<RouteRecommendation>
    fun generateShapeLoops(request: RoutingEngineRequest): List<RouteRecommendation>
    fun editRoute(request: RoutingEngineEditRequest): RoutingEngineEditResult {
        return RoutingEngineEditResult(
            diagnostics = listOf(
                RouteGenerationDiagnostic(
                    code = "EDIT_ENGINE_UNAVAILABLE",
                    message = "OSRM route editing is not available for this routing engine.",
                ),
            ),
        )
    }
    fun healthDetails(): Map<String, Any?>
}
