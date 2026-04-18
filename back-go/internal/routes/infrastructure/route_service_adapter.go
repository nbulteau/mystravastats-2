package infrastructure

import (
	"mystravastats/internal/platform/activityprovider"
	routeApp "mystravastats/internal/routes/application"
	routesDomain "mystravastats/internal/routes/domain"
	"mystravastats/internal/shared/domain/business"
)

// RouteServiceAdapter computes route explorer recommendations from cached activities.
type RouteServiceAdapter struct {
	routingEngine routeApp.RoutingEnginePort
}

func NewRouteServiceAdapter(routingEngine routeApp.RoutingEnginePort) *RouteServiceAdapter {
	return &RouteServiceAdapter{
		routingEngine: routingEngine,
	}
}

func (adapter *RouteServiceAdapter) FindRouteExplorerByYearAndTypes(
	year *int,
	request routesDomain.RouteExplorerRequest,
	activityTypes ...business.ActivityType,
) routesDomain.RouteExplorerResult {
	activities := activityprovider.Get().GetActivitiesByYearAndActivityTypes(year, activityTypes...)
	result := computeRouteExplorerFromActivities(activities, request)

	if adapter.routingEngine == nil || request.StartPoint == nil || request.DistanceTargetKm == nil {
		return result
	}
	if *request.DistanceTargetKm <= 0 {
		return result
	}

	routeType := ""
	if request.RouteType != nil {
		routeType = *request.RouteType
	}
	startDirection := ""
	if request.StartDirection != nil {
		startDirection = *request.StartDirection
	}
	targetMode := ""
	if request.TargetMode != nil {
		targetMode = *request.TargetMode
	}
	limit := request.Limit
	if limit <= 0 {
		limit = 2
	}

	generatedLoops, err := adapter.routingEngine.GenerateTargetLoops(routeApp.RoutingEngineRequest{
		StartPoint:       *request.StartPoint,
		DistanceTargetKm: *request.DistanceTargetKm,
		ElevationTargetM: request.ElevationTargetM,
		StartDirection:   startDirection,
		TargetMode:       targetMode,
		Waypoints:        request.CustomWaypoints,
		RouteType:        routeType,
		Limit:            limit,
	})
	if err != nil {
		result.RoadGraphLoops = []routesDomain.RouteRecommendation{}
		return result
	}
	if len(generatedLoops) == 0 {
		result.RoadGraphLoops = []routesDomain.RouteRecommendation{}
		return result
	}
	result.RoadGraphLoops = generatedLoops
	return result
}
