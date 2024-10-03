package me.nicolas.stravastats.domain.services.statistics

import me.nicolas.stravastats.domain.business.strava.Activity

abstract class Statistic(val name: String, protected val activities: List<Activity>) {
    abstract val value: String

    override fun toString() = value
}

class GlobalStatistic(
    name: String,
    activities: List<Activity>,
    private val formatString: String,
    private val function: (List<Activity>) -> Number,
) : Statistic(name, activities) {

    override val value: String
        get() = formatString.format(function(activities))
}

abstract class ActivityStatistic(
    name: String,
    activities: List<Activity>,
) : Statistic(name, activities) {

    var activity: Activity? = null

    override fun toString() = if (activity != null) {
        "$value - $activity"
    } else {
        "Not available"
    }
}



