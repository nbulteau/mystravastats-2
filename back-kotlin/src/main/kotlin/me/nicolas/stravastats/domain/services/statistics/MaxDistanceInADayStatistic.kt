package me.nicolas.stravastats.domain.services.statistics

import me.nicolas.stravastats.domain.business.strava.Activity
import me.nicolas.stravastats.domain.utils.dateFormatter
import java.time.LocalDate

internal class MaxDistanceInADayStatistic(
    activities: List<Activity>,
) : Statistic(name = "Max distance in a day", activities) {

    private val mostActiveDay: Map.Entry<String, Double>? =
        activities.groupBy { activity -> activity.startDateLocal.substringBefore('T') }
            .mapValues { (_, activities) -> activities.sumOf { activity -> activity.distance } }
            .maxByOrNull { entry: Map.Entry<String, Double> -> entry.value }

    override val value: String
        get() = if (mostActiveDay != null) {
            "%.2f km - ${
                LocalDate.parse(mostActiveDay.key).format(dateFormatter)
            }".format(mostActiveDay.value.div(1000))
        } else {
            "Not available"
        }
}