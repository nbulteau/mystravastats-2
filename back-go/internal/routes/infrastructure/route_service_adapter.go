package infrastructure

import (
	"strings"
	"time"

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

	if adapter.routingEngine == nil || request.StartPoint == nil {
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
	directionStrict := false
	if request.DirectionStrict != nil {
		directionStrict = *request.DirectionStrict
	}
	strictBacktracking := false
	if request.StrictBacktracking != nil {
		strictBacktracking = *request.StrictBacktracking
	}
	backtrackingProfile := ""
	if request.BacktrackingProfile != nil {
		backtrackingProfile = *request.BacktrackingProfile
	}
	shapePolyline := ""
	if request.ShapePolyline != nil {
		shapePolyline = strings.TrimSpace(*request.ShapePolyline)
	}
	limit := request.Limit
	if limit <= 0 {
		limit = 2
	}
	distanceTargetKm := 0.0
	if request.DistanceTargetKm != nil {
		distanceTargetKm = *request.DistanceTargetKm
	}
	engineRequest := routeApp.RoutingEngineRequest{
		StartPoint:          *request.StartPoint,
		DistanceTargetKm:    distanceTargetKm,
		ElevationTargetM:    request.ElevationTargetM,
		StartDirection:      startDirection,
		DirectionStrict:     directionStrict,
		StrictBacktracking:  strictBacktracking,
		BacktrackingProfile: backtrackingProfile,
		TargetMode:          targetMode,
		Waypoints:           request.CustomWaypoints,
		ShapePolyline:       shapePolyline,
		RouteType:           routeType,
		Limit:               limit,
		HistoryBiasEnabled:  routingHistoryBiasEnabled(),
	}
	if engineRequest.HistoryBiasEnabled {
		engineRequest.HistoryProfile = buildRoutingHistoryProfileFromActivities(
			activities,
			routeType,
			time.Now().UTC(),
			routingHistoryHalfLifeDays(),
		)
	}

	if shapePolyline != "" {
		generatedShapeLoops, err := adapter.routingEngine.GenerateShapeLoops(engineRequest)
		if err == nil && len(generatedShapeLoops) > 0 {
			result.ShapeMatches = generatedShapeLoops
		}
		return result
	}

	if distanceTargetKm <= 0 {
		return result
	}
	generatedLoops, err := adapter.routingEngine.GenerateTargetLoops(engineRequest)
	if err != nil {
		// Keep cache-derived road-graph fallbacks when OSRM is unavailable.
		if len(result.RoadGraphLoops) == 0 && len(result.ClosestLoops) > 0 {
			result.RoadGraphLoops = append([]routesDomain.RouteRecommendation{}, result.ClosestLoops...)
		}
		return result
	}
	if len(generatedLoops) == 0 {
		// Keep cache-derived road-graph fallbacks when OSRM returns no route.
		if len(result.RoadGraphLoops) == 0 && len(result.ClosestLoops) > 0 {
			result.RoadGraphLoops = append([]routesDomain.RouteRecommendation{}, result.ClosestLoops...)
		}
		return result
	}
	result.RoadGraphLoops = generatedLoops
	return result
}
