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

internal fun computeSurfaceBreakdown(route: OsrmRoute): RouteSurfaceBreakdown {
    var pavedM = 0.0
    var gravelM = 0.0
    var trailM = 0.0
    var unknownM = 0.0

    route.legs.forEach { leg ->
        leg.steps.forEach { step ->
            val distance = max(0.0, step.distance)
            if (distance <= 0.0) return@forEach
            when (classifySurfaceBucket(step)) {
                "paved" -> pavedM += distance
                "gravel" -> gravelM += distance
                "trail" -> trailM += distance
                else -> unknownM += distance
            }
        }
    }

    if (pavedM + gravelM + trailM + unknownM <= 0.0 && route.distance > 0.0) {
        unknownM = route.distance
    }

    return RouteSurfaceBreakdown(
        pavedM = pavedM,
        gravelM = gravelM,
        trailM = trailM,
        unknownM = unknownM,
    )
}

internal fun mergeSurfaceBreakdowns(left: RouteSurfaceBreakdown, right: RouteSurfaceBreakdown): RouteSurfaceBreakdown {
    return RouteSurfaceBreakdown(
        pavedM = left.pavedM + right.pavedM,
        gravelM = left.gravelM + right.gravelM,
        trailM = left.trailM + right.trailM,
        unknownM = left.unknownM + right.unknownM,
    )
}

internal fun classifySurfaceBucket(step: OsrmStep): String {
    val mode = step.mode.orEmpty().trim().lowercase(Locale.getDefault())
    if (mode.contains("pushing") || mode == "foot" || mode == "walking") {
        return "trail"
    }
    val classes = step.classes
        .asSequence()
        .map { normalizeClassToken(it) }
        .filter { it.isNotBlank() }
        .toSet()

    if (classes.contains("ferry")) {
        return "unknown"
    }
    val surfaceValue = normalizeTagValue(step.surface, "surface")
        .ifBlank { extractTagValueFromClasses(step.classes, "surface") }
    surfaceBucketFromSurfaceTag(surfaceValue)?.let { return it }

    val trackTypeValue = normalizeTagValue(step.tracktype, "tracktype")
        .ifBlank { extractTagValueFromClasses(step.classes, "tracktype") }
    surfaceBucketFromTrackType(trackTypeValue)?.let { return it }

    if (hasAnyClass(classes, "path", "track", "steps", "bridleway", "cycleway_unpaved")) {
        return "trail"
    }
    if (
        hasAnyClass(
            classes,
            "tracktype_grade1", "tracktype=grade1", "tracktype:grade1",
            "grade1",
            "asphalt", "paved", "concrete", "concrete:lanes", "concrete:plates",
            "paving_stones", "sett", "cobblestone", "metal", "wood",
        )
    ) {
        return "paved"
    }
    if (
        hasAnyClass(
            classes,
            "tracktype_grade2", "tracktype=grade2", "tracktype:grade2",
            "tracktype_grade3", "tracktype=grade3", "tracktype:grade3",
            "grade2", "grade3",
        )
    ) {
        return "gravel"
    }
    if (
        hasAnyClass(
            classes,
            "tracktype_grade4", "tracktype=grade4", "tracktype:grade4",
            "tracktype_grade5", "tracktype=grade5", "tracktype:grade5",
            "grade4", "grade5",
        )
    ) {
        return "trail"
    }
    if (hasAnyClass(classes, "unpaved", "gravel", "dirt", "ground", "earth", "compacted", "fine_gravel", "sand", "mud")) {
        return "gravel"
    }
    if (mode == "cycling" || mode == "driving" || mode == "running") {
        return "paved"
    }
    return "unknown"
}

internal fun hasAnyClass(classes: Set<String>, vararg keys: String): Boolean {
    return keys.any { key -> classes.contains(normalizeClassToken(key)) }
}

internal fun normalizeClassToken(raw: String?): String {
    return raw.orEmpty().trim().lowercase(Locale.getDefault()).replace(" ", "_")
}

internal fun normalizeTagValue(raw: String?, key: String): String {
    val normalized = normalizeClassToken(raw)
    if (normalized.isBlank()) {
        return ""
    }
    val keyNormalized = normalizeClassToken(key)
    if (keyNormalized.isBlank()) {
        return normalized
    }
    val prefixes = listOf(
        "$keyNormalized=",
        "$keyNormalized:",
        "${keyNormalized}_",
        "$keyNormalized-",
    )
    for (prefix in prefixes) {
        if (normalized.startsWith(prefix) && normalized.length > prefix.length) {
            return normalized.substring(prefix.length).trim('_', '-', ':')
        }
    }
    return normalized
}

