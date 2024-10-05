package me.nicolas.stravastats.api.dto

import io.swagger.v3.oas.annotations.media.Schema
import me.nicolas.stravastats.domain.business.badges.*
import me.nicolas.stravastats.domain.business.strava.ActivityType

@Schema(description = "Badge check result", name = "BadgeCheckResult")
data class BadgeCheckResultDto(
    val badge: BadgeDto,
    val activity: ActivityDto?,
    val isCompleted: Boolean,
)

fun BadgeCheckResult.toDto(activityType: ActivityType) =
    BadgeCheckResultDto(this.badge.toDto(activityType), this.activity?.toDto(), this.isCompleted)

@Schema(description = "Badge", name = "Badge")
data class BadgeDto(
    val label: String,
    val description: String,
    val type: String,
)

// King of abstract method Badge.toDto
fun Badge.toDto(activityType: ActivityType): BadgeDto {
    return when (this) {
        is DistanceBadge -> this.toDto(activityType)
        is ElevationBadge -> this.toDto(activityType)
        is FamousClimbBadge -> this.toDto(activityType)
        is MovingTimeBadge -> this.toDto(activityType)
    }
}

    fun ElevationBadge.toDto(activityType: ActivityType): BadgeDto {
    return BadgeDto(this.label, this.totalElevationGain.toString(), activityType.name + this.javaClass.simpleName)
}

fun FamousClimbBadge.toDto(activityType: ActivityType): BadgeDto {
    return BadgeDto(this.label, this.name, activityType.name + this.javaClass.simpleName)
}

fun DistanceBadge.toDto(activityType: ActivityType): BadgeDto {
    return BadgeDto(this.label, this.distance.toString(), activityType.name + this.javaClass.simpleName)
}

fun MovingTimeBadge.toDto(activityType: ActivityType): BadgeDto {
    return BadgeDto(this.label, this.movingTime.toString(), activityType.name + this.javaClass.simpleName)
}

