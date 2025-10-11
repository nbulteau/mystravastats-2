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
    fun getFormattedSpeedWithUnits(): String {
        val speed = getFormatedSpeed()
        return if (activityShort.type in runActivities) {
            "${speed}/km"
        } else {
            "$speed km/h"
        }
    }

    fun getFormatedSpeed(): String {
        return if (activityShort.type in runActivities) {
            (seconds * 1000 / distance).formatSeconds()
        } else {
            "%.02f".format(Locale.ENGLISH, distance / seconds * 3600 / 1000)
        }
    }

    fun getMSSpeed(): Double {
        return distance / seconds
    }

    fun getFormattedGradientWithUnit() = "${this.getFormattedGradient()} %"

    fun getFormattedPower() = if (this.averagePower != null) "${this.averagePower} W" else ""

    fun getPower() = this.averagePower

    fun getFormattedGradient() = "%.02f".format(Locale.ENGLISH, getGradient())

    fun getGradient() =100 * deltaAltitude / distance

    fun getDescription() = "${this.label}:" +
            "<ul>" +
            "<li>Distance : %.1f km</li>".format(distance / 1000) +
            "<li>Time : ${seconds.formatSeconds()}</li>" +
            "<li>Speed : ${getFormattedSpeedWithUnits()}</li>" +
            "<li>Gradient: ${getFormattedGradient()}%</li>" +
            "<li>Power: ${if (averagePower != null) getFormattedPower() else "Not available"}</li>" +
            "</ul>"
}