internal fun extractTagValueFromClasses(rawClasses: List<String>, key: String): String {
    val keyNormalized = normalizeClassToken(key)
    if (keyNormalized.isBlank()) {
        return ""
    }
    val prefixes = listOf(
        "$keyNormalized=",
        "$keyNormalized:",
        "${keyNormalized}_",
        "$keyNormalized-",
    )
    for (rawClass in rawClasses) {
        val normalized = normalizeClassToken(rawClass)
        if (normalized.isBlank()) continue
        for (prefix in prefixes) {
            if (normalized.startsWith(prefix) && normalized.length > prefix.length) {
                return normalized.substring(prefix.length).trim('_', '-', ':')
            }
        }
    }
    return ""
}

internal fun surfaceBucketFromSurfaceTag(surface: String): String? {
    return when (normalizeTagValue(surface, "surface")) {
        "" -> null
        "asphalt", "paved", "concrete", "concrete_lanes", "concrete_plates",
        "concrete:lanes", "concrete:plates", "paving_stones", "sett",
        "cobblestone", "metal", "wood", "chipseal" -> "paved"
        "unpaved", "gravel", "fine_gravel", "compacted", "dirt",
        "ground", "earth", "pebblestone", "sand", "mud", "clay" -> "gravel"
        "path", "trail", "steps", "grass", "woodchips" -> "trail"
        else -> null
    }
}

internal fun surfaceBucketFromTrackType(trackType: String): String? {
    return when (normalizeTagValue(trackType, "tracktype")) {
        "" -> null
        "grade1" -> "paved"
        "grade2", "grade3" -> "gravel"
        "grade4", "grade5" -> "trail"
        else -> null
    }
}

internal fun formatSurfaceBreakdown(breakdown: RouteSurfaceBreakdown): String {
    val (paved, gravel, trail, unknown) = breakdown.normalizedRatios()
    return "paved ${(paved * 100.0).roundToInt()}%, " +
        "gravel ${(gravel * 100.0).roundToInt()}%, " +
        "trail ${(trail * 100.0).roundToInt()}%, " +
        "unknown ${(unknown * 100.0).roundToInt()}%"
}

internal fun surfaceMatchScore(routeType: String?, breakdown: RouteSurfaceBreakdown): Double {
    val (paved, gravel, trail, unknown) = breakdown.normalizedRatios()
    val pathRatio = osmClampUnit(gravel + trail)
    var targetPaved = 0.60
    var targetGravel = 0.25
    var targetTrail = 0.15

    when (routeType.orEmpty().trim().uppercase(Locale.getDefault())) {
        "RIDE" -> {
            targetPaved = 0.92
            targetGravel = 0.06
            targetTrail = 0.02
        }
        "GRAVEL" -> {
            // Gravel contract:
            // - minimum 25% paths (gravel + trail)
            // - no hard upper bound once this minimum is reached
            val shortfall = max(0.0, 0.25 - pathRatio)
            val pavedExcess = max(0.0, paved - 0.75)
            val penalty = shortfall * 220.0 + pavedExcess * 36.0 + unknown * 22.0
            return clampScore(100.0 - penalty)
        }
        "MTB" -> {
            // MTB should prefer paths as much as possible.
            val pavedExcess = max(0.0, paved - 0.20)
            val score = 28.0 + pathRatio * 74.0 - unknown * 24.0 - pavedExcess * 48.0
            return clampScore(score)
        }
        "RUN" -> {
            targetPaved = 0.50
            targetGravel = 0.25
            targetTrail = 0.25
        }
        "TRAIL", "HIKE" -> {
            targetPaved = 0.12
            targetGravel = 0.28
            targetTrail = 0.60
        }
    }

    val penalty = abs(paved - targetPaved) * 85.0 +
        abs(gravel - targetGravel) * 78.0 +
        abs(trail - targetTrail) * 92.0 +
        unknown * 35.0
    return clampScore(100.0 - penalty)
}

