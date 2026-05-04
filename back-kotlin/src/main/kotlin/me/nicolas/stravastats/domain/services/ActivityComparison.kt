package me.nicolas.stravastats.domain.services

import me.nicolas.stravastats.domain.business.ActivityType
import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.business.strava.StravaDetailedActivity
import me.nicolas.stravastats.domain.services.activityproviders.IActivityProvider
import java.time.LocalDate
import java.time.OffsetDateTime
import kotlin.math.abs
import kotlin.math.max
import kotlin.math.roundToInt

private const val MAX_CANDIDATES = 5
private const val MAX_SIMILARITY_SCORE = 0.45
private const val SEGMENT_DETAIL_LIMIT = 12
private const val MIN_DISTANCE_SCALE = 1000.0
private const val MIN_ELEVATION_SCALE = 100.0

data class ActivityComparison(
    val status: String,
    val label: String,
    val criteria: ActivityComparisonCriteria,
    val baseline: ActivityComparisonBaseline,
    val deltas: ActivityComparisonDeltas,
    val similarActivities: List<ActivityComparisonActivity>,
    val commonSegments: List<ActivityComparisonSegment>,
)

data class ActivityComparisonCriteria(
    val activityType: String,
    val year: Int,
    val sampleSize: Int,
)

data class ActivityComparisonBaseline(
    val distance: Double,
    val elevationGain: Double,
    val movingTime: Int,
    val averageSpeed: Double,
    val averageHeartrate: Double,
    val averageWatts: Double,
    val averageCadence: Double,
)

data class ActivityComparisonDeltas(
    val distance: Double,
    val elevationGain: Double,
    val movingTime: Int,
    val averageSpeed: Double,
    val averageSpeedPct: Double,
    val averageHeartrate: Double,
    val averageWatts: Double,
    val averageCadence: Double,
)

data class ActivityComparisonActivity(
    val id: Long,
    val name: String,
    val date: String,
    val distance: Double,
    val elevationGain: Double,
    val movingTime: Int,
    val averageSpeed: Double,
    val averageHeartrate: Double,
    val averageWatts: Double,
    val averageCadence: Double,
    val similarityScore: Double,
)

data class ActivityComparisonSegment(
    val id: Long,
    val name: String,
    val matchCount: Int,
    val activityIds: List<Long>,
    val activityNames: List<String>,
)

internal fun buildActivityComparison(
    target: StravaDetailedActivity,
    activityProvider: IActivityProvider,
): ActivityComparison? {
    val activityType = resolveComparisonActivityType(target) ?: return null
    val year = resolveComparisonYear(target) ?: return null

    val selected = activityProvider
        .getActivitiesByActivityTypeAndYear(setOf(activityType), year)
        .withDataQualityCorrections(activityProvider)
        .rankSimilarActivities(target)
        .take(MAX_CANDIDATES)

    if (selected.isEmpty()) {
        return ActivityComparison(
            status = "insufficient-data",
            label = "Not enough similar activities",
            criteria = ActivityComparisonCriteria(activityType.name, year, 0),
            baseline = ActivityComparisonBaseline(0.0, 0.0, 0, 0.0, 0.0, 0.0, 0.0),
            deltas = ActivityComparisonDeltas(0.0, 0.0, 0, 0.0, 0.0, 0.0, 0.0, 0.0),
            similarActivities = emptyList(),
            commonSegments = emptyList(),
        )
    }

    val baseline = buildComparisonBaseline(selected)
    val deltas = buildComparisonDeltas(target, baseline)
    val classification = classifyComparison(deltas)
    return ActivityComparison(
        status = classification.first,
        label = classification.second,
        criteria = ActivityComparisonCriteria(activityType.name, year, selected.size),
        baseline = baseline,
        deltas = deltas,
        similarActivities = selected,
        commonSegments = findCommonSegments(target, selected, activityProvider),
    )
}

private fun resolveComparisonActivityType(target: StravaDetailedActivity): ActivityType? {
    if (target.commute) return ActivityType.Commute
    return listOf(target.sportType, target.type)
        .mapNotNull { value -> runCatching { ActivityType.valueOf(value.trim()) }.getOrNull() }
        .firstOrNull()
}

private fun resolveComparisonYear(target: StravaDetailedActivity): Int? =
    listOf(target.startDateLocal, target.startDate)
        .mapNotNull { value -> parseYear(value) }
        .firstOrNull()

private fun parseYear(value: String): Int? {
    val normalized = value.trim()
    if (normalized.length >= 10) {
        runCatching { LocalDate.parse(normalized.substring(0, 10)).year }.getOrNull()?.let { return it }
    }
    return runCatching { OffsetDateTime.parse(normalized).year }.getOrNull()
}

private fun List<StravaActivity>.rankSimilarActivities(target: StravaDetailedActivity): List<ActivityComparisonActivity> =
    mapNotNull { candidate ->
        if (candidate.id == target.id || candidate.distance <= 0.0 || candidate.movingTime <= 0) {
            return@mapNotNull null
        }
        val score = similarActivityScore(target, candidate)
        if (!score.isFinite() || score > MAX_SIMILARITY_SCORE) {
            return@mapNotNull null
        }
        ActivityComparisonActivity(
            id = candidate.id,
            name = candidate.name,
            date = firstNonEmpty(candidate.startDateLocal, candidate.startDate),
            distance = candidate.distance,
            elevationGain = candidate.totalElevationGain,
            movingTime = candidate.movingTime,
            averageSpeed = candidate.averageSpeed,
            averageHeartrate = candidate.averageHeartrate,
            averageWatts = candidate.averageWatts.toDouble(),
            averageCadence = candidate.averageCadence,
            similarityScore = score,
        )
    }.sortedWith(
        compareBy<ActivityComparisonActivity> { it.similarityScore }
            .thenByDescending { it.date }
    )

