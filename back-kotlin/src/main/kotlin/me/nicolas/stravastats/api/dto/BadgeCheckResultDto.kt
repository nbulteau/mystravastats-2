package me.nicolas.stravastats.api.dto

import com.fasterxml.jackson.annotation.JsonInclude
import io.swagger.v3.oas.annotations.media.Schema
import me.nicolas.stravastats.domain.business.ActivityType
import me.nicolas.stravastats.domain.business.badges.*
import me.nicolas.stravastats.domain.business.representativeBadgeActivityType
import me.nicolas.stravastats.domain.business.strava.StravaActivity

@Schema(description = "Badge check result", name = "BadgeCheckResult")
data class BadgeCheckResultDto(
    val badge: BadgeDto,
    val activities: List<ActivityDto>,
    val nbCheckedActivities: Int,
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

    return BadgeCheckResultDto(this.badge.toDto(activityTypes), activities, nbCheckedActivities)
}

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
    val stream = activity.stream ?: return null
    val latLngData = stream.latlng?.data ?: return null
    val timeData = stream.time.data
    val dataSize = minOf(latLngData.size, timeData.size)
    if (dataSize == 0) {
        return null
    }

    val waypointToleranceMeters = 500
    var seenStart = false
    var startTime = 0

    for (index in 0 until dataSize) {
        val coords = latLngData[index]
        if (coords.size < 2) {
            continue
        }

        if (!seenStart) {
            if (badge.start.haversineInM(coords[0], coords[1]) < waypointToleranceMeters) {
                seenStart = true
                startTime = timeData[index]
            }
            continue
        }

        if (badge.end.haversineInM(coords[0], coords[1]) < waypointToleranceMeters) {
            val duration = timeData[index] - startTime
            if (duration > 0) {
                return duration
            }
        }
    }
    return null
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
