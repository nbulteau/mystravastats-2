package me.nicolas.stravastats.domain.services.statistics

import me.nicolas.stravastats.domain.business.strava.StravaActivity
import java.time.LocalDate

internal class MaxStreakStatistic(
    activities: List<StravaActivity>,
) : Statistic("Max streak", activities) {

    private val maxStreak: Int

    init {
        val uniqueDates = activities
            .mapNotNull { activity ->
                val rawDate = activity.startDateLocal.substringBefore('T')
                runCatching { LocalDate.parse(rawDate) }.getOrNull()
            }
            .toSet()
            .sorted()

        var maxLen = 0
        var currentLen = 0
        var previousDate: LocalDate? = null

        for (date in uniqueDates) {
            currentLen = if (previousDate != null && previousDate.plusDays(1) == date) {
                currentLen + 1
            } else {
                1
            }
            if (currentLen > maxLen) {
                maxLen = currentLen
            }
            previousDate = date
        }

        maxStreak = maxLen
    }

    override val value: String
        get() = maxStreak.toString()

    override fun toString() = value
}
