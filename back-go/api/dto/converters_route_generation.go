package dto

import (
	routesDomain "mystravastats/internal/routes/domain"
	"strings"
)

func ToGeneratedRouteDto(
	recommendation routesDomain.RouteRecommendation,
	score RouteGenerationScoreDto,
	routeType string,
) GeneratedRouteDto {
	var activityID *int64
	if recommendation.Activity.Id != 0 {
		id := recommendation.Activity.Id
		activityID = &id
	}

	title := recommendation.Activity.Name
	if title == "" {
		title = recommendation.RouteID
	}

	return GeneratedRouteDto{
		RouteID:              recommendation.RouteID,
		Title:                title,
		VariantType:          string(recommendation.VariantType),
		RouteType:            routeType,
		DistanceKm:           recommendation.DistanceKm,
		ElevationGainM:       recommendation.ElevationGainM,
		DurationSec:          recommendation.DurationSec,
		EstimatedDurationSec: recommendation.DurationSec,
		Score:                score,
		Reasons:              append([]string(nil), recommendation.Reasons...),
		PreviewLatLng:        recommendation.PreviewLatLng,
		Start:                toRouteCoordinateDto(recommendation.Start),
		End:                  toRouteCoordinateDto(recommendation.End),
		ActivityID:           activityID,
		IsRoadGraphGenerated: isRoadGraphGeneratedRecommendation(recommendation),
	}
}

func isRoadGraphGeneratedRecommendation(recommendation routesDomain.RouteRecommendation) bool {
	if recommendation.VariantType == routesDomain.RouteVariantRoadGraph {
		return true
	}
	for _, reason := range recommendation.Reasons {
		if strings.TrimSpace(reason) == "Generated with OSM road graph (OSRM)" {
			return true
		}
	}
	return false
}

func ToGeneratedRouteFromShapeRemixDto(
	remix routesDomain.ShapeRemixRecommendation,
	score RouteGenerationScoreDto,
	routeType string,
) GeneratedRouteDto {
	title := remix.ID
	if len(remix.Components) > 0 && remix.Components[0].Name != "" {
		title = remix.Components[0].Name
	}

	return GeneratedRouteDto{
		RouteID:              remix.ID,
		Title:                title,
		VariantType:          string(routesDomain.RouteVariantShapeMix),
		RouteType:            routeType,
		DistanceKm:           remix.DistanceKm,
		ElevationGainM:       remix.ElevationGainM,
		DurationSec:          remix.DurationSec,
		EstimatedDurationSec: remix.DurationSec,
		Score:                score,
		Reasons:              append([]string(nil), remix.Reasons...),
		PreviewLatLng:        remix.PreviewLatLng,
		Start:                nil,
		End:                  nil,
		ActivityID:           nil,
		IsRoadGraphGenerated: false,
	}
}
