package me.nicolas.stravastats.domain.business.badges

import me.nicolas.stravastats.domain.business.strava.Activity

/**
 * A BadgeCheckResult is a result of the check of a badge.
 * @param badge the badge that was checked
 * @param activities the activity that completed the badge
 * @param isCompleted a boolean indicating if the badge is completed
 *
 */
data class BadgeCheckResult(
    val badge: Badge,
    val activities: List<Activity>,
    val isCompleted: Boolean,
)