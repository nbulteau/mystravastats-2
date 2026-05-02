package me.nicolas.stravastats.domain.services.csv

import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.business.ActivityType

import java.util.Locale

internal interface ICSVExporter {
    fun supports(activityType: ActivityType): Boolean
    fun export(clientId: String, activities: List<StravaActivity>, year: Int?): String
}

internal abstract class CSVExporter(
    private val activityType: ActivityType,
) : ICSVExporter {

    override fun supports(activityType: ActivityType): Boolean = this.activityType == activityType

    override fun export(clientId: String, activities: List<StravaActivity>, year: Int?): String {
        val filteredActivities = activities
            .filter { activity -> activity.type == activityType.name }

        // if no activities: nothing to do
        if (filteredActivities.isNotEmpty()) {
            return generateHeader() + generateActivities(filteredActivities) + generateFooter(filteredActivities)
        }

        return ""
    }

    protected abstract fun generateActivities(activities: List<StravaActivity>): String
    protected abstract fun generateHeader(): String
    protected abstract fun generateFooter(activities: List<StravaActivity>): String

    protected fun enrichedHeader(): List<String> = listOf(
        "Activity ID",
        "Type",
        "Sport type",
        "Commute",
        "Gear ID",
        "Strava link",
        "Start date local",
        "Moving time",
        "Moving time (seconds)",
        "Elevation gain (m)",
        "Average heart rate",
        "Max heart rate",
        "Average watts",
        "Weighted average watts",
        "Average cadence",
        "Has GPS stream",
        "Has altitude stream",
        "Has heart rate stream",
        "Has power stream",
        "Data quality flags",
    )

    protected fun StravaActivity.enrichedValues(): List<String> = listOf(
        id.toString(),
        type,
        sportType,
        commute.toCsvValue(),
        gearId.orEmpty(),
        stravaActivityLink(),
        startDateLocal,
        movingTime.formatDuration(),
        movingTime.toString(),
        totalElevationGain.formatCsv("%.0f"),
        averageHeartrate.formatCsv("%.0f"),
        maxHeartrate.toString(),
        averageWatts.toString(),
        weightedAverageWatts.toString(),
        averageCadence.formatCsv("%.1f"),
        hasGpsStream().toCsvValue(),
        hasAltitudeStream().toCsvValue(),
        hasHeartRateStream().toCsvValue(),
        hasPowerStream().toCsvValue(),
        dataQualityFlags().joinToString("|"),
    )

    protected fun writeCSVLine(values: List<String>, customQuote: Char = ' '): String {

        val separators = ';'
        var first = true

        val sb = StringBuilder()
        for (value in values) {
            if (!first) {
                sb.append(separators)
            }
            val formattedValue = followCVSFormat(value)
            if (customQuote == ' ') {
                if (value.contains(separators) || value.contains("\"") || value.contains("\n") || value.contains("\r")) {
                    sb.append('"').append(formattedValue).append('"')
                } else {
                    sb.append(formattedValue)
                }
            } else {
                sb.append(customQuote).append(formattedValue).append(customQuote)
            }
            first = false
        }
        sb.append("\n")

        return sb.toString()
    }

    //https://tools.ietf.org/html/rfc4180
    private fun followCVSFormat(value: String): String {

        var result = value
        result = result.replace("\n", " ")
        result = result.replace("\r", " ")
        if (result.contains("\"")) {
            result = result.replace("\"", "\"\"")
        }
        return result
    }

    private fun Boolean.toCsvValue(): String = if (this) "yes" else "no"

    private fun Int.formatDuration(): String {
        val hours = this / 3600
        val minutes = (this % 3600) / 60
        val seconds = this % 60
        return "%02d:%02d:%02d".format(hours, minutes, seconds)
    }

    private fun Double.formatCsv(format: String): String {
        if (!isFinite()) return ""
        return format.format(Locale.US, this)
    }

    private fun StravaActivity.hasGpsStream(): Boolean =
        stream?.latlng?.data?.isNotEmpty() == true

    private fun StravaActivity.hasAltitudeStream(): Boolean =
        stream?.altitude?.data?.isNotEmpty() == true

    private fun StravaActivity.hasHeartRateStream(): Boolean =
        stream?.heartrate?.data?.isNotEmpty() == true

    private fun StravaActivity.hasPowerStream(): Boolean =
        stream?.watts?.data?.isNotEmpty() == true

    private fun StravaActivity.dataQualityFlags(): List<String> {
        val flags = mutableListOf<String>()
        if (distance <= 0.0) flags += "missing_distance"
        if (elapsedTime <= 0) flags += "missing_elapsed_time"
        if (movingTime <= 0) flags += "missing_moving_time"
        if (movingTime > elapsedTime && elapsedTime > 0) flags += "moving_time_gt_elapsed_time"
        if (!averageSpeed.isFinite()) flags += "invalid_average_speed"

        if (stream == null) {
            flags += "missing_stream"
            return flags
        }
        if (!hasGpsStream()) flags += "missing_gps_stream"
        if (!hasAltitudeStream()) flags += "missing_altitude_stream"
        if (!hasHeartRateStream() && averageHeartrate > 0.0) flags += "missing_heart_rate_stream"
        if (!hasPowerStream() && averageWatts > 0) flags += "missing_power_stream"
        return flags
    }

    private fun StravaActivity.stravaActivityLink(): String =
        if (id != 0L && uploadId != 0L) "https://www.strava.com/activities/$id" else ""

}
