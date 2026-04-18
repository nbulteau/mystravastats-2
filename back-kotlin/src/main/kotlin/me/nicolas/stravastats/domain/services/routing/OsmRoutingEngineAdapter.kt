package me.nicolas.stravastats.domain.services.routing

import me.nicolas.stravastats.domain.business.ActivityShort
import me.nicolas.stravastats.domain.business.ActivityType
import me.nicolas.stravastats.domain.business.Coordinates
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

private const val DEFAULT_BASE_URL = "http://localhost:5000"
private const val DEFAULT_TIMEOUT_MS = 3000
private const val MAX_OSRM_CALLS = 16
private const val START_SNAP_TOLERANCE_METERS = 900.0
private const val DIRECTION_TOLERANCE_METERS = 120.0

private data class OsrmRouteResponse(
    val code: String? = null,
    val message: String? = null,
    val routes: List<OsrmRoute> = emptyList(),
)

private data class OsrmRoute(
    val distance: Double = 0.0,
    val duration: Double = 0.0,
    val geometry: OsrmGeometry? = null,
)

private data class OsrmGeometry(
    val type: String? = null,
    val coordinates: List<List<Double>> = emptyList(),
)

private data class OsmScoringProfile(
    val distanceWeight: Double,
    val elevationWeight: Double,
    val directionWeight: Double,
    val diversityWeight: Double,
)

private data class OsrmRouteCandidate(
    val recommendation: RouteRecommendation,
    val directionPenalty: Double,
    val backtrackingRatio: Double,
    val corridorOverlap: Double,
    val segmentDiversity: Double,
    val distanceDeltaRatio: Double,
    val effectiveMatchScore: Double,
)

private data class RouteRelaxationLevel(
    val name: String,
    val maxDirectionPenalty: Double,
    val maxBacktrackingRatio: Double,
    val maxCorridorOverlap: Double,
    val minSegmentDiversity: Double,
    val maxDistanceDeltaRatio: Double,
)

private data class PathSegment(
    val startLat: Double,
    val startLng: Double,
    val endLat: Double,
    val endLng: Double,
    val midLat: Double,
    val midLng: Double,
    val lengthM: Double,
    val bearing: Double,
)

@Component
class OsmRoutingEngineAdapter : RoutingEnginePort {

    private val logger = LoggerFactory.getLogger(OsmRoutingEngineAdapter::class.java)

    private val enabled = readBoolConfig("OSM_ROUTING_ENABLED", true)
    private val debug = readBoolConfig("OSM_ROUTING_DEBUG", false)
    private val baseUrl = readStringConfig("OSM_ROUTING_BASE_URL", DEFAULT_BASE_URL).trim().trimEnd('/')
    private val timeoutMs = readIntConfig("OSM_ROUTING_TIMEOUT_MS", DEFAULT_TIMEOUT_MS).coerceAtLeast(300)
    private val profileOverride = readStringConfig("OSM_ROUTING_PROFILE", "").trim()

    private val mapper = JsonMapper.builder()
        .addModule(KotlinModule.Builder().build())
        .build()

    private val httpClient = HttpClient.newBuilder()
        .connectTimeout(Duration.ofMillis(timeoutMs.toLong()))
        .build()

    override fun generateTargetLoops(request: RoutingEngineRequest): List<RouteRecommendation> {
        if (!enabled || baseUrl.isBlank()) {
            return emptyList()
        }
        if (request.distanceTargetKm <= 0.0 || request.limit <= 0) {
            return emptyList()
        }

        val profile = profileForRouteType(request.routeType)
        if (isCustomTargetMode(request)) {
            return generateCustomWaypointLoops(request, profile)
        }
        val baseBearing = startDirectionToBearing(request.startDirection)
        val baseRadiusKm = max(1.0, request.distanceTargetKm / (2.0 * PI))
        val radiusMultipliers = listOf(1.00, 0.92, 1.08, 0.84, 1.16, 1.24, 0.76, 1.32, 0.68, 1.40, 1.48, 0.60)
        val rotations = listOf(0.0, 22.0, -22.0, 45.0, -45.0, 68.0, -68.0, 95.0, -95.0, 125.0, -125.0, 155.0, -155.0)
        // Keep a high candidate pool even when request.limit is small, otherwise
        // strict anti-backtracking filters only compare near-identical loops.
        // We intentionally explore the full candidate budget so we can keep
        // anti-overlap constraints strict while still finding a route.
        val maxCalls = MAX_OSRM_CALLS

        // Pipeline:
        // 1) generate multiple OSRM candidates around the start point
        // 2) convert each route to scored candidate metrics
        // 3) deduplicate by geometry signature
        // 4) pick top routes with progressive constraint relaxation
        val candidates = mutableListOf<OsrmRouteCandidate>()
        val seenGeometry = mutableSetOf<String>()
        val rejectCounts = mutableMapOf<String, Int>()
        var fetchedRouteCount = 0
        var fetchErrors = 0
        var generatedCount = 0

        for (callIndex in 0 until maxCalls) {
            val radiusKm = baseRadiusKm * radiusMultipliers[callIndex % radiusMultipliers.size]
            val rotation = rotations[callIndex % rotations.size]
            val waypoints = syntheticLoopWaypoints(
                start = request.startPoint,
                radiusKm = radiusKm,
                initialBearing = baseBearing + rotation,
                callIndex = callIndex,
            )
            val routes = runCatching { fetchRoutes(profile, waypoints) }
                .onFailure { error ->
                    fetchErrors++
                    incrementRejectCount(rejectCounts, "OSRM_CALL_FAILED")
                    if (debug) {
                        logger.info(
                            "OSRM target generation call failed: call={} profile={} radiusKm={} rotation={} err={}",
                            callIndex + 1, profile, String.format("%.2f", radiusKm), String.format("%.1f", rotation), error.message
                        )
                    } else {
                        logger.debug("OSRM route generation failed: {}", error.message)
                    }
                }
                .getOrElse { emptyList() }
            fetchedRouteCount += routes.size

            for ((routeIndex, route) in routes.withIndex()) {
                val candidate = toRouteCandidate(request, route, generatedCount + routeIndex, rejectCounts) ?: continue
                val geometryKey = geometrySignature(candidate.recommendation.previewLatLng)
                if (geometryKey.isBlank()) {
                    incrementRejectCount(rejectCounts, "EMPTY_GEOMETRY_SIGNATURE")
                    continue
                }
                if (!seenGeometry.add(geometryKey)) {
                    incrementRejectCount(rejectCounts, "DUPLICATE_GEOMETRY")
                    continue
                }
                candidates += candidate
            }
            generatedCount += routes.size
        }
        val recommendations = selectCandidatesWithRelaxation(request, candidates, rejectCounts)
            .take(request.limit)

        if (debug || recommendations.isEmpty()) {
            val targetElevation = request.elevationTargetM?.let { value -> "${value.roundToInt()}m" } ?: "n/a"
            logger.info(
                "OSRM target generation summary: routeType={} direction={} target={}km/{} calls={} fetched={} accepted={} fetchErrors={} rejects={}",
                request.routeType?.trim()?.uppercase(Locale.getDefault()).orEmpty(),
                request.startDirection?.trim()?.uppercase(Locale.getDefault()).orEmpty(),
                String.format("%.1f", request.distanceTargetKm),
                targetElevation,
                maxCalls,
                fetchedRouteCount,
                recommendations.size,
                fetchErrors,
                formatRejectCounts(rejectCounts),
            )
        }

        return recommendations
    }

