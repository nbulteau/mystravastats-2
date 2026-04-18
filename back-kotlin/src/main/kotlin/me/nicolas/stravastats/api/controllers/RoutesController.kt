package me.nicolas.stravastats.api.controllers

import io.swagger.v3.oas.annotations.Operation
import io.swagger.v3.oas.annotations.media.Content
import io.swagger.v3.oas.annotations.media.Schema
import io.swagger.v3.oas.annotations.responses.ApiResponse
import me.nicolas.stravastats.api.dto.ErrorResponseMessageDto
import me.nicolas.stravastats.api.dto.GenerateRoutesResponseDto
import me.nicolas.stravastats.api.dto.GenerateShapeRoutesRequestDto
import me.nicolas.stravastats.api.dto.GenerateTargetRoutesRequestDto
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
import org.springframework.http.ContentDisposition
import org.springframework.http.HttpHeaders
import org.springframework.http.HttpStatus
import org.springframework.http.MediaType
import org.springframework.http.ResponseEntity
import org.springframework.web.bind.annotation.GetMapping
import org.springframework.web.bind.annotation.PathVariable
import org.springframework.web.bind.annotation.PostMapping
import org.springframework.web.bind.annotation.RequestBody
import org.springframework.web.bind.annotation.RequestMapping
import org.springframework.web.bind.annotation.RequestParam
import org.springframework.web.bind.annotation.RestController
import org.springframework.web.server.ResponseStatusException
import tools.jackson.databind.json.JsonMapper
import tools.jackson.module.kotlin.KotlinModule
import tools.jackson.module.kotlin.readValue
import java.time.Instant
import java.util.concurrent.ConcurrentHashMap
import kotlin.math.abs
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
    }

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

    @PostMapping("/generate/target")
    fun generateTargetRoutes(
        @RequestParam(required = false) activityType: String?,
        @RequestParam(required = false) year: Int?,
        @RequestBody payload: GenerateTargetRoutesRequestDto,
    ): GenerateRoutesResponseDto {
        validateTargetPayload(payload)
        val activityTypes = parseActivityTypesOrDefault(activityType)
        val routeType = normalizeRouteType(payload.routeType)
        val targetMode = normalizeTargetGenerationMode(payload.generationMode) ?: "AUTOMATIC"
        val startDirection = if (targetMode == "CUSTOM") {
            null
        } else {
            normalizeStartDirection(payload.startDirection)
        }
        val variantCount = normalizeVariantCount(payload.variantCount)
        val distanceTarget = payload.distanceTargetKm!!
        val request = RouteExplorerRequest(
            distanceTargetKm = distanceTarget,
            elevationTargetM = payload.elevationTargetM,
            durationTargetMin = null,
            startDirection = startDirection,
            startPoint = payload.startPoint?.toCoordinates(),
            targetMode = targetMode,
            customWaypoints = payload.customWaypoints.orEmpty().map { waypoint -> waypoint.toCoordinates() },
            routeType = routeType,
            season = null,
            limit = variantCount,
            shape = null,
            shapePolyline = null,
            includeRemix = false,
        )
        val result = routeExplorerService.getRouteExplorer(activityTypes, year, request)
        val routes = buildTargetGeneratedRoutes(
            result = result,
            distanceTarget = distanceTarget,
            elevationTarget = payload.elevationTargetM,
            routeType = routeType,
            startDirection = startDirection,
            limit = variantCount,
        )
        val diagnostics = buildTargetGenerationDiagnostics(
            distanceTarget = distanceTarget,
            elevationTarget = payload.elevationTargetM,
            startDirection = startDirection,
            targetMode = targetMode,
            routes = routes,
        )
        cacheGeneratedRoutes(routes)
        return GenerateRoutesResponseDto(
            routes = routes,
            diagnostics = diagnostics,
        )
    }

    @PostMapping("/generate/shape")
    fun generateShapeRoutes(
        @RequestParam(required = false) activityType: String?,
        @RequestParam(required = false) year: Int?,
        @RequestBody payload: GenerateShapeRoutesRequestDto,
    ): GenerateRoutesResponseDto {
        validateShapePayload(payload)
        val activityTypes = parseActivityTypesOrDefault(activityType)
        val routeType = normalizeRouteType(payload.routeType)
        val variantCount = normalizeVariantCount(payload.variantCount)
        val shapeFilter = inferShapeFilter(payload.shapeInputType.orEmpty(), payload.shapeData.orEmpty())
        val request = RouteExplorerRequest(
            distanceTargetKm = payload.distanceTargetKm,
            elevationTargetM = payload.elevationTargetM,
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
            distanceTarget = payload.distanceTargetKm,
            elevationTarget = payload.elevationTargetM,
            routeType = routeType,
            limit = variantCount,
        )
        cacheGeneratedRoutes(routes)
        return GenerateRoutesResponseDto(
            routes = routes,
            diagnostics = emptyList(),
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

    private fun validateTargetPayload(payload: GenerateTargetRoutesRequestDto) {
        val startPoint = payload.startPoint ?: throw ResponseStatusException(HttpStatus.BAD_REQUEST, "startPoint is required")
        if (!isValidLatLng(startPoint.lat, startPoint.lng)) {
            throw ResponseStatusException(HttpStatus.BAD_REQUEST, "startPoint has invalid coordinates")
        }
        val targetMode = normalizeTargetGenerationMode(payload.generationMode)
            ?: throw ResponseStatusException(HttpStatus.BAD_REQUEST, "generationMode must be one of AUTOMATIC/CUSTOM")
        val distanceTarget = payload.distanceTargetKm ?: throw ResponseStatusException(HttpStatus.BAD_REQUEST, "distanceTargetKm is required")
        if (distanceTarget <= 0.0) {
            throw ResponseStatusException(HttpStatus.BAD_REQUEST, "distanceTargetKm must be greater than 0")
        }
        payload.elevationTargetM?.let { elevation ->
            if (elevation < 0.0) {
                throw ResponseStatusException(HttpStatus.BAD_REQUEST, "elevationTargetM must be greater than or equal to 0")
            }
        }
        if (targetMode == "AUTOMATIC") {
            payload.startDirection?.trim()?.takeIf { value -> value.isNotEmpty() }?.let { direction ->
                if (normalizeStartDirection(direction) == null) {
                    throw ResponseStatusException(HttpStatus.BAD_REQUEST, "startDirection must be one of N/S/E/W")
                }
            }
        }
        if (targetMode == "CUSTOM") {
            val customWaypoints = payload.customWaypoints.orEmpty()
            if (customWaypoints.isEmpty()) {
                throw ResponseStatusException(HttpStatus.BAD_REQUEST, "customWaypoints must contain at least one waypoint when generationMode is CUSTOM")
            }
            customWaypoints.forEach { point ->
                if (!isValidLatLng(point.lat, point.lng)) {
                    throw ResponseStatusException(HttpStatus.BAD_REQUEST, "customWaypoints has invalid coordinates")
                }
            }
        }
        payload.variantCount?.let { value ->
            if (value !in 1..MAX_VARIANT_COUNT) {
                throw ResponseStatusException(HttpStatus.BAD_REQUEST, "variantCount must be between 1 and $MAX_VARIANT_COUNT")
            }
        }
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
        payload.distanceTargetKm?.let { distance ->
            if (distance <= 0.0) {
                throw ResponseStatusException(HttpStatus.BAD_REQUEST, "distanceTargetKm must be greater than 0")
            }
        }
        payload.elevationTargetM?.let { elevation ->
            if (elevation < 0.0) {
                throw ResponseStatusException(HttpStatus.BAD_REQUEST, "elevationTargetM must be greater than or equal to 0")
            }
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

    private fun buildTargetGeneratedRoutes(
        result: RouteExplorerResult,
        distanceTarget: Double,
        elevationTarget: Double?,
        routeType: String?,
        startDirection: String?,
        limit: Int,
    ): List<GeneratedRouteDto> {
        val routes = mutableListOf<GeneratedRouteDto>()
        val seen = mutableSetOf<String>()
        // Target mode must return newly generated loops only.
        val ordered = result.roadGraphLoops
        for (recommendation in ordered) {
            if (routes.size >= limit) {
                break
            }
            if (!seen.add(recommendation.routeId)) {
                continue
            }
            val score = buildGeneratedRouteScore(
                recommendation = recommendation,
                distanceTarget = distanceTarget,
                elevationTarget = elevationTarget,
                startDirection = startDirection,
            )
            routes += recommendation.toGeneratedRouteDto(score, routeType, startDirection)
        }
        return routes
    }

    private fun buildTargetGenerationDiagnostics(
        distanceTarget: Double,
        elevationTarget: Double?,
        startDirection: String?,
        targetMode: String?,
        routes: List<GeneratedRouteDto>,
    ): List<RouteGenerationDiagnosticDto> {
        if (routes.isNotEmpty()) {
            return emptyList()
        }

        val diagnostics = mutableListOf(
            RouteGenerationDiagnosticDto(
                code = "NO_CANDIDATE",
                message = "No route candidate matched all current constraints.",
            )
        )

        if (distanceTarget >= 120.0) {
            diagnostics += RouteGenerationDiagnosticDto(
                code = "DISTANCE_TOO_FAR",
                message = "Distance target is high for the current area and may remove most candidates.",
            )
        }

        if (elevationTarget != null && distanceTarget > 0.0) {
            val elevationPerKm = elevationTarget / distanceTarget
            if (elevationPerKm > 35.0) {
                diagnostics += RouteGenerationDiagnosticDto(
                    code = "ELEVATION_TOO_LOW",
                    message = "Requested elevation gain is likely too high for the selected distance and area.",
                )
            }
        }

        if (targetMode == "CUSTOM") {
            diagnostics += RouteGenerationDiagnosticDto(
                code = "CUSTOM_WAYPOINTS_CONFLICT",
                message = "Custom waypoint geometry may be too constrained in this area.",
            )
        }

        if (targetMode == "AUTOMATIC" && !startDirection.isNullOrBlank()) {
            diagnostics += RouteGenerationDiagnosticDto(
                code = "DIRECTION_CONFLICT",
                message = "Strict departure direction can filter out otherwise valid loops.",
            )
        }

        diagnostics += RouteGenerationDiagnosticDto(
            code = "BACKTRACKING_FILTERED",
            message = "Candidates that return over the same segment in reverse are rejected.",
        )

        return diagnostics
    }

    private fun buildShapeGeneratedRoutes(
        result: RouteExplorerResult,
        distanceTarget: Double?,
        elevationTarget: Double?,
        routeType: String?,
        limit: Int,
    ): List<GeneratedRouteDto> {
        val routes = mutableListOf<GeneratedRouteDto>()
        val seen = mutableSetOf<String>()
        val ordered = result.shapeMatches + result.roadGraphLoops + result.closestLoops
        for (recommendation in ordered) {
            if (routes.size >= limit) {
                break
            }
            if (!seen.add(recommendation.routeId)) {
                continue
            }
            val score = buildGeneratedRouteScore(
                recommendation = recommendation,
                distanceTarget = distanceTarget,
                elevationTarget = elevationTarget,
                startDirection = null,
            )
            routes += recommendation.toGeneratedRouteDto(score, routeType, null)
        }

        for (remix in result.shapeRemixes) {
            if (routes.size >= limit) {
                break
            }
            if (!seen.add(remix.id)) {
                continue
            }
            val score = RouteGenerationScoreDto(
                global = clampScore(remix.matchScore),
                distance = clampScore(remix.matchScore),
                elevation = clampScore(remix.matchScore),
                duration = clampScore(remix.matchScore),
                direction = clampScore(remix.matchScore),
                shape = clampScore(remix.matchScore),
                roadFitness = 75.0,
            )
            routes += remix.toGeneratedRouteDto(score, routeType)
        }
        return routes
    }

    private fun buildGeneratedRouteScore(
        recommendation: RouteRecommendation,
        distanceTarget: Double?,
        elevationTarget: Double?,
        startDirection: String?,
    ): RouteGenerationScoreDto {
        val global = clampScore(recommendation.matchScore)
        val distance = if (distanceTarget != null && distanceTarget > 0.0) {
            proximityScore(recommendation.distanceKm, distanceTarget)
        } else {
            global
        }
        val elevation = if (elevationTarget != null && elevationTarget >= 0.0) {
            proximityScore(recommendation.elevationGainM, elevationTarget)
        } else {
            global
        }
        val direction = if (!startDirection.isNullOrBlank() && recommendation.start != null && recommendation.end != null) {
            directionScore(recommendation.start, recommendation.end, startDirection)
        } else {
            global
        }
        val shape = recommendation.shapeScore?.let { value -> clampScore(value * 100.0) } ?: 50.0
        val roadFitness = when {
            recommendation.variantType == RouteVariantType.ROAD_GRAPH -> 100.0
            recommendation.isLoop -> 82.0
            else -> 70.0
        }
        return RouteGenerationScoreDto(
            global = global,
            distance = distance,
            elevation = elevation,
            duration = global,
            direction = direction,
            shape = shape,
            roadFitness = roadFitness,
        )
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

    private fun normalizeTargetGenerationMode(value: String?): String? {
        return when (value?.trim()?.uppercase()) {
            null, "", "AUTOMATIC" -> "AUTOMATIC"
            "CUSTOM" -> "CUSTOM"
            else -> null
        }
    }

    private fun normalizeStartDirection(value: String?): String? {
        val normalized = value?.trim()?.uppercase()
        return when (normalized) {
            "N", "S", "E", "W" -> normalized
            else -> null
        }
    }

    private fun normalizeVariantCount(value: Int?): Int {
        if (value == null) {
            return DEFAULT_VARIANT_COUNT
        }
        return value.coerceIn(1, MAX_VARIANT_COUNT)
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
        if (normalizedInputType != "draw" && normalizedInputType != "polyline") {
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
                null
            }
        }
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

    private fun proximityScore(actual: Double, target: Double): Double {
        if (target <= 0.0) {
            return 50.0
        }
        val deltaRatio = abs(actual - target) / target
        return clampScore(100.0 - (deltaRatio * 100.0))
    }

    private fun directionScore(start: Coordinates, end: Coordinates, expected: String): Double {
        val actual = directionFromCoordinates(start, end) ?: return 50.0
        return if (actual == expected) 100.0 else 40.0
    }

    private fun directionFromCoordinates(start: Coordinates, end: Coordinates): String? {
        val dLat = end.lat - start.lat
        val dLng = end.lng - start.lng
        if (abs(dLat) < 0.0001 && abs(dLng) < 0.0001) {
            return null
        }
        return if (abs(dLat) >= abs(dLng)) {
            if (dLat >= 0.0) "N" else "S"
        } else {
            if (dLng >= 0.0) "E" else "W"
        }
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
        startDirection: String?,
    ): GeneratedRouteDto {
        val title = activity.name.ifBlank { routeId }
        return GeneratedRouteDto(
            routeId = routeId,
            title = title,
            variantType = variantType.name,
            routeType = routeType,
            startDirection = startDirection,
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
            isRoadGraphGenerated = variantType == RouteVariantType.ROAD_GRAPH,
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
            startDirection = null,
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
