package me.nicolas.stravastats.domain.business.strava

import me.nicolas.stravastats.domain.utils.formatSeconds
import java.util.*


/**
 * An effort within an activity.
 */
data class ActivityEffort(
    val activity: Activity,
    val distance: Double,
    val seconds: Int,
    val deltaAltitude: Double,
    val idxStart: Int,
    val idxEnd: Int,
    val averagePower: Int? = null,
) {
    fun getFormattedSpeed(): String {
        val speed = getSpeed()
        return if (activity.type == Run) {
            "${speed}/km"
        } else {
            "$speed km/h"
        }
    }

    fun getSpeed(): String {
        return if (activity.type == Run) {
            (seconds * 1000 / distance).formatSeconds()
        } else {
            "%.02f".format(Locale.ENGLISH, distance / seconds * 3600 / 1000)
        }
    }

    fun getMSSpeed(): String {
        return "%.02f".format(Locale.ENGLISH, distance / seconds)
    }

    fun getFormattedGradient() = "${this.getGradient()} %"

    fun getFormattedPower() = if (this.averagePower != null) "${this.averagePower} Watts" else ""

    fun getGradient() = "%.02f".format(Locale.ENGLISH, 100 * deltaAltitude / distance)
}