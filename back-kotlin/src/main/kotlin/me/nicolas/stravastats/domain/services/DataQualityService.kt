package me.nicolas.stravastats.domain.services

import me.nicolas.stravastats.domain.business.ActivityType
import me.nicolas.stravastats.domain.business.DataQualityCorrection
import me.nicolas.stravastats.domain.business.DataQualityCorrectionBatchSummary
import me.nicolas.stravastats.domain.business.DataQualityCorrectionImpact
import me.nicolas.stravastats.domain.business.DataQualityCorrectionPreview
import me.nicolas.stravastats.domain.business.DataQualityCorrectionSuggestion
import me.nicolas.stravastats.domain.business.DataQualityExclusion
import me.nicolas.stravastats.domain.business.DataQualityIssue
import me.nicolas.stravastats.domain.business.DataQualityReport
import me.nicolas.stravastats.domain.business.DataQualitySummary
import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.business.strava.StravaDetailedActivity
import me.nicolas.stravastats.domain.business.strava.stream.SmoothVelocityStream
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
    fun previewCorrection(issueId: String): DataQualityCorrectionPreview
    fun previewSafeCorrections(): DataQualityCorrectionPreview
    fun applyCorrection(issueId: String): DataQualityReport
    fun applySafeCorrections(): DataQualityReport
    fun revertCorrection(correctionId: String): DataQualityReport
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

    override fun previewCorrection(issueId: String): DataQualityCorrectionPreview {
        val normalizedIssueId = issueId.trim()
        require(normalizedIssueId.isNotEmpty()) { "issueId must not be empty" }
        val context = correctionContext()
        val issue = context.report.issues.firstOrNull { it.id == normalizedIssueId }
            ?: throw IllegalArgumentException("issue $normalizedIssueId not found")
        val correction = buildCorrectionForIssue(issue.activityId?.let { context.activitiesById[it] }, issue)
        val corrections = if (correction != null) listOf(correction) else emptyList()
        val blockingReasons = if (correction == null) listOf("${issue.id} cannot be corrected automatically") else emptyList()
        return DataQualityCorrectionPreview(
            generatedAt = Instant.now().toString(),
            mode = "single",
            summary = summarizeCorrections(corrections, 0, blockingReasons.size),
            corrections = corrections,
            blockingReasons = blockingReasons,
        )
    }

    override fun previewSafeCorrections(): DataQualityCorrectionPreview {
        val context = correctionContext()
        val corrections = mutableListOf<DataQualityCorrection>()
        var manualReviewCount = 0
        var unsupportedIssueCount = 0
        val blockingReasons = mutableListOf<String>()
        context.report.issues.forEach { issue ->
            val correction = buildCorrectionForIssue(issue.activityId?.let { context.activitiesById[it] }, issue)
            when {
                correction?.safety == "safe" -> corrections += correction
                correction?.safety == "manual" -> manualReviewCount++
                else -> {
                    unsupportedIssueCount++
                    blockingReasons += "${issue.id} cannot be corrected automatically"
                }
            }
        }
        val deduped = corrections.associateBy { it.id }.values.sortedWith(correctionComparator())
        return DataQualityCorrectionPreview(
            generatedAt = Instant.now().toString(),
            mode = "safe_batch",
            summary = summarizeCorrections(deduped, manualReviewCount, unsupportedIssueCount),
            corrections = deduped,
            blockingReasons = blockingReasons,
        )
    }

    override fun applyCorrection(issueId: String): DataQualityReport {
        val preview = previewCorrection(issueId)
        val correction = preview.corrections.firstOrNull()
            ?: throw IllegalArgumentException("issue $issueId has no safe correction")
        require(correction.safety == "safe") { "issue $issueId requires manual review" }
        DataQualityCorrectionStorage.saveMerged(activityProvider, listOf(correction))
        return buildReport()
    }

    override fun applySafeCorrections(): DataQualityReport {
        val preview = previewSafeCorrections()
        if (preview.corrections.isNotEmpty()) {
            DataQualityCorrectionStorage.saveMerged(activityProvider, preview.corrections)
        }
        return buildReport()
    }

    override fun revertCorrection(correctionId: String): DataQualityReport {
        val normalizedCorrectionId = correctionId.trim()
        require(normalizedCorrectionId.isNotEmpty()) { "correctionId must not be empty" }
        val corrections = DataQualityCorrectionStorage.load(activityProvider)
        var found = false
        val now = Instant.now().toString()
        val updated = corrections.map { correction ->
            if (correction.id == normalizedCorrectionId) {
                found = true
                correction.copy(status = "reverted", revertedAt = now)
            } else {
                correction
            }
        }
        require(found) { "correction $normalizedCorrectionId not found" }
        DataQualityCorrectionStorage.save(activityProvider, updated.sortedWith(correctionComparator()))
        return buildReport()
    }

    private fun buildReport(): DataQualityReport {
        val context = correctionContext()
        return context.report
    }

    private fun correctionContext(): CorrectionContext {
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
        val corrections = DataQualityCorrectionStorage.load(activityProvider)
        val correctedActivities = activities.withDataQualityCorrections(corrections)
        val exclusionsById = exclusions.associateBy { exclusion -> exclusion.activityId }
        val correctionsByIssueId = corrections.active().associateBy { correction -> correction.issueId }
        val issues = correctedActivities
            .flatMap { activity -> analyzeActivity(provider, sourcePath, activity) }
            .map { issue -> issue.markExcluded(exclusionsById) }
            .map { issue -> issue.markCorrected(correctionsByIssueId) }
            .sortedWith(issueComparator())

        val report = DataQualityReport(
            generatedAt = Instant.now().toString(),
            summary = buildSummary(provider, issues, exclusions, corrections),
            issues = issues,
            exclusions = exclusions.sortedWith(exclusionComparator()),
            corrections = corrections.sortedWith(correctionComparator()),
        )
        return CorrectionContext(
            report = report,
            activitiesById = correctedActivities.associateBy { activity -> activity.id },
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
                return issues.withCorrectionSuggestions(activity)
            }
            if (source != "fit" && source != "gpx") {
                return issues.withCorrectionSuggestions(activity)
            }
            issues += issue(source, sourcePath, activity, "warning", "MISSING_STREAM", "stream", "Activity has no stream data.", "", "Open the source file and verify GPS/time streams are present.")
            return issues.withCorrectionSuggestions(activity)
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
        return issues.withCorrectionSuggestions(activity)
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

    private fun buildSummary(
        provider: String,
        issues: List<DataQualityIssue>,
        exclusions: List<DataQualityExclusion>,
        corrections: List<DataQualityCorrection>,
    ): DataQualitySummary {
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
            correctionCount = corrections.active().size,
            safeCorrectionCount = issues.count { issue -> issue.correction?.let { it.available && it.safety == "safe" } == true },
            manualReviewCount = issues.count { issue -> issue.correction?.let { it.available && it.safety == "manual" } == true },
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

    private fun DataQualityIssue.markCorrected(correctionsByIssueId: Map<String, DataQualityCorrection>): DataQualityIssue {
        val correction = correctionsByIssueId[id] ?: return this
        return copy(corrected = true, correctionAppliedAt = correction.appliedAt)
    }

    private fun List<DataQualityIssue>.withCorrectionSuggestions(activity: StravaActivity): List<DataQualityIssue> {
        return map { issue -> issue.copy(correction = correctionSuggestionForIssue(activity, issue)) }
    }

    private fun correctionSuggestionForIssue(activity: StravaActivity, issue: DataQualityIssue): DataQualityCorrectionSuggestion {
        val correction = buildCorrectionForIssue(activity, issue)
            ?: return DataQualityCorrectionSuggestion(
                available = false,
                safety = "unsupported",
                description = "No local non-destructive correction is available for this issue.",
            )
        return DataQualityCorrectionSuggestion(
            available = true,
            safety = correction.safety,
            type = correction.type,
            label = correctionLabel(correction.type),
            description = correction.reason,
        )
    }

    private fun issueComparator(): Comparator<DataQualityIssue> =
        compareBy<DataQualityIssue> { severityRank(it.severity) }
            .thenByDescending { it.year.orEmpty() }
            .thenBy { it.activityName.orEmpty() }

    private fun exclusionComparator(): Comparator<DataQualityExclusion> =
        compareByDescending<DataQualityExclusion> { it.year.orEmpty() }
            .thenBy { it.activityId }

    private fun correctionComparator(): Comparator<DataQualityCorrection> =
        compareBy<DataQualityCorrection> { it.status }
            .thenByDescending { it.year.orEmpty() }
            .thenBy { it.activityId }
            .thenBy { it.id }

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

private const val altitudeSpikeMeters = 120.0
private const val altitudeSpikeSeconds = 15
private const val altitudeSpikeInterpolationMaxNeighborDeltaMeters = 60.0

private data class CorrectionContext(
    val report: DataQualityReport,
    val activitiesById: Map<Long, StravaActivity>,
)

private fun buildCorrectionForIssue(activity: StravaActivity?, issue: DataQualityIssue): DataQualityCorrection? {
    if (activity == null || issue.activityId == null) return null
    return when (issue.category) {
        "GPS_GLITCH" -> {
            val index = findIsolatedGpsOutlier(activity)
            if (index == null) {
                manualCorrection(issue, "GPS glitch is not isolated enough for a safe automatic fix.")
            } else {
                buildRemoveGpsPointCorrection(activity, issue, index)
            }
        }
        "ALTITUDE_SPIKE" -> {
            val index = findIsolatedAltitudeSpike(activity)
            if (index == null) {
                manualCorrection(issue, "Altitude spike is not isolated enough for a safe automatic fix.")
            } else {
                buildSmoothAltitudeCorrection(activity, issue, index)
            }
        }
        "INVALID_VALUE" -> buildInvalidValueCorrection(activity, issue)
        else -> null
    }
}

private fun buildRemoveGpsPointCorrection(activity: StravaActivity, issue: DataQualityIssue, index: Int): DataQualityCorrection {
    val correction = baseCorrection(activity, issue, "REMOVE_GPS_POINT").copy(
        pointIndexes = listOf(index),
        modifiedFields = listOf("stream.latlng", "stream.distance", "stream.velocitySmooth", "distance", "average_speed", "max_speed"),
        reason = "Remove isolated GPS point $index and recompute distance and speed from remaining coordinates.",
    )
    return correction.copy(impact = impactForCorrection(activity, correction))
}

private fun buildSmoothAltitudeCorrection(activity: StravaActivity, issue: DataQualityIssue, index: Int): DataQualityCorrection {
    val correction = baseCorrection(activity, issue, "SMOOTH_ALTITUDE_SPIKE").copy(
        pointIndexes = listOf(index),
        modifiedFields = listOf("stream.altitude", "total_elevation_gain", "elev_high"),
        reason = "Replace isolated altitude point $index by interpolation and recompute elevation gain.",
    )
    return correction.copy(impact = impactForCorrection(activity, correction))
}

private fun buildInvalidValueCorrection(activity: StravaActivity, issue: DataQualityIssue): DataQualityCorrection? {
    return when (issue.field) {
        "distance" -> {
            if (activity.canRecomputeDistance()) {
                buildRecalculateFromStreamCorrection(activity, issue, activity.recomputeMotionModifiedFields(), "Recompute local distance and speed fields from GPS coordinates.")
            } else {
                manualCorrection(issue, "Distance needs GPS coordinates for a safe local recalculation.")
            }
        }
        "average_speed" -> {
            when {
                issue.severity == "critical" && activity.canRecomputeAverageSpeedFromSummary() ->
                    buildRecalculateFromStreamCorrection(activity, issue, listOf("average_speed"), "Recompute local average speed from distance and moving time.")
                issue.severity == "critical" && activity.canRecomputeAverageSpeedFromStream() ->
                    buildRecalculateFromStreamCorrection(activity, issue, activity.recomputeMotionModifiedFields(), "Recompute local average speed from corrected distance and moving time.")
                activity.fieldHasNonFiniteValue(issue.field) ->
                    buildMaskInvalidValueCorrection(activity, issue, listOf(issue.field))
                else ->
                    manualCorrection(issue, "Unusually high average speed needs manual review before changing records.")
            }
        }
        "max_speed" -> {
            when {
                issue.severity == "critical" && activity.canRecomputeVelocity() ->
                    buildRecalculateFromStreamCorrection(activity, issue, listOf("stream.velocitySmooth", "max_speed"), "Recompute local max speed from GPS and time streams.")
                activity.fieldHasNonFiniteValue(issue.field) ->
                    buildMaskInvalidValueCorrection(activity, issue, listOf(issue.field))
                else ->
                    manualCorrection(issue, "Unusually high max speed needs manual review before changing records.")
            }
        }
        "total_elevation_gain" -> {
            when {
                activity.canRecomputeElevation() ->
                    buildRecalculateFromStreamCorrection(activity, issue, listOf("stream.altitude", "total_elevation_gain", "elev_high"), "Recompute local elevation gain from the altitude stream.")
                activity.fieldHasNonFiniteValue(issue.field) ->
                    buildMaskInvalidValueCorrection(activity, issue, listOf(issue.field))
                else ->
                    manualCorrection(issue, "Elevation gain needs an altitude stream for a safe local recalculation.")
            }
        }
        "elapsed_time", "moving_time" ->
            manualCorrection(issue, "Timing fields need manual review before changing activity duration.")
        else -> null
    }
}

private fun buildRecalculateFromStreamCorrection(
    activity: StravaActivity,
    issue: DataQualityIssue,
    modifiedFields: List<String>,
    reason: String,
): DataQualityCorrection {
    val correction = baseCorrection(activity, issue, "RECALCULATE_FROM_STREAM").copy(
        modifiedFields = modifiedFields,
        reason = reason,
    )
    val groupedCorrection = when {
        modifiedFields.shouldRecomputeDistance() -> correction.copy(id = correctionId("${issue.source}-${activity.id}-motion-stream", "RECALCULATE_FROM_STREAM"))
        modifiedFields.shouldRecomputeElevation() -> correction.copy(id = correctionId("${issue.source}-${activity.id}-elevation-stream", "RECALCULATE_FROM_STREAM"))
        else -> correction
    }
    return groupedCorrection.copy(impact = impactForCorrection(activity, groupedCorrection))
}

private fun buildMaskInvalidValueCorrection(
    activity: StravaActivity,
    issue: DataQualityIssue,
    modifiedFields: List<String>,
): DataQualityCorrection {
    val correction = baseCorrection(activity, issue, "MASK_INVALID_VALUE").copy(
        modifiedFields = modifiedFields,
        reason = "Mask non-serializable ${issue.field} with 0 in the corrected local view; the source activity stays unchanged.",
    )
    return correction.copy(impact = impactForCorrection(activity, correction))
}

private fun baseCorrection(activity: StravaActivity, issue: DataQualityIssue, type: String): DataQualityCorrection {
    return DataQualityCorrection(
        id = correctionId(issue.id, type),
        issueId = issue.id,
        source = issue.source,
        activityId = activity.id,
        activityName = activity.name.trim(),
        activityType = activity.type,
        year = activity.startDateLocal.take(4).ifBlank { activity.startDate.take(4) },
        type = type,
        safety = "safe",
        status = "active",
    )
}

private fun manualCorrection(issue: DataQualityIssue, reason: String): DataQualityCorrection {
    return DataQualityCorrection(
        id = correctionId(issue.id, "RECALCULATE_FROM_STREAM"),
        issueId = issue.id,
        source = issue.source,
        activityId = issue.activityId ?: 0,
        activityName = issue.activityName,
        activityType = issue.activityType,
        year = issue.year,
        type = "RECALCULATE_FROM_STREAM",
        safety = "manual",
        status = "active",
        reason = reason,
    )
}

private fun impactForCorrection(activity: StravaActivity, correction: DataQualityCorrection): DataQualityCorrectionImpact {
    val corrected = activity.applyDataQualityCorrections(listOf(correction))
    return DataQualityCorrectionImpact(
        distanceMetersBefore = activity.distance.finiteOrZero(),
        distanceMetersAfter = corrected.distance.finiteOrZero(),
        elevationMetersBefore = activity.totalElevationGain.finiteOrZero(),
        elevationMetersAfter = corrected.totalElevationGain.finiteOrZero(),
        maxSpeedBefore = activity.maxSpeed.toDouble().finiteOrZero(),
        maxSpeedAfter = corrected.maxSpeed.toDouble().finiteOrZero(),
        distanceDeltaMeters = corrected.distance.finiteOrZero() - activity.distance.finiteOrZero(),
        elevationDeltaMeters = corrected.totalElevationGain.finiteOrZero() - activity.totalElevationGain.finiteOrZero(),
    )
}

private fun correctionId(issueId: String, type: String): String =
    "${issueId.lowercase().replace(" ", "-")}-${type.lowercase()}"

private fun correctionLabel(type: String): String = when (type) {
    "REMOVE_GPS_POINT" -> "Remove GPS point"
    "SMOOTH_ALTITUDE_SPIKE" -> "Smooth altitude spike"
    "MASK_INVALID_VALUE" -> "Mask invalid value"
    else -> "Recalculate from stream"
}

private fun StravaActivity.recomputeMotionModifiedFields(): List<String> {
    val fields = mutableListOf("stream.distance", "distance")
    if (movingTime > 0 || elapsedTime > 0) {
        fields += "average_speed"
    }
    if (canRecomputeVelocity()) {
        fields += listOf("stream.velocitySmooth", "max_speed")
    }
    return fields
}

private fun StravaActivity.fieldHasNonFiniteValue(field: String): Boolean {
    return when (field) {
        "distance" -> !distance.isFinite()
        "average_speed" -> !averageSpeed.isFinite()
        "max_speed" -> !maxSpeed.toDouble().isFinite()
        "total_elevation_gain" -> !totalElevationGain.isFinite()
        else -> false
    }
}

private fun StravaActivity.canRecomputeDistance(): Boolean {
    val points = stream?.latlng?.data ?: return false
    return points.size >= 2
}

private fun StravaActivity.canRecomputeAverageSpeedFromSummary(): Boolean =
    distance.isFinite() && distance > 0.0 && (movingTime > 0 || elapsedTime > 0)

private fun StravaActivity.canRecomputeAverageSpeedFromStream(): Boolean =
    canRecomputeDistance() && (movingTime > 0 || elapsedTime > 0)

private fun StravaActivity.canRecomputeVelocity(): Boolean {
    val times = stream?.time?.data ?: return false
    return canRecomputeDistance() && times.size >= 2
}

private fun StravaActivity.canRecomputeElevation(): Boolean {
    val altitudes = stream?.altitude?.data ?: return false
    return altitudes.isNotEmpty()
}

private fun List<String>.shouldRecomputeDistance(): Boolean =
    any { field -> field == "stream.distance" || field == "distance" }

private fun List<String>.shouldRecomputeAverageSpeed(): Boolean =
    any { field -> field == "average_speed" }

private fun List<String>.shouldRecomputeVelocity(): Boolean =
    any { field -> field == "stream.velocitySmooth" || field == "max_speed" }

private fun List<String>.shouldRecomputeElevation(): Boolean =
    any { field -> field == "stream.altitude" || field == "total_elevation_gain" || field == "elev_high" }

private fun Double.finiteOrZero(): Double = if (isFinite()) this else 0.0

private fun findIsolatedGpsOutlier(activity: StravaActivity): Int? {
    val stream = activity.stream ?: return null
    val points = stream.latlng?.data ?: return null
    val times = stream.time.data
    val limit = minOf(points.size, times.size)
    if (limit < 3) return null
    val threshold = correctionSpeedThreshold(activity.type)
    var bestIndex: Int? = null
    var bestScore = 0.0
    for (index in 1 until limit - 1) {
        val previousSpeed = segmentSpeed(points[index - 1], points[index], times[index] - times[index - 1]) ?: continue
        val nextSpeed = segmentSpeed(points[index], points[index + 1], times[index + 1] - times[index]) ?: continue
        val stitchedSpeed = segmentSpeed(points[index - 1], points[index + 1], times[index + 1] - times[index - 1]) ?: continue
        if (previousSpeed <= threshold || nextSpeed <= threshold || stitchedSpeed > threshold) continue
        val score = previousSpeed + nextSpeed - stitchedSpeed
        if (score > bestScore) {
            bestScore = score
            bestIndex = index
        }
    }
    return bestIndex
}

private fun findIsolatedAltitudeSpike(activity: StravaActivity): Int? {
    val stream = activity.stream ?: return null
    val altitudes = stream.altitude?.data ?: return null
    if (altitudes.size < 3) return null
    val times = stream.time.data
    val limit = if (times.isNotEmpty()) minOf(altitudes.size, times.size) else altitudes.size
    var bestIndex: Int? = null
    var bestDelta = 0.0
    for (index in 1 until limit - 1) {
        val previousDelta = abs(altitudes[index] - altitudes[index - 1])
        val nextDelta = abs(altitudes[index] - altitudes[index + 1])
        val neighborDelta = abs(altitudes[index + 1] - altitudes[index - 1])
        if (previousDelta < altitudeSpikeMeters || nextDelta < altitudeSpikeMeters) continue
        if (neighborDelta > altitudeSpikeInterpolationMaxNeighborDeltaMeters) continue
        if (times.isNotEmpty() && times[index + 1] - times[index - 1] > altitudeSpikeSeconds * 2) continue
        if (previousDelta + nextDelta > bestDelta) {
            bestDelta = previousDelta + nextDelta
            bestIndex = index
        }
    }
    return bestIndex
}

private fun segmentSpeed(previous: List<Double>, current: List<Double>, seconds: Int): Double? {
    if (previous.size < 2 || current.size < 2 || seconds <= 0) return null
    return correctionHaversineMeters(previous[0], previous[1], current[0], current[1]) / seconds
}

private fun correctionSpeedThreshold(activityType: String): Double = when (activityType) {
    "Run", "TrailRun" -> 12.0
    "Hike", "Walk" -> 7.0
    "AlpineSki" -> 45.0
    else -> 35.0
}

private fun correctionHaversineMeters(lat1: Double, lon1: Double, lat2: Double, lon2: Double): Double {
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

private fun summarizeCorrections(
    corrections: Collection<DataQualityCorrection>,
    manualReviewCount: Int,
    unsupportedIssueCount: Int,
): DataQualityCorrectionBatchSummary {
    return DataQualityCorrectionBatchSummary(
        safeCorrectionCount = corrections.size,
        manualReviewCount = manualReviewCount,
        unsupportedIssueCount = unsupportedIssueCount,
        activityCount = corrections.map { it.activityId }.distinct().size,
        distanceDeltaMeters = corrections.sumOf { it.impact.distanceDeltaMeters },
        elevationDeltaMeters = corrections.sumOf { it.impact.elevationDeltaMeters },
        modifiedFields = corrections.flatMap { it.modifiedFields }.distinct().sorted(),
        potentiallyImpactsRecords = corrections.isNotEmpty(),
    )
}

private fun List<DataQualityCorrection>.active(): List<DataQualityCorrection> =
    filterNot { correction -> correction.status == "reverted" }

fun List<StravaActivity>.withDataQualityCorrections(activityProvider: IActivityProvider): List<StravaActivity> {
    return withDataQualityCorrections(DataQualityCorrectionStorage.load(activityProvider))
}

private fun List<StravaActivity>.withDataQualityCorrections(corrections: List<DataQualityCorrection>): List<StravaActivity> {
    if (isEmpty()) return emptyList()
    val activeByActivityId = corrections.active().groupBy { correction -> correction.activityId }
    if (activeByActivityId.isEmpty()) return this
    return map { activity -> activity.applyDataQualityCorrections(activeByActivityId[activity.id].orEmpty()) }
}

private fun StravaActivity.applyDataQualityCorrections(corrections: List<DataQualityCorrection>): StravaActivity {
    return corrections.sortedBy { it.appliedAt.orEmpty() }.fold(this) { current, correction ->
        when (correction.type) {
            "REMOVE_GPS_POINT" -> current.removeGpsPoint(correction.pointIndexes.firstOrNull())
            "SMOOTH_ALTITUDE_SPIKE" -> current.smoothAltitudePoint(correction.pointIndexes.firstOrNull())
            "RECALCULATE_FROM_STREAM" -> current.recalculateFromStream(correction.modifiedFields)
            "MASK_INVALID_VALUE" -> current.maskInvalidValues(correction.modifiedFields)
            else -> current
        }
    }
}

private fun StravaActivity.recalculateFromStream(fields: List<String>): StravaActivity {
    var corrected = this
    if (fields.shouldRecomputeDistance()) {
        corrected = corrected.recomputeDistanceAndSpeed()
    } else {
        if (fields.shouldRecomputeAverageSpeed()) {
            corrected = corrected.recomputeAverageSpeed()
        }
        if (fields.shouldRecomputeVelocity()) {
            corrected = corrected.recomputeMaxSpeed()
        }
    }
    if (fields.shouldRecomputeElevation()) {
        corrected = corrected.recomputeElevation()
    }
    return corrected
}

private fun StravaActivity.maskInvalidValues(fields: List<String>): StravaActivity {
    var corrected = this
    fields.forEach { field ->
        corrected = when (field) {
            "distance" -> if (!corrected.distance.isFinite()) corrected.copy(distance = 0.0) else corrected
            "average_speed" -> if (!corrected.averageSpeed.isFinite()) corrected.copy(averageSpeed = 0.0) else corrected
            "max_speed" -> if (!corrected.maxSpeed.toDouble().isFinite()) corrected.copy(maxSpeed = 0.0f) else corrected
            "total_elevation_gain" -> if (!corrected.totalElevationGain.isFinite()) corrected.copy(totalElevationGain = 0.0) else corrected
            else -> corrected
        }
    }
    return corrected
}

private fun StravaActivity.removeGpsPoint(index: Int?): StravaActivity {
    val stream = stream ?: return this
    val latlng = stream.latlng ?: return this
    val resolvedIndex = index ?: return this
    if (resolvedIndex <= 0 || resolvedIndex >= latlng.data.size - 1) return this
    val originalSize = latlng.data.size
    val updatedStream = stream.copy(
        latlng = latlng.copy(data = latlng.data.removeAtIndex(resolvedIndex), originalSize = originalSize - 1),
        time = if (stream.time.data.size == originalSize) stream.time.copy(data = stream.time.data.removeAtIndex(resolvedIndex), originalSize = originalSize - 1) else stream.time,
        altitude = stream.altitude?.let { altitude ->
            if (altitude.data.size == originalSize) altitude.copy(data = altitude.data.removeAtIndex(resolvedIndex), originalSize = originalSize - 1) else altitude
        },
        moving = stream.moving?.let { moving ->
            if (moving.data.size == originalSize) moving.copy(data = moving.data.removeAtIndex(resolvedIndex), originalSize = originalSize - 1) else moving
        },
        heartrate = stream.heartrate?.let { heartrate ->
            if (heartrate.data.size == originalSize) heartrate.copy(data = heartrate.data.removeAtIndex(resolvedIndex), originalSize = originalSize - 1) else heartrate
        },
        watts = stream.watts?.let { watts ->
            if (watts.data.size == originalSize) watts.copy(data = watts.data.removeAtIndex(resolvedIndex), originalSize = originalSize - 1) else watts
        },
        cadence = stream.cadence?.let { cadence ->
            if (cadence.data.size == originalSize) cadence.copy(data = cadence.data.removeAtIndex(resolvedIndex), originalSize = originalSize - 1) else cadence
        },
    )
    return copy(stream = updatedStream).recomputeDistanceAndSpeed().recomputeElevation()
}

private fun StravaActivity.smoothAltitudePoint(index: Int?): StravaActivity {
    val stream = stream ?: return this
    val altitude = stream.altitude ?: return this
    val resolvedIndex = index ?: return this
    if (resolvedIndex <= 0 || resolvedIndex >= altitude.data.size - 1) return this
    val updatedAltitude = altitude.data.toMutableList()
    updatedAltitude[resolvedIndex] = (updatedAltitude[resolvedIndex - 1] + updatedAltitude[resolvedIndex + 1]) / 2
    return copy(stream = stream.copy(altitude = altitude.copy(data = updatedAltitude))).recomputeElevation()
}

private fun StravaActivity.recomputeDistanceAndSpeed(): StravaActivity {
    val stream = stream ?: return this
    val points = stream.latlng?.data ?: return this
    if (points.isEmpty()) return this
    val distances = MutableList(points.size) { 0.0 }
    for (index in 1 until points.size) {
        val previous = points[index - 1]
        val current = points[index]
        distances[index] = distances[index - 1] + if (previous.size >= 2 && current.size >= 2) {
            correctionHaversineMeters(previous[0], previous[1], current[0], current[1])
        } else {
            0.0
        }
    }
    val distanceMeters = distances.last()
    val velocity = recomputeVelocity(points, stream.time.data)
    val updatedStream = stream.copy(
        distance = stream.distance.copy(data = distances, originalSize = distances.size),
        velocitySmooth = SmoothVelocityStream(velocity, velocity.size, stream.distance.resolution, "time"),
    )
    return copy(
        stream = updatedStream,
        distance = distanceMeters,
        maxSpeed = (velocity.maxOrNull() ?: 0.0).toFloat(),
    ).recomputeAverageSpeed()
}

private fun StravaActivity.recomputeAverageSpeed(): StravaActivity {
    val movingSeconds = movingTime.takeIf { it > 0 } ?: elapsedTime
    if (movingSeconds <= 0 || !distance.isFinite()) return this
    return copy(averageSpeed = distance / movingSeconds)
}

private fun StravaActivity.recomputeMaxSpeed(): StravaActivity {
    val stream = stream ?: return this
    val points = stream.latlng?.data ?: return this
    val velocity = recomputeVelocity(points, stream.time.data)
    return copy(
        stream = stream.copy(
            velocitySmooth = SmoothVelocityStream(velocity, velocity.size, stream.distance.resolution, "time"),
        ),
        maxSpeed = (velocity.maxOrNull() ?: 0.0).toFloat(),
    )
}

private fun recomputeVelocity(points: List<List<Double>>, times: List<Int>): List<Float> {
    val velocity = MutableList(points.size) { 0.0f }
    val limit = minOf(points.size, times.size)
    for (index in 1 until limit) {
        velocity[index] = (segmentSpeed(points[index - 1], points[index], times[index] - times[index - 1]) ?: 0.0).toFloat()
    }
    return velocity
}

private fun StravaActivity.recomputeElevation(): StravaActivity {
    val altitude = stream?.altitude?.data ?: return this
    if (altitude.isEmpty()) return this
    val totalGain = altitude.zipWithNext { previous, current -> current - previous }
        .filter { delta -> delta > 0 }
        .sum()
    return copy(totalElevationGain = totalGain, elevHigh = altitude.maxOrNull() ?: elevHigh)
}

fun StravaDetailedActivity.withDataQualityCorrections(activityProvider: IActivityProvider): StravaDetailedActivity {
    val corrections = DataQualityCorrectionStorage.load(activityProvider).active().filter { correction -> correction.activityId == id }
    if (corrections.isEmpty()) return this
    val corrected = toCorrectionActivity().applyDataQualityCorrections(corrections)
    return copy(
        averageSpeed = corrected.averageSpeed,
        distance = corrected.distance.toInt(),
        elevHigh = corrected.elevHigh,
        maxSpeed = corrected.maxSpeed.toDouble(),
        totalElevationGain = corrected.totalElevationGain.toInt(),
        stream = corrected.stream,
    )
}

private fun StravaDetailedActivity.toCorrectionActivity(): StravaActivity =
    StravaActivity(
        athlete = me.nicolas.stravastats.domain.business.strava.AthleteRef(athlete.id.toInt()),
        averageSpeed = averageSpeed,
        averageCadence = averageCadence,
        averageHeartrate = averageHeartrate,
        maxHeartrate = maxHeartrate,
        averageWatts = averageWatts.toInt(),
        commute = commute,
        distance = distance.toDouble(),
        deviceWatts = deviceWatts,
        elapsedTime = elapsedTime,
        elevHigh = elevHigh,
        id = id,
        kilojoules = kilojoules,
        maxSpeed = maxSpeed.toFloat(),
        movingTime = movingTime,
        name = name,
        startDate = startDate,
        startDateLocal = startDateLocal,
        startLatlng = startLatLng,
        totalElevationGain = totalElevationGain.toDouble(),
        type = type,
        uploadId = uploadId,
        weightedAverageWatts = weightedAverageWatts,
        gearId = gearId,
        stream = stream,
    )

private fun <T> List<T>.removeAtIndex(index: Int): List<T> =
    filterIndexed { currentIndex, _ -> currentIndex != index }

fun List<StravaActivity>.withoutDataQualityExcludedStats(activityProvider: IActivityProvider): List<StravaActivity> {
    if (isEmpty()) return emptyList()
    val correctedActivities = withDataQualityCorrections(activityProvider)
    val exclusions = DataQualityExclusionStorage.load(activityProvider).map { exclusion -> exclusion.activityId }.toSet()
    if (exclusions.isEmpty()) return correctedActivities
    return correctedActivities.filterNot { activity -> exclusions.contains(activity.id) }
}

fun dataQualityExclusionSignature(activityProvider: IActivityProvider): String {
    val file = DataQualityExclusionStorage.file(activityProvider) ?: return "none"
    val exclusionSignature = if (!file.exists()) "none" else "${file.lastModified()}:${file.length()}"
    return "$exclusionSignature|${dataQualityCorrectionSignature(activityProvider)}"
}

fun dataQualityCorrectionSignature(activityProvider: IActivityProvider): String {
    val file = DataQualityCorrectionStorage.file(activityProvider) ?: return "none"
    if (!file.exists()) return "none"
    return "${file.lastModified()}:${file.length()}"
}

fun dataQualityExcludedActivityIds(activityProvider: IActivityProvider): Set<Long> {
    return DataQualityExclusionStorage.load(activityProvider).map { exclusion -> exclusion.activityId }.toSet()
}

private data class DataQualityExclusionFile(
    val exclusions: List<DataQualityExclusion> = emptyList(),
)

private data class DataQualityCorrectionFile(
    val corrections: List<DataQualityCorrection> = emptyList(),
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

private object DataQualityCorrectionStorage {
    private val objectMapper = JsonMapper.builder()
        .addModule(KotlinModule.Builder().build())
        .disable(DeserializationFeature.FAIL_ON_NULL_FOR_PRIMITIVES)
        .build()

    fun load(activityProvider: IActivityProvider): List<DataQualityCorrection> {
        val file = file(activityProvider) ?: return emptyList()
        if (!file.exists()) return emptyList()
        return runCatching {
            objectMapper.readValue(file, DataQualityCorrectionFile::class.java).corrections
        }.getOrElse {
            emptyList()
        }
    }

    fun saveMerged(activityProvider: IActivityProvider, corrections: List<DataQualityCorrection>) {
        val existing = load(activityProvider).associateBy { correction -> correction.id }.toMutableMap()
        val now = Instant.now().toString()
        corrections.forEach { correction ->
            existing[correction.id] = correction.copy(status = "active", appliedAt = now, revertedAt = null)
        }
        save(activityProvider, existing.values.sortedWith(compareBy<DataQualityCorrection> { it.status }.thenByDescending { it.year.orEmpty() }.thenBy { it.activityId }.thenBy { it.id }))
    }

    fun save(activityProvider: IActivityProvider, corrections: List<DataQualityCorrection>) {
        val file = file(activityProvider) ?: return
        file.parentFile?.mkdirs()
        objectMapper.writeValue(file, DataQualityCorrectionFile(corrections))
    }

    fun file(activityProvider: IActivityProvider): File? {
        val identity = runCatching { activityProvider.cacheIdentity() }.getOrNull() ?: return null
        val athleteDirectory = File(identity.cacheRoot, "strava-${identity.athleteId}")
        return File(athleteDirectory, "data-quality-corrections-${identity.athleteId}.json")
    }
}
