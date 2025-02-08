package me.nicolas.stravastats.domain.services.statistics

import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.business.ActivityEffort

/**
 * A [BestEffortTimeStatistic] for 6 minutes that also reports vVO2max.
 *
 * VO2max: Velocity at maximal oxygen uptake.
 */
internal class VO2maxStatistic(
    activities: List<StravaActivity>,
) : BestEffortTimeStatistic("Best VO2max (6 min)", activities, 6 * 60) {

    override fun result(bestActivityEffort: ActivityEffort) =
        super.result(bestActivityEffort) +
                " -- VO2max = %.2f km/h".format(
                    calculateVO2max(
                        bestActivityEffort.distance,
                        bestActivityEffort.seconds
                    )
                )

    private fun calculateVO2max(distance: Double, seconds: Int) = distance / seconds * 3600 / 1000
}