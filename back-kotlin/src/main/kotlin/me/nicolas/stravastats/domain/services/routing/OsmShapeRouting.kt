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

private val shapeJsonMapper = JsonMapper.builder()
    .addModule(KotlinModule.Builder().build())
    .build()

internal fun activityTypeFromRouteType(routeType: String?): ActivityType {
    return when (routeType.orEmpty().trim().uppercase(Locale.getDefault())) {
        "RUN" -> ActivityType.Run
        "TRAIL" -> ActivityType.TrailRun
        "HIKE" -> ActivityType.Hike
        "MTB" -> ActivityType.MountainBikeRide
        "GRAVEL" -> ActivityType.GravelRide
        else -> ActivityType.Ride
    }
}

internal fun destinationFromBearing(start: Coordinates, distanceKm: Double, bearingDegrees: Double): Coordinates {
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

internal fun normalizeBearing(value: Double): Double {
    var normalized = value % 360.0
    if (normalized < 0) normalized += 360.0
    return normalized
}

internal fun startDirectionToBearing(direction: String?): Double {
    return when (direction.orEmpty().trim().uppercase(Locale.getDefault())) {
        "N" -> 0.0
        "E" -> 90.0
        "S" -> 180.0
        "W" -> 270.0
        else -> 0.0
    }
}

internal fun normalizeLongitude(value: Double): Double {
    var normalized = value
    while (normalized < -180.0) normalized += 360.0
    while (normalized > 180.0) normalized -= 360.0
    return normalized
}

internal fun generatedRouteId(points: List<List<Double>>, start: Coordinates, index: Int): String {
    val step = if (points.size > 40) max(1, points.size / 40) else 1
    val signature = buildString {
        append("%.5f|%.5f|%d|".format(Locale.US, start.lat, start.lng, index))
        points.indices.step(step).forEach { idx ->
            append("%.5f,%.5f|".format(Locale.US, points[idx][0], points[idx][1]))
        }
    }
    return "generated-osm-${signature.hashCode().toUInt().toString(16)}"
}

internal fun generatedEditedRouteId(points: List<List<Double>>, sourceRouteId: String): String {
    val step = if (points.size > 40) max(1, points.size / 40) else 1
    val signature = buildString {
        append("edit|")
        append(sourceRouteId.trim())
        append("|")
        points.indices.step(step).forEach { idx ->
            append("%.5f,%.5f|".format(Locale.US, points[idx][0], points[idx][1]))
        }
    }
    return "edited-osm-${signature.hashCode().toUInt().toString(16)}"
}

internal fun parseShapePolylineCoordinates(raw: String): List<Coordinates> {
    val trimmed = raw.trim()
    if (trimmed.isEmpty()) return emptyList()

    var points = runCatching { shapeJsonMapper.readValue<List<List<Double>>>(trimmed) }.getOrElse { emptyList() }
    if (points.isEmpty()) {
        val wrapped = runCatching {
            shapeJsonMapper.readValue<Map<String, List<List<Double>>>>(trimmed)
        }.getOrNull()
        points = wrapped?.get("points")
            ?: wrapped?.get("coordinates")
            ?: wrapped?.get("latLng")
            ?: emptyList()
    }

    if (points.isEmpty()) {
        val fromGpx = parseShapeCoordinatesFromGpx(trimmed)
        if (fromGpx.isNotEmpty()) {
            return fromGpx
        }
        val encodedPolyline = runCatching { shapeJsonMapper.readValue<String>(trimmed).trim() }
            .getOrElse { trimmed }
        val decoded = decodeEncodedPolylineCoordinatesToCoordinates(encodedPolyline)
        if (decoded.isNotEmpty()) {
            return decoded
        }
    }

    return points.mapNotNull { point ->
        if (point.size < 2) return@mapNotNull null
        val lat = point[0]
        val lng = point[1]
        if (lat !in -90.0..90.0 || lng !in -180.0..180.0) return@mapNotNull null
        Coordinates(lat = lat, lng = lng)
    }
}

internal fun parseShapeCoordinatesFromGpx(raw: String): List<Coordinates> {
    val pointRegex = Regex("""<(?:trkpt|rtept|wpt)\b([^>]*)>""", setOf(RegexOption.IGNORE_CASE, RegexOption.DOT_MATCHES_ALL))
    val latRegex = Regex("""\blat\s*=\s*["']([^"']+)["']""", RegexOption.IGNORE_CASE)
    val lonRegex = Regex("""\blon\s*=\s*["']([^"']+)["']""", RegexOption.IGNORE_CASE)

    val points = mutableListOf<Coordinates>()
    pointRegex.findAll(raw).forEach { match ->
        val attributes = match.groupValues.getOrNull(1).orEmpty()
        val latText = latRegex.find(attributes)?.groupValues?.getOrNull(1)?.trim() ?: return@forEach
        val lonText = lonRegex.find(attributes)?.groupValues?.getOrNull(1)?.trim() ?: return@forEach
        val lat = latText.toDoubleOrNull() ?: return@forEach
        val lon = lonText.toDoubleOrNull() ?: return@forEach
        if (lat in -90.0..90.0 && lon in -180.0..180.0) {
            points += Coordinates(lat = lat, lng = lon)
        }
    }
    return points
}

internal fun decodeEncodedPolylineCoordinatesToCoordinates(encodedPolyline: String): List<Coordinates> {
    val points = decodeEncodedPolylineCoordinates(encodedPolyline) ?: return emptyList()
    return points.mapNotNull { point ->
        if (point.size < 2) return@mapNotNull null
        val lat = point[0]
        val lng = point[1]
        if (lat !in -90.0..90.0 || lng !in -180.0..180.0) return@mapNotNull null
        Coordinates(lat = lat, lng = lng)
    }
}

internal fun decodeEncodedPolylineCoordinates(encodedPolyline: String): List<List<Double>>? {
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

internal fun decodePolylineDelta(encoded: String, startIndex: Int): Pair<Int, Int>? {
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

internal fun polylineDistanceKmFromCoordinates(points: List<Coordinates>): Double {
    if (points.size < 2) return 0.0
    var totalMeters = 0.0
    for (index in 0 until points.size - 1) {
        val left = points[index]
        val right = points[index + 1]
        totalMeters += osmHaversineDistanceMeters(left.lat, left.lng, right.lat, right.lng)
    }
    return totalMeters / 1000.0
}

internal fun projectShapePolylineToStart(
    shape: List<Coordinates>,
    start: Coordinates,
    targetDistanceKm: Double,
): List<Coordinates> {
    if (shape.isEmpty()) return emptyList()

    val (shapeCenter, shapeRadiusMeters) = shapeCenterAndRadius(shape)
    var scaleAnchor = shapeCenter
    var projectedBase = shape.map { point -> Coordinates(lat = point.lat, lng = point.lng) }
    if (!preserveGeoreferencedShapePlacement(shape, start, shapeCenter, shapeRadiusMeters)) {
        val deltaLat = start.lat - shapeCenter.lat
        val deltaLng = start.lng - shapeCenter.lng
        projectedBase = shape.map { point ->
            Coordinates(
                lat = point.lat + deltaLat,
                lng = point.lng + deltaLng,
            )
        }
        scaleAnchor = start
    }

    var scale = 1.0
    val shapeDistanceKm = polylineDistanceKmFromCoordinates(projectedBase)
    if (targetDistanceKm > 0.0 && shapeDistanceKm > 0.0) {
        scale = (targetDistanceKm / shapeDistanceKm).coerceIn(0.45, 2.60)
    }

    return projectedBase.map { point ->
        Coordinates(
            lat = scaleAnchor.lat + (point.lat - scaleAnchor.lat) * scale,
            lng = scaleAnchor.lng + (point.lng - scaleAnchor.lng) * scale,
        )
    }
}

internal fun shapeCenterAndRadius(points: List<Coordinates>): Pair<Coordinates, Double> {
    if (points.isEmpty()) {
        return Coordinates(lat = 0.0, lng = 0.0) to 0.0
    }
    val minLat = points.minOf { it.lat }
    val maxLat = points.maxOf { it.lat }
    val minLng = points.minOf { it.lng }
    val maxLng = points.maxOf { it.lng }
    val center = Coordinates(
        lat = (minLat + maxLat) / 2.0,
        lng = (minLng + maxLng) / 2.0,
    )
    val radiusMeters = points.maxOf { point ->
        osmHaversineDistanceMeters(center.lat, center.lng, point.lat, point.lng)
    }
    return center to radiusMeters
}

internal fun preserveGeoreferencedShapePlacement(
    shape: List<Coordinates>,
    start: Coordinates,
    center: Coordinates,
    radiusMeters: Double,
): Boolean {
    val centerDistanceMeters = osmHaversineDistanceMeters(start.lat, start.lng, center.lat, center.lng)
    if (centerDistanceMeters <= max(900.0, radiusMeters * 1.35)) {
        return true
    }
    val nearestPointMeters = shape.minOfOrNull { point ->
        osmHaversineDistanceMeters(start.lat, start.lng, point.lat, point.lng)
    } ?: Double.MAX_VALUE
    return nearestPointMeters <= max(500.0, radiusMeters * 0.35)
}

internal fun buildShapeRoutingVariants(
    shape: List<Coordinates>,
    preferredStart: Coordinates,
): List<ShapeRoutingVariant> {
    if (shape.size < 2) return emptyList()
    val primary = prepareShapeForRouting(shape, preferredStart)
    val variants = mutableListOf<ShapeRoutingVariant>()
    val seenShapes = mutableSetOf<String>()

    fun addVariant(label: String, points: List<Coordinates>) {
        if (points.size < 2) return
        val key = shapeVariantSignature(points)
        if (key.isBlank() || !seenShapes.add(key)) return
        variants += ShapeRoutingVariant(
            label = label,
            shape = points.map { point -> Coordinates(lat = point.lat, lng = point.lng) },
        )
    }

    addVariant("", primary)
    listOf(
        Triple("scale 0.55x", 0.55, 0.0),
        Triple("scale 0.70x", 0.70, 0.0),
        Triple("rotate -12 deg", 1.0, -12.0),
        Triple("rotate 12 deg", 1.0, 12.0),
        Triple("scale 0.85x", 0.85, 0.0),
        Triple("scale 1.15x", 1.15, 0.0),
        Triple("rotate -24 deg", 1.0, -24.0),
        Triple("rotate 24 deg", 1.0, 24.0),
        Triple("scale 1.30x", 1.30, 0.0),
        Triple("rotate -36 deg", 1.0, -36.0),
        Triple("rotate 36 deg", 1.0, 36.0),
    ).forEach { (label, scale, rotationDegrees) ->
        addVariant(label, transformShapePose(primary, scale, rotationDegrees, 0.0, 0.0))
    }
    val (_, radiusMeters) = shapeCenterAndRadius(primary)
    val shiftKm = min(0.45, max(0.18, radiusMeters / 1000.0 * 0.18))
    addVariant("shift north ${"%.2f".format(Locale.US, shiftKm)}km", transformShapePose(primary, 1.0, 0.0, shiftKm, 0.0))
    addVariant("shift east ${"%.2f".format(Locale.US, shiftKm)}km", transformShapePose(primary, 1.0, 0.0, shiftKm, 90.0))
    addVariant("shift south ${"%.2f".format(Locale.US, shiftKm)}km", transformShapePose(primary, 1.0, 0.0, shiftKm, 180.0))
    addVariant("shift west ${"%.2f".format(Locale.US, shiftKm)}km", transformShapePose(primary, 1.0, 0.0, shiftKm, 270.0))
    return variants
}

internal fun shapeVariantSignature(points: List<Coordinates>): String {
    if (points.size < 2) return ""
    return sampleCoordinates(points, 10).joinToString("|") { point ->
        "%.5f,%.5f".format(Locale.US, point.lat, point.lng)
    }
}

internal fun transformShapePose(
    points: List<Coordinates>,
    scale: Double,
    rotationDegrees: Double,
    shiftKm: Double,
    shiftBearingDegrees: Double,
): List<Coordinates> {
    if (points.size < 2) {
        return points.map { point -> Coordinates(lat = point.lat, lng = point.lng) }
    }
    val (center, _) = shapeCenterAndRadius(points)
    val cosLat = cos(Math.toRadians(center.lat))
    if (abs(cosLat) < 0.000001) {
        return points.map { point -> Coordinates(lat = point.lat, lng = point.lng) }
    }
    val safeScale = if (scale > 0.0) scale else 1.0
    val rotationRadians = Math.toRadians(rotationDegrees)
    val cosRotation = cos(rotationRadians)
    val sinRotation = sin(rotationRadians)
    val shiftMeters = shiftKm * 1000.0
    val shiftBearingRadians = Math.toRadians(shiftBearingDegrees)
    val shiftX = sin(shiftBearingRadians) * shiftMeters
    val shiftY = cos(shiftBearingRadians) * shiftMeters
    val transformed = points.map { point ->
        var x = (point.lng - center.lng) * 111320.0 * cosLat
        var y = (point.lat - center.lat) * 111320.0
        x *= safeScale
        y *= safeScale
        val rotatedX = x * cosRotation - y * sinRotation + shiftX
        val rotatedY = x * sinRotation + y * cosRotation + shiftY
        Coordinates(
            lat = center.lat + rotatedY / 111320.0,
            lng = center.lng + rotatedX / (111320.0 * cosLat),
        )
    }
    return if (transformed.all(::isFiniteCoordinate)) {
        transformed
    } else {
        points.map { point -> Coordinates(lat = point.lat, lng = point.lng) }
    }
}

internal fun isFiniteCoordinate(point: Coordinates): Boolean {
    return point.lat.isFinite() &&
        point.lng.isFinite() &&
        point.lat >= -90.0 &&
        point.lat <= 90.0 &&
        point.lng >= -180.0 &&
        point.lng <= 180.0
}

internal fun prepareShapeForRouting(
    shape: List<Coordinates>,
    preferredStart: Coordinates,
): List<Coordinates> {
    return shape.map { point -> Coordinates(lat = point.lat, lng = point.lng) }
}

internal fun polylineDistanceKmFromLatLng(points: List<List<Double>>): Double {
    if (points.size < 2) return 0.0
    var totalMeters = 0.0
    for (index in 0 until points.size - 1) {
        val left = points[index]
        val right = points[index + 1]
        if (left.size < 2 || right.size < 2) {
            continue
        }
        totalMeters += osmHaversineDistanceMeters(left[0], left[1], right[0], right[1])
    }
    return totalMeters / 1000.0
}

internal fun shapeLengthSimilarityScore(routeLengthKm: Double, shapeLengthKm: Double): Double {
    if (routeLengthKm <= 0.0 || shapeLengthKm <= 0.0) {
        return 0.0
    }
    val deltaRatio = abs(routeLengthKm - shapeLengthKm) / max(routeLengthKm, shapeLengthKm)
    return osmClampUnit(1.0 - deltaRatio * 1.35)
}

internal fun sampleCoordinates(points: List<Coordinates>, maxPoints: Int): List<Coordinates> {
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

internal fun sampleCoordinatesByDistance(
    points: List<Coordinates>,
    maxPoints: Int,
    minSpacingMeters: Double,
): List<Coordinates> {
    if (points.size <= 2 || maxPoints <= 0) return points.toList()
    val spacing = minSpacingMeters.coerceAtLeast(1.0)
    val segmentLengths = points.zipWithNext().map { (left, right) ->
        osmHaversineDistanceMeters(left.lat, left.lng, right.lat, right.lng)
    }
    val totalMeters = segmentLengths.sum()
    if (totalMeters <= 0.0) return points.take(1)

    val targetCount = (kotlin.math.floor(totalMeters / spacing).toInt() + 1)
        .coerceAtLeast(2)
        .coerceAtMost(maxPoints)
    if (targetCount >= points.size) return points.toList()

    val intervalMeters = totalMeters / (targetCount - 1).toDouble()
    val sampled = mutableListOf(points.first())
    var nextDistance = intervalMeters
    var traversed = 0.0
    var segmentIndex = 0
    while (sampled.size < targetCount - 1 && segmentIndex < segmentLengths.size) {
        val segmentLength = segmentLengths[segmentIndex]
        if (segmentLength <= 0.0) {
            segmentIndex += 1
            continue
        }
        if (traversed + segmentLength < nextDistance) {
            traversed += segmentLength
            segmentIndex += 1
            continue
        }
        val t = ((nextDistance - traversed) / segmentLength).coerceIn(0.0, 1.0)
        val start = points[segmentIndex]
        val end = points[segmentIndex + 1]
        sampled += Coordinates(
            lat = start.lat + (end.lat - start.lat) * t,
            lng = start.lng + (end.lng - start.lng) * t,
        )
        nextDistance += intervalMeters
    }
    val last = points.last()
    if (sampled.isEmpty() || osmHaversineDistanceMeters(sampled.last().lat, sampled.last().lng, last.lat, last.lng) > 1.0) {
        sampled += last
    }
    return sampled
}

internal fun buildShapeLoopWaypoints(start: Coordinates, shape: List<Coordinates>): List<Coordinates> {
    val sampled = sampleCoordinates(shape, 18)
    val waypoints = mutableListOf(start)
    var previous = start
    for (point in sampled) {
        if (osmHaversineDistanceMeters(previous.lat, previous.lng, point.lat, point.lng) < 80.0) {
            continue
        }
        waypoints += point
        previous = point
    }
    return appendShapeEndWaypoint(waypoints, shape)
}

internal fun buildShapeDenseWaypoints(start: Coordinates, shape: List<Coordinates>): List<Coordinates> {
    if (shape.size < 2) return emptyList()
    val sampled = sampleCoordinates(shape, 28)
    val waypoints = mutableListOf(start)
    var previous = start
    sampled.forEach { point ->
        if (osmHaversineDistanceMeters(previous.lat, previous.lng, point.lat, point.lng) < 60.0) {
            return@forEach
        }
        waypoints += point
        previous = point
    }
    if (waypoints.size < 3) {
        return buildShapeLoopWaypoints(start, shape)
    }
    return appendShapeEndWaypoint(waypoints, shape)
}

internal fun buildShapeFidelityStitchedWaypoints(start: Coordinates, shape: List<Coordinates>): List<Coordinates> {
    if (shape.size < 2) return emptyList()
    val sampled = sampleCoordinatesByDistance(shape, maxPoints = 26, minSpacingMeters = 90.0)
    val waypoints = mutableListOf(start)
    var previous = start
    sampled.forEach { point ->
        if (osmHaversineDistanceMeters(previous.lat, previous.lng, point.lat, point.lng) < 55.0) {
            return@forEach
        }
        waypoints += point
        previous = point
    }
    if (waypoints.size < 3) {
        return buildShapeStitchedWaypoints(start, shape)
    }
    return appendShapeEndWaypoint(waypoints, shape)
}

internal fun buildShapeStitchedWaypoints(start: Coordinates, shape: List<Coordinates>): List<Coordinates> {
    if (shape.size < 2) return emptyList()
    val sampled = sampleCoordinates(shape, 14)
    val waypoints = mutableListOf(start)
    var previous = start
    sampled.forEach { point ->
        if (osmHaversineDistanceMeters(previous.lat, previous.lng, point.lat, point.lng) < 120.0) {
            return@forEach
        }
        waypoints += point
        previous = point
    }
    if (waypoints.size < 3) {
        return buildShapeSimplifiedWaypoints(start, shape)
    }
    return appendShapeEndWaypoint(waypoints, shape)
}

internal fun buildShapeSimplifiedWaypoints(start: Coordinates, shape: List<Coordinates>): List<Coordinates> {
    if (shape.size < 2) return emptyList()
    val sampled = sampleCoordinates(shape, 12)
    val waypoints = mutableListOf(start)
    var previous = start
    sampled.forEach { point ->
        if (osmHaversineDistanceMeters(previous.lat, previous.lng, point.lat, point.lng) < 160.0) {
            return@forEach
        }
        waypoints += point
        previous = point
    }
    if (waypoints.size < 3) {
        return buildShapeLoopWaypoints(start, shape)
    }
    return appendShapeEndWaypoint(waypoints, shape)
}

internal fun buildShapeRoadFirstWaypoints(start: Coordinates, shape: List<Coordinates>): List<Coordinates> {
    if (shape.size < 2) return emptyList()
    val sampled = sampleCoordinates(shape, 20)
    if (sampled.size < 2) return emptyList()

    data class IndexedPoint(
        val index: Int,
        val point: Coordinates,
        val distance: Double,
    )

    val scored = sampled
        .mapIndexedNotNull { index, point ->
            val distance = osmHaversineDistanceMeters(start.lat, start.lng, point.lat, point.lng)
            if (distance < 280.0) return@mapIndexedNotNull null
            IndexedPoint(index = index, point = point, distance = distance)
        }
        .sortedWith(
            compareByDescending<IndexedPoint> { it.distance }
                .thenBy { it.index }
        )
        .take(8)
        .sortedBy { it.index }

    if (scored.isEmpty()) {
        return buildShapeLoopWaypoints(start, shape)
    }

    val waypoints = mutableListOf(start)
    var previous = start
    scored.forEach { entry ->
        if (osmHaversineDistanceMeters(previous.lat, previous.lng, entry.point.lat, entry.point.lng) < 180.0) {
            return@forEach
        }
        waypoints += entry.point
        previous = entry.point
    }

    if (waypoints.size < 3) {
        return buildShapeLoopWaypoints(start, shape)
    }
    return appendShapeEndWaypoint(waypoints, shape)
}

internal data class ShapeBestEffortRoutingStrategy(
    val code: String,
    val label: String,
    val waypoints: List<Coordinates>,
    val bestEffort: Boolean = true,
)

internal fun buildShapeBestEffortRoutingStrategies(
    start: Coordinates,
    shape: List<Coordinates>,
): List<ShapeBestEffortRoutingStrategy> {
    return buildList {
        val simplified = buildShapeBestEffortWaypoints(start, shape)
        if (simplified.size >= 3) {
            add(
                ShapeBestEffortRoutingStrategy(
                    code = SHAPE_MODE_STRATEGY_BEST_EFFORT,
                    label = "simplified sketch fallback",
                    waypoints = simplified,
                )
            )
        }
        val envelope = buildShapeEnvelopeWaypoints(start, shape)
        if (envelope.size >= 3) {
            add(
                ShapeBestEffortRoutingStrategy(
                    code = SHAPE_MODE_STRATEGY_BEST_EFFORT,
                    label = "shape envelope fallback",
                    waypoints = envelope,
                )
            )
        }
    }
}

internal fun buildShapeBestEffortWaypoints(start: Coordinates, shape: List<Coordinates>): List<Coordinates> {
    val sampled = sampleCoordinates(shape, 8)
    val waypoints = mutableListOf(start)
    var previous = start
    sampled.forEach { point ->
        if (osmHaversineDistanceMeters(previous.lat, previous.lng, point.lat, point.lng) < 220.0) {
            return@forEach
        }
        waypoints += point
        previous = point
    }
    return appendShapeEndWaypoint(waypoints, shape)
}

internal fun buildShapeEnvelopeWaypoints(start: Coordinates, shape: List<Coordinates>): List<Coordinates> {
    if (shape.size < 2) {
        return emptyList()
    }
    val (center, radiusMeters) = shapeCenterAndRadius(shape)
    val radiusKm = (radiusMeters / 1000.0).coerceIn(0.55, 5.0)
    val waypoints = mutableListOf(start)
    var previous = start
    listOf(0.0, 90.0, 180.0, 270.0).forEach { bearing ->
        val point = destinationFromBearing(center, radiusKm, bearing)
        if (osmHaversineDistanceMeters(previous.lat, previous.lng, point.lat, point.lng) < 220.0) {
            return@forEach
        }
        waypoints += point
        previous = point
    }
    return appendShapeEndWaypoint(waypoints, shape)
}

internal fun appendShapeEndWaypoint(
    waypoints: MutableList<Coordinates>,
    shape: List<Coordinates>,
): List<Coordinates> {
    if (waypoints.isEmpty()) {
        return shape.map { point -> Coordinates(lat = point.lat, lng = point.lng) }
    }
    val end = shape.lastOrNull() ?: return waypoints
    val last = waypoints.last()
    if (osmHaversineDistanceMeters(last.lat, last.lng, end.lat, end.lng) > 80.0) {
        waypoints += end
    }
    return waypoints
}

internal fun coordinatesToLatLng(points: List<Coordinates>): List<List<Double>> {
    return points.map { point -> listOf(point.lat, point.lng) }
}
