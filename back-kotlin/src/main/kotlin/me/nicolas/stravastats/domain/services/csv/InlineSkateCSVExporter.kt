package me.nicolas.stravastats.domain.services.csv

import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.business.ActivityType

import me.nicolas.stravastats.domain.services.statistics.calculateBestDistanceForTime
import me.nicolas.stravastats.domain.services.statistics.calculateBestTimeForDistance
import me.nicolas.stravastats.domain.utils.formatDate
import me.nicolas.stravastats.domain.utils.formatSeconds

internal class InlineSkateCSVExporter(clientId: String, activities: List<StravaActivity>, year: Int) :
    CSVExporter(clientId, activities, year, ActivityType.InlineSkate) {

    override fun generateHeader(): String {
        return writeCSVLine(
            listOf(
                "Date",
                "Description",
                "DistanceStream (km)",
                "TimeStream",
                "TimeStream (seconds)",
                "Average speed (km/h)",
                "Best 200m (km/h)",
                "Best 400m (km/h)",
                "Best 1000m (km/h)",
                "Best 10000m (km/h)",
                "Best half Marathon (km/h)",
                "Best Marathon (km/h)",
                "Best 30 min (km/h)",
                "Best 1 h (km/h)",
                "Best 2 h (km/h)",
                "Best 3 h (km/h)",
                "Best 4 h (km/h)",
                "Best 5 h (km/h)",
            )
        )
    }

    override fun generateActivities(): String {
        return activities.joinToString("\n") { activity ->
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
                    activity.calculateBestTimeForDistance(10000.0)?.getFormatedSpeed() ?: "",
                    activity.calculateBestTimeForDistance(21097.0)?.getFormatedSpeed() ?: "",
                    activity.calculateBestTimeForDistance(42195.0)?.getFormatedSpeed() ?: "",
                    activity.calculateBestDistanceForTime(30 * 60)?.getFormatedSpeed() ?: "",
                    activity.calculateBestDistanceForTime(60 * 60)?.getFormatedSpeed() ?: "",
                    activity.calculateBestDistanceForTime(2 * 60 * 60)?.getFormatedSpeed() ?: "",
                    activity.calculateBestDistanceForTime(3 * 60 * 60)?.getFormatedSpeed() ?: "",
                    activity.calculateBestDistanceForTime(4 * 60 * 60)?.getFormatedSpeed() ?: "",
                    activity.calculateBestDistanceForTime(5 * 60 * 60)?.getFormatedSpeed() ?: "",
                )
            )
        }
    }

    override fun generateFooter(): String {
        val lastRow = activities.size + 1
        return writeCSVLine(
            listOf(
                ";;" +
                        "=SOMME(\$C2:\$C$lastRow);;" +
                        "=SOMME(\$E2:\$E$lastRow);" +
                        "=MAX(\$F2:\$F$lastRow);" +
                        "=MAX(\$G2:\$G$lastRow);" +
                        "=MAX(\$H2:\$H$lastRow);" +
                        "=MAX(\$I2:\$I$lastRow);" +
                        "=MAX(\$J2:\$J$lastRow);" +
                        "=MAX(\$K2:\$K$lastRow);" +
                        "=MAX(\$L2:\$L$lastRow);" +
                        "=MAX(\$M2:\$M$lastRow);" +
                        "=MAX(\$N2:\$N$lastRow);" +
                        "=MAX(\$O2:\$O$lastRow);" +
                        "=MAX(\$P2:\$P$lastRow);" +
                        "=MAX(\$Q2:\$Q$lastRow);" +
                        "=MAX(\$R2:\$R$lastRow)"
            )
        )
    }
}