package me.nicolas.stravastats.domain.business.badges

import me.nicolas.stravastats.domain.business.strava.StravaActivity

data class ElevationBadge(
    override val label: String,
    val totalElevationGain: Int,
) : Badge(label) {

    override fun check(activities: List<StravaActivity>): Pair<List<StravaActivity>, Boolean> {
        val checkedActivities = activities.filter { activity -> activity.totalElevationGain >= totalElevationGain }

        return Pair(checkedActivities, checkedActivities.isNotEmpty())
    }

    override fun toString() = "${super.toString()}\n$totalElevationGain m"

    companion object {
        private val RIDE_LEVEL_1 = ElevationBadge(
            label = "Ride that climb 1000 m",
            totalElevationGain = 1000
        )
        private val RIDE_LEVEL_2 = ElevationBadge(
            label = "Ride that climb 1500 m",
            totalElevationGain = 1500
        )
        private val RIDE_LEVEL_3 = ElevationBadge(
            label = "Ride that climb 2000 m",
            totalElevationGain = 2000
        )
        private val RIDE_LEVEL_4 = ElevationBadge(
            label = "Ride that climb 2500 m",
            totalElevationGain = 2500
        )
        private val RIDE_LEVEL_5 = ElevationBadge(
            label = "Ride that climb 3000 m",
            totalElevationGain = 3000
        )
        private val RIDE_LEVEL_6 = ElevationBadge(
            label = "Ride that climb 3500 m",
            totalElevationGain = 3500
        )
        val rideBadgeSet = BadgeSet(
            name = "Run that climb",
            badges = listOf(RIDE_LEVEL_1, RIDE_LEVEL_2, RIDE_LEVEL_3, RIDE_LEVEL_4, RIDE_LEVEL_5, RIDE_LEVEL_6)
        )

        private val RUN_LEVEL_1 = ElevationBadge(
            label = "Run that climb",
            totalElevationGain = 250
        )
        private val RUN_LEVEL_2 = ElevationBadge(
            label = "Run that climb",
            totalElevationGain = 500
        )
        private val RUN_LEVEL_3 = ElevationBadge(
            label = "Run that climb",
            totalElevationGain = 1000
        )
        private val RUN_LEVEL_4 = ElevationBadge(
            label = "Run that climb",
            totalElevationGain = 1500
        )
        private val RUN_LEVEL_5 = ElevationBadge(
            label = "Run that climb",
            totalElevationGain = 2000
        )
        val runBadgeSet = BadgeSet(
            name = "Run that climb",
            badges = listOf(RUN_LEVEL_1, RUN_LEVEL_2, RUN_LEVEL_3, RUN_LEVEL_4, RUN_LEVEL_5)
        )

        private val HIKE_LEVEL_1 = ElevationBadge(
            label = "Hike that climb 1000 m",
            totalElevationGain = 1000
        )
        private val HIKE_LEVEL_2 = ElevationBadge(
            label = "Hike that climb 1500 m",
            totalElevationGain = 1500
        )
        private val HIKE_LEVEL_3 = ElevationBadge(
            label = "Hike that climb 2000 m",
            totalElevationGain = 2000
        )
        private val HIKE_LEVEL_4 = ElevationBadge(
            label = "Hike that climb 2500 m",
            totalElevationGain = 2500
        )
        private val HIKE_LEVEL_5 = ElevationBadge(
            label = "Hike that climb 3000 m",
            totalElevationGain = 3000
        )
        val hikeBadgeSet = BadgeSet(
            name = "Run that climb",
            badges = listOf(HIKE_LEVEL_1, HIKE_LEVEL_2, HIKE_LEVEL_3, HIKE_LEVEL_4, HIKE_LEVEL_5)
        )
    }

}