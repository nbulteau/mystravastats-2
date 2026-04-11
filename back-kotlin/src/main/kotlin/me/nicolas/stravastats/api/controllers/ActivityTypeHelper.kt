package me.nicolas.stravastats.api.controllers

import me.nicolas.stravastats.domain.business.ActivityType

fun String.convertToActivityTypeSet(): Set<ActivityType> {
    val validNames = ActivityType.entries.map { it.name }
    return this
        .split('_')
        .map { token ->
            ActivityType.entries.firstOrNull { it.name == token }
                ?: throw IllegalArgumentException(
                    "Unknown activity type: '$token'. Valid types: $validNames"
                )
        }
        .toSet()
        .takeIf { it.isNotEmpty() } ?: throw IllegalArgumentException("Activity type must not be empty")
}