package me.nicolas.stravastats.api.dto

import me.nicolas.stravastats.domain.business.strava.Achievement

data class AchievementDto(
    val effortCount: Int,
    val rank: Int,
    val type: String,
    val typeId: Int,
)

fun Achievement.toDto() = AchievementDto(
    effortCount = effortCount,
    rank = rank,
    type = type,
    typeId = typeId,
)