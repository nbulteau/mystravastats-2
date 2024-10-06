package me.nicolas.stravastats.domain.business.badges

import me.nicolas.stravastats.domain.business.strava.Activity

data class MovingTimeBadge(
    override val label: String,
    val movingTime: Int,
) : Badge(label) {

    override fun check(activities: List<Activity>): Pair<List<Activity>, Boolean> {
        val checkedActivities = activities.filter { activity -> activity.movingTime >= movingTime }

        return Pair(checkedActivities, checkedActivities.isNotEmpty())
    }

    override fun toString(): String {
        return label
    }

    companion object {
        private val LEVEL_1 = MovingTimeBadge(
            label = "Moving time 1 hour",
            movingTime = 3600
        )
        private val LEVEL_2 = MovingTimeBadge(
            label = "Moving time 2 hours",
            movingTime = 7200
        )
        private val LEVEL_3 = MovingTimeBadge(
            label = "Moving time 3 hours",
            movingTime = 10800
        )
        private val LEVEL_4 = MovingTimeBadge(
            label = "Moving time 4 hours",
            movingTime = 14400
        )
        private val LEVEL_5 = MovingTimeBadge(
            label = "Moving time 5 hours",
            movingTime = 18000
        )
        private val LEVEL_6 = MovingTimeBadge(
            label = "Moving time 6 hours",
            movingTime = 21600
        )
        private val LEVEL_7 = MovingTimeBadge(
            label = "Moving time 7 hours",
            movingTime = 25200
        )
        val movingTimeBadgesSet = BadgeSet(
            name = "Run that distance",
            badges = listOf(LEVEL_1, LEVEL_2, LEVEL_3, LEVEL_4, LEVEL_5, LEVEL_6, LEVEL_7)
        )
    }
}
