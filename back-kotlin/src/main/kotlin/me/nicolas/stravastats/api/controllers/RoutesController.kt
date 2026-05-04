package me.nicolas.stravastats.api.controllers

import jakarta.servlet.http.HttpServletResponse
import io.swagger.v3.oas.annotations.Operation
import io.swagger.v3.oas.annotations.media.Content
import io.swagger.v3.oas.annotations.media.Schema
import io.swagger.v3.oas.annotations.responses.ApiResponse
import me.nicolas.stravastats.api.dto.ErrorResponseMessageDto
import me.nicolas.stravastats.api.dto.GenerateRoutesResponseDto
import me.nicolas.stravastats.api.dto.GenerateShapeRoutesRequestDto
import me.nicolas.stravastats.api.dto.GeneratedRouteDto
import me.nicolas.stravastats.api.dto.RouteExplorerResultDto
import me.nicolas.stravastats.api.dto.RouteGenerationScoreDto
import me.nicolas.stravastats.api.dto.RouteGenerationDiagnosticDto
import me.nicolas.stravastats.api.dto.RouteCoordinateDto
import me.nicolas.stravastats.api.dto.RouteStartPointDto
import me.nicolas.stravastats.api.dto.toDto
import me.nicolas.stravastats.domain.business.ActivityType
import me.nicolas.stravastats.domain.business.Coordinates
import me.nicolas.stravastats.domain.business.RouteExplorerRequest
import me.nicolas.stravastats.domain.business.RouteExplorerResult
import me.nicolas.stravastats.domain.business.RouteRecommendation
import me.nicolas.stravastats.domain.business.RouteVariantType
import me.nicolas.stravastats.domain.business.ShapeRemixRecommendation
import me.nicolas.stravastats.domain.services.IRouteExplorerService
import org.slf4j.LoggerFactory
import org.springframework.http.ContentDisposition
import org.springframework.http.HttpHeaders
import org.springframework.http.HttpStatus
import org.springframework.http.MediaType
import org.springframework.http.ResponseEntity
import org.springframework.web.bind.annotation.GetMapping
import org.springframework.web.bind.annotation.PathVariable
import org.springframework.web.bind.annotation.PostMapping
import org.springframework.web.bind.annotation.RequestBody
import org.springframework.web.bind.annotation.RequestHeader
import org.springframework.web.bind.annotation.RequestMapping
import org.springframework.web.bind.annotation.RequestParam
import org.springframework.web.bind.annotation.RestController
import org.springframework.web.server.ResponseStatusException
import tools.jackson.databind.json.JsonMapper
import tools.jackson.module.kotlin.KotlinModule
import tools.jackson.module.kotlin.readValue
import java.time.Instant
import java.util.UUID
import java.util.concurrent.ConcurrentHashMap
import kotlin.math.atan2
import kotlin.math.cos
import kotlin.math.max
import kotlin.math.min
import kotlin.math.round
import kotlin.math.sin
import kotlin.math.sqrt