    private fun generateCustomWaypointLoops(
        request: RoutingEngineRequest,
        profile: String,
    ): List<RouteRecommendation> {
        val rejectCounts = mutableMapOf<String, Int>()
        val waypoints = buildCustomLoopWaypoints(request.startPoint, request.waypoints)
        if (waypoints.size < 3) {
            incrementRejectCount(rejectCounts, "CUSTOM_WAYPOINTS_TOO_FEW")
            return emptyList()
        }

        val routes = runCatching { fetchRoutes(profile, waypoints) }
            .onFailure { error ->
                incrementRejectCount(rejectCounts, "OSRM_CALL_FAILED")
                if (debug) {
                    logger.info(
                        "OSRM custom target generation call failed: profile={} waypoints={} err={}",
                        profile, waypoints.size, error.message
                    )
                } else {
                    logger.debug("OSRM custom target generation failed: {}", error.message)
                }
            }
            .getOrElse { emptyList() }

        val candidates = mutableListOf<OsrmRouteCandidate>()
        val seenGeometry = mutableSetOf<String>()
        routes.forEachIndexed { index, route ->
            val candidate = toRouteCandidate(request, route, index, rejectCounts) ?: return@forEachIndexed
            val geometryKey = geometrySignature(candidate.recommendation.previewLatLng)
            if (geometryKey.isBlank()) {
                incrementRejectCount(rejectCounts, "EMPTY_GEOMETRY_SIGNATURE")
                return@forEachIndexed
            }
            if (!seenGeometry.add(geometryKey)) {
                incrementRejectCount(rejectCounts, "DUPLICATE_GEOMETRY")
                return@forEachIndexed
            }
            candidates += candidate
        }

        val recommendations = selectCandidatesWithRelaxation(request, candidates, rejectCounts)
            .take(request.limit)
            .map { recommendation ->
                recommendation.copy(reasons = recommendation.reasons + "Target mode: custom waypoints")
            }

        if (debug || recommendations.isEmpty()) {
            val targetElevation = request.elevationTargetM?.let { value -> "${value.roundToInt()}m" } ?: "n/a"
            logger.info(
                "OSRM custom target generation summary: routeType={} target={}km/{} customWaypoints={} fetched={} accepted={} rejects={}",
                request.routeType?.trim()?.uppercase(Locale.getDefault()).orEmpty(),
                String.format("%.1f", request.distanceTargetKm),
                targetElevation,
                request.waypoints.size,
                routes.size,
                recommendations.size,
                formatRejectCounts(rejectCounts),
            )
        }

        return recommendations
    }

    override fun healthDetails(): Map<String, Any?> {
        val details = mutableMapOf<String, Any?>(
            "engine" to "osrm",
            "enabled" to enabled,
            "debug" to debug,
            "baseUrl" to baseUrl,
        )
        if (!enabled) {
            details["status"] = "disabled"
            details["reachable"] = false
            return details
        }
        if (baseUrl.isBlank()) {
            details["status"] = "misconfigured"
            details["reachable"] = false
            details["error"] = "OSM_ROUTING_BASE_URL is empty"
            return details
        }

        return runCatching {
            val response = httpClient.send(
                HttpRequest.newBuilder()
                    .uri(URI.create("$baseUrl/"))
                    .timeout(Duration.ofMillis(timeoutMs.toLong()))
                    .GET()
                    .build(),
                HttpResponse.BodyHandlers.ofString(),
            )
            details["statusCode"] = response.statusCode()
            if (response.statusCode() >= 500) {
                details["status"] = "down"
                details["reachable"] = false
            } else {
                details["status"] = "up"
                details["reachable"] = true
                details["profile"] = profileOverride
            }
            details
        }.getOrElse { error ->
            details["status"] = "down"
            details["reachable"] = false
            details["error"] = error.message
            details
        }
    }

    private fun profileForRouteType(routeType: String?): String {
        val override = profileOverride.lowercase(Locale.getDefault())
        if (override.isNotBlank()) {
            return override
        }
        return when (routeType.orEmpty().trim().uppercase(Locale.getDefault())) {
            "RUN", "TRAIL", "HIKE" -> "walking"
            else -> "cycling"
        }
    }

    private fun isCustomTargetMode(request: RoutingEngineRequest): Boolean {
        if (request.targetMode?.trim()?.equals("CUSTOM", ignoreCase = true) == true) {
            return true
        }
        return request.waypoints.isNotEmpty()
    }