internal fun surfaceScoreWeight(routeType: String?): Double {
    return when (routeType.orEmpty().trim().uppercase(Locale.getDefault())) {
        "RIDE" -> 1.10
        "GRAVEL" -> 1.25
        "MTB" -> 1.70
        "TRAIL", "HIKE" -> 1.40
        else -> 0.45
    }
}

internal fun pathPreferenceBonus(routeType: String?, pathRatio: Double): Double {
    return when (routeType.orEmpty().trim().uppercase(Locale.getDefault())) {
        "RIDE" -> {
            // Road rides should avoid off-road sections as much as possible.
            (0.10 - pathRatio) * 35.0
        }
        "MTB" -> {
            // Strongly reward path-heavy candidates for MTB.
            (pathRatio - 0.50) * 60.0
        }
        "GRAVEL" -> {
            // Encourage higher path ratio once the 25% minimum is reached.
            (pathRatio - 0.25) * 30.0
        }
        else -> 0.0
    }
}

internal fun minSegmentDiversityRatio(routeType: String?): Double {
    return when (routeType.orEmpty().trim().uppercase(Locale.getDefault())) {
        "MTB" -> 0.55
        "GRAVEL" -> 0.54
        "RUN" -> 0.35
        "TRAIL" -> 0.46
        "HIKE" -> 0.40
        "WALK" -> 0.42
        else -> 0.32
    }
}

internal fun segmentDiversityRatio(points: List<List<Double>>): Double {
    return evaluateAxisUsage(points).segmentDiversityRatio()
}

internal fun distanceShortfallRatio(distanceKm: Double, targetKm: Double): Double {
    if (targetKm <= 0.0) {
        return 0.0
    }
    val shortfall = targetKm - distanceKm
    if (shortfall <= 0.0) {
        return 0.0
    }
    return shortfall / max(1.0, targetKm)
}

internal fun distanceOvershootRatio(distanceKm: Double, targetKm: Double): Double {
    if (targetKm <= 0.0) {
        return 0.0
    }
    val overshoot = distanceKm - targetKm
    if (overshoot <= 0.0) {
        return 0.0
    }
    return overshoot / max(1.0, targetKm)
}

internal fun outsideStartAxisReuseLimit(routeType: String?, strict: Boolean): Int {
    // P0-02 policy: outside start/finish zone, an axis cannot be reused.
    return 1
}

internal fun allowedOppositeOutsideStartRatio(routeType: String?, strict: Boolean): Double {
    // P0-02 policy: opposite-direction overlap is forbidden outside start zone.
    return 0.0
}

internal fun minimumOppositeReuseMetersForRequest(
    routeType: String?,
    strict: Boolean,
    distanceTargetKm: Double,
): Double {
    val base = max(MIN_OPPOSITE_REUSE_METERS, distanceTargetKm * 6.0)
    return when (routeType.orEmpty().trim().uppercase(Locale.getDefault())) {
        "MTB", "TRAIL", "HIKE" -> max(base, 320.0)
        "GRAVEL" -> max(base, 280.0)
        else -> max(base, 240.0)
    }
}

internal fun requiredPathRatioForRequest(routeType: String?, strict: Boolean): Double {
    val normalized = routeType.orEmpty().trim().uppercase(Locale.getDefault())
    if (normalized != "GRAVEL") {
        return 0.0
    }
    // Gravel contract: keep a 25% path target; fallback to Ride handles impossible cases.
    return 0.25
}

internal fun meetsMinimumDistance(distanceKm: Double, targetKm: Double): Boolean {
    if (targetKm <= 0.0) {
        return true
    }
    // Keep a small tolerance for geometry simplification / snapping noise.
    val toleranceKm = max(0.25, targetKm * 0.02)
    return distanceKm + toleranceKm >= targetKm
}

internal fun fallbackRouteTypes(routeType: String?): List<String> {
    return when (routeType.orEmpty().trim().uppercase(Locale.getDefault())) {
        "MTB" -> listOf("GRAVEL", "RIDE")
        "GRAVEL" -> listOf("RIDE")
        "RIDE" -> emptyList()
        else -> listOf("RIDE")
    }
}