private fun similarActivityScore(target: StravaDetailedActivity, candidate: StravaActivity): Double {
    val distanceScore = ratioDelta(target.distance.toDouble(), candidate.distance, MIN_DISTANCE_SCALE)
    val elevationScore = ratioDelta(target.totalElevationGain.toDouble(), candidate.totalElevationGain, MIN_ELEVATION_SCALE)
    return distanceScore * 0.62 + elevationScore * 0.38
}

private fun ratioDelta(target: Double, value: Double, minScale: Double): Double {
    val denominator = max(abs(target), minScale)
    return abs(value - target) / denominator
}

private fun buildComparisonBaseline(activities: List<ActivityComparisonActivity>): ActivityComparisonBaseline =
    ActivityComparisonBaseline(
        distance = activities.averageOf { it.distance },
        elevationGain = activities.averageOf { it.elevationGain },
        movingTime = activities.averageOf { it.movingTime.toDouble() }.roundToInt(),
        averageSpeed = activities.averageOf { it.averageSpeed },
        averageHeartrate = activities.averageOfIgnoringZero { it.averageHeartrate },
        averageWatts = activities.averageOfIgnoringZero { it.averageWatts },
        averageCadence = activities.averageOfIgnoringZero { it.averageCadence },
    )

private fun buildComparisonDeltas(
    target: StravaDetailedActivity,
    baseline: ActivityComparisonBaseline,
): ActivityComparisonDeltas {
    val speedDelta = finiteDelta(target.averageSpeed, baseline.averageSpeed)
    return ActivityComparisonDeltas(
        distance = finiteDelta(target.distance.toDouble(), baseline.distance),
        elevationGain = finiteDelta(target.totalElevationGain.toDouble(), baseline.elevationGain),
        movingTime = target.movingTime - baseline.movingTime,
        averageSpeed = speedDelta,
        averageSpeedPct = percentageDelta(speedDelta, baseline.averageSpeed),
        averageHeartrate = finiteDelta(target.averageHeartrate, baseline.averageHeartrate),
        averageWatts = finiteDelta(target.averageWatts, baseline.averageWatts),
        averageCadence = finiteDelta(target.averageCadence, baseline.averageCadence),
    )
}

private fun classifyComparison(deltas: ActivityComparisonDeltas): Pair<String, String> =
    when {
        abs(deltas.averageSpeedPct) >= 15.0 -> "atypical" to "Atypical pace for similar activities"
        deltas.averageSpeedPct >= 5.0 -> "faster" to "Faster than similar activities"
        deltas.averageSpeedPct <= -5.0 -> "slower" to "Slower than similar activities"
        else -> "typical" to "In line with similar activities"
    }

private fun findCommonSegments(
    target: StravaDetailedActivity,
    activities: List<ActivityComparisonActivity>,
    activityProvider: IActivityProvider,
): List<ActivityComparisonSegment> {
    val targetSegments = target.segmentEfforts
        .mapNotNull { effort ->
            val segmentId = effort.segment.id
            val name = effort.segment.name.ifBlank { effort.name }
            if (segmentId == 0L || name.isBlank()) null else segmentId to name
        }
        .toMap()
    if (targetSegments.isEmpty()) return emptyList()

    val commonById = mutableMapOf<Long, MutableCommonSegment>()
    activities.forEach { activity ->
        val detailed = activityProvider.getCachedDetailedActivity(activity.id)
            ?.withDataQualityCorrections(activityProvider)
            ?: return@forEach
        val seenForActivity = mutableSetOf<Long>()
        detailed.segmentEfforts.forEach { effort ->
            val segmentId = effort.segment.id
            val segmentName = targetSegments[segmentId] ?: return@forEach
            if (!seenForActivity.add(segmentId)) return@forEach
            val common = commonById.getOrPut(segmentId) {
                MutableCommonSegment(id = segmentId, name = segmentName)
            }
            common.matchCount += 1
            common.activityIds += activity.id
            common.activityNames += activity.name
        }
    }

    return commonById.values
        .map { segment ->
            ActivityComparisonSegment(
                id = segment.id,
                name = segment.name,
                matchCount = segment.matchCount,
                activityIds = segment.activityIds,
                activityNames = segment.activityNames,
            )
        }
        .sortedWith(compareByDescending<ActivityComparisonSegment> { it.matchCount }.thenBy { it.name })
        .take(SEGMENT_DETAIL_LIMIT)
}

private data class MutableCommonSegment(
    val id: Long,
    val name: String,
    var matchCount: Int = 0,
    val activityIds: MutableList<Long> = mutableListOf(),
    val activityNames: MutableList<String> = mutableListOf(),
)

private fun List<ActivityComparisonActivity>.averageOfIgnoringZero(getter: (ActivityComparisonActivity) -> Double): Double =
    map(getter).filter { value -> value.isFinite() && value > 0.0 }.averageOrZero()

private fun List<ActivityComparisonActivity>.averageOf(getter: (ActivityComparisonActivity) -> Double): Double =
    map(getter).filter { value -> value.isFinite() }.averageOrZero()

private fun List<Double>.averageOrZero(): Double =
    if (isEmpty()) 0.0 else average()

private fun finiteDelta(target: Double, baseline: Double): Double =
    if (!target.isFinite() || !baseline.isFinite() || baseline == 0.0) 0.0 else target - baseline

private fun percentageDelta(delta: Double, baseline: Double): Double =
    if (!delta.isFinite() || baseline == 0.0) 0.0 else delta / baseline * 100.0

private fun firstNonEmpty(vararg values: String): String =
    values.firstOrNull { it.isNotBlank() }?.trim().orEmpty()