    private fun buildCustomLoopWaypoints(start: Coordinates, customWaypoints: List<Coordinates>): List<Coordinates> {
        val waypoints = mutableListOf<Coordinates>()
        waypoints += start
        customWaypoints.forEach { point ->
            if (point.lat in -90.0..90.0 && point.lng in -180.0..180.0) {
                waypoints += point
            }
        }
        waypoints += start
        return waypoints
    }

    private fun syntheticLoopWaypoints(
        start: Coordinates,
        radiusKm: Double,
        initialBearing: Double,
        callIndex: Int,
    ): List<Coordinates> {
        // Rotate through multiple waypoint "shapes" so OSRM explores distinct loops
        // instead of repeatedly returning the same corridor.
        val patterns = listOf(
            Pair(listOf(0.0, 120.0, 240.0), listOf(1.00, 1.05, 0.95)),
            Pair(listOf(0.0, 85.0, 170.0, 255.0), listOf(1.10, 0.92, 1.08, 0.88)),
            Pair(listOf(0.0, 70.0, 155.0, 230.0, 300.0), listOf(1.00, 1.20, 0.85, 1.10, 0.90)),
            Pair(listOf(0.0, 60.0, 135.0, 210.0, 285.0), listOf(1.15, 0.90, 1.18, 0.86, 1.00)),
        )
        val pattern = patterns[callIndex % patterns.size]
        val waypoints = mutableListOf<Coordinates>()
        waypoints += start
        pattern.first.forEachIndexed { index, bearingOffset ->
            val scale = pattern.second.getOrNull(index)?.takeIf { it > 0.0 } ?: 1.0
            waypoints += destinationFromBearing(
                start = start,
                distanceKm = radiusKm * scale,
                bearingDegrees = normalizeBearing(initialBearing + bearingOffset),
            )
        }
        waypoints += start
        return waypoints
    }

    private fun fetchRoutes(profile: String, waypoints: List<Coordinates>): List<OsrmRoute> {
        if (waypoints.size < 2) return emptyList()
        val coordinates = waypoints.joinToString(";") { waypoint -> "%.6f,%.6f".format(waypoint.lng, waypoint.lat) }
        val url = "$baseUrl/route/v1/$profile/$coordinates?alternatives=true&steps=false&overview=full&geometries=geojson&continue_straight=true"
        val response = httpClient.send(
            HttpRequest.newBuilder()
                .uri(URI.create(url))
                .timeout(Duration.ofMillis(timeoutMs.toLong()))
                .GET()
                .build(),
            HttpResponse.BodyHandlers.ofString(),
        )
        if (response.statusCode() !in 200..299) {
            throw IllegalStateException("OSRM route API returned status ${response.statusCode()}")
        }
        val payload = mapper.readValue<OsrmRouteResponse>(response.body())
        if (payload.code?.lowercase(Locale.getDefault()) != "ok") {
            throw IllegalStateException("OSRM route API returned code ${payload.code}: ${payload.message}")
        }
        return payload.routes
    }

    private fun toRouteCandidate(
        request: RoutingEngineRequest,
        route: OsrmRoute,
        index: Int,
        rejectCounts: MutableMap<String, Int>,
    ): OsrmRouteCandidate? {
        if (route.distance <= 0.0 || route.geometry == null || route.geometry.coordinates.size < 2) {
            incrementRejectCount(rejectCounts, "INVALID_ROUTE_GEOMETRY")
            return null
        }
        val preview = route.geometry.coordinates.mapNotNull { point ->
            if (point.size < 2) return@mapNotNull null
            val lng = point[0]
            val lat = point[1]
            if (lat !in -90.0..90.0 || lng !in -180.0..180.0) return@mapNotNull null
            listOf(lat, lng)
        }
        if (preview.size < 2) {
            incrementRejectCount(rejectCounts, "INVALID_COORDINATES")
            return null
        }
        if (!startsNearRequestedStart(preview, request.startPoint, START_SNAP_TOLERANCE_METERS)) {
            incrementRejectCount(rejectCounts, "START_TOO_FAR")
            return null
        }

        val start = Coordinates(lat = preview.first()[0], lng = preview.first()[1])
        val end = Coordinates(lat = preview.last()[0], lng = preview.last()[1])
        val distanceKm = route.distance / 1000.0
        val durationSec = route.duration.toInt().coerceAtLeast((distanceKm * 180.0).toInt())
        val directionPenalty = combinedDirectionPenalty(preview, request.startPoint, request.startDirection, DIRECTION_TOLERANCE_METERS)
        val backtrackingRatio = oppositeEdgeTraversalRatio(preview)
        val corridorOverlap = corridorOverlapRatio(preview)
        val diversityRatio = segmentDiversityRatio(preview)
        val distanceDeltaRatio = abs(distanceKm - request.distanceTargetKm) / max(1.0, request.distanceTargetKm)
        val elevationEstimate = request.elevationTargetM?.let { target ->
            val deltaRatio = distanceDeltaRatio
            max(0.0, target * (1.0 - deltaRatio * 0.5))
        } ?: max(0.0, distanceKm * 8.0)
        val matchScore = computeOsmMatchScore(request, distanceKm, elevationEstimate, preview)
        val routeId = generatedRouteId(preview, request.startPoint, index)
        val titleSuffix = if (index > 0) " #${index + 1}" else ""
        val title = "Generated loop near %.4f, %.4f%s".format(request.startPoint.lat, request.startPoint.lng, titleSuffix)

        val reasons = buildList {
            add("Generated with OSM road graph (OSRM)")
            add("Distance delta: ${formatDistanceDelta(distanceKm - request.distanceTargetKm)}")
            add("Segment diversity: ${(diversityRatio * 100.0).roundToInt()}% unique edges")
            add("Directional alignment: ${((1.0 - directionPenalty) * 100.0).roundToInt()}%")
            add("Backtracking: ${(backtrackingRatio * 100.0).roundToInt()}%")
            add("Corridor overlap: ${(corridorOverlap * 100.0).roundToInt()}%")
            request.elevationTargetM?.let { target ->
                add("Elevation estimate: ${formatElevationDelta(elevationEstimate - target)}")
            }
            request.startDirection?.takeIf { it.isNotBlank() }?.let { direction ->
                add("Direction: ${direction.uppercase(Locale.getDefault())}")
            }
        }

        val recommendation = RouteRecommendation(
            routeId = routeId,
            activity = ActivityShort(
                id = 0,
                name = title,
                type = activityTypeFromRouteType(request.routeType),
            ),
            activityDate = DateTimeFormatter.ISO_INSTANT.format(Instant.now()),
            distanceKm = distanceKm,
            elevationGainM = elevationEstimate,
            durationSec = durationSec,
            isLoop = true,
            start = start,
            end = end,
            startArea = "%.4f, %.4f".format(start.lat, start.lng),
            season = seasonFromDate(Instant.now()),
            variantType = RouteVariantType.ROAD_GRAPH,
            matchScore = matchScore,
            reasons = reasons,
            previewLatLng = preview,
            shape = null,
            shapeScore = null,
            experimental = false,
        )
        val effectiveScore = clampScore(
            matchScore -
                directionPenalty * 22.0 -
                backtrackingRatio * 70.0 -
                corridorOverlap * 110.0 -
                max(0.0, minSegmentDiversityRatio(request.routeType) - diversityRatio) * 35.0 -
                max(0.0, distanceDeltaRatio - 0.15) * 45.0,
        )
        // effectiveMatchScore is an internal ranking score (not API score):
        // it aggressively penalizes backtracking and bad directional fit to keep
        // generated loops practical even in relaxed levels.
        return OsrmRouteCandidate(
            recommendation = recommendation,
            directionPenalty = directionPenalty,
            backtrackingRatio = backtrackingRatio,
            corridorOverlap = corridorOverlap,
            segmentDiversity = diversityRatio,
            distanceDeltaRatio = distanceDeltaRatio,
            effectiveMatchScore = effectiveScore,
        )
    }

