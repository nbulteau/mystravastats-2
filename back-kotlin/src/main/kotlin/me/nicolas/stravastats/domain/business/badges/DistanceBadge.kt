package me.nicolas.stravastats.domain.business.badges

import me.nicolas.stravastats.domain.business.strava.Activity

data class DistanceBadge(
    override val label: String,
    val distance: Int,
) : Badge(label) {

    override fun check(activities: List<Activity>): Pair<List<Activity>, Boolean> {
        val checkedActivities = activities.filter { activity -> activity.distance >= distance }

        return Pair(checkedActivities, checkedActivities.isNotEmpty())
    }

    override fun toString() = label

    companion object {
        private val RIDE_LEVEL_1 = DistanceBadge(
            label = "Hit the road 50 km",
            distance = 50000
        )
        private val RIDE_LEVEL_2 = DistanceBadge(
            label = "Hit the road 100 km",
            distance = 100000
        )
        private val RIDE_LEVEL_3 = DistanceBadge(
            label = "Hit the road 150 km",
            distance = 150000
        )
        private val RIDE_LEVEL_4 = DistanceBadge(
            label = "Hit the road 200 km",
            distance = 200000
        )
        private val RIDE_LEVEL_5 = DistanceBadge(
            label = "Hit the road 250 km",
            distance = 250000
        )
        private val RIDE_LEVEL_6 = DistanceBadge(
            label = "Hit the road 300 km",
            distance = 300000
        )
        val rideBadgeSet = BadgeSet(
            name = "Hit the road",
            badges = listOf(RIDE_LEVEL_1, RIDE_LEVEL_2, RIDE_LEVEL_3, RIDE_LEVEL_4, RIDE_LEVEL_5, RIDE_LEVEL_6)
        )

        private val RUN_LEVEL_1 = DistanceBadge(
            label = "Run that distance 10 km",
            distance = 10000
        )
        private val RUN_LEVEL_2 = DistanceBadge(
            label = "Run that distance half Marathon",
            distance = 21097
        )
        private val RUN_LEVEL_3 = DistanceBadge(
            label = "Run that distance 30 km",
            distance = 30000
        )
        private val RUN_LEVEL_4 = DistanceBadge(
            label = "Run that distance Marathon",
            distance = 42195
        )
        val runBadgeSet = BadgeSet(
            name = "Run that distance",
            badges = listOf(RUN_LEVEL_1, RUN_LEVEL_2, RUN_LEVEL_3, RUN_LEVEL_4)
        )

        private val HIKE_LEVEL_1 = DistanceBadge(
            label = "Hike that distance 10 km",
            distance = 10000
        )
        private val HIKE_LEVEL_2 = DistanceBadge(
            label = "Hike that distance 15 km",
            distance = 15000
        )
        private val HIKE_LEVEL_3 = DistanceBadge(
            label = "Hike that distance 20 km",
            distance = 20000
        )
        private val HIKE_LEVEL_4 = DistanceBadge(
            label = "Hike that distance 25 km",
            distance = 25000
        )
        private val HIKE_LEVEL_5 = DistanceBadge(
            label = "Hike that distance 30 km",
            distance = 30000
        )
        private val HIKE_LEVEL_6 = DistanceBadge(
            label = "Hike that distance 35 km",
            distance = 35000
        )
        val hikeBadgeSet = BadgeSet(
            name = "Hike that distance",
            badges = listOf(HIKE_LEVEL_1, HIKE_LEVEL_2, HIKE_LEVEL_3, HIKE_LEVEL_4, HIKE_LEVEL_5, HIKE_LEVEL_6)
        )
    }
}