package me.nicolas.stravastats.api.dto

import com.fasterxml.jackson.annotation.JsonInclude
import io.swagger.v3.oas.annotations.media.Schema
import me.nicolas.stravastats.domain.business.ActivityType
import me.nicolas.stravastats.domain.business.badges.*
import me.nicolas.stravastats.domain.business.representativeBadgeActivityType
import me.nicolas.stravastats.domain.business.strava.StravaActivity
import kotlin.math.round

@Schema(description = "Badge check result", name = "BadgeCheckResult")
@JsonInclude(JsonInclude.Include.NON_NULL)
data class BadgeCheckResultDto(
    val badge: BadgeDto,
    val activities: List<ActivityDto>,
    val nbCheckedActivities: Int,
    val climbDetails: ClimbDetailsDto? = null,
)

fun BadgeCheckResult.toDto(activityTypes: Set<ActivityType>): BadgeCheckResultDto {
    val nbCheckedActivities = this.activities.size
    val representative = selectRepresentativeBadgeActivity(this.badge, this.activities)
    val activities = representative?.let { selected ->
        listOf(
            selected.activity.toDto().copy(
                badgeEffortSeconds = selected.badgeEffortSeconds,
            )
        )
    } ?: emptyList()

    val climbDetails = (this.badge as? FamousClimbBadge)?.let { climbBadge ->
        buildClimbDetailsDto(climbBadge, this.activities)
    }

    return BadgeCheckResultDto(this.badge.toDto(activityTypes), activities, nbCheckedActivities, climbDetails)
}

@JsonInclude(JsonInclude.Include.NON_NULL)
data class ClimbDetailsDto(
    val name: String,
    val country: String,
    val massif: String,
    val sourceUrl: String? = null,
    val summitAltitude: Int,
    val minimumAltitude: Int,
    val lengthKm: Double,
    val totalAscent: Int,
    val difficulty: Int,
    val averageGradient: Double,
    val maximumGradient: Double? = null,
    val profile: List<ClimbProfilePointDto>,
    val ascentCount: Int,
    val bestAscent: ClimbAscentDto? = null,
)

data class ClimbProfilePointDto(
    val distanceKm: Double,
    val elevation: Double,
)

data class ClimbAscentDto(
    val activityId: Long,
    val date: String,
    val durationSeconds: Int,
)

private const val CLIMB_PROFILE_POINT_LIMIT = 64
private const val CLIMB_MAXIMUM_GRADIENT_WINDOW_METERS = 500.0
private const val CLIMB_COMPUTED_MAXIMUM_GRADIENT_CEILING = 20.0
private const val CLIMB_REFERENCE_MAXIMUM_GRADIENT_CEILING = 30.0
private const val CLIMB_WAYPOINT_TOLERANCE_METERS = 500
private const val CLIMB_LENGTH_TOLERANCE_RATIO = 0.35
private const val CLIMB_LENGTH_TOLERANCE_MINIMUM_METERS = 750.0

private data class FamousClimbBounds(
    val startIndex: Int,
    val endIndex: Int,
)

