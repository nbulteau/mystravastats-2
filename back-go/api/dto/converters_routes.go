package dto

import (
	"mystravastats/domain/business"
	routesDomain "mystravastats/internal/routes/domain"
)

func ToRouteExplorerResultDto(result routesDomain.RouteExplorerResult) RouteExplorerResultDto {
	return RouteExplorerResultDto{
		ClosestLoops:   toRouteRecommendationDtos(result.ClosestLoops),
		Variants:       toRouteRecommendationDtos(result.Variants),
		Seasonal:       toRouteRecommendationDtos(result.Seasonal),
		RoadGraphLoops: toRouteRecommendationDtos(result.RoadGraphLoops),
		ShapeMatches:   toRouteRecommendationDtos(result.ShapeMatches),
		ShapeRemixes:   toShapeRemixRecommendationDtos(result.ShapeRemixes),
	}
}

func toRouteRecommendationDtos(recommendations []routesDomain.RouteRecommendation) []RouteRecommendationDto {
	if len(recommendations) == 0 {
		return []RouteRecommendationDto{}
	}

	result := make([]RouteRecommendationDto, len(recommendations))
	for index, recommendation := range recommendations {
		result[index] = RouteRecommendationDto{
			RouteID:        recommendation.RouteID,
			Activity:       toActivityShortDto(recommendation.Activity),
			ActivityDate:   recommendation.ActivityDate,
			DistanceKm:     recommendation.DistanceKm,
			ElevationGainM: recommendation.ElevationGainM,
			DurationSec:    recommendation.DurationSec,
			IsLoop:         recommendation.IsLoop,
			Start:          toRouteCoordinateDto(recommendation.Start),
			End:            toRouteCoordinateDto(recommendation.End),
			StartArea:      recommendation.StartArea,
			Season:         recommendation.Season,
			VariantType:    string(recommendation.VariantType),
			MatchScore:     recommendation.MatchScore,
			Reasons:        append([]string(nil), recommendation.Reasons...),
			PreviewLatLng:  recommendation.PreviewLatLng,
			Shape:          recommendation.Shape,
			ShapeScore:     recommendation.ShapeScore,
			Experimental:   recommendation.Experimental,
		}
	}

	return result
}

func toShapeRemixRecommendationDtos(recommendations []routesDomain.ShapeRemixRecommendation) []ShapeRemixRecommendationDto {
	if len(recommendations) == 0 {
		return []ShapeRemixRecommendationDto{}
	}

	result := make([]ShapeRemixRecommendationDto, len(recommendations))
	for index, recommendation := range recommendations {
		result[index] = ShapeRemixRecommendationDto{
			ID:             recommendation.ID,
			Shape:          recommendation.Shape,
			DistanceKm:     recommendation.DistanceKm,
			ElevationGainM: recommendation.ElevationGainM,
			DurationSec:    recommendation.DurationSec,
			MatchScore:     recommendation.MatchScore,
			Reasons:        append([]string(nil), recommendation.Reasons...),
			Components:     toActivityShortDtos(recommendation.Components),
			PreviewLatLng:  recommendation.PreviewLatLng,
			Experimental:   recommendation.Experimental,
		}
	}

	return result
}

func toRouteCoordinateDto(value *routesDomain.Coordinates) *RouteCoordinateDto {
	if value == nil {
		return nil
	}
	return &RouteCoordinateDto{
		Lat: value.Lat,
		Lng: value.Lng,
	}
}

func toActivityShortDtos(values []business.ActivityShort) []ActivityShortDto {
	if len(values) == 0 {
		return []ActivityShortDto{}
	}

	result := make([]ActivityShortDto, len(values))
	for index, value := range values {
		result[index] = toActivityShortDto(value)
	}
	return result
}

func toActivityShortDto(value business.ActivityShort) ActivityShortDto {
	return ActivityShortDto{
		ID:   value.Id,
		Name: value.Name,
		Type: value.Type.String(),
	}
}
