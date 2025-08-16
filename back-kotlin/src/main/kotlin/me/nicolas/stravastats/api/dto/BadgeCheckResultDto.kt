package me.nicolas.stravastats.api.dto

import io.swagger.v3.oas.annotations.media.Schema
import me.nicolas.stravastats.domain.business.ActivityType
import me.nicolas.stravastats.domain.business.badges.*

@Schema(description = "Badge check result", name = "BadgeCheckResult")
data class BadgeCheckResultDto(
    val badge: BadgeDto,
    val activities: List<ActivityDto>,
    val nbCheckedActivities: Int,
)

fun BadgeCheckResult.toDto(activityTypes: Set<ActivityType>): BadgeCheckResultDto {
    val nbCheckedActivities = this.activities.size
    val activities = this.activities.takeLast(1).map { it.toDto() }

    return BadgeCheckResultDto(this.badge.toDto(activityTypes), activities, nbCheckedActivities)
}

@Schema(description = "Badge", name = "Badge")
data class BadgeDto(
    val label: String,
    val description: String,
    val type: String,
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
    return BadgeDto(this.label, this.name, activityTypes.first().name + this.javaClass.simpleName)
}