internal fun computeOsmMatchScore(
    request: RoutingEngineRequest,
    distanceKm: Double,
    elevationGainM: Double,
    points: List<List<Double>>,
): Double {
    val hasElevationTarget = (request.elevationTargetM ?: 0.0) > 0.0
    val hasDirection = !request.startDirection.isNullOrBlank()
    val profile = buildOsmScoringProfile(request.routeType, hasElevationTarget, hasDirection)

    val distanceComponent = distanceShortfallRatio(distanceKm, request.distanceTargetKm) +
        distanceOvershootRatio(distanceKm, request.distanceTargetKm) * 0.15
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

internal fun buildOsmScoringProfile(
    routeType: String?,
    hasElevationTarget: Boolean,
    hasDirection: Boolean,
): OsmScoringProfile {
    var profile = when (routeType.orEmpty().trim().uppercase(Locale.getDefault())) {
        "MTB" -> OsmScoringProfile(distanceWeight = 0.36, elevationWeight = 0.29, directionWeight = 0.07, diversityWeight = 0.28)
        "GRAVEL" -> OsmScoringProfile(distanceWeight = 0.44, elevationWeight = 0.26, directionWeight = 0.06, diversityWeight = 0.24)
        "RUN" -> OsmScoringProfile(distanceWeight = 0.56, elevationWeight = 0.17, directionWeight = 0.13, diversityWeight = 0.14)
        "TRAIL" -> OsmScoringProfile(distanceWeight = 0.34, elevationWeight = 0.28, directionWeight = 0.10, diversityWeight = 0.28)
        "HIKE" -> OsmScoringProfile(distanceWeight = 0.30, elevationWeight = 0.35, directionWeight = 0.09, diversityWeight = 0.26)
        "WALK" -> OsmScoringProfile(distanceWeight = 0.33, elevationWeight = 0.28, directionWeight = 0.10, diversityWeight = 0.29)
        else -> OsmScoringProfile(distanceWeight = 0.70, elevationWeight = 0.22, directionWeight = 0.06, diversityWeight = 0.02)
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

internal fun normalizeOsmScoringProfile(profile: OsmScoringProfile): OsmScoringProfile {
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

internal fun directionPenaltyFromPreview(points: List<List<Double>>, startDirection: String?): Double {
    val initialBearing = initialBearingFromPreview(points) ?: return 1.0
    val targetBearing = targetBearingFromDirection(startDirection) ?: return 0.0
    val rawDiff = abs(initialBearing - targetBearing)
    val normalizedDiff = if (rawDiff > 180.0) 360.0 - rawDiff else rawDiff
    return normalizedDiff / 180.0
}

internal fun initialBearingFromPreview(points: List<List<Double>>): Double? {
    if (points.size < 2) return null
    val start = points.firstOrNull()?.takeIf { point -> point.size >= 2 } ?: return null
    val startLat = start[0]
    val startLng = start[1]

    for (index in 1 until points.size) {
        val next = points[index]
        if (next.size < 2) continue
        if (osmHaversineDistanceMeters(startLat, startLng, next[0], next[1]) < 35.0) continue
        return bearingDegrees(startLat, startLng, next[0], next[1])
    }

    val fallback = points.lastOrNull()?.takeIf { point -> point.size >= 2 } ?: return null
    return bearingDegrees(startLat, startLng, fallback[0], fallback[1])
}

internal fun targetBearingFromDirection(direction: String?): Double? {
    return when (direction.orEmpty().trim().uppercase(Locale.getDefault())) {
        "N" -> 0.0
        "E" -> 90.0
        "S" -> 180.0
        "W" -> 270.0
        else -> null
    }
}

internal fun bearingDegrees(lat1: Double, lng1: Double, lat2: Double, lng2: Double): Double {
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

internal fun quantizedPointKey(lat: Double, lng: Double): String = "%.5f:%.5f".format(Locale.US, lat, lng)

internal fun canonicalEdgeKey(a: String, b: String): String = if (a < b) "$a|$b" else "$b|$a"

internal fun osmHaversineDistanceMeters(lat1: Double, lng1: Double, lat2: Double, lng2: Double): Double {
    val earthRadiusMeters = 6_371_000.0
    val dLat = Math.toRadians(lat2 - lat1)
    val dLng = Math.toRadians(lng2 - lng1)
    val sinLat = sin(dLat / 2.0)
    val sinLng = sin(dLng / 2.0)
    val a = sinLat * sinLat + cos(Math.toRadians(lat1)) * cos(Math.toRadians(lat2)) * sinLng * sinLng
    val c = 2.0 * atan2(sqrt(a), sqrt(1.0 - a))
    return earthRadiusMeters * c
}
