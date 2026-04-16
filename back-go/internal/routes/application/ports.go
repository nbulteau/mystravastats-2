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
