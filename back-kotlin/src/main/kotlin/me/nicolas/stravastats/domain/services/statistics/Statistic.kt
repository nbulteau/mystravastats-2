package me.nicolas.stravastats.domain.services.statistics

import me.nicolas.stravastats.domain.business.ActivityShort
import me.nicolas.stravastats.domain.business.strava.StravaActivity

abstract class Statistic(val name: String, protected val activities: List<StravaActivity>) {
    abstract val value: String

    override fun toString() = value
}

class GlobalStatistic(
    name: String,
    activities: List<StravaActivity>,
    private val formatString:(Number) -> String,
    private val function: (List<StravaActivity>) -> Number,
) : Statistic(name, activities) {

    override val value: String
        get() = formatString(function(activities))
}

abstract class ActivityStatistic(
    name: String,
    activities: List<StravaActivity>,
) : Statistic(name, activities) {

    var activity: ActivityShort? = null

    override fun toString() = if (activity != null) {
        "$value - $activity"
    } else {
        "Not available"
    }
}



