package me.nicolas.stravastats.domain.services.csv

import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.business.ActivityType

import me.nicolas.stravastats.domain.services.statistics.calculateBestDistanceForTime
import me.nicolas.stravastats.domain.services.statistics.calculateBestTimeForDistance
import me.nicolas.stravastats.domain.utils.formatDate
import me.nicolas.stravastats.domain.utils.formatSeconds

import org.springframework.stereotype.Component

@Component
internal class RunCSVExporter : CSVExporter(ActivityType.Run) {

    override fun generateActivities(activities: List<StravaActivity>): String {
        return activities.joinToString("") { activity ->
            writeCSVLine(
                listOf(
                    activity.startDateLocal.formatDate(),
                    activity.name.trim(),
                    "%.02f".format(activity.distance / 1000),
                    activity.elapsedTime.formatSeconds(),
                    "%d".format(activity.elapsedTime),
                    activity.processAverageSpeed(),
                    activity.calculateBestTimeForDistance(200.0)?.getFormatedSpeed() ?: "",
                    activity.calculateBestTimeForDistance(400.0)?.getFormatedSpeed() ?: "",
                    activity.calculateBestTimeForDistance(1000.0)?.getFormatedSpeed() ?: "",
                    activity.calculateBestTimeForDistance(5000.0)?.getFormatedSpeed() ?: "",
                    activity.calculateBestTimeForDistance(10000.0)?.getFormatedSpeed() ?: "",
                    activity.calculateBestTimeForDistance(21097.0)?.getFormatedSpeed() ?: "",
                    activity.calculateBestTimeForDistance(42195.0)?.getFormatedSpeed() ?: "",
                    activity.calculateBestDistanceForTime(30 * 60)?.getFormatedSpeed() ?: "",
                    activity.calculateBestDistanceForTime(60 * 60)?.getFormatedSpeed() ?: "",
                    activity.calculateBestDistanceForTime(2 * 60 * 60)?.getFormatedSpeed() ?: "",
                    activity.calculateBestDistanceForTime(3 * 60 * 60)?.getFormatedSpeed() ?: "",
                    activity.calculateBestDistanceForTime(4 * 60 * 60)?.getFormatedSpeed() ?: "",
                    activity.calculateBestDistanceForTime(5 * 60 * 60)?.getFormatedSpeed() ?: "",
                    activity.calculateBestDistanceForTime(12 * 60)?.getFormatedSpeed() ?: ""
                ) + activity.enrichedValues()
            )
        }
    }

    override fun generateHeader(): String {
        return writeCSVLine(
            listOf(
                "Date",
                "Description",
                "DistanceStream (km)",
                "TimeStream",
                "TimeStream (seconds)",
                "Average speed (min/km)",
                "Best 200m (min/km)",
                "Best 400m (min/km)",
                "Best 1000m (min/km)",
                "Best 5000m (min/km)",
                "Best 10000m (min/km)",
                "Best half Marathon (min/km)",
                "Best Marathon (min/km)",
                "Best 30 min (min/km)",
                "Best 1 h (min/km)",
                "Best 2 h (min/km)",
                "Best 3 h (min/km)",
                "Best 4 h (min/km)",
                "Best 5 h (min/km)",
                "Best vVO2max = 6 min (min/km)",
            ) + enrichedHeader()
        )
    }

    override fun generateFooter(activities: List<StravaActivity>): String {
        val lastRow = activities.size + 1

        return writeCSVLine(
            listOf(
                ";;" +
                        "=SOMME(\$C2:\$C$lastRow);;" +
                        "=SOMME(\$E2:\$E$lastRow);"
            )
        )
    }
}
