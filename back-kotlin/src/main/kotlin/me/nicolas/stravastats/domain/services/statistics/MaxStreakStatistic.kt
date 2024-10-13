package me.nicolas.stravastats.domain.services.statistics

import me.nicolas.stravastats.domain.business.strava.StravaActivity
import java.time.LocalDate

internal class MaxStreakStatistic(
    activities: List<StravaActivity>,
) : Statistic("Max streak", activities) {

    private val maxStreak: Int

    init {
        var maxLen = 0

        if (activities.isNotEmpty()) {
            val lastDate = LocalDate.parse(activities.first().startDateLocal.substringBefore('T'))
            val firstDate = LocalDate.parse(activities.last().startDateLocal.substringBefore('T'))
            val firstEpochDay = firstDate.toEpochDay()
            val activeDaysSet: Set<Int> = activities
                .map { activity ->
                    val date = LocalDate.parse(activity.startDateLocal.substringBefore('T'))
                    (date.toEpochDay() - firstEpochDay).toInt()
                }.toSet()

            val days = (lastDate.toEpochDay() - firstDate.toEpochDay()).toInt()
            val activeDays = Array(days) { activeDaysSet.contains(it) }

            var currLen = 0
            for (k in 0 until days) {
                if (activeDays[k]) {
                    currLen++
                } else {
                    if (currLen > maxLen) {
                        maxLen = currLen
                    }
                    currLen = 0
                }
            }
        }

        maxStreak = maxLen
    }

    override val value: String
        get() = maxStreak.toString()

    override fun toString() = value
}

