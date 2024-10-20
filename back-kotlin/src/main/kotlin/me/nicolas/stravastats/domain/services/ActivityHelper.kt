package me.nicolas.stravastats.domain.services

import me.nicolas.stravastats.domain.business.ActivityType
import me.nicolas.stravastats.domain.business.strava.StravaActivity

object ActivityHelper {
    /**
     * Smooth a list of doubles
     * @param size the size of the smoothing window
     * @return a list of smoothed doubles
     */
    fun List<Double>.smooth(size: Int = 5): List<Double> {
        val smooth = DoubleArray(this.size)
        for (i in 0 until size) {
            smooth[i] = this[i]
        }
        for (i in size until this.size - size) {
            smooth[i] = this.subList(i - size, i + size).sum() / (2 * size + 1)
        }
        for (i in this.size - size until this.size) {
            smooth[i] = this[i]
        }

        return smooth.toList()
    }

    /**
     * Remove activities that are not in the list of stravaActivity types to consider (i.e. Run, Ride, Hike, etc.)
     * @return a list of activities filtered by type
     * @see StravaActivity
     */
    fun List<StravaActivity>.filterByActivityTypes() = this.filter { activity ->
        ActivityType.entries.any { activity.type == it.name }
    }
}

