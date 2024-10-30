package me.nicolas.stravastats.domain.business

import me.nicolas.stravastats.domain.utils.formatSeconds
import java.util.*


/**
 * An effort within an activity.
 */
class ActivityEffort(
    val distance: Double,
    val seconds: Int,
    val deltaAltitude: Double,
    val idxStart: Int,
    val idxEnd: Int,
    val averagePower: Int? = null,
    val label: String,
    val activityShort: ActivityShort
) {
    fun getFormattedSpeed(): String {
        val speed = getSpeed()
        return if (activityShort.type == ActivityType.Run) {
            "${speed}/km"
        } else {
            "$speed km/h"
        }
    }

    fun getSpeed(): String {
        return if (activityShort.type == ActivityType.Run) {
            (seconds * 1000 / distance).formatSeconds()
        } else {
            "%.02f".format(Locale.ENGLISH, distance / seconds * 3600 / 1000)
        }
    }

    fun getMSSpeed(): String {
        return "%.02f".format(Locale.ENGLISH, distance / seconds)
    }

    fun getFormattedGradient() = "${this.getGradient()} %"

    fun getFormattedPower() = if (this.averagePower != null) "${this.averagePower} W" else ""

    fun getGradient() = "%.02f".format(Locale.ENGLISH, 100 * deltaAltitude / distance)

    fun getDescription() = "${this.label}:" +
            "<ul>" +
            "<li>Distance : %.1f km</li>".format(distance / 1000) +
            "<li>Time : ${seconds.formatSeconds()}</li>" +
            "<li>Speed : ${getFormattedSpeed()}</li>" +
            "<li>Gradient: ${getGradient()}%</li>" +
            "<li>Power: ${if (averagePower != null) getFormattedPower() else "Not available"}</li>" +
            "</ul>"
}

