package me.nicolas.stravastats.domain.business

enum class RouteVariantType {
    CLOSE_MATCH,
    SHORTER,
    LONGER,
    HILLIER,
    SEASONAL,
    ROAD_GRAPH,
    SHAPE_MATCH,
    SHAPE_REMIX,
}

data class Coordinates(
    val lat: Double,
    val lng: Double,
)

data class RouteRecommendation(
    val routeId: String = "",
    val activity: ActivityShort,
    val activityDate: String,
    val distanceKm: Double,
    val elevationGainM: Double,
    val durationSec: Int,
    val isLoop: Boolean,
    val start: Coordinates?,
    val end: Coordinates?,
    val startArea: String,
    val season: String,
    val variantType: RouteVariantType,
    val matchScore: Double,
    val reasons: List<String>,
    val previewLatLng: List<List<Double>>,
    val shape: String?,
    val shapeScore: Double?,
    val experimental: Boolean,
)

data class ShapeRemixRecommendation(
    val id: String,
    val shape: String,
    val distanceKm: Double,
    val elevationGainM: Double,
    val durationSec: Int,
    val matchScore: Double,
    val reasons: List<String>,
    val components: List<ActivityShort>,
    val previewLatLng: List<List<Double>>,
    val experimental: Boolean,
)

data class RouteExplorerRequest(
    val distanceTargetKm: Double?,
    val elevationTargetM: Double?,
    val durationTargetMin: Int?,
    val startDirection: String? = null,
    val startPoint: Coordinates? = null,
    val routeType: String? = null,
    val season: String?,
    val limit: Int,
    val shape: String?,
    val shapePolyline: String? = null,
    val includeRemix: Boolean,
)

data class RouteExplorerResult(
    val closestLoops: List<RouteRecommendation>,
    val variants: List<RouteRecommendation>,
    val seasonal: List<RouteRecommendation>,
    val roadGraphLoops: List<RouteRecommendation>,
    val shapeMatches: List<RouteRecommendation>,
    val shapeRemixes: List<ShapeRemixRecommendation>,
)