private fun buildClimbDetailsDto(
    badge: FamousClimbBadge,
    activities: List<StravaActivity>,
): ClimbDetailsDto {
    var ascentCount = 0
    var bestAscent: ClimbAscentDto? = null
    var profileActivity: StravaActivity? = null
    var profileBounds: FamousClimbBounds? = null

    activities.forEach { activity ->
        val bounds = findFamousClimbBounds(activity, badge) ?: return@forEach
        ascentCount++
        val detectedDuration = famousClimbDurationSeconds(activity, bounds)
        val duration = detectedDuration.takeIf { it > 0 } ?: activity.movingTime
        val candidate = ClimbAscentDto(
            activityId = activity.id,
            date = activity.startDateLocal,
            durationSeconds = duration,
        )

        if (profileActivity == null) {
            profileActivity = activity
            profileBounds = bounds
        }
        if (duration > 0 && betterClimbAscent(candidate, bestAscent)) {
            bestAscent = candidate
            profileActivity = activity
            profileBounds = bounds
        }
    }

    val selectedActivity = profileActivity
    val selectedBounds = profileBounds
    val profile = if (selectedActivity != null && selectedBounds != null) {
        buildClimbProfile(selectedActivity, selectedBounds)
    } else {
        emptyList()
    }
    val maximumGradient = badge.maximumGradient
        .takeIf { it > 0.0 && it <= CLIMB_REFERENCE_MAXIMUM_GRADIENT_CEILING }
        ?.let { round(it * 10.0) / 10.0 }
        ?: if (selectedActivity != null && selectedBounds != null) {
            computeClimbMaximumGradient(selectedActivity, selectedBounds)
        } else {
            null
        }
    val minimumAltitude = (badge.minimumAltitude.takeIf { it > 0 }
        ?: (badge.topOfTheAscent - badge.totalAscent)).coerceAtLeast(0)

    return ClimbDetailsDto(
        name = badge.name,
        country = badge.country,
        massif = badge.massif,
        sourceUrl = badge.sourceUrl.ifBlank { null },
        summitAltitude = badge.topOfTheAscent,
        minimumAltitude = minimumAltitude,
        lengthKm = badge.length,
        totalAscent = badge.totalAscent,
        difficulty = badge.difficulty,
        averageGradient = badge.averageGradient,
        maximumGradient = maximumGradient,
        profile = profile,
        ascentCount = ascentCount,
        bestAscent = bestAscent,
    )
}

private fun betterClimbAscent(candidate: ClimbAscentDto, current: ClimbAscentDto?): Boolean {
    if (current == null || candidate.durationSeconds < current.durationSeconds) return true
    if (candidate.durationSeconds > current.durationSeconds) return false
    if (candidate.date != current.date) {
        return current.date.isBlank() || (candidate.date.isNotBlank() && candidate.date < current.date)
    }
    return candidate.activityId < current.activityId
}

private fun findFamousClimbBounds(
    activity: StravaActivity,
    badge: FamousClimbBadge,
): FamousClimbBounds? {
    val stream = activity.stream ?: return null
    val coordinates = stream.latlng?.data ?: return null
    val distances = stream.distance.data
    val startIndices = mutableListOf<Int>()
    var fallback: FamousClimbBounds? = null
    var bestBounds: FamousClimbBounds? = null
    var bestLengthDelta = Double.POSITIVE_INFINITY
    var scoredCandidate = false
    val referenceLengthMeters = badge.length * 1000.0
    val lengthToleranceMeters = maxOf(
        CLIMB_LENGTH_TOLERANCE_MINIMUM_METERS,
        referenceLengthMeters * CLIMB_LENGTH_TOLERANCE_RATIO,
    )

    coordinates.forEachIndexed { index, coords ->
        if (coords.size < 2) {
            return@forEachIndexed
        }
        if (badge.start.haversineInM(coords[0], coords[1]) < CLIMB_WAYPOINT_TOLERANCE_METERS) {
            startIndices += index
        }
        if (badge.end.haversineInM(coords[0], coords[1]) >= CLIMB_WAYPOINT_TOLERANCE_METERS) {
            return@forEachIndexed
        }

        startIndices.forEach { startIndex ->
            if (startIndex >= index) {
                return@forEach
            }
            val candidate = FamousClimbBounds(startIndex = startIndex, endIndex = index)
            if (fallback == null) {
                fallback = candidate
            }
            if (referenceLengthMeters <= 0.0 || startIndex >= distances.size || index >= distances.size) {
                return@forEach
            }
            val candidateLengthMeters = distances[index] - distances[startIndex]
            if (!candidateLengthMeters.isUsable() || candidateLengthMeters <= 0.0) {
                return@forEach
            }
            scoredCandidate = true
            val lengthDelta = kotlin.math.abs(candidateLengthMeters - referenceLengthMeters)
            if (lengthDelta <= lengthToleranceMeters && lengthDelta < bestLengthDelta) {
                bestBounds = candidate
                bestLengthDelta = lengthDelta
            }
        }
    }

    return bestBounds ?: if (referenceLengthMeters > 0.0 && scoredCandidate) null else fallback
}

