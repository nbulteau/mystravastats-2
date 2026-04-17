package application

import (
	routesDomain "mystravastats/internal/routes/domain"
	"mystravastats/internal/shared/domain/business"
)

// RoutesReader is an outbound port used by routes explorer use cases.
type RoutesReader interface {
	FindRouteExplorerByYearAndTypes(
		year *int,
		request routesDomain.RouteExplorerRequest,
		activityTypes ...business.ActivityType,
	) routesDomain.RouteExplorerResult
}

// RoutingEngineRequest captures the minimum information needed to generate
// road-graph loops from an external routing engine.
type RoutingEngineRequest struct {
	StartPoint       routesDomain.Coordinates
	DistanceTargetKm float64
	ElevationTargetM *float64
	StartDirection   string
	RouteType        string
	Limit            int
}

// RoutingEnginePort is an outbound port for external routing engines
// (OSRM, GraphHopper, ...).
type RoutingEnginePort interface {
	GenerateTargetLoops(request RoutingEngineRequest) ([]routesDomain.RouteRecommendation, error)
	HealthDetails() map[string]any
}
