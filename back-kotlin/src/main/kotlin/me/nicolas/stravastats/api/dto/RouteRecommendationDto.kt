package me.nicolas.stravastats.api.dto

import io.swagger.v3.oas.annotations.media.Schema
import me.nicolas.stravastats.domain.business.Coordinates
import me.nicolas.stravastats.domain.business.RouteExplorerResult
import me.nicolas.stravastats.domain.business.RouteRecommendation
import me.nicolas.stravastats.domain.business.ShapeRemixRecommendation

@Schema(description = "Route coordinate", name = "RouteCoordinate")
data class RouteCoordinateDto(
    val lat: Double,
    val lng: Double,
)

@Schema(description = "Route recommendation", name = "RouteRecommendation")
data class RouteRecommendationDto(
    val routeId: String,
    val activity: ActivityShortDto,
    val activityDate: String,
    val distanceKm: Double,
    val elevationGainM: Double,
    val durationSec: Int,
    val isLoop: Boolean,
    val start: RouteCoordinateDto?,
    val end: RouteCoordinateDto?,
    val startArea: String,
    val season: String,
    val variantType: String,
    val matchScore: Double,
    val reasons: List<String>,
    val previewLatLng: List<List<Double>>,
    val shape: String?,
    val shapeScore: Double?,
    val experimental: Boolean,
)

@Schema(description = "Shape remix recommendation", name = "ShapeRemixRecommendation")
data class ShapeRemixRecommendationDto(
    val id: String,
    val shape: String,
    val distanceKm: Double,
    val elevationGainM: Double,
    val durationSec: Int,
    val matchScore: Double,
    val reasons: List<String>,
    val components: List<ActivityShortDto>,
    val previewLatLng: List<List<Double>>,
    val experimental: Boolean,
)

@Schema(description = "Routes explorer response", name = "RouteExplorerResult")
data class RouteExplorerResultDto(
    val closestLoops: List<RouteRecommendationDto>,
    val variants: List<RouteRecommendationDto>,
    val seasonal: List<RouteRecommendationDto>,
    val roadGraphLoops: List<RouteRecommendationDto>,
    val shapeMatches: List<RouteRecommendationDto>,
    val shapeRemixes: List<ShapeRemixRecommendationDto>,
)

@Schema(description = "Route generation score", name = "RouteGenerationScore")
data class RouteGenerationScoreDto(
    val global: Double,
    val distance: Double,
    val elevation: Double,
    val duration: Double,
    val direction: Double,
    val shape: Double,
    val roadFitness: Double,
)

@Schema(description = "Generated route", name = "GeneratedRoute")
data class GeneratedRouteDto(
    val routeId: String,
    val title: String,
    val variantType: String,
    val routeType: String?,
    val startDirection: String?,
    val distanceKm: Double,
    val elevationGainM: Double,
    val durationSec: Int,
    val estimatedDurationSec: Int,
    val score: RouteGenerationScoreDto,
    val reasons: List<String>,
    val previewLatLng: List<List<Double>>,
    val start: RouteCoordinateDto?,
    val end: RouteCoordinateDto?,
    val activityId: Long?,
    val isRoadGraphGenerated: Boolean,
)

@Schema(description = "Generated routes response", name = "GenerateRoutesResponse")
data class GenerateRoutesResponseDto(
    val routes: List<GeneratedRouteDto>,
)

data class RouteStartPointDto(
    val lat: Double,
    val lng: Double,
)

data class GenerateTargetRoutesRequestDto(
    val startPoint: RouteStartPointDto?,
    val routeType: String?,
    val startDirection: String?,
    val distanceTargetKm: Double?,
    val elevationTargetM: Double?,
    val variantCount: Int?,
)

data class GenerateShapeRoutesRequestDto(
    val shapeInputType: String?,
    val shapeData: String?,
    val startPoint: RouteStartPointDto?,
    val distanceTargetKm: Double?,
    val elevationTargetM: Double?,
    val routeType: String?,
    val variantCount: Int?,
)

fun RouteExplorerResult.toDto(): RouteExplorerResultDto {
    return RouteExplorerResultDto(
        closestLoops = closestLoops.map { recommendation -> recommendation.toDto() },
        variants = variants.map { recommendation -> recommendation.toDto() },
        seasonal = seasonal.map { recommendation -> recommendation.toDto() },
        roadGraphLoops = roadGraphLoops.map { recommendation -> recommendation.toDto() },
        shapeMatches = shapeMatches.map { recommendation -> recommendation.toDto() },
        shapeRemixes = shapeRemixes.map { recommendation -> recommendation.toDto() },
    )
}

private fun RouteRecommendation.toDto(): RouteRecommendationDto {
    return RouteRecommendationDto(
        routeId = routeId,
        activity = activity.toDto(),
        activityDate = activityDate,
        distanceKm = distanceKm,
        elevationGainM = elevationGainM,
        durationSec = durationSec,
        isLoop = isLoop,
        start = start?.toDto(),
        end = end?.toDto(),
        startArea = startArea,
        season = season,
        variantType = variantType.name,
        matchScore = matchScore,
        reasons = reasons,
        previewLatLng = previewLatLng,
        shape = shape,
        shapeScore = shapeScore,
        experimental = experimental,
    )
}

private fun ShapeRemixRecommendation.toDto(): ShapeRemixRecommendationDto {
    return ShapeRemixRecommendationDto(
        id = id,
        shape = shape,
        distanceKm = distanceKm,
        elevationGainM = elevationGainM,
        durationSec = durationSec,
        matchScore = matchScore,
        reasons = reasons,
        components = components.map { activity -> activity.toDto() },
        previewLatLng = previewLatLng,
        experimental = experimental,
    )
}

private fun Coordinates.toDto(): RouteCoordinateDto {
    return RouteCoordinateDto(
        lat = lat,
        lng = lng,
    )
}
