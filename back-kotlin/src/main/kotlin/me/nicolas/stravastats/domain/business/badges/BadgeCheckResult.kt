package me.nicolas.stravastats.domain.business.badges

import me.nicolas.stravastats.domain.business.strava.StravaActivity

/**
 * A BadgeCheckResult is a result of the check of a badge.
 * @param badge the badge that was checked
 * @param activities the stravaActivity that completed the badge
 * @param isCompleted a boolean indicating if the badge is completed
 *
 */
data class BadgeCheckResult(
    val badge: Badge,
    val activities: List<StravaActivity>,
    val isCompleted: Boolean,
)