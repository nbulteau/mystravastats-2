package me.nicolas.stravastats.domain.services

import me.nicolas.stravastats.domain.business.ActivityType
import me.nicolas.stravastats.domain.business.DataQualityExclusion
import me.nicolas.stravastats.domain.business.DataQualityIssue
import me.nicolas.stravastats.domain.business.DataQualityReport
import me.nicolas.stravastats.domain.business.DataQualitySummary
import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.services.activityproviders.IActivityProvider
import org.springframework.stereotype.Service
import tools.jackson.databind.DeserializationFeature
import tools.jackson.databind.json.JsonMapper
import tools.jackson.module.kotlin.KotlinModule
import java.io.File
import java.time.Instant
import java.util.Locale
import kotlin.math.PI
import kotlin.math.abs
import kotlin.math.atan2
import kotlin.math.cos
import kotlin.math.sin
import kotlin.math.sqrt

interface IDataQualityService {
    fun getReport(): DataQualityReport
    fun excludeActivityFromStats(activityId: Long, reason: String?): DataQualityReport
    fun includeActivityInStats(activityId: Long): DataQualityReport
}

@Service
class DataQualityService(
    private val activityProvider: IActivityProvider,
) : IDataQualityService {

    override fun getReport(): DataQualityReport {
        return buildReport()
    }

    override fun excludeActivityFromStats(activityId: Long, reason: String?): DataQualityReport {
        require(activityId > 0) { "activityId must be > 0" }
        val diagnostics = activityProvider.getCacheDiagnostics()
        val provider = diagnostics["provider"]?.toString()?.trim()?.lowercase().orEmpty()
        val activities = activityProvider.getActivitiesByActivityTypeAndYear(ActivityType.values().toSet(), null)
        val activity = activities.firstOrNull { candidate -> candidate.id == activityId }
            ?: throw IllegalArgumentException("activity $activityId not found")
        val exclusionsById = DataQualityExclusionStorage.load(activityProvider).associateBy { exclusion -> exclusion.activityId }.toMutableMap()
        exclusionsById[activityId] = DataQualityExclusion(
            activityId = activityId,
            source = provider.uppercase(),
            activityName = activity.name.trim(),
            activityType = activity.type,
            year = activity.startDateLocal.take(4).ifBlank { activity.startDate.take(4) },
            reason = reason?.trim()?.takeIf { it.isNotEmpty() } ?: "Excluded from statistics after data quality audit.",
            excludedAt = Instant.now().toString(),
        )
        DataQualityExclusionStorage.save(activityProvider, exclusionsById.values.sortedWith(exclusionComparator()))
        return buildReport()
    }

    override fun includeActivityInStats(activityId: Long): DataQualityReport {
        require(activityId > 0) { "activityId must be > 0" }
        val exclusionsById = DataQualityExclusionStorage.load(activityProvider).associateBy { exclusion -> exclusion.activityId }.toMutableMap()
        exclusionsById.remove(activityId)
        DataQualityExclusionStorage.save(activityProvider, exclusionsById.values.sortedWith(exclusionComparator()))
        return buildReport()
    }

    private fun buildReport(): DataQualityReport {
        val diagnostics = activityProvider.getCacheDiagnostics()
        val provider = diagnostics["provider"]?.toString()?.trim()?.lowercase().orEmpty()
        val sourcePath = listOfNotNull(
            diagnostics["fitDirectory"]?.toString(),
            diagnostics["gpxDirectory"]?.toString(),
            diagnostics["cacheRoot"]?.toString(),
            activityProvider.cacheIdentity()?.cacheRoot,
        ).firstOrNull().orEmpty()

        val activities = activityProvider.getActivitiesByActivityTypeAndYear(ActivityType.values().toSet(), null)
        val exclusions = DataQualityExclusionStorage.load(activityProvider)
        val exclusionsById = exclusions.associateBy { exclusion -> exclusion.activityId }
        val issues = activities
            .flatMap { activity -> analyzeActivity(provider, sourcePath, activity) }
            .map { issue -> issue.markExcluded(exclusionsById) }
            .sortedWith(issueComparator())

        return DataQualityReport(
            generatedAt = Instant.now().toString(),
            summary = buildSummary(provider, issues, exclusions),
            issues = issues,
            exclusions = exclusions.sortedWith(exclusionComparator()),
        )
    }

    private fun analyzeActivity(source: String, sourcePath: String, activity: StravaActivity): List<DataQualityIssue> {
        val issues = mutableListOf<DataQualityIssue>()
        if (activity.distance <= 0.0 || !activity.distance.isFinite()) {
            issues += issue(source, sourcePath, activity, "critical", "INVALID_VALUE", "distance", "Activity distance is missing or invalid.", activity.distance.formatRaw(), "Check the source file or exclude the activity from statistics.")
        }
        if (activity.elapsedTime <= 0) {
            issues += issue(source, sourcePath, activity, "critical", "INVALID_VALUE", "elapsed_time", "Elapsed time is missing or invalid.", activity.elapsedTime.toString(), "Check the source file timing data.")
        }
        if (activity.movingTime <= 0) {
            issues += issue(source, sourcePath, activity, "warning", "INVALID_VALUE", "moving_time", "Moving time is missing or zero.", activity.movingTime.toString(), "Use elapsed time as fallback only if the activity has no pauses.")
        }
        if (activity.movingTime > activity.elapsedTime && activity.elapsedTime > 0) {
            issues += issue(source, sourcePath, activity, "warning", "INCONSISTENT_TIME", "moving_time", "Moving time is greater than elapsed time.", "${activity.movingTime} > ${activity.elapsedTime}", "Prefer elapsed time or recompute moving time from stream data.")
        }
        if (!activity.averageSpeed.isFinite()) {
            issues += issue(source, sourcePath, activity, "critical", "INVALID_VALUE", "average_speed", "Average speed is not serializable.", activity.averageSpeed.formatRaw(), "Sanitize NaN/Inf values before exposing the activity.")
        } else if (activity.averageSpeed > speedThreshold(activity.type)) {
            issues += issue(source, sourcePath, activity, "warning", "INVALID_VALUE", "average_speed", "Average speed is unusually high.", "%.1f km/h".format(Locale.US, activity.averageSpeed * 3.6), "Inspect GPS glitches or timing data before trusting speed statistics.")
        }
        val maxSpeed = activity.maxSpeed.toDouble()
        if (!maxSpeed.isFinite()) {
            issues += issue(source, sourcePath, activity, "critical", "INVALID_VALUE", "max_speed", "Max speed is not serializable.", maxSpeed.formatRaw(), "Sanitize NaN/Inf values before exposing the activity.")
        } else if (maxSpeed > speedThreshold(activity.type) * 1.3) {
            issues += issue(source, sourcePath, activity, "warning", "INVALID_VALUE", "max_speed", "Max speed is unusually high.", "%.1f km/h".format(Locale.US, maxSpeed * 3.6), "Inspect GPS glitches before trusting speed records.")
        }
        if (!activity.totalElevationGain.isFinite()) {
            issues += issue(source, sourcePath, activity, "critical", "INVALID_VALUE", "total_elevation_gain", "Elevation gain is not serializable.", activity.totalElevationGain.formatRaw(), "Recompute elevation from altitude stream or SRTM.")
        }

        issues += analyzeDerivedSpeed(source, sourcePath, activity)

        val stream = activity.stream
        if (stream == null) {
            if (source == "strava" && activity.uploadId > 0) {
                issues += issue(source, sourcePath, activity, "info", "MISSING_STREAM", "stream", "Detailed stream is missing from the local cache.", "", "Download missing streams from Strava when API access is available.")
                return issues
            }
            if (source != "fit" && source != "gpx") {
                return issues
            }
            issues += issue(source, sourcePath, activity, "warning", "MISSING_STREAM", "stream", "Activity has no stream data.", "", "Open the source file and verify GPS/time streams are present.")
            return issues
        }

        if (stream.distance.data.isEmpty()) {
            issues += issue(source, sourcePath, activity, "critical", "MISSING_STREAM_FIELD", "stream.distance", "Distance stream field is missing.", "", "Recompute distance from GPS points when possible.")
        }
        if (stream.time.data.isEmpty()) {
            issues += issue(source, sourcePath, activity, "critical", "MISSING_STREAM_FIELD", "stream.time", "Time stream field is missing.", "", "The activity cannot be checked for speed glitches without time data.")
        }
        if (requiresRouteStream(activity) && stream.latlng?.data.isNullOrEmpty()) {
            issues += issue(source, sourcePath, activity, "warning", "MISSING_STREAM_FIELD", "stream.latlng", "GPS trace stream field is missing.", "", "Map and route-based analysis will be unavailable.")
        }
        if (stream.altitude?.data.isNullOrEmpty()) {
            issues += issue(source, sourcePath, activity, "warning", "MISSING_STREAM_FIELD", "stream.altitude", "Altitude stream field is missing.", "", "Use SRTM elevation fallback or recompute D+.")
        }
        if (stream.heartrate == null && activity.averageHeartrate > 0.0) {
            issues += issue(source, sourcePath, activity, "info", "STREAM_DATA_COVERAGE", "stream.heartrate", "Average heart rate exists but heart-rate samples are not available.", activity.averageHeartrate.formatRaw(), "Heart-rate charts will be incomplete.")
        }
        if (stream.watts == null && activity.averageWatts > 0 && (source != "strava" || activity.deviceWatts)) {
            issues += issue(source, sourcePath, activity, "info", "STREAM_DATA_COVERAGE", "stream.watts", "Average power exists but power samples are not available.", activity.averageWatts.toString(), "Power charts will be incomplete.")
        }

        val timeSize = stream.time.data.size
        val gpsSize = stream.latlng?.data?.size ?: 0
        if (gpsSize > 0 && timeSize > 0 && abs(gpsSize - timeSize) > 1) {
            issues += issue(source, sourcePath, activity, "warning", "MISSING_STREAM_FIELD", "stream.latlng", "GPS and time stream fields have inconsistent sizes.", "gps=$gpsSize time=$timeSize", "Trim or resample streams before detailed analysis.")
        }
        val altitudeSize = stream.altitude?.data?.size ?: 0
        if (altitudeSize > 0 && timeSize > 0 && abs(altitudeSize - timeSize) > 1) {
            issues += issue(source, sourcePath, activity, "warning", "MISSING_STREAM_FIELD", "stream.altitude", "Altitude and time stream fields have inconsistent sizes.", "altitude=$altitudeSize time=$timeSize", "Trim or resample streams before elevation analysis.")
        }

        issues += analyzeGpsGlitch(source, sourcePath, activity)
        issues += analyzeAltitudeSpike(source, sourcePath, activity)
        return issues
    }

    private fun analyzeDerivedSpeed(source: String, sourcePath: String, activity: StravaActivity): List<DataQualityIssue> {
        val movingTime = activity.movingTime.takeIf { it > 0 } ?: activity.elapsedTime
        if (movingTime <= 0 || activity.distance <= 0.0) return emptyList()
        val speed = activity.distance / movingTime
        return if (speed > speedThreshold(activity.type)) {
            listOf(issue(source, sourcePath, activity, "warning", "INVALID_VALUE", "average_speed", "Computed average speed is unusually high.", "%.1f km/h".format(Locale.US, speed * 3.6), "Inspect GPS glitches or timing data before trusting speed statistics."))
        } else {
            emptyList()
        }
    }

    private fun analyzeGpsGlitch(source: String, sourcePath: String, activity: StravaActivity): List<DataQualityIssue> {
        val stream = activity.stream ?: return emptyList()
        val points = stream.latlng?.data ?: return emptyList()
        val times = stream.time.data
        val limit = minOf(points.size, times.size)
        if (limit < 2) return emptyList()

        var maxSpeed = 0.0
        var maxIndex = 0
        for (index in 1 until limit) {
            val previous = points[index - 1]
            val current = points[index]
            if (previous.size < 2 || current.size < 2) continue
            val deltaSeconds = times[index] - times[index - 1]
            if (deltaSeconds <= 0) continue
            val speed = haversineMeters(previous[0], previous[1], current[0], current[1]) / deltaSeconds
            if (speed > maxSpeed) {
                maxSpeed = speed
                maxIndex = index
            }
        }

        return if (maxSpeed > speedThreshold(activity.type)) {
            listOf(issue(source, sourcePath, activity, "warning", "GPS_GLITCH", "stream.latlng", "GPS trace contains an impossible speed jump.", "%.1f km/h near point %d".format(Locale.US, maxSpeed * 3.6, maxIndex), "Mark the segment as suspicious or smooth/remove the point locally."))
        } else {
            emptyList()
        }
    }

    private fun analyzeAltitudeSpike(source: String, sourcePath: String, activity: StravaActivity): List<DataQualityIssue> {
        val stream = activity.stream ?: return emptyList()
        val altitudes = stream.altitude?.data ?: return emptyList()
        if (altitudes.size < 2) return emptyList()
        val times = stream.time.data
        val limit = if (times.isNotEmpty()) minOf(altitudes.size, times.size) else altitudes.size

        var maxDelta = 0.0
        var maxIndex = 0
        for (index in 1 until limit) {
            val delta = abs(altitudes[index] - altitudes[index - 1])
            if (delta > maxDelta) {
                maxDelta = delta
                maxIndex = index
            }
            if (delta >= 120.0 && (times.isEmpty() || times[index] - times[index - 1] <= 15)) {
                return listOf(issue(source, sourcePath, activity, "warning", "ALTITUDE_SPIKE", "stream.altitude", "Altitude stream contains a sharp spike.", "%.0f m near point %d".format(Locale.US, maxDelta, maxIndex), "Smooth altitude locally or recompute elevation from SRTM."))
            }
        }
        return emptyList()
    }

    private fun buildSummary(provider: String, issues: List<DataQualityIssue>, exclusions: List<DataQualityExclusion>): DataQualitySummary {
        val bySeverity = mutableMapOf("critical" to 0, "warning" to 0, "info" to 0)
        issues.forEach { issue -> bySeverity[issue.severity] = (bySeverity[issue.severity] ?: 0) + 1 }
        val byCategory = issues.groupingBy { issue -> issue.category }.eachCount()
        val status = when {
            provider.isBlank() -> "not_applicable"
            (bySeverity["critical"] ?: 0) > 0 -> "critical"
            (bySeverity["warning"] ?: 0) > 0 -> "warning"
            else -> "ok"
        }
        return DataQualitySummary(
            status = status,
            provider = provider,
            issueCount = issues.size,
            impactedActivities = issues.mapNotNull { issue -> issue.activityId }.distinct().size,
            excludedActivities = exclusions.map { exclusion -> exclusion.activityId }.distinct().size,
            bySeverity = bySeverity,
            byCategory = byCategory,
            topIssues = issues.take(5),
        )
    }

    private fun issue(
        source: String,
        sourcePath: String,
        activity: StravaActivity,
        severity: String,
        category: String,
        field: String,
        message: String,
        rawValue: String,
        suggestion: String,
    ): DataQualityIssue {
        return DataQualityIssue(
            id = "${source}-${activity.id}-${category}-${field.replace(".", "-")}",
            source = source.uppercase(),
            activityId = activity.id,
            activityName = activity.name.trim(),
            activityType = activity.type,
            year = activity.startDateLocal.take(4).ifBlank { activity.startDate.take(4) },
            filePath = sourcePath,
            severity = severity,
            category = category,
            field = field,
            message = message,
            rawValue = rawValue.ifBlank { null },
            suggestion = suggestion,
        )
    }

    private fun DataQualityIssue.markExcluded(exclusionsById: Map<Long, DataQualityExclusion>): DataQualityIssue {
        val activityId = activityId ?: return this
        val exclusion = exclusionsById[activityId] ?: return this
        return copy(excludedFromStats = true, excludedAt = exclusion.excludedAt)
    }

    private fun issueComparator(): Comparator<DataQualityIssue> =
        compareBy<DataQualityIssue> { severityRank(it.severity) }
            .thenByDescending { it.year.orEmpty() }
            .thenBy { it.activityName.orEmpty() }

    private fun exclusionComparator(): Comparator<DataQualityExclusion> =
        compareByDescending<DataQualityExclusion> { it.year.orEmpty() }
            .thenBy { it.activityId }

    private fun severityRank(severity: String): Int = when (severity) {
        "critical" -> 0
        "warning" -> 1
        else -> 2
    }

    private fun speedThreshold(activityType: String): Double = when (activityType) {
        "Run", "TrailRun" -> 12.0
        "Hike", "Walk" -> 7.0
        "AlpineSki" -> 45.0
        else -> 35.0
    }

    private fun requiresRouteStream(activity: StravaActivity): Boolean {
        return activity.type != "VirtualRide" && activity.sportType != "VirtualRide"
    }

    private fun Double.formatRaw(): String = when {
        isNaN() -> "NaN"
        this == Double.POSITIVE_INFINITY -> "+Inf"
        this == Double.NEGATIVE_INFINITY -> "-Inf"
        else -> "%.2f".format(Locale.US, this)
    }

    private fun haversineMeters(lat1: Double, lon1: Double, lat2: Double, lon2: Double): Double {
        val lat1Rad = lat1 * PI / 180
        val lat2Rad = lat2 * PI / 180
        val deltaLat = (lat2 - lat1) * PI / 180
        val deltaLon = (lon2 - lon1) * PI / 180
        val a = sin(deltaLat / 2) * sin(deltaLat / 2) +
                cos(lat1Rad) * cos(lat2Rad) *
                sin(deltaLon / 2) * sin(deltaLon / 2)
        val c = 2 * atan2(sqrt(a), sqrt(1 - a))
        return 6371e3 * c
    }
}