    private fun selectCandidatesWithRelaxation(
        request: RoutingEngineRequest,
        candidates: List<OsrmRouteCandidate>,
        rejectCounts: MutableMap<String, Int>,
    ): List<RouteRecommendation> {
        if (candidates.isEmpty()) {
            return emptyList()
        }
        val sortedCandidates = candidates.sortedWith(
            compareBy<OsrmRouteCandidate> { it.corridorOverlap }
                .thenBy { it.backtrackingRatio }
                .thenByDescending { it.effectiveMatchScore }
                .thenBy { it.directionPenalty }
                .thenByDescending { it.recommendation.matchScore }
                .thenBy { it.distanceDeltaRatio }
                .thenBy { it.recommendation.routeId },
        )
        // Levels are evaluated in order: strict -> balanced -> relaxed -> fallback.
        // We fill results incrementally: if strict cannot fill the target limit,
        // next levels progressively loosen constraints while keeping quality.
        val levels = buildRouteRelaxationLevels(
            routeType = request.routeType,
            hasDirection = !request.startDirection.isNullOrBlank(),
        )
        val selected = mutableListOf<RouteRecommendation>()
        val selectedIds = mutableSetOf<String>()

        for (level in levels) {
            if (selected.size >= request.limit) break
            for (candidate in sortedCandidates) {
                if (selected.size >= request.limit) break
                if (!selectedIds.add(candidate.recommendation.routeId)) continue
                if (candidate.directionPenalty > level.maxDirectionPenalty) {
                    incrementRejectCount(rejectCounts, "DIRECTION_CONSTRAINT")
                    selectedIds.remove(candidate.recommendation.routeId)
                    continue
                }
                if (candidate.backtrackingRatio > level.maxBacktrackingRatio) {
                    incrementRejectCount(rejectCounts, "OPPOSITE_EDGE_TRAVERSAL")
                    selectedIds.remove(candidate.recommendation.routeId)
                    continue
                }
                if (candidate.corridorOverlap > level.maxCorridorOverlap) {
                    incrementRejectCount(rejectCounts, "CORRIDOR_OVERLAP")
                    selectedIds.remove(candidate.recommendation.routeId)
                    continue
                }
                if (candidate.segmentDiversity < level.minSegmentDiversity) {
                    incrementRejectCount(rejectCounts, "LOW_SEGMENT_DIVERSITY")
                    selectedIds.remove(candidate.recommendation.routeId)
                    continue
                }
                if (candidate.distanceDeltaRatio > level.maxDistanceDeltaRatio) {
                    incrementRejectCount(rejectCounts, "DISTANCE_CONSTRAINT")
                    selectedIds.remove(candidate.recommendation.routeId)
                    continue
                }
                selected += candidate.recommendation.copy(
                    reasons = candidate.recommendation.reasons + "Selection profile: ${level.name}",
                )
            }
        }
        return selected
    }

