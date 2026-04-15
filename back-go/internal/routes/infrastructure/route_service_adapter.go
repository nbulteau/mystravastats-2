package infrastructure

import (
	"mystravastats/domain/business"
	"mystravastats/internal/platform/activityprovider"
	routesDomain "mystravastats/internal/routes/domain"
)

// RouteServiceAdapter computes route explorer recommendations from cached activities.
type RouteServiceAdapter struct{}

func NewRouteServiceAdapter() *RouteServiceAdapter {
	return &RouteServiceAdapter{}
}

func (adapter *RouteServiceAdapter) FindRouteExplorerByYearAndTypes(
	year *int,
	request routesDomain.RouteExplorerRequest,
	activityTypes ...business.ActivityType,
) routesDomain.RouteExplorerResult {
	activities := activityprovider.Get().GetActivitiesByYearAndActivityTypes(year, activityTypes...)
	return computeRouteExplorerFromActivities(activities, request)
}
