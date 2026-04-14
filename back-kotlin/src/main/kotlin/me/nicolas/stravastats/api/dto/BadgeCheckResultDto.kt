package me.nicolas.stravastats.api.dto

import com.fasterxml.jackson.annotation.JsonInclude
import io.swagger.v3.oas.annotations.media.Schema
import me.nicolas.stravastats.domain.business.ActivityType
import me.nicolas.stravastats.domain.business.badges.*
import me.nicolas.stravastats.domain.business.strava.StravaActivity

@Schema(description = "Badge check result", name = "BadgeCheckResult")
data class BadgeCheckResultDto(
    val badge: BadgeDto,
    val activities: List<ActivityDto>,
    val nbCheckedActivities: Int,
)

fun BadgeCheckResult.toDto(activityTypes: Set<ActivityType>): BadgeCheckResultDto {
    val nbCheckedActivities = this.activities.size
    val representativeActivity = selectRepresentativeActivity(this.badge, this.activities)
    val activities = representativeActivity?.let { listOf(it.toDto()) } ?: emptyList()

    return BadgeCheckResultDto(this.badge.toDto(activityTypes), activities, nbCheckedActivities)
}

private fun selectRepresentativeActivity(badge: Badge, activities: List<StravaActivity>): StravaActivity? {
    if (activities.isEmpty()) {
        return null
    }

    return when (badge) {
        is FamousClimbBadge -> activities
            .filter { it.movingTime > 0 }
            .minByOrNull { it.movingTime }
            ?: activities.last()

        else -> activities.last()
    }
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
    return BadgeDto(this.label, this.totalElevationGain.toString(), activityTypes.first().name + this.javaClass.simpleName)
}

private fun DistanceBadge.toDto(activityTypes: Set<ActivityType>): BadgeDto {
    // TODO: handle case multiple activity types
    return BadgeDto(this.label, this.distance.toString(), activityTypes.first().name + this.javaClass.simpleName)
}

private fun MovingTimeBadge.toDto(activityTypes: Set<ActivityType>): BadgeDto {
    // TODO: handle case multiple activity types
    return BadgeDto(this.label, this.movingTime.toString(), activityTypes.first().name + this.javaClass.simpleName)
}

private fun FamousClimbBadge.toDto(activityTypes: Set<ActivityType>): BadgeDto {
    // TODO: handle case multiple activity types
    return BadgeDto(
        label = this.label,
        description = this.name,
        type = activityTypes.first().name + this.javaClass.simpleName,
        category = this.category,
    )
}