private fun famousClimbDurationSeconds(
    activity: StravaActivity,
    bounds: FamousClimbBounds,
): Int {
    val times = activity.stream?.time?.data ?: return 0
    if (bounds.startIndex < 0 || bounds.endIndex <= bounds.startIndex || bounds.endIndex >= times.size) {
        return 0
    }
    return (times[bounds.endIndex] - times[bounds.startIndex]).coerceAtLeast(0)
}

private fun buildClimbProfile(
    activity: StravaActivity,
    bounds: FamousClimbBounds,
): List<ClimbProfilePointDto> {
    val stream = activity.stream ?: return emptyList()
    val altitudes = stream.altitude?.data ?: return emptyList()
    val distances = stream.distance.data
    if (
        bounds.startIndex < 0 ||
        bounds.endIndex <= bounds.startIndex ||
        bounds.endIndex >= altitudes.size ||
        bounds.endIndex >= distances.size
    ) {
        return emptyList()
    }

    val startDistance = distances[bounds.startIndex]
    val endDistance = distances[bounds.endIndex]
    if (!startDistance.isUsable() || !endDistance.isUsable() || endDistance <= startDistance) {
        return emptyList()
    }

    val pointCount = bounds.endIndex - bounds.startIndex + 1
    val sampleCount = minOf(pointCount, CLIMB_PROFILE_POINT_LIMIT)
    return (0 until sampleCount).mapNotNull { sampleIndex ->
        val offset = if (sampleCount > 1) {
            round(sampleIndex.toDouble() * (pointCount - 1) / (sampleCount - 1)).toInt()
        } else {
            0
        }
        val index = bounds.startIndex + offset
        val distance = distances[index]
        val altitude = altitudes[index]
        if (!distance.isUsable() || !altitude.isUsable()) {
            null
        } else {
            ClimbProfilePointDto(
                distanceKm = round(((distance - startDistance) / 1000.0) * 1000.0) / 1000.0,
                elevation = round(altitude * 10.0) / 10.0,
            )
        }
    }
}

// Use a rolling window of at least 500 m and reject implausible road gradients.
// A reference catalogue value takes priority when the climb provides one.
private fun computeClimbMaximumGradient(
    activity: StravaActivity,
    bounds: FamousClimbBounds,
): Double? {
    val stream = activity.stream ?: return null
    val altitudes = stream.altitude?.data ?: return null
    val distances = stream.distance.data
    if (
        bounds.startIndex < 0 ||
        bounds.endIndex <= bounds.startIndex ||
        bounds.endIndex >= altitudes.size ||
        bounds.endIndex >= distances.size
    ) {
        return null
    }

    var windowStart = bounds.startIndex
    var maximumGradient = 0.0
    for (index in (bounds.startIndex + 1)..bounds.endIndex) {
        while (
            windowStart + 1 < index &&
            distances[index] - distances[windowStart + 1] >= CLIMB_MAXIMUM_GRADIENT_WINDOW_METERS
        ) {
            windowStart += 1
        }
        val distanceDelta = distances[index] - distances[windowStart]
        val altitudeDelta = altitudes[index] - altitudes[windowStart]
        if (!distanceDelta.isUsable() || !altitudeDelta.isUsable() || distanceDelta < CLIMB_MAXIMUM_GRADIENT_WINDOW_METERS) {
            continue
        }
        val gradient = (altitudeDelta / distanceDelta) * 100.0
        if (gradient.isUsable() && gradient > maximumGradient && gradient <= CLIMB_COMPUTED_MAXIMUM_GRADIENT_CEILING) {
            maximumGradient = gradient
        }
    }

    return maximumGradient.takeIf { it > 0 }?.let { round(it * 10.0) / 10.0 }
}

