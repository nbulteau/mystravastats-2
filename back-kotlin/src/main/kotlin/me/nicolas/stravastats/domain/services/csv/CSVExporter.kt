package me.nicolas.stravastats.domain.services.csv

import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.business.ActivityType

import java.io.File
import java.io.FileWriter

internal abstract class CSVExporter(
    clientId: String,
    activities: List<StravaActivity>,
    year: Int,
    activityType: ActivityType,
) {

    protected val activities: List<StravaActivity> = activities
        .filter { activity -> activity.type == activityType.name }

    private val writer: FileWriter = FileWriter(File("$clientId-$activityType-$year.csv"))

    fun export(): String {
        // if no activities : nothing to do
        if (activities.isNotEmpty()) {
            writer.use {
                return generateHeader() + generateActivities() + generateFooter()
            }
        }

        return ""
    }

    protected abstract fun generateActivities(): String
    protected abstract fun generateHeader(): String
    protected abstract fun generateFooter(): String

    protected fun writeCSVLine(values: List<String>, customQuote: Char = ' '): String {

        val separators = ';'
        var first = true

        val sb = StringBuilder()
        for (value in values) {
            if (!first) {
                sb.append(separators)
            }
            if (customQuote == ' ') {
                sb.append(followCVSFormat(value))
            } else {
                sb.append(customQuote).append(followCVSFormat(value)).append(customQuote)
            }
            first = false
        }
        sb.append("\n")
        writer.append(sb.toString())

        return sb.toString()
    }

    //https://tools.ietf.org/html/rfc4180
    private fun followCVSFormat(value: String): String {

        var result = value
        result = result.replace("\n", " ")
        if (result.contains("\"")) {
            result = result.replace("\"", "\"\"")
        }
        return result
    }

}