fun List<StravaActivity>.withoutDataQualityExcludedStats(activityProvider: IActivityProvider): List<StravaActivity> {
    if (isEmpty()) return emptyList()
    val exclusions = DataQualityExclusionStorage.load(activityProvider).map { exclusion -> exclusion.activityId }.toSet()
    if (exclusions.isEmpty()) return this
    return filterNot { activity -> exclusions.contains(activity.id) }
}

fun dataQualityExclusionSignature(activityProvider: IActivityProvider): String {
    val file = DataQualityExclusionStorage.file(activityProvider) ?: return "none"
    if (!file.exists()) return "none"
    return "${file.lastModified()}:${file.length()}"
}

fun dataQualityExcludedActivityIds(activityProvider: IActivityProvider): Set<Long> {
    return DataQualityExclusionStorage.load(activityProvider).map { exclusion -> exclusion.activityId }.toSet()
}

private data class DataQualityExclusionFile(
    val exclusions: List<DataQualityExclusion> = emptyList(),
)

private object DataQualityExclusionStorage {
    private val objectMapper = JsonMapper.builder()
        .addModule(KotlinModule.Builder().build())
        .disable(DeserializationFeature.FAIL_ON_NULL_FOR_PRIMITIVES)
        .build()

    fun load(activityProvider: IActivityProvider): List<DataQualityExclusion> {
        val file = file(activityProvider) ?: return emptyList()
        if (!file.exists()) return emptyList()
        return runCatching {
            objectMapper.readValue(file, DataQualityExclusionFile::class.java).exclusions
        }.getOrElse {
            emptyList()
        }
    }

    fun save(activityProvider: IActivityProvider, exclusions: List<DataQualityExclusion>) {
        val file = file(activityProvider) ?: return
        file.parentFile?.mkdirs()
        objectMapper.writeValue(file, DataQualityExclusionFile(exclusions))
    }

    fun file(activityProvider: IActivityProvider): File? {
        val identity = runCatching { activityProvider.cacheIdentity() }.getOrNull() ?: return null
        val athleteDirectory = File(identity.cacheRoot, "strava-${identity.athleteId}")
        return File(athleteDirectory, "data-quality-exclusions-${identity.athleteId}.json")
    }
}
