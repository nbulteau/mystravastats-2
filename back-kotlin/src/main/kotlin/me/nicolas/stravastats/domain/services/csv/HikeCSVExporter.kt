package me.nicolas.stravastats.domain.services.csv

import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.business.ActivityType

import me.nicolas.stravastats.domain.services.statistics.calculateBestDistanceForTime
import me.nicolas.stravastats.domain.services.statistics.calculateBestElevationForDistance
import me.nicolas.stravastats.domain.services.statistics.calculateBestTimeForDistance
import me.nicolas.stravastats.domain.utils.formatDate
import me.nicolas.stravastats.domain.utils.formatSeconds

internal class HikeCSVExporter(clientId: String, activities: List<StravaActivity>, year: Int) :
    CSVExporter(clientId, activities, year, ActivityType.Hike) {

    override fun generateHeader(): String {
        return writeCSVLine(
            listOf(
                "Date",
                "Description",
                "DistanceStream (km)",
                "TimeStream",
                "TimeStream (seconds)",
                "Average speed (km/h)",
                "Elevation (m)",
                "Highest point (m)",
                "Best 1000m (km/h)",
                "Best 1 h (km/h)",
                "Max gradient for 250 m (%)",
                "Max gradient for 500 m (%)",
                "Max gradient for 1000 m (%)",
                "Max gradient for 5 km (%)",
                "Max gradient for 10 km (%)",
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
                    "%.0f".format(activity.totalElevationGain),
                    "%.0f".format(activity.elevHigh),
                    activity.calculateBestTimeForDistance(1000.0)?.getSpeed() ?: "",
                    activity.calculateBestDistanceForTime(60 * 60)?.getSpeed() ?: "",
                    activity.calculateBestElevationForDistance(250.0)?.getGradient() ?: "",
                    activity.calculateBestElevationForDistance(500.0)?.getGradient() ?: "",
                    activity.calculateBestElevationForDistance(1000.0)?.getGradient() ?: "",
                    activity.calculateBestElevationForDistance(5000.0)?.getGradient() ?: "",
                    activity.calculateBestElevationForDistance(10000.0)?.getGradient() ?: "",
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
                        "=MAX(\$M2:\$M$lastRow)"
            )
        )
    }
}