@RestController
@RequestMapping("/routes", produces = [MediaType.APPLICATION_JSON_VALUE])
@Schema(description = "Routes explorer controller", name = "RoutesController")
class RoutesController(
    private val routeExplorerService: IRouteExplorerService,
) {

    companion object {
        private const val DEFAULT_VARIANT_COUNT = 2
        private const val MAX_VARIANT_COUNT = 24
        private const val GENERATED_ROUTE_CACHE_TTL_SECONDS = 6 * 3600L
        private const val REQUEST_ID_HEADER = "X-Request-Id"
    }

    private val logger = LoggerFactory.getLogger(RoutesController::class.java)

    private data class CachedGeneratedRoute(
        val name: String,
        val points: List<List<Double>>,
        val expiresAt: Instant,
    )

    private val generatedRouteCache = ConcurrentHashMap<String, CachedGeneratedRoute>()
    private val shapeMapper = JsonMapper.builder()
        .addModule(KotlinModule.Builder().build())
        .build()

    @Operation(
        description = "Get route recommendations based on historical cached activities",
        summary = "Get route recommendations",
        responses = [
            ApiResponse(
                responseCode = "200",
                description = "Routes recommendations found",
                content = [Content(
                    mediaType = MediaType.APPLICATION_JSON_VALUE,
                    schema = Schema(implementation = RouteExplorerResultDto::class)
                )]
            ),
            ApiResponse(
                responseCode = "400",
                description = "Invalid parameters",
                content = [Content(
                    mediaType = MediaType.APPLICATION_JSON_VALUE,
                    schema = Schema(implementation = ErrorResponseMessageDto::class)
                )]
            )
        ]
    )
    @GetMapping("/recommendations")
    fun getRouteRecommendations(
        @RequestParam(required = true) activityType: String,
        @RequestParam(required = false) year: Int?,
        @RequestParam(required = false) distanceTargetKm: Double?,
        @RequestParam(required = false) elevationTargetM: Double?,
        @RequestParam(required = false) durationTargetMin: Int?,
        @RequestParam(required = false) startDirection: String?,
        @RequestParam(required = false) startLat: Double?,
        @RequestParam(required = false) startLng: Double?,
        @RequestParam(required = false) routeType: String?,
        @RequestParam(required = false) season: String?,
        @RequestParam(required = false) shape: String?,
        @RequestParam(required = false) shapePolyline: String?,
        @RequestParam(required = false, defaultValue = "false") includeRemix: Boolean,
        @RequestParam(required = false, defaultValue = "6") limit: Int,
    ): RouteExplorerResultDto {
        val activityTypes = activityType.convertToActivityTypeSet()
        val preferredStart = parseOptionalStartPoint(startLat, startLng)
        val request = RouteExplorerRequest(
            distanceTargetKm = distanceTargetKm,
            elevationTargetM = elevationTargetM,
            durationTargetMin = durationTargetMin,
            startDirection = startDirection,
            startPoint = preferredStart,
            routeType = routeType,
            season = season,
            limit = limit,
            shape = shape,
            shapePolyline = shapePolyline,
            includeRemix = includeRemix,
        )
        return routeExplorerService.getRouteExplorer(activityTypes, year, request).toDto()
    }

    @PostMapping("/generate/shape")
    fun generateShapeRoutes(
        @RequestParam(required = false) activityType: String?,
        @RequestParam(required = false) year: Int?,
        @RequestBody payload: GenerateShapeRoutesRequestDto,
        @RequestHeader(name = REQUEST_ID_HEADER, required = false) requestIdHeader: String?,
        response: HttpServletResponse,
    ): GenerateRoutesResponseDto {
        val requestId = resolveRouteRequestId(requestIdHeader)
        response.setHeader(REQUEST_ID_HEADER, requestId)
        val startedAtNs = System.nanoTime()
        validateShapePayload(payload)
        val activityTypes = parseActivityTypesOrDefault(activityType)
        val routeType = normalizeRouteType(payload.routeType)
        val variantCount = normalizeVariantCount(payload.variantCount)
        val shapeFilter = inferShapeFilter(payload.shapeInputType.orEmpty(), payload.shapeData.orEmpty())
        val request = RouteExplorerRequest(
            distanceTargetKm = null,
            elevationTargetM = null,
            durationTargetMin = null,
            startDirection = null,
            startPoint = payload.startPoint?.toCoordinates(),
            routeType = routeType,
            season = null,
            limit = variantCount,
            shape = shapeFilter,
            shapePolyline = payload.shapeData?.trim()?.takeIf { value -> value.isNotBlank() },
            includeRemix = true,
        )
        val result = routeExplorerService.getRouteExplorer(activityTypes, year, request)
        val routes = buildShapeGeneratedRoutes(
            result = result,
            routeType = routeType,
            limit = variantCount,
        )
        val diagnostics = buildShapeGenerationDiagnostics(
            routes = routes,
            routeType = routeType,
            shapeInputType = payload.shapeInputType,
            shapeFilter = shapeFilter,
            requestId = requestId,
            ignoredCandidateCount = countIgnoredShapeGenerationCandidates(result),
        )
        cacheGeneratedRoutes(routes)
        logRouteGenerationSummary(
            mode = "shape",
            requestId = requestId,
            routeType = routeType,
            requestMode = payload.shapeInputType,
            variantCount = variantCount,
            routes = routes,
            diagnostics = diagnostics,
            elapsedMs = (System.nanoTime() - startedAtNs) / 1_000_000,
        )
        return GenerateRoutesResponseDto(
            routes = routes,
            diagnostics = diagnostics,
        )
    }

    @GetMapping("/recommendations/gpx", produces = ["application/gpx+xml"])
    fun getRouteRecommendationGpx(
        @RequestParam(required = true) activityType: String,
        @RequestParam(required = false) year: Int?,
        @RequestParam(required = false) distanceTargetKm: Double?,
        @RequestParam(required = false) elevationTargetM: Double?,
        @RequestParam(required = false) durationTargetMin: Int?,
        @RequestParam(required = false) startDirection: String?,
        @RequestParam(required = false) startLat: Double?,
        @RequestParam(required = false) startLng: Double?,
        @RequestParam(required = false) routeType: String?,
        @RequestParam(required = false) season: String?,
        @RequestParam(required = false) shape: String?,
        @RequestParam(required = false) shapePolyline: String?,
        @RequestParam(required = false, defaultValue = "false") includeRemix: Boolean,
        @RequestParam(required = false, defaultValue = "6") limit: Int,
        @RequestParam(required = true) routeId: String,
    ): ResponseEntity<String> {
        val activityTypes = activityType.convertToActivityTypeSet()
        val preferredStart = parseOptionalStartPoint(startLat, startLng)
        val request = RouteExplorerRequest(
            distanceTargetKm = distanceTargetKm,
            elevationTargetM = elevationTargetM,
            durationTargetMin = durationTargetMin,
            startDirection = startDirection,
            startPoint = preferredStart,
            routeType = routeType,
            season = season,
            limit = limit,
            shape = shape,
            shapePolyline = shapePolyline,
            includeRemix = includeRemix,
        )
        val result = routeExplorerService.getRouteExplorer(activityTypes, year, request)
        val route = result.closestLoops.asSequence()
            .plus(result.variants.asSequence())
            .plus(result.seasonal.asSequence())
            .plus(result.roadGraphLoops.asSequence())
            .plus(result.shapeMatches.asSequence())
            .firstOrNull { recommendation -> recommendation.routeId == routeId }
        val remix = result.shapeRemixes.firstOrNull { recommendation -> recommendation.id == routeId }

        val routeName = route?.activity?.name ?: remix?.components?.firstOrNull()?.name?.let { name -> "Remix - $name" }
        val points = route?.previewLatLng ?: remix?.previewLatLng
        if (routeName.isNullOrBlank() || points.isNullOrEmpty()) {
            throw ResponseStatusException(HttpStatus.NOT_FOUND, "No route found for routeId=$routeId with current filters")
        }

        val gpx = toGpx(routeName, points)
        val fileName = sanitizeRouteFileName(routeId.ifBlank { "route" })
        val headers = HttpHeaders()
        headers.contentType = MediaType.parseMediaType("application/gpx+xml")
        headers.contentDisposition = ContentDisposition.attachment().filename("$fileName.gpx").build()
        return ResponseEntity.ok()
            .headers(headers)
            .body(gpx)
    }

    @GetMapping("/{routeId}/gpx", produces = ["application/gpx+xml"])
    fun getGeneratedRouteGpx(
        @PathVariable routeId: String,
    ): ResponseEntity<String> {
        val cachedRoute = getCachedGeneratedRoute(routeId)
            ?: throw ResponseStatusException(HttpStatus.NOT_FOUND, "No generated route found for routeId=$routeId")
        val gpx = toGpx(cachedRoute.name, cachedRoute.points)
        val fileName = sanitizeRouteFileName(routeId.ifBlank { "route" })
        val headers = HttpHeaders()
        headers.contentType = MediaType.parseMediaType("application/gpx+xml")
        headers.contentDisposition = ContentDisposition.attachment().filename("$fileName.gpx").build()
        return ResponseEntity.ok()
            .headers(headers)
            .body(gpx)
    }

    private fun validateShapePayload(payload: GenerateShapeRoutesRequestDto) {
        val shapeInputType = payload.shapeInputType?.trim()?.lowercase()
            ?: throw ResponseStatusException(HttpStatus.BAD_REQUEST, "shapeInputType is required")
        if (shapeInputType !in setOf("draw", "gpx", "svg", "polyline")) {
            throw ResponseStatusException(HttpStatus.BAD_REQUEST, "shapeInputType must be one of draw/gpx/svg/polyline")
        }
        val shapeData = payload.shapeData?.trim()
        if (shapeData.isNullOrEmpty()) {
            throw ResponseStatusException(HttpStatus.BAD_REQUEST, "shapeData is required")
        }
        payload.startPoint?.let { startPoint ->
            if (!isValidLatLng(startPoint.lat, startPoint.lng)) {
                throw ResponseStatusException(HttpStatus.BAD_REQUEST, "startPoint has invalid coordinates")
            }
        }
        payload.variantCount?.let { value ->
            if (value !in 1..MAX_VARIANT_COUNT) {
                throw ResponseStatusException(HttpStatus.BAD_REQUEST, "variantCount must be between 1 and $MAX_VARIANT_COUNT")
            }
        }
    }

    private fun parseActivityTypesOrDefault(activityType: String?): Set<ActivityType> {
        val raw = activityType?.trim().orEmpty()
        if (raw.isBlank()) {
            return setOf(
                ActivityType.Ride,
                ActivityType.GravelRide,
                ActivityType.MountainBikeRide,
                ActivityType.Commute,
                ActivityType.VirtualRide,
                ActivityType.Run,
                ActivityType.TrailRun,
                ActivityType.Hike,
                ActivityType.Walk,
            )
        }
        return raw.convertToActivityTypeSet()
    }

    private fun buildSuccessfulGenerationDiagnostics(routes: List<GeneratedRouteDto>): List<RouteGenerationDiagnosticDto> {
        val diagnostics = mutableListOf<RouteGenerationDiagnosticDto>()
        val seenCodes = mutableSetOf<String>()
        fun appendOnce(code: String, message: String) {
            if (!seenCodes.add(code)) {
                return
            }
            diagnostics += RouteGenerationDiagnosticDto(
                code = code,
                message = message,
            )
        }

        routes.forEach { route ->
            route.reasons.forEach { reason ->
                val normalized = reason.trim()
                when {
                    normalized.startsWith("Direction relaxed:") -> appendOnce(
                        code = "DIRECTION_RELAXED",
                        message = "Direction constraint was relaxed to return a valid route.",
                    )
                    normalized.startsWith("Anti-backtracking relaxed:") -> appendOnce(
                        code = "BACKTRACKING_RELAXED",
                        message = "Anti-backtracking constraints were relaxed to return a valid route.",
                    )
                    normalized.startsWith("Route type fallback:") -> appendOnce(
                        code = "ROUTE_TYPE_FALLBACK",
                        message = normalized,
                    )
                    normalized.startsWith("Start snapped to nearest routable point") -> appendOnce(
                        code = "START_POINT_SNAPPED",
                        message = normalized,
                    )
                    normalized == "Generation engine fallback: legacy synthetic waypoints" -> appendOnce(
                        code = "ENGINE_FALLBACK_LEGACY",
                        message = "Legacy waypoint generator was used as fallback.",
                    )
                    normalized.startsWith("Selection profile: best-effort-soft") -> appendOnce(
                        code = "SELECTION_RELAXED",
                        message = "Selection constraints were softened to preserve route availability.",
                    )
                    normalized.startsWith("Selection profile: directional-best-effort") -> appendOnce(
                        code = "DIRECTION_BEST_EFFORT",
                        message = "Directional constraints were softened to preserve route availability.",
                    )
                    normalized.startsWith("Selection profile: art-fit-diagnostic") -> appendOnce(
                        code = "ART_FIT_RETRACE_ALLOWED",
                        message = "Strava Art selected drawing fit first; retrace is reported as rideability context.",
                    )
                    normalized.contains("Selection profile: emergency-fallback") -> appendOnce(
                        code = "EMERGENCY_FALLBACK",
                        message = "Emergency fallback selected the best available generated route.",
                    )
                    normalized == "Generation fallback: historical route cache" -> appendOnce(
                        code = "ENGINE_CACHE_FALLBACK",
                        message = "Road-graph generation was unavailable, historical cache routes were returned.",
                    )
                }
            }
        }

        return diagnostics
    }

    private fun buildShapeGeneratedRoutes(
        result: RouteExplorerResult,
        routeType: String?,
        limit: Int,
    ): List<GeneratedRouteDto> {
        val routes = mutableListOf<GeneratedRouteDto>()
        val seen = mutableSetOf<String>()
        val ordered = (result.shapeMatches + result.roadGraphLoops)
            .filter { recommendation -> isShapeGeneratedRouteCandidate(recommendation) }
        for (recommendation in ordered) {
            if (!seen.add(recommendation.routeId)) {
                continue
            }
            val score = buildGeneratedRouteScore(recommendation)
            routes += recommendation.toGeneratedRouteDto(score, routeType)
        }
        return routes
            .sortedWith(
                compareByDescending<GeneratedRouteDto> { route -> route.score.shape }
                    .thenByDescending { route -> route.score.global }
                    .thenByDescending { route -> route.score.roadFitness }
                    .thenBy { route -> route.distanceKm }
                    .thenBy { route -> route.routeId }
            )
            .take(limit)
    }

    private fun isShapeGeneratedRouteCandidate(recommendation: RouteRecommendation): Boolean {
        if (recommendation.variantType !in setOf(RouteVariantType.SHAPE_MATCH, RouteVariantType.ROAD_GRAPH)) {
            return false
        }
        return recommendation.reasons.any { reason -> reason.trim().startsWith("Shape mode:") }
    }

    private fun countIgnoredShapeGenerationCandidates(result: RouteExplorerResult): Int {
        return result.closestLoops.size +
            result.shapeRemixes.size +
            result.shapeMatches.count { recommendation -> !isShapeGeneratedRouteCandidate(recommendation) } +
            result.roadGraphLoops.count { recommendation -> !isShapeGeneratedRouteCandidate(recommendation) }
    }

    private fun buildShapeGenerationDiagnostics(
        routes: List<GeneratedRouteDto>,
        routeType: String?,
        shapeInputType: String?,
        shapeFilter: String?,
        requestId: String,
        ignoredCandidateCount: Int,
    ): List<RouteGenerationDiagnosticDto> {
        if (routes.isNotEmpty()) {
            return buildSuccessfulGenerationDiagnostics(routes)
        }

        val diagnostics = mutableListOf(
            RouteGenerationDiagnosticDto(
                code = "NO_CANDIDATE",
                message = "No route candidate matched the provided shape.",
            )
        )
        if (ignoredCandidateCount > 0) {
            diagnostics += RouteGenerationDiagnosticDto(
                code = "NON_SHAPE_CANDIDATES_IGNORED",
                message = "Historical or non-shape route candidates were ignored because Strava Art only returns OSRM routes generated from the drawing.",
            )
        }

        val parts = mutableListOf(
            "${normalizeRouteType(routeType)} shape=${shapeFilter?.takeIf { value -> value.isNotBlank() } ?: "UNKNOWN"}",
        )
        shapeInputType?.trim()?.takeIf { value -> value.isNotBlank() }?.let { value ->
            parts += "input=${value.lowercase()}"
        }

        diagnostics += RouteGenerationDiagnosticDto(
            code = "FAILURE_SUMMARY",
            message = "No route generated (${parts.joinToString(", ")}). Try simplifying the shape or moving the start point. requestId=$requestId",
        )

        return diagnostics
    }

    private fun buildGeneratedRouteScore(recommendation: RouteRecommendation): RouteGenerationScoreDto {
        val global = clampScore(recommendation.matchScore)
        val shape = recommendation.shapeScore?.let { value -> clampScore(value * 100.0) } ?: 50.0
        val roadFitness = parseSurfaceFitnessReason(recommendation.reasons) ?: when {
            recommendation.variantType == RouteVariantType.ROAD_GRAPH -> 100.0
            recommendation.isLoop -> 82.0
            else -> 70.0
        }
        return RouteGenerationScoreDto(
            global = global,
            distance = global,
            elevation = global,
            duration = global,
            direction = global,
            shape = shape,
            roadFitness = roadFitness,
        )
    }

    private fun parseSurfaceFitnessReason(reasons: List<String>): Double? {
        reasons.forEach { reason ->
            val normalized = reason.trim()
            if (!normalized.startsWith("Surface fitness:", ignoreCase = false)) {
                return@forEach
            }
            val payload = normalized.removePrefix("Surface fitness:").trim().removeSuffix("%").trim()
            payload.toDoubleOrNull()?.let { value ->
                return clampScore(value)
            }
        }
        return null
    }

    private fun cacheGeneratedRoutes(routes: List<GeneratedRouteDto>) {
        cleanupGeneratedRouteCache()
        val expiry = Instant.now().plusSeconds(GENERATED_ROUTE_CACHE_TTL_SECONDS)
        routes.forEach { route ->
            if (route.routeId.isBlank() || route.previewLatLng.size < 2) {
                return@forEach
            }
            generatedRouteCache[route.routeId] = CachedGeneratedRoute(
                name = route.title,
                points = route.previewLatLng,
                expiresAt = expiry,
            )
        }
    }

    private fun getCachedGeneratedRoute(routeId: String): CachedGeneratedRoute? {
        cleanupGeneratedRouteCache()
        return generatedRouteCache[routeId]
    }

    private fun cleanupGeneratedRouteCache() {
        val now = Instant.now()
        generatedRouteCache.entries.removeIf { (_, entry) -> entry.expiresAt.isBefore(now) }
    }

    private fun normalizeRouteType(value: String?): String? {
        val normalized = value?.trim()?.uppercase()
        return when (normalized) {
            "RIDE", "MTB", "GRAVEL", "RUN", "TRAIL", "HIKE" -> normalized
            else -> "RIDE"
        }
    }

    private fun normalizeVariantCount(value: Int?): Int {
        if (value == null) {
            return DEFAULT_VARIANT_COUNT
        }
        return value.coerceIn(1, MAX_VARIANT_COUNT)
    }

    private fun resolveRouteRequestId(requestIdHeader: String?): String {
        val candidate = requestIdHeader?.trim().orEmpty()
        if (candidate.isNotEmpty()) {
            return candidate
        }
        return "route-${UUID.randomUUID()}"
    }

    private fun logRouteGenerationSummary(
        mode: String,
        requestId: String,
        routeType: String?,
        requestMode: String?,
        variantCount: Int,
        routes: List<GeneratedRouteDto>,
        diagnostics: List<RouteGenerationDiagnosticDto>,
        elapsedMs: Long,
    ) {
        logger.info(
            "category=routes requestId={} mode={} requestMode={} routeType={} variantCount={} generatedRoutes={} diagnostics={} routeReasons={} durationMs={}",
            requestId,
            mode,
            logValue(requestMode),
            logValue(routeType),
            variantCount,
            routes.size,
            diagnosticsCodeSummary(diagnostics),
            routeReasonSummary(routes),
            elapsedMs,
        )
    }

    private fun diagnosticsCodeSummary(diagnostics: List<RouteGenerationDiagnosticDto>): String {
        if (diagnostics.isEmpty()) {
            return "none"
        }
        val codes = diagnostics
            .mapNotNull { diagnostic -> diagnostic.code.trim().takeIf { code -> code.isNotEmpty() } }
            .distinct()
        if (codes.isEmpty()) {
            return "none"
        }
        return codes.joinToString("|")
    }

    private fun routeReasonSummary(routes: List<GeneratedRouteDto>): String {
        if (routes.isEmpty()) {
            return "none"
        }
        val reasons = routes
            .asSequence()
            .flatMap { route -> route.reasons.asSequence() }
            .map { reason -> reason.trim() }
            .filter { reason -> reason.isNotEmpty() }
            .distinct()
            .take(6)
            .toList()
        if (reasons.isEmpty()) {
            return "none"
        }
        return reasons.joinToString("|")
    }

    private fun logValue(value: String?): String {
        return value?.trim()?.takeIf { candidate -> candidate.isNotEmpty() } ?: "none"
    }

    private fun parseOptionalStartPoint(startLat: Double?, startLng: Double?): Coordinates? {
        if (startLat == null && startLng == null) {
            return null
        }
        if (startLat == null || startLng == null) {
            throw ResponseStatusException(HttpStatus.BAD_REQUEST, "startLat and startLng must be provided together")
        }
        if (!isValidLatLng(startLat, startLng)) {
            throw ResponseStatusException(HttpStatus.BAD_REQUEST, "invalid startLat/startLng coordinates")
        }
        return Coordinates(lat = startLat, lng = startLng)
    }

    private fun inferShapeFilter(shapeInputType: String, shapeData: String): String? {
        val normalizedInputType = shapeInputType.trim().lowercase()
        if (normalizedInputType != "draw" && normalizedInputType != "polyline" && normalizedInputType != "gpx") {
            return null
        }
        val points = parseShapeCoordinates(shapeData) ?: return null
        if (points.size < 2) {
            return null
        }
        return inferShapeFromCoordinates(points)
    }

    private fun parseShapeCoordinates(shapeData: String): List<List<Double>>? {
        val trimmed = shapeData.trim()
        if (trimmed.isEmpty()) {
            return null
        }
        return try {
            val rawPoints: List<List<Double>> = shapeMapper.readValue(trimmed)
            sanitizeShapeCoordinates(rawPoints)
        } catch (_: Exception) {
            try {
                val wrapped: Map<String, List<List<Double>>> = shapeMapper.readValue(trimmed)
                val points = wrapped["points"] ?: wrapped["coordinates"] ?: wrapped["latLng"] ?: return null
                sanitizeShapeCoordinates(points)
            } catch (_: Exception) {
                parseShapeCoordinatesFromGpx(trimmed)?.let { points ->
                    return sanitizeShapeCoordinates(points)
                }
                val encodedPolyline = try {
                    shapeMapper.readValue<String>(trimmed).trim()
                } catch (_: Exception) {
                    trimmed
                }
                decodeEncodedPolylineCoordinates(encodedPolyline)?.let { points ->
                    sanitizeShapeCoordinates(points)
                }
            }
        }
    }

    private fun parseShapeCoordinatesFromGpx(raw: String): List<List<Double>>? {
        val pointRegex = Regex("""<(?:trkpt|rtept|wpt)\b([^>]*)>""", setOf(RegexOption.IGNORE_CASE, RegexOption.DOT_MATCHES_ALL))
        val latRegex = Regex("""\blat\s*=\s*["']([^"']+)["']""", RegexOption.IGNORE_CASE)
        val lonRegex = Regex("""\blon\s*=\s*["']([^"']+)["']""", RegexOption.IGNORE_CASE)
        val points = mutableListOf<List<Double>>()
        pointRegex.findAll(raw).forEach { match ->
            val attributes = match.groupValues.getOrNull(1).orEmpty()
            val latText = latRegex.find(attributes)?.groupValues?.getOrNull(1)?.trim() ?: return@forEach
            val lonText = lonRegex.find(attributes)?.groupValues?.getOrNull(1)?.trim() ?: return@forEach
            val lat = latText.toDoubleOrNull() ?: return@forEach
            val lon = lonText.toDoubleOrNull() ?: return@forEach
            if (isValidLatLng(lat, lon)) {
                points += listOf(lat, lon)
            }
        }
        if (points.isEmpty()) {
            return null
        }
        return points
    }

    private fun decodeEncodedPolylineCoordinates(encodedPolyline: String): List<List<Double>>? {
        val encoded = encodedPolyline.trim()
        if (encoded.isEmpty()) {
            return null
        }
        val points = mutableListOf<List<Double>>()
        var index = 0
        var lat = 0
        var lng = 0
        while (index < encoded.length) {
            val latDelta = decodePolylineDelta(encoded, index) ?: return null
            index = latDelta.second
            val lngDelta = decodePolylineDelta(encoded, index) ?: return null
            index = lngDelta.second
            lat += latDelta.first
            lng += lngDelta.first
            points += listOf(lat / 1e5, lng / 1e5)
        }
        if (points.isEmpty()) {
            return null
        }
        return points
    }

    private fun decodePolylineDelta(encoded: String, startIndex: Int): Pair<Int, Int>? {
        var result = 0
        var shift = 0
        var index = startIndex
        while (index < encoded.length) {
            val chunk = encoded[index].code - 63
            if (chunk < 0) {
                return null
            }
            result = result or ((chunk and 0x1f) shl shift)
            shift += 5
            index += 1
            if (chunk < 0x20) {
                val delta = if ((result and 1) == 1) {
                    (result shr 1).inv()
                } else {
                    result shr 1
                }
                return delta to index
            }
        }
        return null
    }

    private fun sanitizeShapeCoordinates(points: List<List<Double>>): List<List<Double>> {
        return points.filter { point ->
            point.size >= 2 && isValidLatLng(point[0], point[1])
        }.map { point ->
            listOf(point[0], point[1])
        }
    }

    private fun inferShapeFromCoordinates(points: List<List<Double>>): String {
        if (points.size < 2) {
            return "POINT_TO_POINT"
        }
        val start = Coordinates(points.first()[0], points.first()[1])
        val end = Coordinates(points.last()[0], points.last()[1])
        val startEndDistance = haversineDistanceMeters(start, end)
        var pathDistance = 0.0
        var maxFromStart = 0.0
        for (index in 1 until points.size) {
            val previous = Coordinates(points[index - 1][0], points[index - 1][1])
            val next = Coordinates(points[index][0], points[index][1])
            val segment = haversineDistanceMeters(previous, next)
            pathDistance += segment
            val startDistance = haversineDistanceMeters(start, next)
            maxFromStart = max(maxFromStart, startDistance)
        }
        val loopThreshold = max(350.0, pathDistance * 0.08)
        if (startEndDistance <= loopThreshold) {
            return "LOOP"
        }
        if (maxFromStart > 0.0 && startEndDistance <= max(220.0, maxFromStart * 0.18)) {
            return "OUT_AND_BACK"
        }
        return "POINT_TO_POINT"
    }

    private fun haversineDistanceMeters(left: Coordinates, right: Coordinates): Double {
        val lat1 = Math.toRadians(left.lat)
        val lat2 = Math.toRadians(right.lat)
        val dLat = Math.toRadians(right.lat - left.lat)
        val dLng = Math.toRadians(right.lng - left.lng)

        val a = sin(dLat / 2.0) * sin(dLat / 2.0) +
            cos(lat1) * cos(lat2) * sin(dLng / 2.0) * sin(dLng / 2.0)
        val c = 2.0 * atan2(sqrt(a), sqrt(1.0 - a))
        return 6371000.0 * c
    }

    private fun isValidLatLng(lat: Double, lng: Double): Boolean {
        return lat in -90.0..90.0 && lng in -180.0..180.0
    }

    private fun clampScore(value: Double): Double {
        val normalized = min(100.0, max(0.0, value))
        return round(normalized * 10.0) / 10.0
    }

    private fun RouteStartPointDto.toCoordinates(): Coordinates = Coordinates(
        lat = lat,
        lng = lng,
    )

    private fun RouteRecommendation.toGeneratedRouteDto(
        score: RouteGenerationScoreDto,
        routeType: String?,
    ): GeneratedRouteDto {
        val title = activity.name.ifBlank { routeId }
        return GeneratedRouteDto(
            routeId = routeId,
            title = title,
            variantType = variantType.name,
            routeType = routeType,
            distanceKm = distanceKm,
            elevationGainM = elevationGainM,
            durationSec = durationSec,
            estimatedDurationSec = durationSec,
            score = score,
            reasons = reasons,
            previewLatLng = previewLatLng,
            start = start?.let { coordinate -> RouteCoordinateDto(lat = coordinate.lat, lng = coordinate.lng) },
            end = end?.let { coordinate -> RouteCoordinateDto(lat = coordinate.lat, lng = coordinate.lng) },
            activityId = activity.id.takeIf { value -> value != 0L },
            isRoadGraphGenerated = variantType == RouteVariantType.ROAD_GRAPH ||
                reasons.any { reason -> reason.trim() == "Generated with OSM road graph (OSRM)" },
        )
    }

    private fun ShapeRemixRecommendation.toGeneratedRouteDto(
        score: RouteGenerationScoreDto,
        routeType: String?,
    ): GeneratedRouteDto {
        val title = components.firstOrNull()?.name?.ifBlank { id } ?: id
        return GeneratedRouteDto(
            routeId = id,
            title = title,
            variantType = RouteVariantType.SHAPE_REMIX.name,
            routeType = routeType,
            distanceKm = distanceKm,
            elevationGainM = elevationGainM,
            durationSec = durationSec,
            estimatedDurationSec = durationSec,
            score = score,
            reasons = reasons,
            previewLatLng = previewLatLng,
            start = null,
            end = null,
            activityId = null,
            isRoadGraphGenerated = false,
        )
    }

    private fun toGpx(name: String, points: List<List<Double>>): String {
        val validPoints = points.filter { point ->
            point.size >= 2 &&
                point[0] in -90.0..90.0 &&
                point[1] in -180.0..180.0
        }
        if (validPoints.size < 2) {
            throw ResponseStatusException(HttpStatus.BAD_REQUEST, "at least 2 valid points are required to export GPX")
        }

        val safeName = escapeXml(name.ifBlank { "MyStravaStats route" })
        val trkPoints = validPoints.joinToString("\n") { point ->
            "      <trkpt lat=\"%.7f\" lon=\"%.7f\"></trkpt>".format(point[0], point[1])
        }
        return """
            <?xml version="1.0" encoding="UTF-8"?>
            <gpx version="1.1" creator="MyStravaStats" xmlns="http://www.topografix.com/GPX/1/1">
              <trk>
                <name>$safeName</name>
                <trkseg>
            $trkPoints
                </trkseg>
              </trk>
            </gpx>
        """.trimIndent()
    }

    private fun escapeXml(value: String): String {
        return value
            .replace("&", "&amp;")
            .replace("<", "&lt;")
            .replace(">", "&gt;")
            .replace("\"", "&quot;")
            .replace("'", "&apos;")
    }

    private fun sanitizeRouteFileName(value: String): String {
        val normalized = value.trim().lowercase()
        if (normalized.isBlank()) {
            return "route"
        }
        return normalized
            .replace(" ", "-")
            .replace("/", "-")
            .replace("\\", "-")
            .replace(":", "-")
            .replace(";", "-")
            .replace(",", "-")
            .replace("\"", "")
            .replace("'", "")
            .replace("(", "")
            .replace(")", "")
            .replace("[", "")
            .replace("]", "")
            .trim('-', '.', '_')
            .ifBlank { "route" }
    }
}
