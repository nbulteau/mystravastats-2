package me.nicolas.stravastats.domain.services.activityproviders

import me.nicolas.stravastats.domain.business.ActivityType
import me.nicolas.stravastats.domain.business.AthletePerformanceSettings
import me.nicolas.stravastats.domain.business.HeartRateZoneSettings
import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.business.strava.StravaDetailedActivity
import me.nicolas.stravastats.domain.business.strava.stream.Stream
import me.nicolas.stravastats.domain.services.toStravaDetailedActivity
import java.time.Instant
import java.time.OffsetDateTime
import java.time.ZoneId
import kotlin.math.abs
import kotlin.math.atan2
import kotlin.math.cos
import kotlin.math.max
import kotlin.math.sin
import kotlin.math.sqrt

data class CompositeActivitySource(
    val name: String,
    val provider: IActivityProvider,
)

class CompositeActivityProvider(
    sources: List<CompositeActivitySource>,
) : AbstractActivityProvider(), AutoCloseable {
    private val sources = sources
        .filter { source -> source.name.isNotBlank() }
        .map { source -> source.copy(name = source.name.trim().lowercase()) }
    private val sourcePriority = this.sources.mapIndexed { index, source -> source.name to index }.toMap()
    private val settingsSource = this.sources.firstOrNull()?.provider

    private var recordsByActivityId: Map<Long, CompositeRecord> = emptyMap()
    private var diagnostics = CompositeDiagnostics()

    init {
        stravaAthlete = this.sources.firstOrNull()?.provider?.athlete()
            ?: throw IllegalArgumentException("CompositeActivityProvider requires at least one source")
        rebuild()
    }

    private fun rebuild() {
        val clusters = mutableListOf<ActivityCluster>()
        val sourceSummaries = mutableListOf<Map<String, Any?>>()

        for (source in sources) {
            val sourceActivities = source.provider.getActivitiesByActivityTypeAndYear(ActivityType.values().toSet(), null)
            val sourceDiagnostics = source.provider.getCacheDiagnostics()
            sourceSummaries.add(
                mapOf(
                    "provider" to source.name,
                    "athleteId" to (source.provider.cacheIdentity()?.athleteId ?: sourceDiagnostics["athleteId"]),
                    "cacheRoot" to (source.provider.cacheIdentity()?.cacheRoot ?: sourceDiagnostics["cacheRoot"]),
                    "activities" to sourceActivities.size,
                    "availableYearBins" to sourceDiagnostics["availableYearBins"],
                )
            )

            for (activity in sourceActivities) {
                val item = SourceActivity(source, activity, activityMatchMetadataFor(activity))
                val matchingCluster = clusters.firstOrNull { cluster ->
                    sourceActivitiesMatch(cluster.items.first(), item)
                }
                if (matchingCluster == null) {
                    clusters.add(ActivityCluster(mutableListOf(item)))
                } else {
                    matchingCluster.items.add(item)
                }
            }
        }

        val records = clusters.map { cluster -> mergeCluster(cluster) }
        activities = records
            .map { record -> record.activity }
            .sortedByDescending { activity -> activitySortTime(activity)?.toEpochMilli() ?: Long.MIN_VALUE }
        recordsByActivityId = records.associateBy { record -> record.activity.id }
        diagnostics = CompositeDiagnostics(
            matchedActivities = records.count { record -> record.sources.size > 1 },
            localOnlyActivities = records.count { record -> record.sources.none { source -> source.provider == SOURCE_STRAVA } },
            conflictCount = records.sumOf { record -> record.conflicts.size },
            conflictSamples = records.flatMap { record -> record.conflicts }.take(12),
            sourceSummaries = sourceSummaries,
        )
    }

    private fun mergeCluster(cluster: ActivityCluster): CompositeRecord {
        val orderedItems = cluster.items.sortedBy { item -> sourcePriority[item.source.name] ?: Int.MAX_VALUE }
        val primary = orderedItems.first()
        val bestStream = orderedItems
            .mapNotNull { item -> item.activity.stream?.let { stream -> item.source.name to stream } }
            .maxByOrNull { (source, stream) -> streamScore(source, stream) }
            ?.second

        val merged = orderedItems.drop(1).fold(primary.activity.copy(stream = bestStream)) { current, item ->
            enrichMissingFields(current, item.activity)
        }
        val conflicts = orderedItems.drop(1).flatMap { item ->
            detectConflicts(primary.activity, item.activity, item.source.name)
        }

        return CompositeRecord(
            activity = merged,
            primaryProvider = primary.source.name,
            primaryId = primary.activity.id,
            sources = orderedItems.map { item -> item.toRef() },
            confidence = if (orderedItems.size > 1) "high" else "single_source",
            conflicts = conflicts,
        )
    }

    override fun getDetailedActivity(activityId: Long): StravaDetailedActivity? {
        val record = recordsByActivityId[activityId] ?: return null
        val source = sources.firstOrNull { source -> source.name == record.primaryProvider }
        val detailed = source?.provider?.getDetailedActivity(record.primaryId)
            ?: record.activity.toStravaDetailedActivity()
        return enrichDetailedActivity(detailed, record.activity)
    }

    override fun getCachedDetailedActivity(activityId: Long): StravaDetailedActivity? {
        val record = recordsByActivityId[activityId] ?: return null
        val source = sources.firstOrNull { source -> source.name == record.primaryProvider }
        val detailed = source?.provider?.getCachedDetailedActivity(record.primaryId)
            ?: return getDetailedActivity(activityId)
        return enrichDetailedActivity(detailed, record.activity)
    }

    override fun getHeartRateZoneSettings(): HeartRateZoneSettings {
        return settingsSource?.getHeartRateZoneSettings() ?: HeartRateZoneSettings()
    }

    override fun saveHeartRateZoneSettings(settings: HeartRateZoneSettings): HeartRateZoneSettings {
        return settingsSource?.saveHeartRateZoneSettings(settings) ?: settings
    }

    override fun getPerformanceSettings(): AthletePerformanceSettings {
        return settingsSource?.getPerformanceSettings() ?: AthletePerformanceSettings()
    }

    override fun savePerformanceSettings(settings: AthletePerformanceSettings): AthletePerformanceSettings {
        return settingsSource?.savePerformanceSettings(settings) ?: settings
    }

    override fun getCacheDiagnostics(): Map<String, Any?> {
        val activeProviders = sources.map { source -> source.name }
        return mapOf(
            "timestamp" to Instant.now().toString(),
            "provider" to "composite",
            "athleteId" to cacheIdentity().athleteId,
            "cacheRoot" to "composite",
            "activities" to activities.size,
            "availableYearBins" to availableYearBins(activities),
            "composite" to mapOf(
                "active" to true,
                "activeProviders" to activeProviders,
                "sources" to diagnostics.sourceSummaries,
                "matchedActivities" to diagnostics.matchedActivities,
                "localOnlyActivities" to diagnostics.localOnlyActivities,
                "conflictCount" to diagnostics.conflictCount,
                "conflictSamples" to diagnostics.conflictSamples,
                "futureProviders" to listOf("ridewithgps", "tcx"),
            ),
        )
    }

    override fun cacheIdentity(): ActivityProviderCacheIdentity {
        return ActivityProviderCacheIdentity(
            cacheRoot = sources.joinToString(";") { source ->
                val identity = source.provider.cacheIdentity()
                "${source.name}=${identity?.cacheRoot ?: "unknown"}"
            },
            athleteId = sources.joinToString("+") { source ->
                val identity = source.provider.cacheIdentity()
                "${source.name}:${identity?.athleteId ?: "unknown"}"
            },
        )
    }

    override fun reload(): Boolean {
        sources.forEach { source -> source.provider.reload() }
        rebuild()
        return true
    }

    override fun close() {
        sources.forEach { source ->
            (source.provider as? AutoCloseable)?.close()
        }
    }

    private data class SourceActivity(
        val source: CompositeActivitySource,
        val activity: StravaActivity,
        val match: ActivityMatchMetadata,
    ) {
        fun toRef() = ActivitySourceRef(
            provider = source.name,
            activityId = activity.id,
            startDateLocal = activity.startDateLocal,
            distance = activity.distance,
            movingTime = activity.movingTime,
            hasStream = activity.stream != null,
        )
    }

    private data class ActivityMatchMetadata(
        val sportFamily: String,
        val startTime: Instant?,
    )

    private data class ActivityCluster(
        val items: MutableList<SourceActivity>,
    )

    private data class CompositeRecord(
        val activity: StravaActivity,
        val primaryProvider: String,
        val primaryId: Long,
        val sources: List<ActivitySourceRef>,
        val confidence: String,
        val conflicts: List<MergeConflict>,
    )

    private data class ActivitySourceRef(
        val provider: String,
        val activityId: Long,
        val startDateLocal: String,
        val distance: Double,
        val movingTime: Int,
        val hasStream: Boolean,
    )

    private data class MergeConflict(
        val field: String,
        val primary: String,
        val other: String,
        val source: String,
    )

    private data class CompositeDiagnostics(
        val matchedActivities: Int = 0,
        val localOnlyActivities: Int = 0,
        val conflictCount: Int = 0,
        val conflictSamples: List<MergeConflict> = emptyList(),
        val sourceSummaries: List<Map<String, Any?>> = emptyList(),
    )

    companion object {
        private const val SOURCE_STRAVA = "strava"
        private const val SOURCE_FIT = "fit"
        private const val SOURCE_GPX = "gpx"
        private const val SAME_ACTIVITY_START_TOLERANCE_SECONDS = 10 * 60L
        private const val SAME_ACTIVITY_TIMEZONE_OFFSET_TOLERANCE_SECONDS = 2 * 60L
        private const val SAME_ACTIVITY_TIMEZONE_NAME = "Europe/Paris"
        private val SAME_ACTIVITY_TIMEZONE = ZoneId.of(SAME_ACTIVITY_TIMEZONE_NAME)
        private val SAME_ACTIVITY_FALLBACK_TIMEZONE_OFFSETS_SECONDS = listOf(60 * 60L, 2 * 60 * 60L)

        private fun activitiesMatch(left: StravaActivity, right: StravaActivity): Boolean {
            return activityValuesMatch(left, right, activityMatchMetadataFor(left), activityMatchMetadataFor(right))
        }

        private fun sourceActivitiesMatch(left: SourceActivity, right: SourceActivity): Boolean {
            return activityValuesMatch(left.activity, right.activity, left.match, right.match)
        }

        private fun activityValuesMatch(
            left: StravaActivity,
            right: StravaActivity,
            leftMatch: ActivityMatchMetadata,
            rightMatch: ActivityMatchMetadata,
        ): Boolean {
            if (leftMatch.sportFamily != rightMatch.sportFamily) return false

            val leftTime = leftMatch.startTime
            val rightTime = rightMatch.startTime
            if (leftTime == null || rightTime == null) return false
            if (!startTimesCompatible(leftTime, rightTime)) return false

            if (!summaryMetricsCompatible(left, right)) return false
            return startLocationCompatible(left.startLatlng, right.startLatlng)
        }

        private fun activityMatchMetadataFor(activity: StravaActivity): ActivityMatchMetadata {
            return ActivityMatchMetadata(
                sportFamily = sportFamily(activity),
                startTime = activitySortTime(activity),
            )
        }

        private fun startTimesCompatible(left: Instant, right: Instant): Boolean {
            val delta = abs(left.epochSecond - right.epochSecond)
            if (delta <= SAME_ACTIVITY_START_TOLERANCE_SECONDS) return true
            return timezoneOffsetsForStartTimes(left, right).any { offset ->
                abs(delta - offset) <= SAME_ACTIVITY_TIMEZONE_OFFSET_TOLERANCE_SECONDS
            }
        }

        private fun timezoneOffsetsForStartTimes(left: Instant, right: Instant): List<Long> {
            val offsets = listOf(left, right)
                .map { instant -> timezoneOffsetForInstant(instant) }
                .filter { offset -> offset > 0 }
                .distinct()
            return offsets.ifEmpty { SAME_ACTIVITY_FALLBACK_TIMEZONE_OFFSETS_SECONDS }
        }

        private fun timezoneOffsetForInstant(value: Instant): Long {
            return abs(SAME_ACTIVITY_TIMEZONE.rules.getOffset(value).totalSeconds.toLong())
        }

        private fun distanceCompatible(left: Double, right: Double): Boolean {
            if (left <= 0.0 || right <= 0.0) return false
            val delta = abs(left - right)
            val limit = max(500.0, max(left, right) * 0.05)
            return delta <= limit
        }

        private fun summaryMetricsCompatible(left: StravaActivity, right: StravaActivity): Boolean {
            if (left.distance > 0.0 && right.distance > 0.0) {
                return distanceCompatible(left.distance, right.distance)
            }
            return durationCompatible(left.movingTime, right.movingTime)
        }

        private fun durationCompatible(left: Int, right: Int): Boolean {
            if (left <= 0 || right <= 0) return false
            val delta = abs(left - right).toDouble()
            val limit = max(120.0, max(left, right).toDouble() * 0.10)
            return delta <= limit
        }

        private fun startLocationCompatible(left: List<Double>?, right: List<Double>?): Boolean {
            if (!validLatLng(left) || !validLatLng(right)) return true
            return haversineMeters(left!![0], left[1], right!![0], right[1]) <= 1000.0
        }

        private fun detectConflicts(primary: StravaActivity, other: StravaActivity, source: String): List<MergeConflict> {
            val conflicts = mutableListOf<MergeConflict>()
            if (primary.distance > 0.0 && other.distance > 0.0) {
                val delta = abs(primary.distance - other.distance)
                if (delta > max(250.0, max(primary.distance, other.distance) * 0.02)) {
                    conflicts.add(MergeConflict("distance", "%.0f".format(primary.distance), "%.0f".format(other.distance), source))
                }
            }
            if (primary.movingTime > 0 && other.movingTime > 0) {
                val delta = abs(primary.movingTime - other.movingTime).toDouble()
                if (delta > max(60.0, max(primary.movingTime, other.movingTime).toDouble() * 0.05)) {
                    conflicts.add(MergeConflict("moving_time", primary.movingTime.toString(), other.movingTime.toString(), source))
                }
            }
            if (validLatLng(primary.startLatlng) && validLatLng(other.startLatlng)) {
                val delta = haversineMeters(primary.startLatlng!![0], primary.startLatlng[1], other.startLatlng!![0], other.startLatlng[1])
                if (delta > 250.0) {
                    conflicts.add(MergeConflict("start_latlng", "0m", "%.0fm".format(delta), source))
                }
            }
            return conflicts
        }

        private fun streamScore(source: String, stream: Stream): Int {
            var score = 1
            score += (stream.latlng?.data?.size ?: 0) * 3
            score += stream.altitude?.data?.size ?: 0
            if (!stream.heartrate?.data.isNullOrEmpty()) score += 3000
            if (!stream.cadence?.data.isNullOrEmpty()) score += 1500
            if (!stream.watts?.data.isNullOrEmpty()) score += 3000
            score += when (source) {
                SOURCE_FIT -> 500
                SOURCE_GPX -> 250
                else -> 0
            }
            return score
        }

        private fun enrichMissingFields(primary: StravaActivity, other: StravaActivity): StravaActivity {
            return primary.copy(
                averageCadence = if (primary.averageCadence == 0.0) other.averageCadence else primary.averageCadence,
                averageHeartrate = if (primary.averageHeartrate == 0.0) other.averageHeartrate else primary.averageHeartrate,
                maxHeartrate = if (primary.maxHeartrate == 0) other.maxHeartrate else primary.maxHeartrate,
                averageWatts = if (primary.averageWatts == 0) other.averageWatts else primary.averageWatts,
                weightedAverageWatts = if (primary.weightedAverageWatts == 0) other.weightedAverageWatts else primary.weightedAverageWatts,
                kilojoules = if (primary.kilojoules == 0.0) other.kilojoules else primary.kilojoules,
                elevHigh = if (primary.elevHigh == 0.0) other.elevHigh else primary.elevHigh,
                totalElevationGain = if (primary.totalElevationGain == 0.0) other.totalElevationGain else primary.totalElevationGain,
                startLatlng = if (primary.startLatlng.isNullOrEmpty()) other.startLatlng else primary.startLatlng,
                maxSpeed = if (primary.maxSpeed == 0f) other.maxSpeed else primary.maxSpeed,
                deviceWatts = primary.deviceWatts || other.deviceWatts,
            )
        }

        private fun enrichDetailedActivity(detailed: StravaDetailedActivity, activity: StravaActivity): StravaDetailedActivity {
            return detailed.copy(
                stream = activity.stream ?: detailed.stream,
                averageCadence = if (detailed.averageCadence == 0.0) activity.averageCadence else detailed.averageCadence,
                averageHeartrate = if (detailed.averageHeartrate == 0.0) activity.averageHeartrate else detailed.averageHeartrate,
                maxHeartrate = if (detailed.maxHeartrate == 0) activity.maxHeartrate else detailed.maxHeartrate,
                averageWatts = if (detailed.averageWatts == 0.0) activity.averageWatts.toDouble() else detailed.averageWatts,
                weightedAverageWatts = if (detailed.weightedAverageWatts == 0) activity.weightedAverageWatts else detailed.weightedAverageWatts,
            )
        }

        private fun availableYearBins(activities: List<StravaActivity>): List<String> {
            return activities
                .mapNotNull { activity ->
                    activity.startDateLocal.takeIf { it.length >= 4 }?.substring(0, 4)
                        ?: activity.startDate.takeIf { it.length >= 4 }?.substring(0, 4)
                }
                .distinct()
                .sorted()
        }

        private fun activitySortTime(activity: StravaActivity): Instant? {
            return parseInstant(activity.startDateLocal) ?: parseInstant(activity.startDate)
        }

        private fun parseInstant(value: String): Instant? {
            return try {
                Instant.parse(value)
            } catch (_: Exception) {
                try {
                    OffsetDateTime.parse(value).toInstant()
                } catch (_: Exception) {
                    null
                }
            }
        }

        private fun sportFamily(activity: StravaActivity): String {
            return when (activity.sportType.ifBlank { activity.type }) {
                ActivityType.Ride.name,
                ActivityType.GravelRide.name,
                ActivityType.MountainBikeRide.name,
                ActivityType.VirtualRide.name -> "ride"
                ActivityType.Run.name,
                ActivityType.TrailRun.name -> "run"
                ActivityType.Hike.name,
                ActivityType.Walk.name -> "walk"
                else -> activity.sportType.ifBlank { activity.type }
            }
        }

        private fun validLatLng(value: List<Double>?): Boolean {
            return value != null && value.size >= 2 && value[0] in -90.0..90.0 && value[1] in -180.0..180.0
        }

        private fun haversineMeters(lat1: Double, lon1: Double, lat2: Double, lon2: Double): Double {
            val earthRadiusMeters = 6_371_000.0
            val lat1Rad = Math.toRadians(lat1)
            val lat2Rad = Math.toRadians(lat2)
            val deltaLat = Math.toRadians(lat2 - lat1)
            val deltaLon = Math.toRadians(lon2 - lon1)
            val a = sin(deltaLat / 2) * sin(deltaLat / 2) +
                cos(lat1Rad) * cos(lat2Rad) * sin(deltaLon / 2) * sin(deltaLon / 2)
            return earthRadiusMeters * 2 * atan2(sqrt(a), sqrt(1 - a))
        }
    }
}
