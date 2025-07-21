package me.nicolas.stravastats.api.controllers

import me.nicolas.stravastats.domain.business.ActivityType

fun String.convertToActivityTypeSet(): Set<ActivityType> {
    val activityTypes = this
        .split('_').map { ActivityType.valueOf(it) }.toSet()
        .takeIf { it.isNotEmpty() } ?: throw IllegalArgumentException("Activity type must not be empty")

    return activityTypes
}