    private fun buildRouteRelaxationLevels(routeType: String?, hasDirection: Boolean): List<RouteRelaxationLevel> {
        val baseMinDiversity = minSegmentDiversityRatio(routeType)
        val strictDirection = if (hasDirection) 0.22 else 1.0
        val balancedDirection = if (hasDirection) 0.38 else 1.0
        val relaxedDirection = if (hasDirection) 0.55 else 1.0
        return listOf(
            RouteRelaxationLevel(
                name = "strict",
                maxDirectionPenalty = strictDirection,
                maxBacktrackingRatio = 0.003,
                maxCorridorOverlap = 0.008,
                minSegmentDiversity = baseMinDiversity,
                maxDistanceDeltaRatio = 0.35,
            ),
            RouteRelaxationLevel(
                name = "balanced",
                maxDirectionPenalty = balancedDirection,
                maxBacktrackingRatio = 0.012,
                maxCorridorOverlap = 0.016,
                minSegmentDiversity = max(0.22, baseMinDiversity - 0.08),
                maxDistanceDeltaRatio = 0.60,
            ),
            RouteRelaxationLevel(
                name = "relaxed",
                maxDirectionPenalty = relaxedDirection,
                maxBacktrackingRatio = 0.028,
                maxCorridorOverlap = 0.026,
                minSegmentDiversity = max(0.12, baseMinDiversity - 0.18),
                maxDistanceDeltaRatio = 1.00,
            ),
            RouteRelaxationLevel(
                name = "fallback",
                maxDirectionPenalty = 1.0,
                maxBacktrackingRatio = 0.07,
                maxCorridorOverlap = 0.035,
                minSegmentDiversity = 0.08,
                maxDistanceDeltaRatio = 2.20,
            ),
        )
    }

    private fun activityTypeFromRouteType(routeType: String?): ActivityType {
        return when (routeType.orEmpty().trim().uppercase(Locale.getDefault())) {
            "RUN" -> ActivityType.Run
            "TRAIL" -> ActivityType.TrailRun
            "HIKE" -> ActivityType.Hike
            "MTB" -> ActivityType.MountainBikeRide
            "GRAVEL" -> ActivityType.GravelRide
            else -> ActivityType.Ride
        }
    }

    private fun destinationFromBearing(start: Coordinates, distanceKm: Double, bearingDegrees: Double): Coordinates {
        val lat1 = Math.toRadians(start.lat)
        val lon1 = Math.toRadians(start.lng)
        val bearing = Math.toRadians(bearingDegrees)
        val angularDistance = distanceKm / 6371.0

        val lat2 = asin(sin(lat1) * cos(angularDistance) + cos(lat1) * sin(angularDistance) * cos(bearing))
        val lon2 = lon1 + atan2(
            sin(bearing) * sin(angularDistance) * cos(lat1),
            cos(angularDistance) - sin(lat1) * sin(lat2),
        )

        return Coordinates(
            lat = Math.toDegrees(lat2),
            lng = normalizeLongitude(Math.toDegrees(lon2)),
        )
    }

    private fun normalizeBearing(value: Double): Double {
        var normalized = value % 360.0
        if (normalized < 0) normalized += 360.0
        return normalized
    }

    private fun startDirectionToBearing(direction: String?): Double {
        return when (direction.orEmpty().trim().uppercase(Locale.getDefault())) {
            "N" -> 0.0
            "E" -> 90.0
            "S" -> 180.0
            "W" -> 270.0
            else -> 0.0
        }
    }

    private fun normalizeLongitude(value: Double): Double {
        var normalized = value
        while (normalized < -180.0) normalized += 360.0
        while (normalized > 180.0) normalized -= 360.0
        return normalized
    }

    private fun generatedRouteId(points: List<List<Double>>, start: Coordinates, index: Int): String {
        val step = if (points.size > 40) max(1, points.size / 40) else 1
        val signature = buildString {
            append("%.5f|%.5f|%d|".format(start.lat, start.lng, index))
            points.indices.step(step).forEach { idx ->
                append("%.5f,%.5f|".format(points[idx][0], points[idx][1]))
            }
        }
        return "generated-osm-${signature.hashCode().toUInt().toString(16)}"
    }

    private fun geometrySignature(points: List<List<Double>>): String {
        if (points.isEmpty()) return ""
        val step = if (points.size > 60) max(1, points.size / 60) else 1
        return buildString {
            points.indices.step(step).forEach { idx ->
                val point = points[idx]
                if (point.size >= 2) {
                    append("%.5f,%.5f|".format(point[0], point[1]))
                }
            }
        }
    }

    private fun formatDistanceDelta(deltaKm: Double): String {
        val absolute = abs(deltaKm)
        return if (absolute < 1.0) {
            "${round(absolute * 1000.0).toInt()} m"
        } else {
            "${"%.2f".format(absolute)} km"
        }
    }

    private fun formatElevationDelta(deltaM: Double): String {
        return "${round(abs(deltaM)).toInt()} m"
    }

    private fun clampScore(value: Double): Double {
        val normalized = min(100.0, max(0.0, value))
        return round(normalized * 10.0) / 10.0
    }

    private fun seasonFromDate(date: Instant): String {
        return when (date.atZone(ZoneOffset.UTC).monthValue) {
            12, 1, 2 -> "WINTER"
            3, 4, 5 -> "SPRING"
            6, 7, 8 -> "SUMMER"
            else -> "AUTUMN"
        }
    }

    private fun readConfigValue(key: String): String? {
        val fromEnv = System.getenv(key)?.trim()
        if (!fromEnv.isNullOrEmpty()) {
            return fromEnv
        }

        val dotEnv = File(".env")
        if (!dotEnv.exists() || !dotEnv.isFile) {
            return null
        }

        return dotEnv.useLines { lines ->
            lines
                .map { it.trim() }
                .filter { it.isNotEmpty() && !it.startsWith("#") && it.contains("=") }
                .map { line ->
                    val separator = line.indexOf('=')
                    val envKey = line.substring(0, separator).trim()
                    val envValue = line.substring(separator + 1).trim().trim('"', '\'')
                    envKey to envValue
                }
                .firstOrNull { (envKey, _) -> envKey == key }
                ?.second
                ?.takeIf { it.isNotEmpty() }
        }
    }

    private fun readStringConfig(key: String, fallback: String): String {
        return readConfigValue(key) ?: fallback
    }

    private fun readBoolConfig(key: String, fallback: Boolean): Boolean {
        val normalized = readConfigValue(key)?.lowercase(Locale.getDefault()) ?: return fallback
        return when (normalized) {
            "1", "true", "yes", "y", "on" -> true
            "0", "false", "no", "n", "off" -> false
            else -> fallback
        }
    }

