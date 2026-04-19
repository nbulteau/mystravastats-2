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
private const val DEFAULT_V3_ENABLED = true
private const val MAX_OSRM_CALLS = 24
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
    val edgeReuseRatio: Double,
    val maxAxisReuseCount: Int,
    val maxAxisReuseRatio: Double,
    val segmentDiversity: Double,
    val distanceDeltaRatio: Double,
    val effectiveMatchScore: Double,
)

private data class RouteRelaxationLevel(
    val name: String,
    val maxDirectionPenalty: Double,
    val maxBacktrackingRatio: Double,
    val maxCorridorOverlap: Double,
    val maxEdgeReuseRatio: Double,
    val maxAxisReuseCount: Int,
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

private data class NormalizedShapePoint(
    var x: Double,
    var y: Double,
)

@Component
class OsmRoutingEngineAdapter : RoutingEnginePort {

    private val logger = LoggerFactory.getLogger(OsmRoutingEngineAdapter::class.java)

    private val enabled = readBoolConfig("OSM_ROUTING_ENABLED", true)
    private val v3Enabled = readBoolConfig("OSM_ROUTING_V3_ENABLED", DEFAULT_V3_ENABLED)
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
        var usedLegacyFallback = false
        if (isCustomTargetMode(request)) {
            return generateCustomWaypointLoops(request, profile)
        }
        if (v3Enabled) {
            val disjointRecommendations = generateTargetLoopsDisjoint(request, profile)
            if (disjointRecommendations.isNotEmpty()) {
                return disjointRecommendations
            }
            usedLegacyFallback = true
            if (debug) {
                logger.info("OSRM target generation v3 produced no valid route, falling back to legacy generator")
            }
        }
        val baseBearing = startDirectionToBearing(request.startDirection)
        val hasDirection = !request.startDirection.isNullOrBlank()
        val directionStrict = hasDirection && request.directionStrict
        val baseRadiusKm = max(1.0, request.distanceTargetKm / (2.0 * PI))
        val radiusMultipliers = listOf(1.00, 0.92, 1.08, 0.84, 1.16, 1.24, 0.76, 1.32, 0.68, 1.40, 1.48, 0.60)
        var rotations = listOf(0.0, 22.0, -22.0, 45.0, -45.0, 68.0, -68.0, 95.0, -95.0, 125.0, -125.0, 155.0, -155.0)
        if (hasDirection) {
            // With direction in automatic mode, keep rotations tight around the
            // requested bearing to preserve a clear global orientation.
            rotations = listOf(0.0, 8.0, -8.0, 15.0, -15.0, 24.0, -24.0, 32.0, -32.0)
            if (directionStrict) {
                // Strict mode keeps the directional cone narrower.
                rotations = listOf(0.0, 5.0, -5.0, 10.0, -10.0, 16.0, -16.0)
            }
        }
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
                startDirection = request.startDirection,
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
        var recommendations = selectCandidatesWithRelaxation(request, candidates, rejectCounts)
            .take(request.limit)
        if (usedLegacyFallback) {
            recommendations = recommendations.map { recommendation ->
                recommendation.copy(
                    reasons = recommendation.reasons + "Generation engine fallback: legacy synthetic waypoints",
                )
            }
        }

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

    override fun generateShapeLoops(request: RoutingEngineRequest): List<RouteRecommendation> {
        if (!enabled || baseUrl.isBlank()) {
            return emptyList()
        }
        if (request.limit <= 0) {
            return emptyList()
        }

        val shapePolyline = request.shapePolyline?.trim().orEmpty()
        if (shapePolyline.isBlank()) {
            return emptyList()
        }
        val rawShape = parseShapePolylineCoordinates(shapePolyline)
        if (rawShape.size < 2) {
            return emptyList()
        }

        var distanceTargetKm = request.distanceTargetKm
        if (distanceTargetKm <= 0.0) {
            distanceTargetKm = polylineDistanceKmFromCoordinates(rawShape)
        }
        if (distanceTargetKm <= 0.0) {
            distanceTargetKm = 20.0
        }

        val projectedShape = projectShapePolylineToStart(
            shape = rawShape,
            start = request.startPoint,
            targetDistanceKm = distanceTargetKm,
        )
        val waypoints = buildShapeLoopWaypoints(request.startPoint, projectedShape)
        if (waypoints.size < 3) {
            return emptyList()
        }

        val profile = profileForRouteType(request.routeType)
        val routes = runCatching { fetchRoutes(profile, waypoints) }
            .onFailure { error ->
                if (debug) {
                    logger.info(
                        "OSRM shape generation call failed: profile={} waypoints={} err={}",
                        profile, waypoints.size, error.message
                    )
                } else {
                    logger.debug("OSRM shape generation failed: {}", error.message)
                }
            }
            .getOrElse { emptyList() }

        val shapeRequest = request.copy(
            distanceTargetKm = distanceTargetKm,
            startDirection = null,
            directionStrict = false,
        )
        val shapePreview = coordinatesToLatLng(projectedShape)
        val rejectCounts = mutableMapOf<String, Int>()
        val candidates = mutableListOf<OsrmRouteCandidate>()
        val seenGeometry = mutableSetOf<String>()

        routes.forEachIndexed { index, route ->
            val candidate = toRouteCandidate(shapeRequest, route, index, rejectCounts) ?: return@forEachIndexed
            val geometryKey = geometrySignature(candidate.recommendation.previewLatLng)
            if (geometryKey.isBlank()) {
                incrementRejectCount(rejectCounts, "EMPTY_GEOMETRY_SIGNATURE")
                return@forEachIndexed
            }
            if (!seenGeometry.add(geometryKey)) {
                incrementRejectCount(rejectCounts, "DUPLICATE_GEOMETRY")
                return@forEachIndexed
            }

            val shapeScore = shapeSimilarityScore(candidate.recommendation.previewLatLng, shapePreview)
            val recommendation = candidate.recommendation.copy(
                variantType = RouteVariantType.SHAPE_MATCH,
                shape = "CUSTOM_SHAPE",
                shapeScore = shapeScore,
                matchScore = clampScore(
                    candidate.recommendation.matchScore * 0.35 +
                        shapeScore * 100.0 * 0.65 -
                        candidate.backtrackingRatio * 28.0 -
                        candidate.corridorOverlap * 35.0 -
                        candidate.edgeReuseRatio * 40.0 -
                        candidate.maxAxisReuseRatio * 48.0,
                ),
                reasons = candidate.recommendation.reasons + listOf(
                    "Shape similarity: ${(shapeScore * 100.0).roundToInt()}%",
                    "Shape mode: projected waypoints",
                ),
            )
            candidates += candidate.copy(
                recommendation = recommendation,
                effectiveMatchScore = clampScore(
                    recommendation.matchScore -
                        candidate.backtrackingRatio * 95.0 -
                        candidate.corridorOverlap * 125.0 -
                        candidate.edgeReuseRatio * 140.0 -
                        candidate.maxAxisReuseRatio * 170.0,
                ),
            )
        }

        val recommendations = selectCandidatesWithRelaxation(shapeRequest, candidates, rejectCounts)
            .take(request.limit)

        if (debug || recommendations.isEmpty()) {
            logger.info(
                "OSRM shape generation summary: routeType={} shapePoints={} waypoints={} fetched={} accepted={} rejects={}",
                request.routeType?.trim()?.uppercase(Locale.getDefault()).orEmpty(),
                rawShape.size,
                waypoints.size,
                routes.size,
                recommendations.size,
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

    private fun generateTargetLoopsDisjoint(
        request: RoutingEngineRequest,
        profile: String,
    ): List<RouteRecommendation> {
        val anchors = sampleTargetAnchors(request)
        if (anchors.isEmpty()) {
            return emptyList()
        }
        val hardAxisReuseCap = disjointHardAxisReuseCap(request)

        val rejectCounts = mutableMapOf<String, Int>()
        val candidates = mutableListOf<OsrmRouteCandidate>()
        val seenGeometry = mutableSetOf<String>()
        val maxCandidates = max(24, request.limit * 12)
        var candidateIndex = 0
        var fetchedRouteCount = 0
        var fetchErrors = 0

        loop@ for ((anchorIndex, anchor) in anchors.withIndex()) {
            val outboundRoutes = runCatching {
                fetchRoutes(profile, listOf(request.startPoint, anchor))
            }.onFailure {
                fetchErrors++
                incrementRejectCount(rejectCounts, "OSRM_CALL_FAILED")
            }.getOrElse { emptyList() }
            fetchedRouteCount += outboundRoutes.size
            if (outboundRoutes.isEmpty()) {
                incrementRejectCount(rejectCounts, "NO_OUTBOUND_ROUTE")
                continue
            }

            val outboundCandidates = outboundRoutes.take(3)
            for ((outboundIndex, outboundRoute) in outboundCandidates.withIndex()) {
                val outboundPreview = osrmRouteToPreviewPoints(outboundRoute)
                if (outboundPreview.size < 2) {
                    incrementRejectCount(rejectCounts, "INVALID_OUTBOUND_GEOMETRY")
                    continue
                }

                val returnVariants = buildReturnWaypointVariants(
                    anchor = anchor,
                    start = request.startPoint,
                    startDirection = request.startDirection,
                    seed = anchorIndex + outboundIndex,
                ).take(4)

                for (returnWaypoints in returnVariants) {
                    val inboundRoutes = runCatching {
                        fetchRoutes(profile, returnWaypoints)
                    }.onFailure {
                        fetchErrors++
                        incrementRejectCount(rejectCounts, "OSRM_CALL_FAILED")
                    }.getOrElse { emptyList() }
                    fetchedRouteCount += inboundRoutes.size
                    if (inboundRoutes.isEmpty()) {
                        incrementRejectCount(rejectCounts, "NO_INBOUND_ROUTE")
                        continue
                    }

                    for (inboundRoute in inboundRoutes.take(2)) {
                        val inboundPreview = osrmRouteToPreviewPoints(inboundRoute)
                        if (inboundPreview.size < 2) {
                            incrementRejectCount(rejectCounts, "INVALID_INBOUND_GEOMETRY")
                            continue
                        }
                        val combinedPreview = mergeRoutePreviews(outboundPreview, inboundPreview)
                        if (combinedPreview.size < 2) {
                            incrementRejectCount(rejectCounts, "INVALID_COMBINED_GEOMETRY")
                            continue
                        }

                        val axisStats = evaluateAxisUsage(combinedPreview)
                        // Construction-phase hard rules for v3:
                        // 1) never accept opposite traversal on same axis
                        // 2) cap repeated traversal of a single axis
                        if (axisStats.conflictingAxisCount > 0) {
                            incrementRejectCount(rejectCounts, "NO_DISJOINT_LOOP")
                            continue
                        }
                        if (axisStats.maxAxisReuseCount > hardAxisReuseCap) {
                            incrementRejectCount(rejectCounts, "AXIS_REUSE_HARD_REJECT")
                            continue
                        }

                        val totalDistanceKm = (outboundRoute.distance + inboundRoute.distance) / 1000.0
                        val totalDurationSec = (outboundRoute.duration + inboundRoute.duration).roundToInt()
                        val candidate = toRouteCandidateFromPreview(
                            request = request,
                            preview = combinedPreview,
                            distanceKm = totalDistanceKm,
                            durationSec = totalDurationSec,
                            index = candidateIndex++,
                            rejectCounts = rejectCounts,
                        ) ?: continue
                        val geometryKey = geometrySignature(candidate.recommendation.previewLatLng)
                        if (geometryKey.isBlank()) {
                            incrementRejectCount(rejectCounts, "EMPTY_GEOMETRY_SIGNATURE")
                            continue
                        }
                        if (!seenGeometry.add(geometryKey)) {
                            incrementRejectCount(rejectCounts, "DUPLICATE_GEOMETRY")
                            continue
                        }

                        candidates += candidate.copy(
                            recommendation = candidate.recommendation.copy(
                                reasons = candidate.recommendation.reasons + "Generation engine: disjoint anchors (v3)",
                            ),
                        )
                        if (candidates.size >= maxCandidates) {
                            break@loop
                        }
                    }
                }
            }
        }

        val recommendations = selectCandidatesWithRelaxation(request, candidates, rejectCounts)
            .take(request.limit)

        if (debug || recommendations.isEmpty()) {
            val targetElevation = request.elevationTargetM?.let { value -> "${value.roundToInt()}m" } ?: "n/a"
            logger.info(
                "OSRM target generation v3 summary: routeType={} direction={} target={}km/{} anchors={} fetched={} accepted={} fetchErrors={} rejects={}",
                request.routeType?.trim()?.uppercase(Locale.getDefault()).orEmpty(),
                request.startDirection?.trim()?.uppercase(Locale.getDefault()).orEmpty(),
                String.format("%.1f", request.distanceTargetKm),
                targetElevation,
                anchors.size,
                fetchedRouteCount,
                recommendations.size,
                fetchErrors,
                formatRejectCounts(rejectCounts),
            )
        }

        return recommendations
    }

    private fun sampleTargetAnchors(request: RoutingEngineRequest): List<Coordinates> {
        val baseBearing = startDirectionToBearing(request.startDirection)
        val hasDirection = !request.startDirection.isNullOrBlank()
        val directionStrict = hasDirection && request.directionStrict
        val baseRadiusKm = max(1.0, request.distanceTargetKm / (2.0 * PI))
        val radiusMultipliers = listOf(1.00, 0.92, 1.08, 0.84, 1.16, 1.24, 0.76, 1.32, 0.68, 1.40, 1.48, 0.60)
        var rotations = listOf(0.0, 22.0, -22.0, 45.0, -45.0, 68.0, -68.0, 95.0, -95.0, 125.0, -125.0, 155.0, -155.0)
        if (hasDirection) {
            rotations = listOf(0.0, 8.0, -8.0, 15.0, -15.0, 24.0, -24.0, 32.0, -32.0)
            if (directionStrict) {
                rotations = listOf(0.0, 5.0, -5.0, 10.0, -10.0, 16.0, -16.0)
            }
        }

        val anchors = mutableListOf<Coordinates>()
        val seen = mutableSetOf<String>()
        for (callIndex in 0 until MAX_OSRM_CALLS) {
            val radiusKm = baseRadiusKm * radiusMultipliers[callIndex % radiusMultipliers.size]
            val rotation = rotations[callIndex % rotations.size]
            val anchor = destinationFromBearing(
                start = request.startPoint,
                distanceKm = radiusKm,
                bearingDegrees = normalizeBearing(baseBearing + rotation),
            )
            val key = quantizedPointKey(anchor.lat, anchor.lng)
            if (!seen.add(key)) continue
            anchors += anchor
        }
        return anchors
    }

    private fun buildReturnWaypointVariants(
        anchor: Coordinates,
        start: Coordinates,
        startDirection: String?,
        seed: Int,
    ): List<List<Coordinates>> {
        val distanceKm = max(1.0, haversineDistanceMeters(anchor.lat, anchor.lng, start.lat, start.lng) / 1000.0)
        val directBearing = bearingDegrees(anchor.lat, anchor.lng, start.lat, start.lng)
        val offsets = listOf(58.0, -58.0, 92.0, -92.0, 125.0, -125.0, 155.0, -155.0)
        val scales = listOf(0.48, 0.48, 0.56, 0.56, 0.68, 0.68, 0.80, 0.80)

        val variants = mutableListOf<List<Coordinates>>()
        variants += listOf(anchor, start)
        val shift = if (offsets.isEmpty()) 0 else seed.mod(offsets.size)
        for (index in offsets.indices) {
            val offsetIndex = (shift + index) % offsets.size
            var pivotBearing = normalizeBearing(directBearing + offsets[offsetIndex])
            if (!startDirection.isNullOrBlank()) {
                val directionBearing = startDirectionToBearing(startDirection)
                // Keep global orientation while forcing a clear outbound/inbound separation.
                pivotBearing = normalizeBearing(pivotBearing * 0.72 + directionBearing * 0.28)
            }
            val pivot = destinationFromBearing(
                start = anchor,
                distanceKm = distanceKm * scales[offsetIndex],
                bearingDegrees = pivotBearing,
            )
            variants += listOf(anchor, pivot, start)
        }
        return variants
    }

    private fun osrmRouteToPreviewPoints(route: OsrmRoute): List<List<Double>> {
        val coordinates = route.geometry?.coordinates ?: return emptyList()
        return coordinates.mapNotNull { point ->
            if (point.size < 2) return@mapNotNull null
            val lng = point[0]
            val lat = point[1]
            if (lat !in -90.0..90.0 || lng !in -180.0..180.0) return@mapNotNull null
            listOf(lat, lng)
        }
    }

    private fun mergeRoutePreviews(outbound: List<List<Double>>, inbound: List<List<Double>>): List<List<Double>> {
        if (outbound.isEmpty()) return inbound
        if (inbound.isEmpty()) return outbound
        val merged = outbound.toMutableList()
        var inboundStartIndex = 0
        val inboundStart = inbound.first()
        val outboundEnd = outbound.last()
        if (
            inboundStart.size >= 2 &&
            outboundEnd.size >= 2 &&
            haversineDistanceMeters(inboundStart[0], inboundStart[1], outboundEnd[0], outboundEnd[1]) <= 20.0
        ) {
            inboundStartIndex = 1
        }
        for (index in inboundStartIndex until inbound.size) {
            merged += inbound[index]
        }
        return merged
    }

    override fun healthDetails(): Map<String, Any?> {
        val details = mutableMapOf<String, Any?>(
            "engine" to "osrm",
            "enabled" to enabled,
            "v3Enabled" to v3Enabled,
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
        startDirection: String?,
        callIndex: Int,
    ): List<Coordinates> {
        // Rotate through multiple waypoint "shapes" so OSRM explores distinct loops
        // instead of repeatedly returning the same corridor.
        val circularPatterns = listOf(
            Pair(listOf(0.0, 120.0, 240.0), listOf(1.00, 1.05, 0.95)),
            Pair(listOf(0.0, 85.0, 170.0, 255.0), listOf(1.10, 0.92, 1.08, 0.88)),
            Pair(listOf(0.0, 70.0, 155.0, 230.0, 300.0), listOf(1.00, 1.20, 0.85, 1.10, 0.90)),
            Pair(listOf(0.0, 60.0, 135.0, 210.0, 285.0), listOf(1.15, 0.90, 1.18, 0.86, 1.00)),
        )
        // Directional patterns keep waypoints in the forward half of the compass
        // (relative to requested direction). This guides the loop's global heading.
        val directionalPatterns = listOf(
            Pair(listOf(0.0, 28.0, 56.0, -28.0, -56.0), listOf(1.18, 1.06, 1.06, 0.90, 0.90)),
            Pair(listOf(12.0, 40.0, 70.0, -12.0, -40.0, -70.0), listOf(1.20, 1.20, 1.00, 1.00, 0.82, 0.82)),
            Pair(listOf(0.0, 22.0, 48.0, 78.0, -22.0, -48.0, -78.0), listOf(1.14, 1.12, 1.12, 0.98, 0.98, 0.78, 0.78)),
            Pair(listOf(6.0, 34.0, 62.0, -6.0, -34.0, -62.0), listOf(1.24, 1.24, 1.05, 1.05, 0.86, 0.86)),
        )
        val hasDirection = !startDirection.isNullOrBlank()
        val pattern = if (hasDirection) {
            directionalPatterns[callIndex % directionalPatterns.size]
        } else {
            circularPatterns[callIndex % circularPatterns.size]
        }
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
        val preview = osrmRouteToPreviewPoints(route)
        if (preview.size < 2) {
            incrementRejectCount(rejectCounts, "INVALID_COORDINATES")
            return null
        }
        val distanceKm = route.distance / 1000.0
        val durationSec = route.duration.toInt().coerceAtLeast((distanceKm * 180.0).toInt())
        return toRouteCandidateFromPreview(
            request = request,
            preview = preview,
            distanceKm = distanceKm,
            durationSec = durationSec,
            index = index,
            rejectCounts = rejectCounts,
        )
    }

    private fun toRouteCandidateFromPreview(
        request: RoutingEngineRequest,
        preview: List<List<Double>>,
        distanceKm: Double,
        durationSec: Int,
        index: Int,
        rejectCounts: MutableMap<String, Int>,
    ): OsrmRouteCandidate? {
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
        val directionPenalty = combinedDirectionPenalty(preview, request.startPoint, request.startDirection, DIRECTION_TOLERANCE_METERS)
        val axisStats = evaluateAxisUsage(preview)
        val backtrackingRatio = axisStats.oppositeTraversalRatio()
        val corridorOverlap = corridorOverlapRatio(preview)
        val edgeReuse = axisStats.reuseRatio()
        val maxAxisReuseCount = axisStats.maxAxisReuseCount
        val maxAxisReuseRatio = axisStats.maxAxisReuseRatio()
        val diversityRatio = axisStats.segmentDiversityRatio()
        val distanceDeltaRatio = abs(distanceKm - request.distanceTargetKm) / max(1.0, request.distanceTargetKm)
        if (backtrackingRatio > 0.32 || corridorOverlap > 0.30 || edgeReuse > 0.28 || maxAxisReuseCount > 8) {
            incrementRejectCount(rejectCounts, "EXCESSIVE_RETRACE")
            return null
        }
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
            add("Axis retrace: ${(edgeReuse * 100.0).roundToInt()}%")
            add("Max axis reuse: ${maxAxisReuseCount}x")
            request.elevationTargetM?.let { target ->
                add("Elevation estimate: ${formatElevationDelta(elevationEstimate - target)}")
            }
            request.startDirection?.takeIf { it.isNotBlank() }?.let { direction ->
                add("Direction: ${direction.uppercase(Locale.getDefault())}")
            }
            add("Anti-backtracking: native ultra")
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
            durationSec = durationSec.coerceAtLeast((distanceKm * 180.0).toInt()),
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
                backtrackingRatio * 90.0 -
                corridorOverlap * 140.0 -
                edgeReuse * 150.0 -
                maxAxisReuseRatio * 180.0 -
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
            edgeReuseRatio = edgeReuse,
            maxAxisReuseCount = maxAxisReuseCount,
            maxAxisReuseRatio = maxAxisReuseRatio,
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
        val limit = request.limit.coerceAtLeast(1)
        val sortedCandidates = candidates.sortedWith(
            compareBy<OsrmRouteCandidate> { it.corridorOverlap }
                .thenBy { it.backtrackingRatio }
                .thenBy { it.edgeReuseRatio }
                .thenBy { it.maxAxisReuseCount }
                .thenByDescending { it.effectiveMatchScore }
                .thenBy { it.directionPenalty }
                .thenByDescending { it.recommendation.matchScore }
                .thenBy { it.distanceDeltaRatio }
                .thenBy { it.recommendation.routeId },
        )
        // Levels are evaluated in order: strict -> balanced -> relaxed -> fallback.
        // We fill results incrementally: if strict cannot fill the target limit,
        // next levels progressively loosen constraints while keeping quality.
        val hasDirection = !request.startDirection.isNullOrBlank()
        val levels = buildRouteRelaxationLevels(
            routeType = request.routeType,
            hasDirection = hasDirection,
            directionStrict = request.directionStrict,
            distanceTargetKm = request.distanceTargetKm,
        )
        val selected = mutableListOf<RouteRecommendation>()
        val selectedIds = mutableSetOf<String>()

        for (level in levels) {
            if (selected.size >= limit) break
            for (candidate in sortedCandidates) {
                if (selected.size >= limit) break
                if (selectedIds.contains(candidate.recommendation.routeId)) continue
                if (candidate.directionPenalty > level.maxDirectionPenalty) {
                    incrementRejectCount(rejectCounts, "DIRECTION_CONSTRAINT")
                    continue
                }
                if (candidate.backtrackingRatio > level.maxBacktrackingRatio) {
                    incrementRejectCount(rejectCounts, "OPPOSITE_EDGE_TRAVERSAL")
                    continue
                }
                if (candidate.corridorOverlap > level.maxCorridorOverlap) {
                    incrementRejectCount(rejectCounts, "CORRIDOR_OVERLAP")
                    continue
                }
                if (candidate.edgeReuseRatio > level.maxEdgeReuseRatio) {
                    incrementRejectCount(rejectCounts, "EDGE_REUSE")
                    continue
                }
                if (candidate.maxAxisReuseCount > level.maxAxisReuseCount) {
                    incrementRejectCount(rejectCounts, "MAX_AXIS_REUSE")
                    continue
                }
                if (candidate.segmentDiversity < level.minSegmentDiversity) {
                    incrementRejectCount(rejectCounts, "LOW_SEGMENT_DIVERSITY")
                    continue
                }
                if (candidate.distanceDeltaRatio > level.maxDistanceDeltaRatio) {
                    incrementRejectCount(rejectCounts, "DISTANCE_CONSTRAINT")
                    continue
                }
                selectedIds += candidate.recommendation.routeId
                selected += candidate.recommendation.copy(
                    reasons = candidate.recommendation.reasons + "Selection profile: ${level.name}",
                )
            }
        }

        // Safety net: when strict/balanced/relaxed/fallback all reject candidates,
        // return best-ranked loops with softer anti-overlap thresholds instead of 0 result.
        val (softAxisCap, directionalAxisCap) = bestEffortAxisReuseCaps(
            distanceTargetKm = request.distanceTargetKm,
            hasDirection = hasDirection,
            directionStrict = request.directionStrict,
        )
        if (selected.size < limit) {
            var softMaxBacktracking = 0.10
            var softMaxCorridor = 0.08
            var softMaxEdgeReuse = 0.08
            var softMaxDirection = 1.0
            // Directional generation naturally creates more corridor pressure.
            // We relax slightly, but stay far from permissive settings.
            if (hasDirection) {
                softMaxBacktracking = 0.14
                softMaxCorridor = 0.11
                softMaxEdgeReuse = 0.10
                softMaxDirection = 0.40
            }
            appendBestEffortCandidates(
                sortedCandidates = sortedCandidates,
                selected = selected,
                selectedIds = selectedIds,
                limit = limit,
                maxDirectionPenalty = softMaxDirection,
                maxBacktrackingRatio = softMaxBacktracking,
                maxCorridorOverlap = softMaxCorridor,
                maxEdgeReuseRatio = softMaxEdgeReuse,
                maxAxisReuseCount = softAxisCap,
                profileName = "best-effort-soft",
            )
        }
        if (selected.size < limit && hasDirection) {
            // Last safety net in directional mode: keep anti-retrace filters, but relax them
            // just enough to avoid returning zero route too often.
            appendBestEffortCandidates(
                sortedCandidates = sortedCandidates,
                selected = selected,
                selectedIds = selectedIds,
                limit = limit,
                maxDirectionPenalty = 0.52,
                maxBacktrackingRatio = 0.18,
                maxCorridorOverlap = 0.14,
                maxEdgeReuseRatio = 0.13,
                maxAxisReuseCount = directionalAxisCap,
                profileName = "directional-best-effort",
            )
        }

        return selected
    }

    private fun appendBestEffortCandidates(
        sortedCandidates: List<OsrmRouteCandidate>,
        selected: MutableList<RouteRecommendation>,
        selectedIds: MutableSet<String>,
        limit: Int,
        maxDirectionPenalty: Double,
        maxBacktrackingRatio: Double,
        maxCorridorOverlap: Double,
        maxEdgeReuseRatio: Double,
        maxAxisReuseCount: Int,
        profileName: String,
    ) {
        for (candidate in sortedCandidates) {
            if (selected.size >= limit) break
            if (selectedIds.contains(candidate.recommendation.routeId)) continue
            if (candidate.directionPenalty > maxDirectionPenalty) continue
            if (candidate.backtrackingRatio > maxBacktrackingRatio) continue
            if (candidate.corridorOverlap > maxCorridorOverlap) continue
            if (candidate.edgeReuseRatio > maxEdgeReuseRatio) continue
            if (candidate.maxAxisReuseCount > maxAxisReuseCount) continue
            selectedIds += candidate.recommendation.routeId
            selected += candidate.recommendation.copy(
                reasons = candidate.recommendation.reasons + "Selection profile: $profileName",
            )
        }
    }

    private fun buildRouteRelaxationLevels(
        routeType: String?,
        hasDirection: Boolean,
        directionStrict: Boolean,
        distanceTargetKm: Double,
    ): List<RouteRelaxationLevel> {
        var baseMinDiversity = minSegmentDiversityRatio(routeType)
        val strictDirection = if (hasDirection) 0.18 else 1.0
        val balancedDirection = if (hasDirection) 0.28 else 1.0
        val relaxedDirection = if (hasDirection) 0.40 else 1.0
        val fallbackDirection = if (hasDirection) 0.52 else 1.0
        val strictLevelDirection = if (hasDirection && directionStrict) 0.10 else strictDirection
        val balancedLevelDirection = if (hasDirection && directionStrict) 0.16 else balancedDirection
        val relaxedLevelDirection = if (hasDirection && directionStrict) 0.22 else relaxedDirection
        val fallbackLevelDirection = if (hasDirection && directionStrict) 0.30 else fallbackDirection

        // Native ultra anti-backtracking policy (always-on).
        baseMinDiversity = (baseMinDiversity + 0.12).coerceAtMost(0.95)
        val strictBacktrackingRatio = 0.0010
        val balancedBacktrackingRatio = 0.0030
        val relaxedBacktrackingRatio = 0.0070
        val fallbackBacktrackingRatio = 0.015
        val strictCorridorOverlap = 0.003
        val balancedCorridorOverlap = 0.007
        val relaxedCorridorOverlap = 0.012
        val fallbackCorridorOverlap = 0.018
        val strictEdgeReuseRatio = 0.008
        val balancedEdgeReuseRatio = 0.020
        val relaxedEdgeReuseRatio = 0.040
        val fallbackEdgeReuseRatio = 0.065
        val (strictAxisCap, balancedAxisCap, relaxedAxisCap, fallbackAxisCap) = adaptiveAxisReuseThresholds(
            distanceTargetKm = distanceTargetKm,
            hasDirection = hasDirection,
            directionStrict = directionStrict,
        )

        return listOf(
            RouteRelaxationLevel(
                name = "strict",
                maxDirectionPenalty = strictLevelDirection,
                maxBacktrackingRatio = strictBacktrackingRatio,
                maxCorridorOverlap = strictCorridorOverlap,
                maxEdgeReuseRatio = strictEdgeReuseRatio,
                maxAxisReuseCount = strictAxisCap,
                minSegmentDiversity = baseMinDiversity,
                maxDistanceDeltaRatio = 0.35,
            ),
            RouteRelaxationLevel(
                name = "balanced",
                maxDirectionPenalty = balancedLevelDirection,
                maxBacktrackingRatio = balancedBacktrackingRatio,
                maxCorridorOverlap = balancedCorridorOverlap,
                maxEdgeReuseRatio = balancedEdgeReuseRatio,
                maxAxisReuseCount = balancedAxisCap,
                minSegmentDiversity = max(0.22, baseMinDiversity - 0.08),
                maxDistanceDeltaRatio = 0.60,
            ),
            RouteRelaxationLevel(
                name = "relaxed",
                maxDirectionPenalty = relaxedLevelDirection,
                maxBacktrackingRatio = relaxedBacktrackingRatio,
                maxCorridorOverlap = relaxedCorridorOverlap,
                maxEdgeReuseRatio = relaxedEdgeReuseRatio,
                maxAxisReuseCount = relaxedAxisCap,
                minSegmentDiversity = max(0.12, baseMinDiversity - 0.18),
                maxDistanceDeltaRatio = 1.00,
            ),
            RouteRelaxationLevel(
                name = "fallback",
                maxDirectionPenalty = fallbackLevelDirection,
                maxBacktrackingRatio = fallbackBacktrackingRatio,
                maxCorridorOverlap = fallbackCorridorOverlap,
                maxEdgeReuseRatio = fallbackEdgeReuseRatio,
                maxAxisReuseCount = fallbackAxisCap,
                minSegmentDiversity = 0.08,
                maxDistanceDeltaRatio = 2.20,
            ),
        )
    }

    private fun adaptiveAxisReuseThresholds(
        distanceTargetKm: Double,
        hasDirection: Boolean,
        directionStrict: Boolean,
    ): List<Int> {
        var strictCap = 2
        var balancedCap = 3
        var relaxedCap = 4
        var fallbackCap = 5
        when {
            distanceTargetKm >= 130.0 -> {
                strictCap = 4
                balancedCap = 5
                relaxedCap = 6
                fallbackCap = 8
            }
            distanceTargetKm >= 90.0 -> {
                strictCap = 3
                balancedCap = 4
                relaxedCap = 6
                fallbackCap = 7
            }
            distanceTargetKm >= 60.0 -> {
                strictCap = 3
                balancedCap = 4
                relaxedCap = 5
                fallbackCap = 6
            }
            distanceTargetKm >= 30.0 -> {
                strictCap = 2
                balancedCap = 3
                relaxedCap = 5
                fallbackCap = 6
            }
        }
        if (hasDirection) {
            strictCap++
            balancedCap++
            relaxedCap++
            fallbackCap++
        }
        if (directionStrict) {
            strictCap++
            balancedCap++
        }
        return listOf(
            strictCap.coerceIn(2, 6),
            balancedCap.coerceIn(3, 7),
            relaxedCap.coerceIn(4, 8),
            fallbackCap.coerceIn(5, 9),
        )
    }

    private fun bestEffortAxisReuseCaps(distanceTargetKm: Double, hasDirection: Boolean, directionStrict: Boolean): Pair<Int, Int> {
        val fallbackCap = adaptiveAxisReuseThresholds(distanceTargetKm, hasDirection, directionStrict)[3]
        val softCap = (fallbackCap + 1).coerceIn(6, 10)
        val directionalCap = (fallbackCap + 2).coerceIn(7, 11)
        return softCap to directionalCap
    }

    private fun disjointHardAxisReuseCap(request: RoutingEngineRequest): Int {
        val hasDirection = !request.startDirection.isNullOrBlank()
        val thresholds = adaptiveAxisReuseThresholds(request.distanceTargetKm, hasDirection, request.directionStrict)
        val relaxedCap = thresholds[2]
        val fallbackCap = thresholds[3]
        // Construction phase should stay tighter than post-selection fallback.
        return max(relaxedCap, fallbackCap - 1).coerceIn(4, 8)
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

    private fun parseShapePolylineCoordinates(raw: String): List<Coordinates> {
        val trimmed = raw.trim()
        if (!trimmed.startsWith("[")) return emptyList()
        val points = runCatching { mapper.readValue<List<List<Double>>>(trimmed) }.getOrElse { emptyList() }
        return points.mapNotNull { point ->
            if (point.size < 2) return@mapNotNull null
            val lat = point[0]
            val lng = point[1]
            if (lat !in -90.0..90.0 || lng !in -180.0..180.0) return@mapNotNull null
            Coordinates(lat = lat, lng = lng)
        }
    }

    private fun polylineDistanceKmFromCoordinates(points: List<Coordinates>): Double {
        if (points.size < 2) return 0.0
        var totalMeters = 0.0
        for (index in 0 until points.size - 1) {
            val left = points[index]
            val right = points[index + 1]
            totalMeters += haversineDistanceMeters(left.lat, left.lng, right.lat, right.lng)
        }
        return totalMeters / 1000.0
    }

    private fun projectShapePolylineToStart(
        shape: List<Coordinates>,
        start: Coordinates,
        targetDistanceKm: Double,
    ): List<Coordinates> {
        if (shape.isEmpty()) return emptyList()
        val translated = buildList(shape.size) {
            val deltaLat = start.lat - shape.first().lat
            val deltaLng = start.lng - shape.first().lng
            shape.forEach { point ->
                add(
                    Coordinates(
                        lat = point.lat + deltaLat,
                        lng = point.lng + deltaLng,
                    )
                )
            }
        }

        var scale = 1.0
        val shapeDistanceKm = polylineDistanceKmFromCoordinates(translated)
        if (targetDistanceKm > 0.0 && shapeDistanceKm > 0.0) {
            scale = (targetDistanceKm / shapeDistanceKm).coerceIn(0.45, 2.60)
        }

        return buildList(translated.size) {
            add(start)
            for (index in 1 until translated.size) {
                val point = translated[index]
                add(
                    Coordinates(
                        lat = start.lat + (point.lat - start.lat) * scale,
                        lng = start.lng + (point.lng - start.lng) * scale,
                    )
                )
            }
        }
    }

    private fun sampleCoordinates(points: List<Coordinates>, maxPoints: Int): List<Coordinates> {
        if (points.size <= maxPoints || maxPoints <= 0) {
            return points
        }
        val step = max(1, ceil(points.size.toDouble() / maxPoints.toDouble()).toInt())
        val sampled = mutableListOf<Coordinates>()
        for (index in points.indices step step) {
            sampled += points[index]
        }
        val lastSample = sampled.lastOrNull()
        val lastPoint = points.last()
        if (lastSample == null || lastSample.lat != lastPoint.lat || lastSample.lng != lastPoint.lng) {
            sampled += lastPoint
        }
        return sampled
    }

    private fun buildShapeLoopWaypoints(start: Coordinates, shape: List<Coordinates>): List<Coordinates> {
        val sampled = sampleCoordinates(shape, 10)
        val waypoints = mutableListOf(start)
        var previous = start
        for (index in 1 until sampled.size) {
            val point = sampled[index]
            if (haversineDistanceMeters(previous.lat, previous.lng, point.lat, point.lng) < 120.0) {
                continue
            }
            waypoints += point
            previous = point
        }
        waypoints += start
        return waypoints
    }

    private fun coordinatesToLatLng(points: List<Coordinates>): List<List<Double>> {
        return points.map { point -> listOf(point.lat, point.lng) }
    }

    private fun shapeSimilarityScore(routePoints: List<List<Double>>, shapePoints: List<List<Double>>): Double {
        val normalizedRoute = normalizeShapePolyline(samplePolylinePoints(routePoints, 90))
        val normalizedShape = normalizeShapePolyline(samplePolylinePoints(shapePoints, 90))
        if (normalizedRoute.size < 2 || normalizedShape.size < 2) {
            return 0.0
        }
        val meanForward = meanNearestShapeDistance(normalizedShape, normalizedRoute)
        val meanBackward = meanNearestShapeDistance(normalizedRoute, normalizedShape)
        val distance = (meanForward + meanBackward) / 2.0
        val score = 1.0 - (distance / 1.35)
        return clampUnit(score)
    }

    private fun normalizeShapePolyline(points: List<List<Double>>): List<NormalizedShapePoint> {
        if (points.isEmpty()) return emptyList()
        var sumLat = 0.0
        var sumLng = 0.0
        var count = 0
        points.forEach { point ->
            if (point.size < 2) return@forEach
            sumLat += point[0]
            sumLng += point[1]
            count++
        }
        if (count == 0) return emptyList()
        val centerLat = sumLat / count.toDouble()
        val centerLng = sumLng / count.toDouble()
        val cosLat = cos(Math.toRadians(centerLat))
        var maxRadius = 0.0
        val normalized = mutableListOf<NormalizedShapePoint>()
        points.forEach { point ->
            if (point.size < 2) return@forEach
            val x = (point[1] - centerLng) * 111320.0 * cosLat
            val y = (point[0] - centerLat) * 111320.0
            val radius = sqrt(x * x + y * y)
            if (radius > maxRadius) {
                maxRadius = radius
            }
            normalized += NormalizedShapePoint(x = x, y = y)
        }
        if (maxRadius < 1.0) {
            maxRadius = 1.0
        }
        normalized.forEach { point ->
            point.x /= maxRadius
            point.y /= maxRadius
        }
        return normalized
    }

    private fun meanNearestShapeDistance(from: List<NormalizedShapePoint>, to: List<NormalizedShapePoint>): Double {
        if (from.isEmpty() || to.isEmpty()) return 1.0
        var total = 0.0
        from.forEach { left ->
            var minDistance = Double.MAX_VALUE
            to.forEach { right ->
                val dx = left.x - right.x
                val dy = left.y - right.y
                val distance = sqrt(dx * dx + dy * dy)
                if (distance < minDistance) {
                    minDistance = distance
                }
            }
            total += minDistance
        }
        return total / from.size.toDouble()
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
        // We combine three direction signals:
        // - initial heading alignment (bearing-based)
        // - half-plane violations (did the route go too much in the opposite side)
        // - global lobe dominance (does the whole loop stay mostly in requested direction)
        // Taking the max keeps direction enforcement robust in dense urban grids.
        // Bearing is intentionally softened because local street orientation near
        // the start can be briefly opposite to the desired global direction.
        val bearingPenalty = directionPenaltyFromPreview(points, direction)
        val halfPlanePenalty = halfPlaneViolationRatio(points, start, direction, toleranceMeters)
        val lobePenalty = directionalLobePenalty(points, start, direction)
        return max(max(bearingPenalty * 0.65, halfPlanePenalty), lobePenalty)
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

    private fun directionalLobePenalty(
        points: List<List<Double>>,
        start: Coordinates,
        direction: String?,
    ): Double {
        val normalized = direction.orEmpty().trim().uppercase(Locale.getDefault())
        if (normalized.isBlank() || points.isEmpty()) return 0.0

        var desiredExtent = 0.0
        var oppositeExtent = 0.0
        var sumProjection = 0.0
        var projectionCount = 0

        for (point in points) {
            if (point.size < 2) continue
            val projection = directionProjectionMeters(point[0], point[1], start, normalized) ?: continue
            if (projection > desiredExtent) {
                desiredExtent = projection
            }
            if (projection < 0 && -projection > oppositeExtent) {
                oppositeExtent = -projection
            }
            sumProjection += projection
            projectionCount++
        }

        if (projectionCount == 0) return 0.0

        // Dominance asks: "how much of the route envelope is on requested side?"
        // 1.0 means full dominance on requested side, 0.5 is symmetric, 0 is opposite.
        var dominancePenalty = 0.0
        val totalExtent = desiredExtent + oppositeExtent
        if (totalExtent > 1.0) {
            val dominanceRatio = desiredExtent / totalExtent
            dominancePenalty = clampUnit((0.63 - dominanceRatio) / 0.63)
        }

        // Average projection guard: route center of mass should not drift opposite.
        var avgPenalty = 0.0
        if (desiredExtent > 1.0) {
            val avgProjection = sumProjection / projectionCount.toDouble()
            avgPenalty = clampUnit((-avgProjection) / max(desiredExtent * 0.25, 1.0))
        }

        return max(dominancePenalty, avgPenalty)
    }

    private fun directionProjectionMeters(
        lat: Double,
        lng: Double,
        start: Coordinates,
        normalizedDirection: String,
    ): Double? {
        val latMeters = (lat - start.lat) * 111320.0
        val lngMeters = (lng - start.lng) * 111320.0 * cos(Math.toRadians(start.lat))
        return when (normalizedDirection) {
            "N" -> latMeters
            "S" -> -latMeters
            "E" -> lngMeters
            "W" -> -lngMeters
            else -> null
        }
    }

    private fun clampUnit(value: Double): Double {
        return when {
            value <= 0.0 -> 0.0
            value >= 1.0 -> 1.0
            else -> value
        }
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

    private data class AxisTraversal(
        val axisId: String,
        val isForward: Boolean,
    )

    private data class AxisUsageSummary(
        val totalTraversals: Int,
        val uniqueAxisCount: Int,
        val conflictingAxisCount: Int,
        val reusedTraversals: Int,
        val maxAxisReuseCount: Int,
    ) {
        fun oppositeTraversalRatio(): Double {
            if (totalTraversals == 0) return 0.0
            return conflictingAxisCount.toDouble() / totalTraversals.toDouble()
        }

        fun reuseRatio(): Double {
            if (totalTraversals == 0) return 0.0
            return reusedTraversals.toDouble() / totalTraversals.toDouble()
        }

        fun segmentDiversityRatio(): Double {
            if (totalTraversals == 0) return 0.0
            return uniqueAxisCount.toDouble() / totalTraversals.toDouble()
        }

        fun maxAxisReuseRatio(): Double {
            if (totalTraversals == 0) return 0.0
            return maxAxisReuseCount.toDouble() / totalTraversals.toDouble()
        }
    }

    private fun hasOppositeEdgeTraversal(points: List<List<Double>>): Boolean {
        return evaluateAxisUsage(points).conflictingAxisCount > 0
    }

    private fun evaluateAxisUsage(points: List<List<Double>>): AxisUsageSummary {
        val traversals = extractAxisTraversals(points)
        if (traversals.isEmpty()) {
            return AxisUsageSummary(
                totalTraversals = 0,
                uniqueAxisCount = 0,
                conflictingAxisCount = 0,
                reusedTraversals = 0,
                maxAxisReuseCount = 0,
            )
        }
        val axisCounts = mutableMapOf<String, Int>()
        val axisDirections = mutableMapOf<String, Int>()
        var maxAxisReuseCount = 0

        traversals.forEach { traversal ->
            val count = (axisCounts[traversal.axisId] ?: 0) + 1
            axisCounts[traversal.axisId] = count
            if (count > maxAxisReuseCount) {
                maxAxisReuseCount = count
            }
            val currentDirectionMask = axisDirections[traversal.axisId] ?: 0
            val updatedDirectionMask = if (traversal.isForward) {
                currentDirectionMask or 0b01
            } else {
                currentDirectionMask or 0b10
            }
            axisDirections[traversal.axisId] = updatedDirectionMask
        }

        var conflictingAxisCount = 0
        var reusedTraversals = 0
        axisCounts.forEach { (axisId, count) ->
            if ((axisDirections[axisId] ?: 0) == 0b11) {
                conflictingAxisCount++
            }
            if (count > 1) {
                reusedTraversals += count - 1
            }
        }

        return AxisUsageSummary(
            totalTraversals = traversals.size,
            uniqueAxisCount = axisCounts.size,
            conflictingAxisCount = conflictingAxisCount,
            reusedTraversals = reusedTraversals,
            maxAxisReuseCount = maxAxisReuseCount,
        )
    }

    private fun extractAxisTraversals(points: List<List<Double>>): List<AxisTraversal> {
        if (points.size < 3) return emptyList()
        return buildList(points.size - 1) {
            for (index in 0 until points.size - 1) {
                val from = points[index]
                val to = points[index + 1]
                if (from.size < 2 || to.size < 2) continue
                val fromId = quantizedPointKey(from[0], from[1])
                val toId = quantizedPointKey(to[0], to[1])
                if (fromId == toId) continue
                add(
                    AxisTraversal(
                        axisId = canonicalEdgeKey(fromId, toId),
                        isForward = fromId < toId,
                    ),
                )
            }
        }
    }

    private fun oppositeEdgeTraversalRatio(points: List<List<Double>>): Double {
        return evaluateAxisUsage(points).oppositeTraversalRatio()
    }

    private fun edgeReuseRatio(points: List<List<Double>>): Double {
        return evaluateAxisUsage(points).reuseRatio()
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
        val axisStats = evaluateAxisUsage(points)
        if (axisStats.totalTraversals == 0) return false
        // Allow local loops, but reject routes that hammer the exact same axis too often.
        if (axisStats.maxAxisReuseCount > 3) return false
        return axisStats.segmentDiversityRatio() >= minSegmentDiversityRatio(routeType)
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
        return evaluateAxisUsage(points).segmentDiversityRatio()
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
