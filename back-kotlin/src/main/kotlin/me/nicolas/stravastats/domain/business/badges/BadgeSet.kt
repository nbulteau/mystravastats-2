package me.nicolas.stravastats.domain.business.badges

import me.nicolas.stravastats.domain.business.strava.StravaActivity


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
    fun check(activities: List<StravaActivity>): List<BadgeCheckResult> {
        val results = badges.map { badge ->
            val (checkedActivities, isCompleted) = badge.check(activities)
            BadgeCheckResult(badge, checkedActivities, isCompleted)
        }
        return deduplicateFamousClimbActivities(results)
    }

    operator fun plus(anotherBadgeSet: BadgeSet): BadgeSet {
        return BadgeSet(name = name, badges = badges + anotherBadgeSet.badges)
    }

    private data class FamousClimbGroupKey(
        val name: String,
        val latitude: Double,
        val longitude: Double,
    )

    private data class FamousClimbActivityKey(
        val group: FamousClimbGroupKey,
        val activityId: Long,
    )

    private data class FamousClimbActivityWinner(
        val resultIndex: Int,
        val quality: Double,
    )

    private fun deduplicateFamousClimbActivities(results: List<BadgeCheckResult>): List<BadgeCheckResult> {
        val winners = mutableMapOf<FamousClimbActivityKey, FamousClimbActivityWinner>()
        results.forEachIndexed { resultIndex, result ->
            val badge = result.badge as? FamousClimbBadge ?: return@forEachIndexed
            val group = FamousClimbGroupKey(badge.name.trim().lowercase(), badge.end.latitude, badge.end.longitude)
            result.activities.forEach { activity ->
                val quality = badge.matchQuality(activity) ?: return@forEach
                val key = FamousClimbActivityKey(group, activity.id)
                val winner = winners[key]
                if (winner == null || quality < winner.quality) {
                    winners[key] = FamousClimbActivityWinner(resultIndex, quality)
                }
            }
        }

        return results.mapIndexed { resultIndex, result ->
            val badge = result.badge as? FamousClimbBadge ?: return@mapIndexed result
            val group = FamousClimbGroupKey(badge.name.trim().lowercase(), badge.end.latitude, badge.end.longitude)
            val filteredActivities = result.activities.filter { activity ->
                winners[FamousClimbActivityKey(group, activity.id)]?.resultIndex == resultIndex
            }
            result.copy(activities = filteredActivities, isCompleted = filteredActivities.isNotEmpty())
        }
    }
}
