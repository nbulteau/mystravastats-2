package me.nicolas.stravastats.domain.business.badges

import me.nicolas.stravastats.domain.business.strava.Activity


enum class BadgeSetEnum {
    FAMOUS, GENERAL
}

/**
 * A BadgeSet is a collection of badges.
 * @param name the name of the badge set
 * @param badges the list of badges
 */
data class BadgeSet(val name: String, private val badges: List<Badge> = listOf()) {

    /**
     * Check all the badges of the set.
     * @param activities the list of activities to check
     * @return a list of BadgeCheckResult
     */
    fun check(activities: List<Activity>): List<BadgeCheckResult> {
        return badges.map { badge ->
            val (activity, isCompleted) = badge.check(activities)
            BadgeCheckResult(badge, activity, isCompleted)
        }
    }

    operator fun plus(anotherBadgeSet: BadgeSet): BadgeSet {
        return BadgeSet(name = name, badges = badges + anotherBadgeSet.badges)
    }
}