private fun Double.isUsable(): Boolean = !isNaN() && !isInfinite()

private data class SelectedBadgeActivity(
    val activity: StravaActivity,
    val badgeEffortSeconds: Int? = null,
)

private fun selectRepresentativeBadgeActivity(badge: Badge, activities: List<StravaActivity>): SelectedBadgeActivity? {
    if (activities.isEmpty()) {
        return null
    }

    return when (badge) {
        is FamousClimbBadge -> selectBestFamousClimbActivity(badge, activities)
        else -> SelectedBadgeActivity(activity = activities.last())
    }
}

private fun selectBestFamousClimbActivity(
    badge: FamousClimbBadge,
    activities: List<StravaActivity>,
): SelectedBadgeActivity {
    val bestEffort = activities.mapNotNull { activity ->
        computeFamousClimbEffortSeconds(activity, badge)?.let { effort ->
            SelectedBadgeActivity(activity = activity, badgeEffortSeconds = effort)
        }
    }.minByOrNull { it.badgeEffortSeconds ?: Int.MAX_VALUE }

    if (bestEffort != null) {
        return bestEffort
    }

    val fallback = activities
        .filter { it.movingTime > 0 }
        .minByOrNull { it.movingTime }
        ?: activities.last()
    return SelectedBadgeActivity(activity = fallback)
}

private fun computeFamousClimbEffortSeconds(
    activity: StravaActivity,
    badge: FamousClimbBadge,
): Int? {
    val bounds = findFamousClimbBounds(activity, badge) ?: return null
    return famousClimbDurationSeconds(activity, bounds).takeIf { it > 0 }
}

@JsonInclude(JsonInclude.Include.NON_NULL)
@Schema(description = "Badge", name = "Badge")
data class BadgeDto(
    val label: String,
    val description: String,
    val type: String,
    val category: String? = null,
)

// King of abstract method Badge.toDto
fun Badge.toDto(activityTypes: Set<ActivityType>): BadgeDto {
    return when (this) {
        is DistanceBadge -> this.toDto(activityTypes)
        is ElevationBadge -> this.toDto(activityTypes)
        is MovingTimeBadge -> this.toDto(activityTypes)
        is HikingBadge -> this.toDto(activityTypes)
        is FamousClimbBadge -> this.toDto(activityTypes)
    }
}

private fun ElevationBadge.toDto(activityTypes: Set<ActivityType>): BadgeDto {
    return BadgeDto(this.label, this.totalElevationGain.toString(), badgeType(activityTypes, this.javaClass.simpleName))
}

private fun DistanceBadge.toDto(activityTypes: Set<ActivityType>): BadgeDto {
    return BadgeDto(this.label, this.distance.toString(), badgeType(activityTypes, this.javaClass.simpleName))
}

private fun MovingTimeBadge.toDto(activityTypes: Set<ActivityType>): BadgeDto {
    return BadgeDto(this.label, this.movingTime.toString(), badgeType(activityTypes, this.javaClass.simpleName))
}

private fun HikingBadge.toDto(activityTypes: Set<ActivityType>): BadgeDto {
    return BadgeDto(this.label, this.description, badgeType(activityTypes, this.javaClass.simpleName))
}

private fun FamousClimbBadge.toDto(activityTypes: Set<ActivityType>): BadgeDto {
    return BadgeDto(
        label = this.label,
        description = this.name,
        type = badgeType(activityTypes, this.javaClass.simpleName),
        category = this.category,
    )
}

private fun badgeType(activityTypes: Set<ActivityType>, badgeClassName: String): String {
    val representativeActivityType = activityTypes.representativeBadgeActivityType()
    return "${representativeActivityType?.name.orEmpty()}$badgeClassName"
}
