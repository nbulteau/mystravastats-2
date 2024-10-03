package me.nicolas.stravastats.api.dto

import io.swagger.v3.oas.annotations.media.Schema
import me.nicolas.stravastats.domain.business.badges.Badge
import me.nicolas.stravastats.domain.business.badges.BadgeCheckResult
import me.nicolas.stravastats.domain.business.strava.Activity

@Schema(description = "Badge check result", name = "BadgeCheckResult")
data class BadgeCheckResultDto(
    val badge: Badge,
    val activity: Activity?,
    val isCompleted: Boolean,
)

fun BadgeCheckResult.toDto() = BadgeCheckResultDto(this.badge, this.activity, this.isCompleted)

