package me.nicolas.stravastats.domain.services

import me.nicolas.stravastats.domain.RuntimeConfig
import me.nicolas.stravastats.domain.business.ActivityShort
import me.nicolas.stravastats.domain.business.ActivityType
import me.nicolas.stravastats.domain.business.Coordinates
import me.nicolas.stravastats.domain.business.RouteExplorerRequest
import me.nicolas.stravastats.domain.business.RouteExplorerResult
import me.nicolas.stravastats.domain.business.RouteRecommendation
import me.nicolas.stravastats.domain.business.RouteVariantType
import me.nicolas.stravastats.domain.business.ShapeRemixRecommendation
import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.services.activityproviders.IActivityProvider
import me.nicolas.stravastats.domain.services.routing.RoutingEnginePort
import me.nicolas.stravastats.domain.services.routing.RoutingHistoryProfile
import me.nicolas.stravastats.domain.services.routing.RoutingEngineRequest
import me.nicolas.stravastats.domain.services.routing.buildRoutingHistoryProfile
import org.springframework.stereotype.Service
import java.time.Instant
import java.time.LocalDate
import java.time.LocalDateTime
import java.time.OffsetDateTime
import java.time.ZoneOffset
import java.time.format.DateTimeFormatter
import java.util.PriorityQueue
import kotlin.math.PI
import kotlin.math.abs
import kotlin.math.atan2
import kotlin.math.cos
import kotlin.math.max
import kotlin.math.min
import kotlin.math.pow
import kotlin.math.round
import kotlin.math.roundToInt
import kotlin.math.sin
import kotlin.math.sqrt

interface IRouteExplorerService {
    fun getRouteExplorer(activityTypes: Set<ActivityType>, year: Int?, request: RouteExplorerRequest): RouteExplorerResult
}

