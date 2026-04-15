package application

import (
	"mystravastats/domain/business"
	routesDomain "mystravastats/internal/routes/domain"
)

// RoutesReader is an outbound port used by routes explorer use cases.
type RoutesReader interface {
	FindRouteExplorerByYearAndTypes(
		year *int,
		request routesDomain.RouteExplorerRequest,
		activityTypes ...business.ActivityType,
	) routesDomain.RouteExplorerResult
}
