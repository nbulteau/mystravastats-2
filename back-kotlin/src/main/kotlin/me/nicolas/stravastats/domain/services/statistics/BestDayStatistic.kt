package me.nicolas.stravastats.domain.services.statistics

import me.nicolas.stravastats.domain.business.strava.Activity
import me.nicolas.stravastats.domain.utils.dateFormatter
import java.time.LocalDate

internal class BestDayStatistic(
    name: String,
    activities: List<Activity>,
    private val formatString: String,
    private val function: (List<Activity>) -> Pair<String, Number>?,
) : Statistic(name, activities) {

    override val value: String
        get() {
            val pair = function(activities)
            return if (pair != null) {
                val date = LocalDate.parse(pair.first)
                formatString.format(date.format(dateFormatter), pair.second)
            } else {
                "Not available"
            }
        }

    override fun toString() = value
}