    private fun readIntConfig(key: String, fallback: Int): Int {
        return readConfigValue(key)?.toIntOrNull() ?: fallback
    }

    private fun startsNearRequestedStart(
        points: List<List<Double>>,
        start: Coordinates,
        toleranceMeters: Double,
    ): Boolean {
        if (points.isEmpty()) return false
        val first = points.first()
        if (first.size < 2) return false
        return haversineDistanceMeters(first[0], first[1], start.lat, start.lng) <= toleranceMeters
    }

    private fun respectsHalfPlaneDirection(
        points: List<List<Double>>,
        start: Coordinates,
        direction: String?,
        toleranceMeters: Double,
    ): Boolean {
        val normalized = direction.orEmpty().trim().uppercase(Locale.getDefault())
        if (normalized.isBlank() || points.isEmpty()) return true

        val latTolerance = toleranceMeters / 111320.0
        val lngTolerance = toleranceMeters / max(1000.0, 111320.0 * cos(Math.toRadians(start.lat)))

        return when (normalized) {
            "N" -> points.all { point -> point.size < 2 || point[0] >= start.lat - latTolerance }
            "S" -> points.all { point -> point.size < 2 || point[0] <= start.lat + latTolerance }
            "E" -> points.all { point -> point.size < 2 || point[1] >= start.lng - lngTolerance }
            "W" -> points.all { point -> point.size < 2 || point[1] <= start.lng + lngTolerance }
            else -> true
        }
    }

    private fun combinedDirectionPenalty(
        points: List<List<Double>>,
        start: Coordinates,
        direction: String?,
        toleranceMeters: Double,
    ): Double {
        if (direction.isNullOrBlank()) {
            return 0.0
        }
        // We combine two direction signals:
        // - initial heading alignment (bearing-based)
        // - half-plane violations (did the route go too much in the opposite side)
        // Taking the max keeps direction enforcement robust in dense urban grids.
        val bearingPenalty = directionPenaltyFromPreview(points, direction)
        val halfPlanePenalty = halfPlaneViolationRatio(points, start, direction, toleranceMeters)
        return max(bearingPenalty, halfPlanePenalty)
    }

    private fun halfPlaneViolationRatio(
        points: List<List<Double>>,
        start: Coordinates,
        direction: String?,
        toleranceMeters: Double,
    ): Double {
        val normalized = direction.orEmpty().trim().uppercase(Locale.getDefault())
        if (normalized.isBlank() || points.isEmpty()) return 0.0

        val latTolerance = toleranceMeters / 111320.0
        val lngTolerance = toleranceMeters / max(1000.0, 111320.0 * cos(Math.toRadians(start.lat)))
        var total = 0
        var violations = 0

        for (point in points) {
            if (point.size < 2) continue
            total++
            when (normalized) {
                "N" -> if (point[0] < start.lat - latTolerance) violations++
                "S" -> if (point[0] > start.lat + latTolerance) violations++
                "E" -> if (point[1] < start.lng - lngTolerance) violations++
                "W" -> if (point[1] > start.lng + lngTolerance) violations++
            }
        }
        if (total == 0) return 0.0
        return violations.toDouble() / total.toDouble()
    }

    private fun corridorOverlapRatio(points: List<List<Double>>): Double {
        if (points.size < 4) return 0.0
        val sampled = samplePolylinePoints(points, 260)
        val segments = buildPathSegments(sampled)
        if (segments.size < 2) return 0.0

        val flagged = BooleanArray(segments.size)
        for (i in segments.indices) {
            // Skip only immediate neighbors to avoid counting normal local curvature as overlap.
            for (j in 0 until (i - 1).coerceAtLeast(0)) {
                if (segmentsLikelySameCorridor(segments[i], segments[j])) {
                    flagged[i] = true
                    flagged[j] = true
                }
            }
        }
        val overlapped = flagged.count { it }
        return overlapped.toDouble() / segments.size.toDouble()
    }

    private fun samplePolylinePoints(points: List<List<Double>>, maxPoints: Int): List<List<Double>> {
        if (points.size <= maxPoints || maxPoints <= 0) {
            return points
        }
        val step = max(1, ceil(points.size.toDouble() / maxPoints.toDouble()).toInt())
        val sampled = mutableListOf<List<Double>>()
        for (index in points.indices step step) {
            sampled += points[index]
        }
        val lastPoint = points.last()
        val lastSample = sampled.lastOrNull()
        if (lastSample == null || lastSample.size < 2 || lastPoint.size < 2 ||
            lastSample[0] != lastPoint[0] || lastSample[1] != lastPoint[1]
        ) {
            sampled += lastPoint
        }
        return sampled
    }

    private fun buildPathSegments(points: List<List<Double>>): List<PathSegment> {
        val segments = mutableListOf<PathSegment>()
        for (index in 0 until points.size - 1) {
            val from = points[index]
            val to = points[index + 1]
            if (from.size < 2 || to.size < 2) continue

            val lengthM = haversineDistanceMeters(from[0], from[1], to[0], to[1])
            if (lengthM < 12.0) continue

            segments += PathSegment(
                startLat = from[0],
                startLng = from[1],
                endLat = to[0],
                endLng = to[1],
                midLat = (from[0] + to[0]) / 2.0,
                midLng = (from[1] + to[1]) / 2.0,
                lengthM = lengthM,
                bearing = bearingDegrees(from[0], from[1], to[0], to[1]),
            )
        }
        return segments
    }

