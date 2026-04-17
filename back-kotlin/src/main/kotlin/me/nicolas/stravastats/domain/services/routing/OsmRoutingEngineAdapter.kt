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
import kotlin.math.cos
import kotlin.math.max
import kotlin.math.min
import kotlin.math.round
import kotlin.math.sin
import kotlin.math.sqrt

private const val DEFAULT_BASE_URL = "http://localhost:5000"
private const val DEFAULT_TIMEOUT_MS = 3000
private const val MAX_OSRM_CALLS = 16
private const val START_SNAP_TOLERANCE_METERS = 500.0
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

@Component
class OsmRoutingEngineAdapter : RoutingEnginePort {

    private val logger = LoggerFactory.getLogger(OsmRoutingEngineAdapter::class.java)

    private val enabled = readBoolConfig("OSM_ROUTING_ENABLED", true)
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
        val baseBearing = startDirectionToBearing(request.startDirection)
        val baseRadiusKm = max(1.0, request.distanceTargetKm / (2.0 * PI))
        val radiusMultipliers = listOf(1.00, 0.90, 1.10, 0.80, 1.20, 1.30, 0.70, 1.40)
        val rotations = listOf(0.0, 18.0, -18.0, 35.0, -35.0, 52.0, -52.0, 70.0, -70.0)
        val maxCalls = min(MAX_OSRM_CALLS, request.limit * 2 + 2)

        val recommendations = mutableListOf<RouteRecommendation>()
        val seenGeometry = mutableSetOf<String>()
        var generatedCount = 0

        for (callIndex in 0 until maxCalls) {
            if (recommendations.size >= request.limit) break
            val radiusKm = baseRadiusKm * radiusMultipliers[callIndex % radiusMultipliers.size]
            val rotation = rotations[callIndex % rotations.size]
            val waypoints = syntheticLoopWaypoints(
                start = request.startPoint,
                radiusKm = radiusKm,
                initialBearing = baseBearing + rotation,
            )
            val routes = runCatching { fetchRoutes(profile, waypoints) }
                .onFailure { logger.debug("OSRM route generation failed: {}", it.message) }
                .getOrElse { emptyList() }

            for ((routeIndex, route) in routes.withIndex()) {
                val recommendation = toRouteRecommendation(request, route, generatedCount + routeIndex) ?: continue
                val geometryKey = geometrySignature(recommendation.previewLatLng)
                if (geometryKey.isBlank() || !seenGeometry.add(geometryKey)) continue
                recommendations += recommendation
                if (recommendations.size >= request.limit) break
            }
            generatedCount += routes.size
        }

        return recommendations
    }

    override fun healthDetails(): Map<String, Any?> {
        val details = mutableMapOf<String, Any?>(
            "engine" to "osrm",
            "enabled" to enabled,
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

    private fun syntheticLoopWaypoints(
        start: Coordinates,
        radiusKm: Double,
        initialBearing: Double,
    ): List<Coordinates> {
        val bearing1 = normalizeBearing(initialBearing)
        val bearing2 = normalizeBearing(initialBearing + 120.0)
        val bearing3 = normalizeBearing(initialBearing + 240.0)
        return listOf(
            start,
            destinationFromBearing(start, radiusKm, bearing1),
            destinationFromBearing(start, radiusKm * 1.05, bearing2),
            destinationFromBearing(start, radiusKm * 0.95, bearing3),
            start,
        )
    }

    private fun fetchRoutes(profile: String, waypoints: List<Coordinates>): List<OsrmRoute> {
        if (waypoints.size < 2) return emptyList()
        val coordinates = waypoints.joinToString(";") { waypoint -> "%.6f,%.6f".format(waypoint.lng, waypoint.lat) }
        val url = "$baseUrl/route/v1/$profile/$coordinates?alternatives=true&steps=false&overview=full&geometries=geojson&continue_straight=false"
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

    private fun toRouteRecommendation(
        request: RoutingEngineRequest,
        route: OsrmRoute,
        index: Int,
    ): RouteRecommendation? {
        if (route.distance <= 0.0 || route.geometry == null || route.geometry.coordinates.size < 2) return null
        val preview = route.geometry.coordinates.mapNotNull { point ->
            if (point.size < 2) return@mapNotNull null
            val lng = point[0]
            val lat = point[1]
            if (lat !in -90.0..90.0 || lng !in -180.0..180.0) return@mapNotNull null
            listOf(lat, lng)
        }
        if (preview.size < 2) return null
        if (!startsNearRequestedStart(preview, request.startPoint, START_SNAP_TOLERANCE_METERS)) return null
        if (!respectsHalfPlaneDirection(preview, request.startPoint, request.startDirection, DIRECTION_TOLERANCE_METERS)) return null
        if (hasOppositeEdgeTraversal(preview)) return null

        val start = Coordinates(lat = preview.first()[0], lng = preview.first()[1])
        val end = Coordinates(lat = preview.last()[0], lng = preview.last()[1])
        val distanceKm = route.distance / 1000.0
        val durationSec = route.duration.toInt().coerceAtLeast((distanceKm * 180.0).toInt())
        val elevationEstimate = request.elevationTargetM?.let { target ->
            val deltaRatio = abs(distanceKm - request.distanceTargetKm) / max(1.0, request.distanceTargetKm)
            max(0.0, target * (1.0 - deltaRatio * 0.5))
        } ?: max(0.0, distanceKm * 8.0)
        val score = clampScore(
            100.0 - (abs(distanceKm - request.distanceTargetKm) / max(1.0, request.distanceTargetKm)) * 100.0
        )
        val routeId = generatedRouteId(preview, request.startPoint, index)
        val titleSuffix = if (index > 0) " #${index + 1}" else ""
        val title = "Generated loop near %.4f, %.4f%s".format(request.startPoint.lat, request.startPoint.lng, titleSuffix)

        val reasons = buildList {
            add("Generated with OSM road graph (OSRM)")
            add("Distance delta: ${formatDistanceDelta(distanceKm - request.distanceTargetKm)}")
            request.elevationTargetM?.let { target ->
                add("Elevation estimate: ${formatElevationDelta(elevationEstimate - target)}")
            }
            request.startDirection?.takeIf { it.isNotBlank() }?.let { direction ->
                add("Departure direction: ${direction.uppercase(Locale.getDefault())}")
            }
        }

        return RouteRecommendation(
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
            matchScore = score,
            reasons = reasons,
            previewLatLng = preview,
            shape = null,
            shapeScore = null,
            experimental = false,
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

    private fun hasOppositeEdgeTraversal(points: List<List<Double>>): Boolean {
        if (points.size < 3) return false

        data class EdgeDirection(var forward: Boolean = false, var reverse: Boolean = false)
        val seen = mutableMapOf<String, EdgeDirection>()

        for (index in 0 until points.size - 1) {
            val from = points[index]
            val to = points[index + 1]
            if (from.size < 2 || to.size < 2) continue

            val fromId = quantizedPointKey(from[0], from[1])
            val toId = quantizedPointKey(to[0], to[1])
            if (fromId == toId) continue

            val edgeKey = canonicalEdgeKey(fromId, toId)
            val edge = seen.getOrPut(edgeKey) { EdgeDirection() }
            if (fromId < toId) {
                if (edge.reverse) return true
                edge.forward = true
            } else {
                if (edge.forward) return true
                edge.reverse = true
            }
        }

        return false
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
