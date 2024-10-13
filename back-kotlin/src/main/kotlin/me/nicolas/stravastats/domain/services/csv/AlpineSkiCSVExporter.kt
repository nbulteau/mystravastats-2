package me.nicolas.stravastats.domain.services.csv

import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.business.ActivityType

import me.nicolas.stravastats.domain.services.statistics.calculateBestDistanceForTime
import me.nicolas.stravastats.domain.services.statistics.calculateBestElevationForDistance
import me.nicolas.stravastats.domain.services.statistics.calculateBestTimeForDistance
import me.nicolas.stravastats.domain.utils.formatDate
import me.nicolas.stravastats.domain.utils.formatSeconds

internal class AlpineSkiCSVExporter(clientId: String, activities: List<StravaActivity>, year: Int) :
    CSVExporter(clientId, activities, year, ActivityType.AlpineSki) {

    override fun generateHeader(): String {
        return writeCSVLine(
            listOf(
                "Date",
                "Description",
                "Distance (km)",
                "Time",
                "Time (seconds)",
                "Speed (km/h)",
                "Best 250m (km/h)",
                "Best 500m (km/h)",
                "Best 1000m (km/h)",
                "Best 5km (km/h)",
                "Best 10km (km/h)",
                "Best 20km (km/h)",
                "Best 50km (km/h)",
                "Best 100km (km/h)",
                "Best 30 min (km/h)",
                "Best 1 h (km/h)",
                "Best 2 h (km/h)",
                "Best 3 h (km/h)",
                "Best 4 h (km/h)",
                "Best 5 h (km/h)",
                "Max gradient for 250 m (%)",
                "Max gradient for 500 m (%)",
                "Max gradient for 1000 m (%)",
                "Max gradient for 5 km (%)",
                "Max gradient for 10 km (%)",
                "Max gradient for 20 km (%)",
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
                    activity.getSpeed(),
                    activity.calculateBestTimeForDistance(250.0)?.getSpeed() ?: "",
                    activity.calculateBestTimeForDistance(500.0)?.getSpeed() ?: "",
                    activity.calculateBestTimeForDistance(1000.0)?.getSpeed() ?: "",
                    activity.calculateBestTimeForDistance(5000.0)?.getSpeed() ?: "",
                    activity.calculateBestTimeForDistance(10000.0)?.getSpeed() ?: "",
                    activity.calculateBestTimeForDistance(20000.0)?.getSpeed() ?: "",
                    activity.calculateBestTimeForDistance(50000.0)?.getSpeed() ?: "",
                    activity.calculateBestTimeForDistance(100000.0)?.getSpeed() ?: "",
                    activity.calculateBestDistanceForTime(30 * 60)?.getSpeed() ?: "",
                    activity.calculateBestDistanceForTime(60 * 60)?.getSpeed() ?: "",
                    activity.calculateBestDistanceForTime(2 * 60 * 60)?.getSpeed() ?: "",
                    activity.calculateBestDistanceForTime(3 * 60 * 60)?.getSpeed() ?: "",
                    activity.calculateBestDistanceForTime(4 * 60 * 60)?.getSpeed() ?: "",
                    activity.calculateBestDistanceForTime(5 * 60 * 60)?.getSpeed() ?: "",
                    activity.calculateBestElevationForDistance(250.0)?.getGradient() ?: "",
                    activity.calculateBestElevationForDistance(500.0)?.getGradient() ?: "",
                    activity.calculateBestElevationForDistance(1000.0)?.getGradient() ?: "",
                    activity.calculateBestElevationForDistance(5000.0)?.getGradient() ?: "",
                    activity.calculateBestElevationForDistance(10000.0)?.getGradient() ?: "",
                    activity.calculateBestElevationForDistance(20000.0)?.getGradient() ?: "",
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
                        "=MAX(\$R2:\$R$lastRow);" +
                        "=MAX(\$S2:\$S$lastRow);" +
                        "=MAX(\$T2:\$T$lastRow);" +
                        "=MAX(\$U2:\$U$lastRow);" +
                        "=MAX(\$V2:\$V$lastRow);" +
                        "=MAX(\$W2:\$W$lastRow);" +
                        "=MAX(\$X2:\$X$lastRow);" +
                        "=MAX(\$Y2:\$Y$lastRow);" +
                        "=MAX(\$Z2:\$Z$lastRow)"
            )
        )
    }
}