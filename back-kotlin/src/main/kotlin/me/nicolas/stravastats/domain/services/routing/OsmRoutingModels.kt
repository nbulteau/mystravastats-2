package me.nicolas.stravastats.domain.services.routing

import me.nicolas.stravastats.domain.RuntimeConfig
import me.nicolas.stravastats.domain.business.ActivityShort
import me.nicolas.stravastats.domain.business.ActivityType
import me.nicolas.stravastats.domain.business.Coordinates
import me.nicolas.stravastats.domain.business.RouteGenerationDiagnostic
import me.nicolas.stravastats.domain.business.RouteRecommendation
import me.nicolas.stravastats.domain.business.RouteVariantType
import org.slf4j.LoggerFactory
import org.springframework.stereotype.Component
import tools.jackson.databind.json.JsonMapper
import tools.jackson.module.kotlin.KotlinModule
import tools.jackson.module.kotlin.readValue
import java.io.File
import java.net.URI
import java.net.http.HttpClient
import java.net.http.HttpRequest
import java.net.http.HttpResponse
import java.time.Duration
import java.time.Instant
import java.time.ZoneOffset
import java.time.format.DateTimeFormatter
import java.util.Locale
import kotlin.math.PI
import kotlin.math.abs
import kotlin.math.asin
import kotlin.math.atan2
import kotlin.math.ceil
import kotlin.math.cos
import kotlin.math.max
import kotlin.math.min
import kotlin.math.round
import kotlin.math.roundToInt
import kotlin.math.sin
import kotlin.math.sqrt

internal const val DEFAULT_BASE_URL = "http://localhost:5000"
internal const val DEFAULT_TIMEOUT_MS = 3000
internal const val DEFAULT_V3_ENABLED = true
internal const val MAX_OSRM_CALLS = 24
internal const val START_SNAP_TOLERANCE_METERS = 900.0
internal const val FALLBACK_START_SNAP_TOLERANCE_METERS = 4000.0
internal const val DIRECTION_TOLERANCE_METERS = 120.0
internal const val BACKTRACKING_START_ZONE_METERS = 2000.0
internal const val MIN_AXIS_SEGMENT_LENGTH_METERS = 25.0
internal const val MIN_OPPOSITE_REUSE_METERS = 120.0
internal const val SHAPE_MODE_STRATEGY_SHAPE_FIRST = "shape-first"
internal const val SHAPE_MODE_STRATEGY_MAP_MATCH = "shape-map-match"
internal const val SHAPE_MODE_STRATEGY_ROAD_SNAP = "shape-road-snap"
internal const val SHAPE_MODE_STRATEGY_STITCHED = "shape-stitched"
internal const val SHAPE_MODE_STRATEGY_SIMPLIFIED = "shape-simplified"
internal const val SHAPE_MODE_STRATEGY_ROAD_FIRST = "road-first"
internal const val SHAPE_MODE_STRATEGY_BEST_EFFORT = "shape-best-effort"
internal const val MAX_SHAPE_TRACE_VARIANTS = 8
internal const val MAX_SHAPE_BEST_EFFORT_VARIANTS = 8
internal const val SHAPE_COVERAGE_SAMPLE_POINTS = 6
internal const val SHAPE_COVERAGE_MAX_SNAP_METERS = 5000.0
internal const val EDIT_CONTROL_SNAP_MAX_METERS = 900.0
internal const val EDIT_MIN_CONTROL_SPACING_METERS = 15.0
internal const val DEFAULT_EXTRACT_PROFILE_FILE = "./osm/region.osrm.profile"
internal const val FALLBACK_EXTRACT_PROFILE_FILE = "../osm/region.osrm.profile"

internal data class OsrmRouteResponse(
    val code: String? = null,
    val message: String? = null,
    val routes: List<OsrmRoute> = emptyList(),
)

internal data class OsrmMatchResponse(
    val code: String? = null,
    val message: String? = null,
    val matchings: List<OsrmRoute> = emptyList(),
)

internal data class OsrmNearestResponse(
    val code: String? = null,
    val message: String? = null,
    val waypoints: List<OsrmNearestWaypoint> = emptyList(),
)

internal data class OsrmNearestWaypoint(
    val distance: Double = 0.0,
    val location: List<Double> = emptyList(),
)

internal data class OsrmRoute(
    val distance: Double = 0.0,
    val duration: Double = 0.0,
    val geometry: OsrmGeometry? = null,
    val legs: List<OsrmLeg> = emptyList(),
)

internal data class OsrmGeometry(
    val type: String? = null,
    val coordinates: List<List<Double>> = emptyList(),
)

internal data class OsrmLeg(
    val steps: List<OsrmStep> = emptyList(),
)

internal data class OsrmStep(
    val distance: Double = 0.0,
    val mode: String? = null,
    val classes: List<String> = emptyList(),
    val surface: String? = null,
    val tracktype: String? = null,
)

internal data class OsmScoringProfile(
    val distanceWeight: Double,
    val elevationWeight: Double,
    val directionWeight: Double,
    val diversityWeight: Double,
)

internal data class OsrmRouteCandidate(
    val recommendation: RouteRecommendation,
    val directionPenalty: Double,
    val backtrackingRatio: Double,
    val corridorOverlap: Double,
    val edgeReuseRatio: Double,
    val maxAxisReuseCount: Int,
    val maxAxisReuseRatio: Double,
    val segmentDiversity: Double,
    val distanceDeltaRatio: Double,
    val pathRatio: Double,
    val historyReuseScore: Double = 0.0,
    val effectiveMatchScore: Double,
)

internal data class RouteRelaxationLevel(
    val name: String,
    val maxDirectionPenalty: Double,
    val maxBacktrackingRatio: Double,
    val maxCorridorOverlap: Double,
    val maxEdgeReuseRatio: Double,
    val maxAxisReuseCount: Int,
    val minSegmentDiversity: Double,
    val maxDistanceDeltaRatio: Double,
)

internal data class RouteSurfaceBreakdown(
    val pavedM: Double = 0.0,
    val gravelM: Double = 0.0,
    val trailM: Double = 0.0,
    val unknownM: Double = 0.0,
) {
    fun totalDistanceM(): Double = pavedM + gravelM + trailM + unknownM

    fun normalizedRatios(): List<Double> {
        val total = totalDistanceM()
        if (total <= 0.0) {
            return listOf(0.0, 0.0, 0.0, 1.0)
        }
        return listOf(
            pavedM / total,
            gravelM / total,
            trailM / total,
            unknownM / total,
        )
    }

    fun pathRatio(): Double {
        val (_, gravel, trail, _) = normalizedRatios()
        return (gravel + trail).coerceIn(0.0, 1.0)
    }
}

internal data class PathSegment(
    val startLat: Double,
    val startLng: Double,
    val endLat: Double,
    val endLng: Double,
    val midLat: Double,
    val midLng: Double,
    val lengthM: Double,
    val bearing: Double,
)

internal data class NormalizedShapePoint(
    var x: Double,
    var y: Double,
)

internal data class ShapeSimilarityBreakdown(
    val score: Double,
    val contourScore: Double,
    val anchoredScore: Double,
    val orderedScore: Double,
    val centroidScore: Double,
    val corridorScore: Double = 0.0,
    val lengthScore: Double = 0.0,
)

internal data class ShapeModeScoringConfig(
    val baseMatchWeight: Double,
    val shapeWeight: Double,
    val lowSimilarityThreshold: Double,
    val lowSimilarityPenaltyRate: Double,
)

internal data class ShapeRoutingVariant(
    val label: String,
    val shape: List<Coordinates>,
)