@Service
class RouteExplorerService(
    activityProvider: IActivityProvider,
    private val routingEngine: RoutingEnginePort,
) : IRouteExplorerService, AbstractStravaService(activityProvider) {

    companion object {
        private const val DEFAULT_ROUTE_LIMIT = 5
        private const val MAX_ROUTE_LIMIT = 24
        private const val PREVIEW_POINT_MAX_SIZE = 120
        private const val DEFAULT_HISTORY_HALF_LIFE_DAYS = 75
    }

    override fun getRouteExplorer(
        activityTypes: Set<ActivityType>,
        year: Int?,
        request: RouteExplorerRequest,
    ): RouteExplorerResult {
        val activities = activityProvider.getActivitiesByActivityTypeAndYear(activityTypes, year)
            .withDataQualityCorrections(activityProvider)
        val candidates = buildRouteCandidates(activities)
        val limit = normalizeLimit(request.limit)
        val generatedWithoutCache = buildRoadGraphRecommendationsFromEngine(
            request = request,
            distanceTarget = request.distanceTargetKm ?: 0.0,
            elevationTarget = request.elevationTargetM ?: 0.0,
            limit = limit,
            fallback = emptyList(),
            historyBiasEnabled = false,
            historyProfile = null,
        )
        if (candidates.isEmpty()) {
            val generatedShapeWithoutCache = buildShapeMatchRecommendationsFromEngine(
                request = request,
                distanceTarget = request.distanceTargetKm ?: 0.0,
                elevationTarget = request.elevationTargetM ?: 0.0,
                limit = limit,
                historyBiasEnabled = false,
                historyProfile = null,
            )
            return RouteExplorerResult(
                closestLoops = emptyList(),
                variants = emptyList(),
                seasonal = emptyList(),
                roadGraphLoops = generatedWithoutCache,
                shapeMatches = generatedShapeWithoutCache,
                shapeRemixes = emptyList(),
            )
        }

        val distanceTarget = request.distanceTargetKm?.takeIf { value -> value > 0 }
            ?: median(candidates.map { candidate -> candidate.distanceKm }, 45.0)
        val elevationTarget = request.elevationTargetM?.takeIf { value -> value > 0 }
            ?: median(candidates.map { candidate -> candidate.elevationGainM }, 600.0)
        val durationTargetSec = request.durationTargetMin?.takeIf { value -> value > 0 }?.times(60)
            ?: median(candidates.map { candidate -> candidate.durationSec.toDouble() }, 2.5 * 3600.0).roundToInt()
        val routeType = normalizeRouteType(request.routeType)
        val startDirection = normalizeStartDirection(request.startDirection)
        val preferredStart = normalizePreferredStartPoint(request.startPoint)
        val historyBiasEnabled = isHistoryBiasEnabled()
        val historyProfile = if (historyBiasEnabled) {
            buildRoutingHistoryProfile(
                activities = activities,
                routeType = routeType,
                now = Instant.now(),
                halfLifeDays = historyHalfLifeDays().toDouble(),
            )
        } else {
            null
        }
        val scoringProfile = buildRouteScoringProfile(routeType, startDirection, preferredStart != null)

        val seasonFilter = normalizeSeason(request.season)
        val shapeFilter = normalizeShape(request.shape)
        val baseCandidates = filterBySeason(candidates, seasonFilter).ifEmpty { candidates }

        val closest = buildClosestLoopRecommendations(
            baseCandidates,
            distanceTarget,
            elevationTarget,
            durationTargetSec,
            scoringProfile,
            startDirection,
            preferredStart,
            limit,
        )
        val variants = buildSmartVariants(
            baseCandidates,
            distanceTarget,
            elevationTarget,
            durationTargetSec,
            scoringProfile,
            startDirection,
            preferredStart,
        )
        val seasonal = buildSeasonalRecommendations(candidates, seasonFilter, distanceTarget, elevationTarget, durationTargetSec, limit)
        val roadGraphFromCache = buildRoadGraphRecommendations(
            candidates = baseCandidates,
            distanceTarget = distanceTarget,
            elevationTarget = elevationTarget,
            durationTargetSec = durationTargetSec,
            scoringProfile = scoringProfile,
            startDirection = startDirection,
            preferredStart = preferredStart,
            limit = limit,
        )
        val roadGraphFallback = if (roadGraphFromCache.isNotEmpty()) {
            roadGraphFromCache
        } else {
            closest.take(limit)
        }
        val roadGraphLoops = buildRoadGraphRecommendationsFromEngine(
            request = request,
            distanceTarget = distanceTarget,
            elevationTarget = elevationTarget,
            limit = limit,
            fallback = roadGraphFallback,
            historyBiasEnabled = historyBiasEnabled,
            historyProfile = historyProfile,
        )
        val shapeMatchesFromCache = buildShapeMatchRecommendations(baseCandidates, shapeFilter, distanceTarget, elevationTarget, durationTargetSec, limit)
        val shapeMatchesFromEngine = buildShapeMatchRecommendationsFromEngine(
            request = request,
            distanceTarget = distanceTarget,
            elevationTarget = elevationTarget,
            limit = limit,
            historyBiasEnabled = historyBiasEnabled,
            historyProfile = historyProfile,
        )
        val shapeMatches = mergeRouteRecommendations(
            primary = shapeMatchesFromEngine,
            secondary = shapeMatchesFromCache,
            limit = limit,
        )
        val remixes = if (request.includeRemix) {
            buildShapeRemixRecommendations(baseCandidates, distanceTarget, elevationTarget, durationTargetSec, limit)
        } else {
            emptyList()
        }

        return RouteExplorerResult(
            closestLoops = closest,
            variants = variants,
            seasonal = seasonal,
            roadGraphLoops = roadGraphLoops,
            shapeMatches = shapeMatches,
            shapeRemixes = remixes,
        )
    }

    private fun buildRoadGraphRecommendationsFromEngine(
        request: RouteExplorerRequest,
        distanceTarget: Double,
        elevationTarget: Double,
        limit: Int,
        fallback: List<RouteRecommendation>,
        historyBiasEnabled: Boolean,
        historyProfile: RoutingHistoryProfile?,
    ): List<RouteRecommendation> {
        val start = request.startPoint
        if (start == null || distanceTarget <= 0.0 || limit <= 0) {
            return fallback
        }

        val generated = runCatching {
            routingEngine.generateTargetLoops(
                RoutingEngineRequest(
                    startPoint = start,
                    distanceTargetKm = distanceTarget,
                    elevationTargetM = request.elevationTargetM ?: elevationTarget,
                    startDirection = request.startDirection,
                    directionStrict = request.strictDirection,
                    strictBacktracking = request.strictBacktracking,
                    backtrackingProfile = request.backtrackingProfile,
                    targetMode = request.targetMode,
                    waypoints = request.customWaypoints,
                    shapePolyline = request.shapePolyline,
                    routeType = request.routeType,
                    limit = limit,
                    historyBiasEnabled = historyBiasEnabled,
                    historyProfile = historyProfile,
                )
            )
        }.getOrElse {
            emptyList()
        }

        return if (generated.isNotEmpty()) generated else fallback
    }

    private fun buildShapeMatchRecommendationsFromEngine(
        request: RouteExplorerRequest,
        distanceTarget: Double,
        elevationTarget: Double,
        limit: Int,
        historyBiasEnabled: Boolean,
        historyProfile: RoutingHistoryProfile?,
    ): List<RouteRecommendation> {
        val start = request.startPoint
        val shapePolyline = request.shapePolyline?.trim()
        if (start == null || shapePolyline.isNullOrBlank() || limit <= 0) {
            return emptyList()
        }
        return runCatching {
            routingEngine.generateShapeLoops(
                RoutingEngineRequest(
                    startPoint = start,
                    distanceTargetKm = request.distanceTargetKm ?: distanceTarget,
                    elevationTargetM = request.elevationTargetM ?: elevationTarget,
                    startDirection = null,
                    directionStrict = false,
                    strictBacktracking = request.strictBacktracking,
                    backtrackingProfile = request.backtrackingProfile,
                    targetMode = request.targetMode,
                    waypoints = emptyList(),
                    shapePolyline = shapePolyline,
                    routeType = request.routeType,
                    limit = limit,
                    historyBiasEnabled = historyBiasEnabled,
                    historyProfile = historyProfile,
                )
            )
        }.getOrElse { emptyList() }
    }

    private fun isHistoryBiasEnabled(): Boolean {
        return readBooleanConfig("OSM_ROUTING_HISTORY_BIAS_ENABLED", false)
    }

    private fun historyHalfLifeDays(): Int {
        val configured = readIntConfig("OSM_ROUTING_HISTORY_HALF_LIFE_DAYS", DEFAULT_HISTORY_HALF_LIFE_DAYS)
        return configured.coerceAtLeast(1)
    }

    private fun readBooleanConfig(key: String, fallback: Boolean): Boolean {
        val normalized = readStringConfig(key)?.trim()?.lowercase() ?: return fallback
        return when (normalized) {
            "1", "true", "yes", "y", "on" -> true
            "0", "false", "no", "n", "off" -> false
            else -> fallback
        }
    }

    private fun readIntConfig(key: String, fallback: Int): Int {
        return readStringConfig(key)?.trim()?.toIntOrNull() ?: fallback
    }

    private fun readStringConfig(key: String): String? {
        return RuntimeConfig.readConfigValue(key)
    }

    private fun mergeRouteRecommendations(
        primary: List<RouteRecommendation>,
        secondary: List<RouteRecommendation>,
        limit: Int,
    ): List<RouteRecommendation> {
        if (limit <= 0) {
            return emptyList()
        }
        val result = mutableListOf<RouteRecommendation>()
        val seen = mutableSetOf<String>()
        for (recommendation in primary + secondary) {
            if (result.size >= limit) break
            if (!seen.add(recommendation.routeId)) continue
            result += recommendation
        }
        return result
    }

    private fun buildRouteCandidates(activities: List<StravaActivity>): List<RouteCandidate> {
        return activities.asSequence()
            .filter { activity -> activity.distance > 0.0 }
            .mapNotNull { activity ->
                val rawDate = firstNonEmpty(activity.startDateLocal, activity.startDate) ?: return@mapNotNull null
                val parsedDate = parseDate(rawDate)
                val activityDate = extractSortableDay(rawDate) ?: rawDate
                val distanceKm = activity.distance / 1000.0
                val elevationGainM = max(0.0, activity.totalElevationGain)
                val durationSec = when {
                    activity.movingTime > 0 -> activity.movingTime
                    activity.elapsedTime > 0 -> activity.elapsedTime
                    else -> (distanceKm * 180.0).roundToInt()
                }

                val start = toCoordinates(activity.startLatlng)
                val end = extractEndCoordinates(activity)
                val isLoop = detectLoop(start, end, distanceKm)
                val preview = buildPreview(activity, start, end, isLoop)
                val shapeWithScore = classifyShape(preview, isLoop)
                RouteCandidate(
                    activity = activity,
                    date = parsedDate,
                    activityDate = activityDate,
                    distanceKm = distanceKm,
                    elevationGainM = elevationGainM,
                    durationSec = durationSec,
                    isLoop = isLoop,
                    start = start,
                    end = end,
                    startArea = startAreaLabel(start),
                    season = seasonFromDate(parsedDate),
                    previewLatLng = preview,
                    shape = shapeWithScore.first,
                    shapeScore = shapeWithScore.second,
                )
            }
            .sortedWith(compareByDescending<RouteCandidate> { candidate -> candidate.date ?: Instant.EPOCH }.thenByDescending { candidate -> candidate.activity.id })
            .toList()
    }

    private fun buildClosestLoopRecommendations(
        candidates: List<RouteCandidate>,
        distanceTarget: Double,
        elevationTarget: Double,
        durationTargetSec: Int,
        scoringProfile: RouteScoringProfile,
        startDirection: String?,
        preferredStart: Coordinates?,
        limit: Int,
    ): List<RouteRecommendation> {
        val loopCandidates = candidates.filter { candidate -> candidate.isLoop }.ifEmpty { candidates }

        return loopCandidates
            .map { candidate ->
                val score = closenessScore(
                    candidate = candidate,
                    distanceTarget = distanceTarget,
                    elevationTarget = elevationTarget,
                    durationTargetSec = durationTargetSec,
                    scoringProfile = scoringProfile,
                    startDirection = startDirection,
                    preferredStart = preferredStart,
                )
                candidate to score
            }
            .sortedWith(compareByDescending<Pair<RouteCandidate, Double>> { (_, score) -> score }.thenByDescending { (candidate, _) -> candidate.date ?: Instant.EPOCH })
            .take(limit)
            .map { (candidate, score) ->
                val reasons = mutableListOf(
                    "Distance delta: ${formatDistanceDelta(candidate.distanceKm - distanceTarget)}",
                    "Elevation delta: ${formatElevationDelta(candidate.elevationGainM - elevationTarget)}",
                )
                if (startDirection != null) {
                    reasons += "Direction: ${startDirectionLabel(startDirection)}"
                }
                if (preferredStart != null) {
                    val startDistanceKm = startDistanceKm(candidate, preferredStart)
                    if (startDistanceKm.isFinite()) {
                        reasons += "Start proximity: ${formatDistanceDelta(startDistanceKm)}"
                    }
                }
                toRouteRecommendation(
                    candidate = candidate,
                    variantType = RouteVariantType.CLOSE_MATCH,
                    score = score,
                    reasons = reasons,
                    experimental = false,
                )
            }
            .distinctBy { recommendation -> recommendation.activity.id }
    }

    private fun buildSmartVariants(
        candidates: List<RouteCandidate>,
        distanceTarget: Double,
        elevationTarget: Double,
        durationTargetSec: Int,
        scoringProfile: RouteScoringProfile,
        startDirection: String?,
        preferredStart: Coordinates?,
    ): List<RouteRecommendation> {
        val shorter = pickBestVariant(
            candidates = candidates,
            filter = { candidate -> candidate.distanceKm < distanceTarget * 0.95 },
            distanceTarget = distanceTarget,
            elevationTarget = elevationTarget,
            durationTargetSec = durationTargetSec,
            scoringProfile = scoringProfile,
            startDirection = startDirection,
            preferredStart = preferredStart,
        )

        val longer = pickBestVariant(
            candidates = candidates,
            filter = { candidate -> candidate.distanceKm > distanceTarget * 1.05 },
            distanceTarget = distanceTarget,
            elevationTarget = elevationTarget,
            durationTargetSec = durationTargetSec,
            scoringProfile = scoringProfile,
            startDirection = startDirection,
            preferredStart = preferredStart,
        )

        val hillier = pickBestVariant(
            candidates = candidates,
            filter = { candidate ->
                if (candidate.elevationGainM < max(elevationTarget + 120.0, elevationTarget * 1.15)) {
                    false
                } else {
                    val distanceDelta = abs(candidate.distanceKm - distanceTarget)
                    distanceDelta <= max(distanceTarget * 0.45, 15.0)
                }
            },
            distanceTarget = distanceTarget,
            elevationTarget = elevationTarget,
            durationTargetSec = durationTargetSec,
            scoringProfile = scoringProfile,
            startDirection = startDirection,
            preferredStart = preferredStart,
        )

        return buildList {
            shorter?.let { (candidate, score) ->
                add(
                    toRouteRecommendation(
                        candidate = candidate,
                        variantType = RouteVariantType.SHORTER,
                        score = score,
                        reasons = listOf(
                            "About ${formatDistanceDelta(distanceTarget - candidate.distanceKm)} shorter than your target",
                            "Estimated duration ${formatDuration(candidate.durationSec)}",
                        ),
                        experimental = false,
                    )
                )
            }
            longer?.let { (candidate, score) ->
                add(
                    toRouteRecommendation(
                        candidate = candidate,
                        variantType = RouteVariantType.LONGER,
                        score = score,
                        reasons = listOf(
                            "About ${formatDistanceDelta(candidate.distanceKm - distanceTarget)} longer than your target",
                            "Good endurance extension (+${formatDurationDelta(candidate.durationSec - durationTargetSec)})",
                        ),
                        experimental = false,
                    )
                )
            }
            hillier?.let { (candidate, score) ->
                add(
                    toRouteRecommendation(
                        candidate = candidate,
                        variantType = RouteVariantType.HILLIER,
                        score = score,
                        reasons = listOf(
                            "+${formatElevationDelta(candidate.elevationGainM - elevationTarget)} elevation vs target",
                            "Climbing-focused variant",
                        ),
                        experimental = false,
                    )
                )
            }
        }.distinctBy { recommendation -> recommendation.activity.id }
    }

    private fun buildSeasonalRecommendations(
        candidates: List<RouteCandidate>,
        seasonFilter: String?,
        distanceTarget: Double,
        elevationTarget: Double,
        durationTargetSec: Int,
        limit: Int,
    ): List<RouteRecommendation> {
        val season = seasonFilter ?: seasonFromDate(Instant.now())
        val filtered = filterBySeason(candidates, season)
        if (filtered.isEmpty()) {
            return emptyList()
        }

        return filtered
            .map { candidate ->
                val score = closenessScore(candidate, distanceTarget, elevationTarget, durationTargetSec)
                candidate to score
            }
            .sortedWith(compareByDescending<Pair<RouteCandidate, Double>> { (_, score) -> score }.thenByDescending { (candidate, _) -> candidate.date ?: Instant.EPOCH })
            .take(limit)
            .map { (candidate, score) ->
                toRouteRecommendation(
                    candidate = candidate,
                    variantType = RouteVariantType.SEASONAL,
                    score = score,
                    reasons = listOf(
                        "Seasonal fit: ${seasonLabel(season)}",
                        "Similar profile to your historical rides in this season",
                    ),
                    experimental = false,
                )
            }
            .distinctBy { recommendation -> recommendation.activity.id }
    }

    private fun buildRoadGraphRecommendations(
        candidates: List<RouteCandidate>,
        distanceTarget: Double,
        elevationTarget: Double,
        durationTargetSec: Int,
        scoringProfile: RouteScoringProfile,
        startDirection: String?,
        preferredStart: Coordinates?,
        limit: Int,
    ): List<RouteRecommendation> {
        if (candidates.isEmpty()) {
            return emptyList()
        }
        val sourceCandidates = candidates
            .filter { candidate -> candidate.previewLatLng.size >= 8 }
            .let { filtered ->
                if (preferredStart == null) {
                    filtered
                } else {
                    filtered.sortedBy { candidate -> startDistanceKm(candidate, preferredStart) }
                }
            }
            .take(120)
        if (sourceCandidates.isEmpty()) {
            return emptyList()
        }
        val graph = buildRoadGraph(sourceCandidates)
        if (graph.nodes.isEmpty()) {
            return emptyList()
        }

        val elevationPerKm = estimateElevationPerKm(sourceCandidates)
        val durationPerKm = estimateDurationPerKm(sourceCandidates)
        val scored = mutableListOf<Pair<RouteRecommendation, Double>>()
        val seenRouteIds = mutableSetOf<String>()
        val seenGeometries = mutableSetOf<String>()

        for (leftIndex in 0 until sourceCandidates.size) {
            val left = sourceCandidates[leftIndex]
            for (rightIndex in leftIndex + 1 until sourceCandidates.size) {
                val right = sourceCandidates[rightIndex]
                val built = buildRoadGraphRecommendationFromPair(
                    graph = graph,
                    left = left,
                    right = right,
                    distanceTarget = distanceTarget,
                    elevationTarget = elevationTarget,
                    durationTargetSec = durationTargetSec,
                    scoringProfile = scoringProfile,
                    startDirection = startDirection,
                    preferredStart = preferredStart,
                    elevationPerKm = elevationPerKm,
                    durationPerKm = durationPerKm,
                ) ?: continue

                if (!seenRouteIds.add(built.first.routeId)) {
                    continue
                }
                val geometryKey = routeGeometrySignature(built.first.previewLatLng)
                if (geometryKey.isBlank() || !seenGeometries.add(geometryKey)) {
                    continue
                }
                scored += built
                if (scored.size >= MAX_ROUTE_LIMIT * 4) {
                    break
                }
            }
            if (scored.size >= MAX_ROUTE_LIMIT * 4) {
                break
            }
        }

        return scored
            .sortedWith(
                compareByDescending<Pair<RouteRecommendation, Double>> { (_, score) -> score }
                    .thenBy { (recommendation, _) -> recommendation.routeId }
            )
            .take(min(limit, 6))
            .map { (recommendation, _) -> recommendation }
    }

    private fun buildRoadGraphRecommendationFromPair(
        graph: RoadGraph,
        left: RouteCandidate,
        right: RouteCandidate,
        distanceTarget: Double,
        elevationTarget: Double,
        durationTargetSec: Int,
        scoringProfile: RouteScoringProfile,
        startDirection: String?,
        preferredStart: Coordinates?,
        elevationPerKm: Double,
        durationPerKm: Double,
    ): Pair<RouteRecommendation, Double>? {
        if (left.previewLatLng.size < 4 || right.previewLatLng.size < 4) {
            return null
        }
        val leftStart = left.previewLatLng.firstOrNull() ?: return null
        val leftEnd = left.previewLatLng.lastOrNull() ?: return null
        val rightStart = right.previewLatLng.firstOrNull() ?: return null
        val rightEnd = right.previewLatLng.lastOrNull() ?: return null

        val connectorA = shortestGraphPath(graph, leftEnd, rightStart) ?: return null
        val connectorB = shortestGraphPath(graph, rightEnd, leftStart) ?: return null

        var merged = mergePreview(left.previewLatLng, connectorA)
        merged = mergePreview(merged, right.previewLatLng)
        merged = mergePreview(merged, connectorB)
        if (merged.size < 6) {
            return null
        }
        if (preferredStart != null) {
            merged = anchorLoopToPreferredStart(merged, preferredStart) ?: return null
        }
        if (merged.size < 6) {
            return null
        }
        if (!isLoopWithoutHeavySegmentReuse(merged)) {
            return null
        }
        val preview = sampleLatLng(merged, PREVIEW_POINT_MAX_SIZE)

        val distanceKm = pathDistanceMeters(merged) / 1000.0
        if (distanceKm < 3.0) {
            return null
        }
        val estimatedElevation = max(0.0, distanceKm * elevationPerKm)
        if (!isTargetMatchAcceptable(distanceKm, estimatedElevation, distanceTarget, elevationTarget)) {
            return null
        }
        val generatedName = "Generated loop near ${firstNonEmpty(left.startArea, right.startArea, "your start point")}"
        val estimatedDuration = (distanceKm * durationPerKm).roundToInt()
        val scoreCandidate = RouteCandidate(
            activity = left.activity.copy(id = 0L, name = generatedName),
            date = left.date,
            activityDate = left.activityDate,
            distanceKm = distanceKm,
            elevationGainM = estimatedElevation,
            durationSec = estimatedDuration,
            isLoop = true,
            start = toCoordinates(merged.first()),
            end = toCoordinates(merged.last()),
            startArea = left.startArea,
            season = left.season,
            previewLatLng = preview,
            shape = "LOOP",
            shapeScore = 0.85,
        )

        val score = closenessScore(
            candidate = scoreCandidate,
            distanceTarget = distanceTarget,
            elevationTarget = elevationTarget,
            durationTargetSec = durationTargetSec,
            scoringProfile = scoringProfile,
            startDirection = startDirection,
            preferredStart = preferredStart,
        )
        if (score < 20.0) {
            return null
        }

        val recommendation = toRouteRecommendation(
            candidate = scoreCandidate,
            variantType = RouteVariantType.ROAD_GRAPH,
            score = score,
            reasons = listOf(
                "Generated on cache road-graph (beta)",
                "Built from local road-network connectivity",
                "Estimated profile: ${formatDistanceDelta(distanceKm)} / ${formatElevationDelta(estimatedElevation)}",
            ),
            experimental = true,
        ).copy(
            routeId = "road-graph-${minLong(left.activity.id, right.activity.id)}-${maxLong(left.activity.id, right.activity.id)}"
        )
        return recommendation to score
    }

    private fun buildRoadGraph(candidates: List<RouteCandidate>): RoadGraph {
        val nodes = mutableMapOf<String, RoadGraphNode>()
        val edges = mutableMapOf<String, MutableList<RoadGraphEdge>>()

        candidates.forEach { candidate ->
            val preview = candidate.previewLatLng
            if (preview.size < 2) {
                return@forEach
            }
            for (index in 0 until preview.size - 1) {
                val left = preview[index]
                val right = preview[index + 1]
                val leftNode = quantizedRoadNode(left) ?: continue
                val rightNode = quantizedRoadNode(right) ?: continue
                if (leftNode.id == rightNode.id) {
                    continue
                }
                nodes[leftNode.id] = leftNode
                nodes[rightNode.id] = rightNode
                val distance = distanceBetween(
                    Coordinates(leftNode.lat, leftNode.lng),
                    Coordinates(rightNode.lat, rightNode.lng)
                )
                if (distance <= 0.0 || distance.isInfinite() || distance.isNaN()) {
                    continue
                }
                edges.computeIfAbsent(leftNode.id) { mutableListOf() }.add(RoadGraphEdge(rightNode.id, distance))
                edges.computeIfAbsent(rightNode.id) { mutableListOf() }.add(RoadGraphEdge(leftNode.id, distance))
            }
        }
        return RoadGraph(nodes = nodes, edges = edges)
    }

    private fun quantizedRoadNode(point: List<Double>): RoadGraphNode? {
        if (point.size < 2) {
            return null
        }
        val lat = point[0]
        val lng = point[1]
        if (lat !in -90.0..90.0 || lng !in -180.0..180.0) {
            return null
        }
        val roundedLat = round(lat * 10000.0) / 10000.0
        val roundedLng = round(lng * 10000.0) / 10000.0
        val id = "%.4f,%.4f".format(roundedLat, roundedLng)
        return RoadGraphNode(id = id, lat = roundedLat, lng = roundedLng)
    }

    private fun shortestGraphPath(graph: RoadGraph, from: List<Double>, to: List<Double>): List<List<Double>>? {
        val startNode = quantizedRoadNode(from) ?: return null
        val endNode = quantizedRoadNode(to) ?: return null
        if (!graph.nodes.containsKey(startNode.id) || !graph.nodes.containsKey(endNode.id)) {
            return null
        }
        if (startNode.id == endNode.id) {
            val node = graph.nodes[startNode.id] ?: return null
            return listOf(listOf(node.lat, node.lng))
        }

        val distances = mutableMapOf(startNode.id to 0.0)
        val parents = mutableMapOf<String, String>()
        val visited = mutableSetOf<String>()
        val queue = PriorityQueue(compareBy<RoadPathState> { state -> state.distance })
        queue.add(RoadPathState(nodeId = startNode.id, distance = 0.0))

        while (queue.isNotEmpty()) {
            val current = queue.poll()
            if (!visited.add(current.nodeId)) {
                continue
            }
            if (current.nodeId == endNode.id) {
                break
            }
            graph.edges[current.nodeId].orEmpty().forEach { edge ->
                val nextDistance = current.distance + edge.distance
                val knownDistance = distances[edge.to]
                if (knownDistance == null || nextDistance < knownDistance) {
                    distances[edge.to] = nextDistance
                    parents[edge.to] = current.nodeId
                    queue.add(RoadPathState(nodeId = edge.to, distance = nextDistance))
                }
            }
        }
        if (!distances.containsKey(endNode.id)) {
            return null
        }

        val ids = mutableListOf(endNode.id)
        var cursor = endNode.id
        while (cursor != startNode.id) {
            val parent = parents[cursor] ?: return null
            ids.add(parent)
            cursor = parent
        }
        ids.reverse()
        return ids.mapNotNull { id ->
            graph.nodes[id]?.let { node -> listOf(node.lat, node.lng) }
        }
    }

    private fun pathDistanceMeters(points: List<List<Double>>): Double {
        if (points.size < 2) {
            return 0.0
        }
        var total = 0.0
        for (index in 0 until points.size - 1) {
            val from = toCoordinates(points[index]) ?: continue
            val to = toCoordinates(points[index + 1]) ?: continue
            total += distanceBetween(from, to)
        }
        return total
    }

    private fun estimateElevationPerKm(candidates: List<RouteCandidate>): Double {
        val values = candidates
            .filter { candidate -> candidate.distanceKm > 0.0 }
            .map { candidate -> candidate.elevationGainM / candidate.distanceKm }
        return median(values, 12.0)
    }

    private fun estimateDurationPerKm(candidates: List<RouteCandidate>): Double {
        val values = candidates
            .filter { candidate -> candidate.distanceKm > 0.0 && candidate.durationSec > 0 }
            .map { candidate -> candidate.durationSec.toDouble() / candidate.distanceKm }
        return median(values, 190.0)
    }

    private fun buildShapeMatchRecommendations(
        candidates: List<RouteCandidate>,
        shapeFilter: String?,
        distanceTarget: Double,
        elevationTarget: Double,
        durationTargetSec: Int,
        limit: Int,
    ): List<RouteRecommendation> {
        if (shapeFilter.isNullOrBlank()) {
            return emptyList()
        }

        return candidates
            .filter { candidate -> shapeMatches(candidate, shapeFilter) }
            .map { candidate ->
                val closeness = closenessScore(candidate, distanceTarget, elevationTarget, durationTargetSec)
                val shapeScore = candidate.shapeScore * 100.0
                val score = shapeScore * 0.65 + closeness * 0.35
                candidate to score
            }
            .sortedWith(compareByDescending<Pair<RouteCandidate, Double>> { (_, score) -> score }.thenByDescending { (candidate, _) -> candidate.date ?: Instant.EPOCH })
            .take(limit)
            .map { (candidate, score) ->
                toRouteRecommendation(
                    candidate = candidate,
                    variantType = RouteVariantType.SHAPE_MATCH,
                    score = score,
                    reasons = listOf(
                        "Shape match: ${shapeFilter.lowercase().replace('_', ' ')}",
                        "Route geometry confidence ${(candidate.shapeScore * 100.0).roundToInt()}%",
                    ),
                    experimental = false,
                )
            }
            .distinctBy { recommendation -> recommendation.activity.id }
    }

    private fun buildShapeRemixRecommendations(
        candidates: List<RouteCandidate>,
        distanceTarget: Double,
        elevationTarget: Double,
        durationTargetSec: Int,
        limit: Int,
    ): List<ShapeRemixRecommendation> {
        val eligible = candidates
            .filter { candidate -> candidate.start != null && candidate.end != null && candidate.previewLatLng.size >= 2 }
            .sortedByDescending { candidate -> candidate.date ?: Instant.EPOCH }
            .take(140)

        if (eligible.size < 2) {
            return emptyList()
        }

        val remixCandidates = mutableListOf<Pair<ShapeRemixRecommendation, Double>>()
        val seen = mutableSetOf<String>()

        for (i in eligible.indices) {
            for (j in i + 1 until eligible.size) {
                val left = eligible[i]
                val right = eligible[j]
                if (left.activity.id == right.activity.id) {
                    continue
                }

                val remix = buildRemixPair(left, right, distanceTarget, elevationTarget, durationTargetSec) ?: continue
                if (remix.second < 40.0) {
                    continue
                }
                if (!seen.add(remix.first.id)) {
                    continue
                }
                remixCandidates += remix
            }
        }

        return remixCandidates
            .sortedByDescending { (_, score) -> score }
            .take(limit)
            .map { (remix, _) -> remix }
    }

    private fun buildRemixPair(
        left: RouteCandidate,
        right: RouteCandidate,
        distanceTarget: Double,
        elevationTarget: Double,
        durationTargetSec: Int,
    ): Pair<ShapeRemixRecommendation, Double>? {
        val connectorA = distanceBetween(left.end, right.start)
        val connectorB = distanceBetween(right.end, left.start)
        val totalConnector = connectorA + connectorB
        if (totalConnector > 7000.0) {
            return null
        }

        val remixDistance = left.distanceKm + right.distanceKm + (totalConnector / 1000.0) * 0.25
        val remixElevation = left.elevationGainM + right.elevationGainM
        val remixDurationSec = left.durationSec + right.durationSec + (totalConnector / 6.0).roundToInt()

        val mergedPreview = sampleLatLng((left.previewLatLng + right.previewLatLng), PREVIEW_POINT_MAX_SIZE)
        val shapeWithScore = classifyShape(mergedPreview, true)
        val remixCandidate = RouteCandidate(
            activity = left.activity,
            date = left.date,
            activityDate = left.activityDate,
            distanceKm = remixDistance,
            elevationGainM = remixElevation,
            durationSec = remixDurationSec,
            isLoop = true,
            start = left.start,
            end = left.end,
            startArea = left.startArea,
            season = left.season,
            previewLatLng = mergedPreview,
            shape = shapeWithScore.first,
            shapeScore = shapeWithScore.second,
        )

        val closeness = closenessScore(remixCandidate, distanceTarget, elevationTarget, durationTargetSec)
        val score = closeness * 0.7 + shapeWithScore.second * 100.0 * 0.3

        val sortedIds = listOf(left.activity.id, right.activity.id).sorted()
        val remixId = "remix-${sortedIds[0]}-${sortedIds[1]}"

        val remix = ShapeRemixRecommendation(
            id = remixId,
            shape = shapeWithScore.first,
            distanceKm = round(remixDistance * 100) / 100,
            elevationGainM = round(remixElevation),
            durationSec = remixDurationSec,
            matchScore = round(score * 10) / 10,
            reasons = listOf(
                "Synthetic loop from ${left.activity.name} + ${right.activity.name}",
                "Connector cost: ${"%.1f".format(totalConnector / 1000.0)} km",
            ),
            components = listOf(toActivityShort(left.activity), toActivityShort(right.activity)),
            previewLatLng = mergedPreview,
            experimental = true,
        )

        return remix to score
    }

    private fun pickBestVariant(
        candidates: List<RouteCandidate>,
        filter: (RouteCandidate) -> Boolean,
        distanceTarget: Double,
        elevationTarget: Double,
        durationTargetSec: Int,
        scoringProfile: RouteScoringProfile,
        startDirection: String?,
        preferredStart: Coordinates?,
    ): Pair<RouteCandidate, Double>? {
        return candidates
            .asSequence()
            .filter { candidate -> filter(candidate) }
            .map { candidate ->
                candidate to closenessScore(
                    candidate = candidate,
                    distanceTarget = distanceTarget,
                    elevationTarget = elevationTarget,
                    durationTargetSec = durationTargetSec,
                    scoringProfile = scoringProfile,
                    startDirection = startDirection,
                    preferredStart = preferredStart,
                )
            }
            .maxByOrNull { (_, score) -> score }
    }

    private fun toRouteRecommendation(
        candidate: RouteCandidate,
        variantType: RouteVariantType,
        score: Double,
        reasons: List<String>,
        experimental: Boolean,
    ): RouteRecommendation {
        val shape = candidate.shape.ifBlank { "UNKNOWN" }
        return RouteRecommendation(
            routeId = routeRecommendationId(candidate, variantType),
            activity = toActivityShort(candidate.activity),
            activityDate = candidate.activityDate,
            distanceKm = round(candidate.distanceKm * 100) / 100,
            elevationGainM = round(candidate.elevationGainM),
            durationSec = candidate.durationSec,
            isLoop = candidate.isLoop,
            start = candidate.start,
            end = candidate.end,
            startArea = candidate.startArea,
            season = candidate.season,
            variantType = variantType,
            matchScore = round(score * 10) / 10,
            reasons = reasons,
            previewLatLng = candidate.previewLatLng,
            shape = shape,
            shapeScore = candidate.shapeScore * 100.0,
            experimental = experimental,
        )
    }

    private fun buildFallbackRoadGraphRecommendation(
        candidates: List<RouteCandidate>,
        distanceTarget: Double,
        elevationTarget: Double,
        durationTargetSec: Int,
        scoringProfile: RouteScoringProfile,
        startDirection: String?,
        preferredStart: Coordinates?,
        elevationPerKm: Double,
        durationPerKm: Double,
    ): Pair<RouteRecommendation, Double>? {
        if (candidates.isEmpty()) {
            return null
        }

        val rankedCandidates = candidates
            .filter { candidate -> candidate.previewLatLng.size >= 2 }
            .filter { candidate -> preferredStart == null || startDistanceKm(candidate, preferredStart) <= 5.0 }
            .map { candidate ->
                val score = closenessScore(
                    candidate = candidate,
                    distanceTarget = distanceTarget,
                    elevationTarget = elevationTarget,
                    durationTargetSec = durationTargetSec,
                    scoringProfile = scoringProfile,
                    startDirection = startDirection,
                    preferredStart = preferredStart,
                )
                candidate to score
            }
            .sortedWith(compareByDescending<Pair<RouteCandidate, Double>> { (_, score) -> score }.thenByDescending { (candidate, _) -> candidate.date ?: Instant.EPOCH })
            .take(12)

        for ((candidate, _) in rankedCandidates) {
            var preview = buildOutAndBackPreview(candidate.previewLatLng, distanceTarget)
            if (preview.size < 4) {
                continue
            }
            if (preferredStart != null) {
                preview = anchorLoopToPreferredStart(preview, preferredStart) ?: continue
            }
            if (preview.size < 4) {
                continue
            }

            val distanceKm = pathDistanceMeters(preview) / 1000.0
            if (distanceKm < 2.0) {
                continue
            }

            val estimatedElevation = max(0.0, distanceKm * elevationPerKm)
            if (!isTargetMatchAcceptable(distanceKm, estimatedElevation, distanceTarget, elevationTarget)) {
                continue
            }
            val estimatedDuration = (distanceKm * durationPerKm).roundToInt()
            val start = toCoordinates(preview.first())
            val scoreCandidate = RouteCandidate(
                activity = candidate.activity.copy(id = 0L, name = "Generated out-and-back near ${firstNonEmpty(candidate.startArea, "your start point")}"),
                date = candidate.date,
                activityDate = candidate.activityDate,
                distanceKm = distanceKm,
                elevationGainM = estimatedElevation,
                durationSec = estimatedDuration,
                isLoop = true,
                start = start,
                end = start,
                startArea = candidate.startArea,
                season = candidate.season,
                previewLatLng = preview,
                shape = "LOOP",
                shapeScore = 0.78,
            )

            val score = closenessScore(
                candidate = scoreCandidate,
                distanceTarget = distanceTarget,
                elevationTarget = elevationTarget,
                durationTargetSec = durationTargetSec,
                scoringProfile = scoringProfile,
                startDirection = startDirection,
                preferredStart = preferredStart,
            )
            val recommendation = toRouteRecommendation(
                candidate = scoreCandidate,
                variantType = RouteVariantType.ROAD_GRAPH,
                score = score,
                reasons = listOf(
                    "Generated fallback route (out-and-back loop)",
                    "Built from local cached roads",
                    "Estimated profile: ${formatDistanceDelta(distanceKm)} / ${formatElevationDelta(estimatedElevation)}",
                ),
                experimental = true,
            ).copy(routeId = "road-graph-fallback-${candidate.activity.id}-${(distanceTarget * 10.0).roundToInt()}")

            return recommendation to score
        }

        return null
    }

    private fun routeRecommendationId(candidate: RouteCandidate, variantType: RouteVariantType): String {
        val type = variantType.name.lowercase()
        return when {
            candidate.activity.id > 0L -> "route-${candidate.activity.id}-$type"
            candidate.activityDate.isNotBlank() -> "route-${candidate.activityDate}-$type"
            else -> "route-${System.currentTimeMillis()}-$type"
        }
    }

    private fun toActivityShort(activity: StravaActivity): ActivityShort {
        val activityType = when {
            activity.commute -> ActivityType.Commute
            else -> runCatching { ActivityType.valueOf(activity.sportType) }
                .recoverCatching { ActivityType.valueOf(activity.type) }
                .getOrDefault(ActivityType.Ride)
        }
        return ActivityShort(
            id = activity.id,
            name = activity.name,
            type = activityType,
        )
    }

    private fun extractEndCoordinates(activity: StravaActivity): Coordinates? {
        val latLng = activity.stream?.latlng?.data
        if (latLng.isNullOrEmpty()) {
            return null
        }
        return toCoordinates(latLng.lastOrNull())
    }

    private fun toCoordinates(values: List<Double>?): Coordinates? {
        if (values == null || values.size < 2) {
            return null
        }
        val lat = values[0]
        val lng = values[1]
        return if (lat in -90.0..90.0 && lng in -180.0..180.0) {
            Coordinates(lat = lat, lng = lng)
        } else {
            null
        }
    }

    private fun detectLoop(start: Coordinates?, end: Coordinates?, distanceKm: Double): Boolean {
        if (start == null || end == null) {
            return false
        }
        val distance = distanceBetween(start, end)
        val threshold = max(250.0, distanceKm * 1000.0 * 0.08)
        return distance <= threshold
    }

    private fun buildPreview(
        activity: StravaActivity,
        start: Coordinates?,
        end: Coordinates?,
        isLoop: Boolean,
    ): List<List<Double>> {
        val stream = activity.stream?.latlng?.data
        if (!stream.isNullOrEmpty()) {
            return sampleLatLng(stream, PREVIEW_POINT_MAX_SIZE)
        }

        return buildList {
            start?.let { coordinates -> add(listOf(coordinates.lat, coordinates.lng)) }
            end?.let { coordinates ->
                if (isEmpty() || first()[0] != coordinates.lat || first()[1] != coordinates.lng) {
                    add(listOf(coordinates.lat, coordinates.lng))
                }
            }
            if (isLoop && start != null && end != null) {
                add(listOf(start.lat, start.lng))
            }
        }
    }

    private fun sampleLatLng(raw: List<List<Double>>, maxPoints: Int): List<List<Double>> {
        val validPoints = raw
            .filter { point -> point.size >= 2 && point[0] in -90.0..90.0 && point[1] in -180.0..180.0 }
            .map { point -> listOf(point[0], point[1]) }
        if (validPoints.size <= maxPoints) {
            return validPoints
        }
        if (maxPoints <= 1) {
            return validPoints.take(1)
        }

        val sampled = mutableListOf<List<Double>>()
        val step = (validPoints.size - 1).toDouble() / (maxPoints - 1).toDouble()
        var lastIndex = -1
        for (position in 0 until maxPoints) {
            var index = round(position * step).toInt()
            if (index >= validPoints.size) {
                index = validPoints.size - 1
            }
            if (index == lastIndex) {
                continue
            }
            sampled += validPoints[index]
            lastIndex = index
        }
        return sampled
    }

    private fun classifyShape(preview: List<List<Double>>, isLoop: Boolean): Pair<String, Double> {
        if (preview.size < 2) {
            return if (isLoop) "LOOP" to 0.55 else "POINT_TO_POINT" to 0.35
        }
        if (looksLikeFigureEight(preview)) {
            return "FIGURE_EIGHT" to 0.84
        }
        if (looksLikeOutAndBack(preview)) {
            return "OUT_AND_BACK" to 0.82
        }
        if (isLoop) {
            return "LOOP" to 0.78
        }

        val start = Coordinates(preview.first()[0], preview.first()[1])
        val end = Coordinates(preview.last()[0], preview.last()[1])
        val latDelta = end.lat - start.lat
        val lngDelta = end.lng - start.lng
        if (abs(latDelta) > abs(lngDelta) * 1.35) {
            return if (latDelta >= 0) "NORTHBOUND" to 0.68 else "SOUTHBOUND" to 0.68
        }
        if (abs(lngDelta) > abs(latDelta) * 1.35) {
            return if (lngDelta >= 0) "EASTBOUND" to 0.68 else "WESTBOUND" to 0.68
        }
        return "POINT_TO_POINT" to 0.62
    }

    private fun looksLikeOutAndBack(preview: List<List<Double>>): Boolean {
        if (preview.size < 6) {
            return false
        }
        val start = Coordinates(preview.first()[0], preview.first()[1])
        val end = Coordinates(preview.last()[0], preview.last()[1])
        if (distanceBetween(start, end) > 320.0) {
            return false
        }

        var maxDistance = 0.0
        var maxIndex = 0
        preview.forEachIndexed { index, point ->
            val currentDistance = distanceBetween(start, Coordinates(point[0], point[1]))
            if (currentDistance > maxDistance) {
                maxDistance = currentDistance
                maxIndex = index
            }
        }
        if (maxDistance < 900.0) {
            return false
        }
        val progress = maxIndex.toDouble() / (preview.size - 1).toDouble()
        return progress in 0.25..0.75
    }

    private fun looksLikeFigureEight(preview: List<List<Double>>): Boolean {
        if (preview.size < 10) {
            return false
        }
        val start = Coordinates(preview.first()[0], preview.first()[1])
        val end = Coordinates(preview.last()[0], preview.last()[1])
        if (distanceBetween(start, end) > 360.0) {
            return false
        }
        val mid = preview[preview.size / 2]
        val center = centroid(preview)
        return distanceBetween(Coordinates(mid[0], mid[1]), Coordinates(center[0], center[1])) <= 180.0
    }

    private fun centroid(preview: List<List<Double>>): List<Double> {
        val lat = preview.sumOf { point -> point[0] } / preview.size.toDouble()
        val lng = preview.sumOf { point -> point[1] } / preview.size.toDouble()
        return listOf(lat, lng)
    }

    private fun shapeMatches(candidate: RouteCandidate, shapeFilter: String): Boolean {
        return if (shapeFilter == "LOOP") {
            candidate.isLoop || candidate.shape == "LOOP"
        } else {
            candidate.shape == shapeFilter
        }
    }

    private fun closenessScore(
        candidate: RouteCandidate,
        distanceTarget: Double,
        elevationTarget: Double,
        durationTargetSec: Int,
    ): Double {
        return closenessScore(
            candidate = candidate,
            distanceTarget = distanceTarget,
            elevationTarget = elevationTarget,
            durationTargetSec = durationTargetSec,
            scoringProfile = buildRouteScoringProfile(null, null, hasPreferredStart = false),
            startDirection = null,
            preferredStart = null,
        )
    }

    private fun closenessScore(
        candidate: RouteCandidate,
        distanceTarget: Double,
        elevationTarget: Double,
        durationTargetSec: Int,
        scoringProfile: RouteScoringProfile,
        startDirection: String?,
        preferredStart: Coordinates?,
    ): Double {
        val profile = normalizeScoringProfile(scoringProfile)
        val distanceComponent = abs(candidate.distanceKm - distanceTarget) / max(distanceTarget, 1.0)
        val elevationComponent = abs(candidate.elevationGainM - elevationTarget) / max(elevationTarget, 200.0)
        val durationComponent = abs(candidate.durationSec.toDouble() - durationTargetSec.toDouble()) / max(durationTargetSec.toDouble(), 1800.0)
        val directionComponent = directionPenaltyComponent(candidate, startDirection)
        val startPointComponent = startPointPenaltyComponent(candidate, preferredStart)
        val weighted = distanceComponent * profile.distanceWeight +
            elevationComponent * profile.elevationWeight +
            durationComponent * profile.durationWeight +
            directionComponent * profile.directionWeight +
            startPointComponent * profile.startPointWeight
        return max(0.0, 100.0 - weighted * 100.0)
    }

    private fun buildRouteScoringProfile(routeType: String?, startDirection: String?, hasPreferredStart: Boolean): RouteScoringProfile {
        val normalizedType = routeType?.trim()?.uppercase()
        val (baseDistance, baseElevation, baseDuration) = when (normalizedType) {
            "MTB" -> Triple(0.44, 0.39, 0.17)
            "GRAVEL" -> Triple(0.48, 0.34, 0.18)
            "RUN" -> Triple(0.45, 0.22, 0.33)
            "TRAIL" -> Triple(0.36, 0.40, 0.24)
            "HIKE" -> Triple(0.30, 0.45, 0.25)
            else -> Triple(0.52, 0.30, 0.18)
        }

        val directionWeight = if (startDirection.isNullOrBlank()) {
            0.0
        } else {
            when (normalizedType) {
                "MTB" -> 0.10
                "GRAVEL" -> 0.09
                "RUN" -> 0.10
                "TRAIL" -> 0.12
                "HIKE" -> 0.12
                else -> 0.08
            }
        }

        val startPointWeight = if (hasPreferredStart) {
            when (normalizedType) {
                "RUN", "TRAIL", "HIKE" -> 0.22
                "MTB", "GRAVEL" -> 0.16
                else -> 0.14
            }
        } else {
            0.0
        }

        val core = max(0.05, 1.0 - directionWeight - startPointWeight)
        return normalizeScoringProfile(
            RouteScoringProfile(
                distanceWeight = baseDistance * core,
                elevationWeight = baseElevation * core,
                durationWeight = baseDuration * core,
                directionWeight = directionWeight,
                startPointWeight = startPointWeight,
            )
        )
    }

    private fun normalizeScoringProfile(profile: RouteScoringProfile): RouteScoringProfile {
        val total = profile.distanceWeight + profile.elevationWeight + profile.durationWeight + profile.directionWeight + profile.startPointWeight
        if (total <= 0.0) {
            return RouteScoringProfile(
                distanceWeight = 0.5,
                elevationWeight = 0.3,
                durationWeight = 0.2,
                directionWeight = 0.0,
                startPointWeight = 0.0,
            )
        }
        return RouteScoringProfile(
            distanceWeight = profile.distanceWeight / total,
            elevationWeight = profile.elevationWeight / total,
            durationWeight = profile.durationWeight / total,
            directionWeight = profile.directionWeight / total,
            startPointWeight = profile.startPointWeight / total,
        )
    }

    private fun normalizeRouteType(value: String?): String? {
        return when (value?.trim()?.uppercase()) {
            "RIDE", "MTB", "GRAVEL", "RUN", "TRAIL", "HIKE" -> value.trim().uppercase()
            else -> null
        }
    }

    private fun normalizeStartDirection(value: String?): String? {
        return when (value?.trim()?.uppercase()) {
            "N", "S", "E", "W" -> value.trim().uppercase()
            else -> null
        }
    }

    private fun startDirectionLabel(value: String): String {
        return when (value) {
            "N" -> "North"
            "S" -> "South"
            "E" -> "East"
            "W" -> "West"
            else -> "Any"
        }
    }

    private fun directionPenaltyComponent(candidate: RouteCandidate, startDirection: String?): Double {
        if (startDirection.isNullOrBlank()) {
            return 0.0
        }
        val initialBearing = initialBearingDegrees(candidate) ?: return 1.0
        val targetBearing = when (startDirection) {
            "N" -> 0.0
            "E" -> 90.0
            "S" -> 180.0
            "W" -> 270.0
            else -> return 1.0
        }
        val rawDiff = abs(initialBearing - targetBearing)
        val normalizedDiff = if (rawDiff > 180.0) 360.0 - rawDiff else rawDiff
        return normalizedDiff / 180.0
    }

    private fun normalizePreferredStartPoint(value: Coordinates?): Coordinates? {
        if (value == null) {
            return null
        }
        return if (value.lat in -90.0..90.0 && value.lng in -180.0..180.0) {
            Coordinates(lat = value.lat, lng = value.lng)
        } else {
            null
        }
    }

    private fun startPointPenaltyComponent(candidate: RouteCandidate, preferredStart: Coordinates?): Double {
        if (preferredStart == null) {
            return 0.0
        }
        val distanceKm = startDistanceKm(candidate, preferredStart)
        if (!distanceKm.isFinite()) {
            return 1.0
        }
        return min(1.0, max(0.0, distanceKm / 18.0))
    }

    private fun startDistanceKm(candidate: RouteCandidate, preferredStart: Coordinates?): Double {
        if (preferredStart == null || candidate.start == null) {
            return Double.POSITIVE_INFINITY
        }
        val distanceMeters = distanceBetween(candidate.start, preferredStart)
        if (!distanceMeters.isFinite() || distanceMeters == Double.MAX_VALUE) {
            return Double.POSITIVE_INFINITY
        }
        return distanceMeters / 1000.0
    }

    private fun initialBearingDegrees(candidate: RouteCandidate): Double? {
        if (candidate.previewLatLng.size < 2) {
            return null
        }
        val start = candidate.previewLatLng.firstOrNull()?.takeIf { point -> point.size >= 2 } ?: return null
        val startLat = start[0]
        val startLng = start[1]
        for (index in 1 until candidate.previewLatLng.size) {
            val next = candidate.previewLatLng[index]
            if (next.size < 2) {
                continue
            }
            val nextLat = next[0]
            val nextLng = next[1]
            if (distanceBetween(Coordinates(startLat, startLng), Coordinates(nextLat, nextLng)) < 35.0) {
                continue
            }
            return bearingDegrees(startLat, startLng, nextLat, nextLng)
        }
        val fallback = candidate.previewLatLng.lastOrNull()?.takeIf { point -> point.size >= 2 } ?: return null
        return bearingDegrees(startLat, startLng, fallback[0], fallback[1])
    }

    private fun bearingDegrees(lat1: Double, lng1: Double, lat2: Double, lng2: Double): Double {
        val lat1r = lat1 * PI / 180.0
        val lat2r = lat2 * PI / 180.0
        val deltaLng = (lng2 - lng1) * PI / 180.0
        val y = sin(deltaLng) * cos(lat2r)
        val x = cos(lat1r) * sin(lat2r) - sin(lat1r) * cos(lat2r) * cos(deltaLng)
        var bearing = atan2(y, x) * 180.0 / PI
        if (bearing < 0.0) {
            bearing += 360.0
        }
        return bearing
    }

    private fun normalizeLimit(limit: Int): Int {
        return when {
            limit <= 0 -> DEFAULT_ROUTE_LIMIT
            limit > MAX_ROUTE_LIMIT -> MAX_ROUTE_LIMIT
            else -> limit
        }
    }

    private fun normalizeSeason(value: String?): String? {
        val normalized = value?.trim()?.uppercase() ?: return null
        return when (normalized) {
            "WINTER", "SPRING", "SUMMER" -> normalized
            "AUTUMN", "FALL" -> "AUTUMN"
            else -> null
        }
    }

    private fun seasonFromDate(value: Instant?): String {
        val month = value?.atZone(ZoneOffset.UTC)?.monthValue ?: return ""
        return when (month) {
            12, 1, 2 -> "WINTER"
            3, 4, 5 -> "SPRING"
            6, 7, 8 -> "SUMMER"
            else -> "AUTUMN"
        }
    }

    private fun seasonLabel(value: String): String {
        return when (value) {
            "WINTER" -> "Winter"
            "SPRING" -> "Spring"
            "SUMMER" -> "Summer"
            "AUTUMN" -> "Autumn"
            else -> "All seasons"
        }
    }

    private fun normalizeShape(value: String?): String? {
        val normalized = value?.trim()?.uppercase()?.replace("-", "_")?.replace(" ", "_") ?: return null
        return when (normalized) {
            "LOOP", "OUT_AND_BACK", "POINT_TO_POINT", "FIGURE_EIGHT",
            "NORTHBOUND", "SOUTHBOUND", "EASTBOUND", "WESTBOUND" -> normalized

            else -> null
        }
    }

    private fun filterBySeason(candidates: List<RouteCandidate>, season: String?): List<RouteCandidate> {
        if (season.isNullOrBlank()) {
            return candidates
        }
        return candidates.filter { candidate -> candidate.season == season }
    }

    private fun formatDistanceDelta(delta: Double): String = "%.1f km".format(abs(delta))
    private fun formatElevationDelta(delta: Double): String = "%.0f m".format(abs(delta))
    private fun formatDurationDelta(deltaSec: Int): String = formatDuration(abs(deltaSec))

    private fun formatDuration(durationSec: Int): String {
        if (durationSec <= 0) {
            return "0m"
        }
        val hours = durationSec / 3600
        val minutes = (durationSec % 3600) / 60
        return if (hours > 0) {
            "${hours}h${minutes.toString().padStart(2, '0')}m"
        } else {
            "${minutes}m"
        }
    }

    private fun mergePreview(left: List<List<Double>>, right: List<List<Double>>): List<List<Double>> {
        if (left.isEmpty()) {
            return right
        }
        if (right.isEmpty()) {
            return left
        }
        val merged = left.toMutableList()
        val rightHead = right.first()
        if (merged.lastOrNull()?.size ?: 0 >= 2 &&
            rightHead.size >= 2 &&
            (merged.last()[0] != rightHead[0] || merged.last()[1] != rightHead[1])
        ) {
            merged += rightHead
        }
        merged += right.drop(1)
        return merged
    }

    private fun buildOutAndBackPreview(points: List<List<Double>>, targetDistanceKm: Double): List<List<Double>> {
        if (points.size < 2) {
            return emptyList()
        }
        val outboundTargetKm = max(2.0, targetDistanceKm / 2.0)
        val outbound = mutableListOf(points.first())
        var accumulatedKm = 0.0
        for (index in 1 until points.size) {
            val previous = points[index - 1]
            val next = points[index]
            if (previous.size < 2 || next.size < 2) {
                continue
            }
            accumulatedKm += distanceBetween(
                Coordinates(previous[0], previous[1]),
                Coordinates(next[0], next[1])
            ) / 1000.0
            outbound += next
            if (accumulatedKm >= outboundTargetKm) {
                break
            }
        }

        if (outbound.size < 2 && points.size >= 2) {
            outbound += points[1]
        }
        val loop = mutableListOf<List<Double>>()
        loop += outbound
        for (index in outbound.size - 2 downTo 0) {
            loop += outbound[index]
        }
        return sampleLatLng(loop, PREVIEW_POINT_MAX_SIZE)
    }

    private fun inflateLoopToTargetDistance(points: List<List<Double>>, targetDistanceKm: Double): List<List<Double>> {
        if (points.size < 2 || targetDistanceKm <= 0.0) {
            return points
        }
        val baseDistanceKm = pathDistanceMeters(points) / 1000.0
        if (baseDistanceKm <= 0.0 || baseDistanceKm >= targetDistanceKm * 0.9) {
            return points
        }
        var laps = (targetDistanceKm / baseDistanceKm).roundToInt()
        if (laps < 1) {
            laps = 1
        }
        if (laps > 8) {
            laps = 8
        }
        if (laps == 1) {
            return points
        }
        val expanded = mutableListOf<List<Double>>()
        expanded += points
        repeat(laps - 1) {
            expanded += points.drop(1)
        }
        return expanded
    }

    private fun anchorLoopToPreferredStart(points: List<List<Double>>, preferredStart: Coordinates): List<List<Double>>? {
        if (points.size < 2) {
            return null
        }
        var bestIndex = -1
        var bestDistance = Double.MAX_VALUE
        points.forEachIndexed { index, point ->
            if (point.size < 2) {
                return@forEachIndexed
            }
            val distance = distanceBetween(preferredStart, Coordinates(point[0], point[1]))
            if (distance < bestDistance) {
                bestDistance = distance
                bestIndex = index
            }
        }
        if (bestIndex < 0 || bestDistance > 1500.0) {
            return null
        }

        val rotated = mutableListOf<List<Double>>()
        rotated += points.drop(bestIndex)
        rotated += points.take(bestIndex)
        if (rotated.size < 2) {
            return null
        }
        val anchored = buildList {
            add(listOf(preferredStart.lat, preferredStart.lng))
            addAll(rotated)
            add(listOf(preferredStart.lat, preferredStart.lng))
        }
        val normalized = removeConsecutiveDuplicatePoints(anchored)
        if (normalized.size < 4) {
            return null
        }
        val firstId = quantizedPointId(normalized.first())
        val lastId = quantizedPointId(normalized.last())
        if (firstId.isNotBlank() && lastId.isNotBlank() && firstId != lastId) {
            return normalized + listOf(normalized.first())
        }
        return normalized
    }

    @Suppress("UNUSED_PARAMETER")
    private fun isTargetMatchAcceptable(
        actualDistanceKm: Double,
        actualElevationM: Double,
        targetDistanceKm: Double,
        targetElevationM: Double,
    ): Boolean {
        if (targetDistanceKm > 0.0) {
            val minDistanceKm = max(2.0, targetDistanceKm * 0.30)
            val maxDistanceKm = targetDistanceKm * 2.60
            if (actualDistanceKm < minDistanceKm || actualDistanceKm > maxDistanceKm) {
                return false
            }
        }
        return true
    }

    private fun routeGeometrySignature(points: List<List<Double>>): String {
        if (points.size < 2) {
            return ""
        }
        val sampled = sampleLatLng(points, 12)
        if (sampled.size < 2) {
            return ""
        }
        val parts = sampled.mapNotNull { point ->
            if (point.size < 2) {
                null
            } else {
                "%.3f,%.3f".format(point[0], point[1])
            }
        }.toMutableList()
        if (parts.isEmpty()) {
            return ""
        }
        val distanceKm = pathDistanceMeters(points) / 1000.0
        parts += "d=%.1f".format(round(distanceKm * 10.0) / 10.0)
        return parts.joinToString("|")
    }

    private fun isLoopWithoutHeavySegmentReuse(points: List<List<Double>>): Boolean {
        val normalized = removeConsecutiveDuplicatePoints(points)
        if (normalized.size < 4) {
            return false
        }
        val firstId = quantizedPointId(normalized.first())
        val lastId = quantizedPointId(normalized.last())
        if (firstId.isBlank() || lastId.isBlank() || firstId != lastId) {
            return false
        }

        val seenSegments = mutableMapOf<String, Int>()
        var repeatedSegments = 0
        var lastLeft = ""
        var lastRight = ""
        for (index in 0 until normalized.size - 1) {
            val left = quantizedPointId(normalized[index])
            val right = quantizedPointId(normalized[index + 1])
            if (left.isBlank() || right.isBlank() || left == right) {
                return false
            }
            if (index > 0 && left == lastRight && right == lastLeft) {
                return false
            }
            val segmentKey = normalizedSegmentKey(left, right)
            if ((seenSegments[segmentKey] ?: 0) > 0) {
                repeatedSegments++
            }
            seenSegments[segmentKey] = (seenSegments[segmentKey] ?: 0) + 1
            lastLeft = left
            lastRight = right
        }
        val maxRepeatedSegments = max(1, normalized.size / 12)
        return repeatedSegments <= maxRepeatedSegments
    }

    private fun quantizedPointId(point: List<Double>): String {
        val node = quantizedRoadNode(point) ?: return ""
        return node.id
    }

    private fun normalizedSegmentKey(left: String, right: String): String {
        return if (left <= right) {
            "$left|$right"
        } else {
            "$right|$left"
        }
    }

    private fun removeConsecutiveDuplicatePoints(points: List<List<Double>>): List<List<Double>> {
        if (points.size <= 1) {
            return points
        }
        val result = mutableListOf<List<Double>>()
        var lastId = ""
        for (point in points) {
            val id = quantizedPointId(point)
            if (id.isBlank()) {
                continue
            }
            if (result.isNotEmpty() && id == lastId) {
                continue
            }
            lastId = id
            result += point
        }
        return result
    }

    private fun firstNonEmpty(vararg values: String?): String? {
        return values.firstOrNull { value -> !value.isNullOrBlank() }?.trim()
    }


    private fun parseDate(value: String): Instant? {
        return runCatching { OffsetDateTime.parse(value).toInstant() }
            .recoverCatching { Instant.parse(value) }
            .recoverCatching {
                LocalDateTime.parse(
                    value,
                    DateTimeFormatter.ofPattern("yyyy-MM-dd'T'HH:mm:ss.SSSSSS")
                ).toInstant(ZoneOffset.UTC)
            }
            .recoverCatching {
                LocalDateTime.parse(
                    value,
                    DateTimeFormatter.ofPattern("yyyy-MM-dd'T'HH:mm:ss")
                ).toInstant(ZoneOffset.UTC)
            }
            .getOrNull()
    }

    private fun startAreaLabel(coordinates: Coordinates?): String {
        return coordinates?.let { point -> "%.2f, %.2f".format(point.lat, point.lng) } ?: "Unknown start"
    }

    private fun distanceBetween(left: Coordinates?, right: Coordinates?): Double {
        if (left == null || right == null) {
            return Double.MAX_VALUE
        }
        val lat1 = left.lat * PI / 180.0
        val lat2 = right.lat * PI / 180.0
        val deltaLat = (right.lat - left.lat) * PI / 180.0
        val deltaLng = (right.lng - left.lng) * PI / 180.0
        val a = sin(deltaLat / 2).pow(2) + cos(lat1) * cos(lat2) * sin(deltaLng / 2).pow(2)
        val c = 2 * atan2(sqrt(a), sqrt(1 - a))
        return 6371000.0 * c
    }

    private fun median(values: List<Double>, fallback: Double): Double {
        if (values.isEmpty()) {
            return fallback
        }
        val sorted = values.sorted()
        val middle = sorted.size / 2
        return if (sorted.size % 2 == 0) {
            (sorted[middle - 1] + sorted[middle]) / 2.0
        } else {
            sorted[middle]
        }
    }

    private fun minLong(left: Long, right: Long): Long {
        return if (left < right) left else right
    }

    private fun maxLong(left: Long, right: Long): Long {
        return if (left > right) left else right
    }

    private data class RouteCandidate(
        val activity: StravaActivity,
        val date: Instant?,
        val activityDate: String,
        val distanceKm: Double,
        val elevationGainM: Double,
        val durationSec: Int,
        val isLoop: Boolean,
        val start: Coordinates?,
        val end: Coordinates?,
        val startArea: String,
        val season: String,
        val previewLatLng: List<List<Double>>,
        val shape: String,
        val shapeScore: Double,
    )

    private data class RouteScoringProfile(
        val distanceWeight: Double,
        val elevationWeight: Double,
        val durationWeight: Double,
        val directionWeight: Double,
        val startPointWeight: Double,
    )

    private data class RoadGraphNode(
        val id: String,
        val lat: Double,
        val lng: Double,
    )

    private data class RoadGraphEdge(
        val to: String,
        val distance: Double,
    )

    private data class RoadGraph(
        val nodes: Map<String, RoadGraphNode>,
        val edges: Map<String, List<RoadGraphEdge>>,
    )

    private data class RoadPathState(
        val nodeId: String,
        val distance: Double,
    )
}
