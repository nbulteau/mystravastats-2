package me.nicolas.stravastats.domain.business.badges

import me.nicolas.stravastats.domain.business.strava.Activity
import me.nicolas.stravastats.domain.services.statistics.MaxElevationStatistic

data class ElevationBadge(
    override val label: String,
    val totalElevationGain: Int,
) : Badge(label) {

    override fun check(activities: List<Activity>): Pair<Activity?, Boolean> {
        val maxElevationStatistic = MaxElevationStatistic(activities)
        val isChecked = if (maxElevationStatistic.activity?.totalElevationGain != null) {
            maxElevationStatistic.activity?.totalElevationGain!! >= totalElevationGain
        } else {
            false
        }
        return Pair(null, isChecked)
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
    }

}