    private fun segmentsLikelySameCorridor(left: PathSegment, right: PathSegment): Boolean {
        val midpointToleranceMeters = 50.0
        val endpointToleranceMeters = 80.0
        val midpointDistance = haversineDistanceMeters(left.midLat, left.midLng, right.midLat, right.midLng)
        if (midpointDistance > midpointToleranceMeters) return false

        val leftToRightStart = haversineDistanceMeters(left.startLat, left.startLng, right.startLat, right.startLng)
        val leftToRightEnd = haversineDistanceMeters(left.startLat, left.startLng, right.endLat, right.endLng)
        val rightToLeftStart = haversineDistanceMeters(left.endLat, left.endLng, right.startLat, right.startLng)
        val rightToLeftEnd = haversineDistanceMeters(left.endLat, left.endLng, right.endLat, right.endLng)
        if (
            min(leftToRightStart, leftToRightEnd) > endpointToleranceMeters ||
            min(rightToLeftStart, rightToLeftEnd) > endpointToleranceMeters
        ) {
            return false
        }

        var bearingDiff = abs(left.bearing - right.bearing)
        if (bearingDiff > 180.0) bearingDiff = 360.0 - bearingDiff
        if (bearingDiff > 22.0 && bearingDiff < 158.0) return false

        val maxLength = max(left.lengthM, right.lengthM)
        val minLength = min(left.lengthM, right.lengthM)
        if (minLength <= 0.0 || maxLength / minLength > 6.0) return false
        return true
    }

    private fun hasOppositeEdgeTraversal(points: List<List<Double>>): Boolean {
        return oppositeEdgeTraversalRatio(points) > 0.0
    }

    private fun oppositeEdgeTraversalRatio(points: List<List<Double>>): Double {
        if (points.size < 3) return 0.0
        data class EdgeDirection(var forward: Boolean = false, var reverse: Boolean = false)
        val seen = mutableMapOf<String, EdgeDirection>()
        var totalEdges = 0

        for (index in 0 until points.size - 1) {
            val from = points[index]
            val to = points[index + 1]
            if (from.size < 2 || to.size < 2) continue

            val fromId = quantizedPointKey(from[0], from[1])
            val toId = quantizedPointKey(to[0], to[1])
            if (fromId == toId) continue

            totalEdges++
            val edgeKey = canonicalEdgeKey(fromId, toId)
            val edge = seen.getOrPut(edgeKey) { EdgeDirection() }
            if (fromId < toId) {
                edge.forward = true
            } else {
                edge.reverse = true
            }
        }
        if (totalEdges == 0) return 0.0
        val conflictingEdges = seen.values.count { edge -> edge.forward && edge.reverse }
        return conflictingEdges.toDouble() / totalEdges.toDouble()
    }

    private fun incrementRejectCount(rejectCounts: MutableMap<String, Int>, reason: String) {
        val normalizedReason = reason.trim()
        if (normalizedReason.isBlank()) {
            return
        }
        rejectCounts[normalizedReason] = (rejectCounts[normalizedReason] ?: 0) + 1
    }

    private fun formatRejectCounts(rejectCounts: Map<String, Int>): String {
        if (rejectCounts.isEmpty()) {
            return "none"
        }
        return rejectCounts.entries
            .sortedWith(compareByDescending<Map.Entry<String, Int>> { entry -> entry.value }.thenBy { entry -> entry.key })
            .joinToString(", ") { entry -> "${entry.key}=${entry.value}" }
    }

    private fun hasMinimumSegmentDiversity(points: List<List<Double>>, routeType: String?): Boolean {
        if (points.size < 3) return false

        val maxEdgeReuse = 3
        val segmentUsage = mutableMapOf<String, Int>()
        var totalEdges = 0
        var uniqueEdges = 0

        for (index in 0 until points.size - 1) {
            val from = points[index]
            val to = points[index + 1]
            if (from.size < 2 || to.size < 2) continue

            val fromId = quantizedPointKey(from[0], from[1])
            val toId = quantizedPointKey(to[0], to[1])
            if (fromId == toId) continue

            totalEdges++
            val segmentKey = canonicalEdgeKey(fromId, toId)
            val count = (segmentUsage[segmentKey] ?: 0) + 1
            segmentUsage[segmentKey] = count
            if (count == 1) {
                uniqueEdges++
            }
            if (count > maxEdgeReuse) {
                return false
            }
        }

        if (totalEdges == 0) return false
        return uniqueEdges.toDouble() / totalEdges.toDouble() >= minSegmentDiversityRatio(routeType)
    }

    private fun minSegmentDiversityRatio(routeType: String?): Double {
        return when (routeType.orEmpty().trim().uppercase(Locale.getDefault())) {
            "MTB" -> 0.40
            "GRAVEL" -> 0.42
            "RUN" -> 0.35
            "TRAIL" -> 0.30
            "HIKE" -> 0.28
            else -> 0.45
        }
    }

    private fun segmentDiversityRatio(points: List<List<Double>>): Double {
        if (points.size < 2) return 0.0

        var totalEdges = 0
        val uniqueEdges = mutableSetOf<String>()
        for (index in 0 until points.size - 1) {
            val from = points[index]
            val to = points[index + 1]
            if (from.size < 2 || to.size < 2) continue

            val fromId = quantizedPointKey(from[0], from[1])
            val toId = quantizedPointKey(to[0], to[1])
            if (fromId == toId) continue

            totalEdges++
            uniqueEdges += canonicalEdgeKey(fromId, toId)
        }
        if (totalEdges == 0) return 0.0
        return uniqueEdges.size.toDouble() / totalEdges.toDouble()
    }

    private fun computeOsmMatchScore(
        request: RoutingEngineRequest,
        distanceKm: Double,
        elevationGainM: Double,
        points: List<List<Double>>,
    ): Double {
        val hasElevationTarget = (request.elevationTargetM ?: 0.0) > 0.0
        val hasDirection = !request.startDirection.isNullOrBlank()
        val profile = buildOsmScoringProfile(request.routeType, hasElevationTarget, hasDirection)

        val distanceComponent = abs(distanceKm - request.distanceTargetKm) / max(1.0, request.distanceTargetKm)
        val elevationComponent = if (hasElevationTarget) {
            abs(elevationGainM - (request.elevationTargetM ?: 0.0)) / max((request.elevationTargetM ?: 0.0), 150.0)
        } else {
            0.0
        }
        val directionComponent = if (hasDirection) {
            directionPenaltyFromPreview(points, request.startDirection)
        } else {
            0.0
        }
        val diversityComponent = 1.0 - segmentDiversityRatio(points)

        val weighted = distanceComponent * profile.distanceWeight +
            elevationComponent * profile.elevationWeight +
            directionComponent * profile.directionWeight +
            diversityComponent * profile.diversityWeight
        return clampScore(100.0 - weighted * 100.0)
    }

