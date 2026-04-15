package application

import (
	"mystravastats/domain/business"
	routesDomain "mystravastats/internal/routes/domain"
)

type GetRouteExplorerUseCase struct {
	reader RoutesReader
}

func NewGetRouteExplorerUseCase(reader RoutesReader) *GetRouteExplorerUseCase {
	return &GetRouteExplorerUseCase{
		reader: reader,
	}
}

func (uc *GetRouteExplorerUseCase) Execute(
	year *int,
	request routesDomain.RouteExplorerRequest,
	activityTypes []business.ActivityType,
) routesDomain.RouteExplorerResult {
	result := uc.reader.FindRouteExplorerByYearAndTypes(year, request, activityTypes...)

	if result.ClosestLoops == nil {
		result.ClosestLoops = []routesDomain.RouteRecommendation{}
	}
	if result.Variants == nil {
		result.Variants = []routesDomain.RouteRecommendation{}
	}
	if result.Seasonal == nil {
		result.Seasonal = []routesDomain.RouteRecommendation{}
	}
	if result.ShapeMatches == nil {
		result.ShapeMatches = []routesDomain.RouteRecommendation{}
	}
	if result.ShapeRemixes == nil {
		result.ShapeRemixes = []routesDomain.ShapeRemixRecommendation{}
	}

	return result
}