    private fun buildOsmScoringProfile(
        routeType: String?,
        hasElevationTarget: Boolean,
        hasDirection: Boolean,
    ): OsmScoringProfile {
        var profile = when (routeType.orEmpty().trim().uppercase(Locale.getDefault())) {
            "MTB" -> OsmScoringProfile(distanceWeight = 0.48, elevationWeight = 0.38, directionWeight = 0.09, diversityWeight = 0.05)
            "GRAVEL" -> OsmScoringProfile(distanceWeight = 0.54, elevationWeight = 0.33, directionWeight = 0.08, diversityWeight = 0.05)
            "RUN" -> OsmScoringProfile(distanceWeight = 0.55, elevationWeight = 0.20, directionWeight = 0.15, diversityWeight = 0.10)
            "TRAIL" -> OsmScoringProfile(distanceWeight = 0.42, elevationWeight = 0.33, directionWeight = 0.15, diversityWeight = 0.10)
            "HIKE" -> OsmScoringProfile(distanceWeight = 0.34, elevationWeight = 0.41, directionWeight = 0.15, diversityWeight = 0.10)
            else -> OsmScoringProfile(distanceWeight = 0.58, elevationWeight = 0.30, directionWeight = 0.08, diversityWeight = 0.04)
        }

        if (!hasElevationTarget) {
            profile = profile.copy(
                distanceWeight = profile.distanceWeight + profile.elevationWeight * 0.70,
                diversityWeight = profile.diversityWeight + profile.elevationWeight * 0.30,
                elevationWeight = 0.0,
            )
        }
        if (!hasDirection) {
            profile = profile.copy(
                distanceWeight = profile.distanceWeight + profile.directionWeight * 0.60,
                diversityWeight = profile.diversityWeight + profile.directionWeight * 0.40,
                directionWeight = 0.0,
            )
        }

        return normalizeOsmScoringProfile(profile)
    }

    private fun normalizeOsmScoringProfile(profile: OsmScoringProfile): OsmScoringProfile {
        val total = profile.distanceWeight + profile.elevationWeight + profile.directionWeight + profile.diversityWeight
        if (total <= 0.0) {
            return OsmScoringProfile(distanceWeight = 0.72, elevationWeight = 0.20, directionWeight = 0.04, diversityWeight = 0.04)
        }
        return OsmScoringProfile(
            distanceWeight = profile.distanceWeight / total,
            elevationWeight = profile.elevationWeight / total,
            directionWeight = profile.directionWeight / total,
            diversityWeight = profile.diversityWeight / total,
        )
    }

    private fun directionPenaltyFromPreview(points: List<List<Double>>, startDirection: String?): Double {
        val initialBearing = initialBearingFromPreview(points) ?: return 1.0
        val targetBearing = targetBearingFromDirection(startDirection) ?: return 0.0
        val rawDiff = abs(initialBearing - targetBearing)
        val normalizedDiff = if (rawDiff > 180.0) 360.0 - rawDiff else rawDiff
        return normalizedDiff / 180.0
    }

    private fun initialBearingFromPreview(points: List<List<Double>>): Double? {
        if (points.size < 2) return null
        val start = points.firstOrNull()?.takeIf { point -> point.size >= 2 } ?: return null
        val startLat = start[0]
        val startLng = start[1]

        for (index in 1 until points.size) {
            val next = points[index]
            if (next.size < 2) continue
            if (haversineDistanceMeters(startLat, startLng, next[0], next[1]) < 35.0) continue
            return bearingDegrees(startLat, startLng, next[0], next[1])
        }

        val fallback = points.lastOrNull()?.takeIf { point -> point.size >= 2 } ?: return null
        return bearingDegrees(startLat, startLng, fallback[0], fallback[1])
    }

    private fun targetBearingFromDirection(direction: String?): Double? {
        return when (direction.orEmpty().trim().uppercase(Locale.getDefault())) {
            "N" -> 0.0
            "E" -> 90.0
            "S" -> 180.0
            "W" -> 270.0
            else -> null
        }
    }

    private fun bearingDegrees(lat1: Double, lng1: Double, lat2: Double, lng2: Double): Double {
        val lat1r = Math.toRadians(lat1)
        val lat2r = Math.toRadians(lat2)
        val deltaLng = Math.toRadians(lng2 - lng1)
        val y = sin(deltaLng) * cos(lat2r)
        val x = cos(lat1r) * sin(lat2r) - sin(lat1r) * cos(lat2r) * cos(deltaLng)
        var bearing = atan2(y, x) * 180.0 / PI
        if (bearing < 0.0) {
            bearing += 360.0
        }
        return bearing
    }

    private fun quantizedPointKey(lat: Double, lng: Double): String = "%.5f:%.5f".format(lat, lng)

    private fun canonicalEdgeKey(a: String, b: String): String = if (a < b) "$a|$b" else "$b|$a"

    private fun haversineDistanceMeters(lat1: Double, lng1: Double, lat2: Double, lng2: Double): Double {
        val earthRadiusMeters = 6_371_000.0
        val dLat = Math.toRadians(lat2 - lat1)
        val dLng = Math.toRadians(lng2 - lng1)
        val sinLat = sin(dLat / 2.0)
        val sinLng = sin(dLng / 2.0)
        val a = sinLat * sinLat + cos(Math.toRadians(lat1)) * cos(Math.toRadians(lat2)) * sinLng * sinLng
        val c = 2.0 * atan2(sqrt(a), sqrt(1.0 - a))
        return earthRadiusMeters * c
    }
}
