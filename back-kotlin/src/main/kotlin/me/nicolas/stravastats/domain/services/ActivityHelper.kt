package me.nicolas.stravastats.domain.services

import me.nicolas.stravastats.domain.business.ActivityEffort
import me.nicolas.stravastats.domain.business.ActivityShort
import me.nicolas.stravastats.domain.business.ActivityType
import me.nicolas.stravastats.domain.business.SlopeType
import me.nicolas.stravastats.domain.business.strava.*
import me.nicolas.stravastats.domain.services.statistics.calculateBestDistanceForTime
import me.nicolas.stravastats.domain.services.statistics.calculateBestElevationForDistance
import me.nicolas.stravastats.domain.services.statistics.calculateBestTimeForDistance
import me.nicolas.stravastats.domain.utils.inDateTimeFormatter
import java.time.LocalDate
import java.time.LocalDateTime
import java.time.Month
import java.time.format.TextStyle
import java.time.temporal.WeekFields
import java.util.*
import kotlin.math.abs

object ActivityHelper {
    /**
     * Smooth a list of doubles using a centered sliding window.
     * @param size radius of the smoothing window (number of elements on each side). If <= 0 the original list is returned.
     * @return a list of smoothed doubles
     */
    fun List<Double>.smooth(size: Int = 2): List<Double> {
        if (size <= 0) return this.toList()
        if (this.isEmpty()) return emptyList()

        val n = this.size
        val result = MutableList(n) { 0.0 }

        for (i in 0 until n) {
            val left = maxOf(0, i - size)
            val right = minOf(n - 1, i + size)
            var sum = 0.0
            for (j in left..right) {
                sum += this[j]
            }
            result[i] = sum / (right - left + 1)
        }

        return result
    }

    /**
     * Remove activities that are not in the list of activity types to consider (i.e.: Run, Ride, Hike, etc.)
     * @return a list of activities filtered by type
     * @see StravaActivity
     */
    fun List<StravaActivity>.filterByActivityTypes() = this.filter { activity ->
        ActivityType.entries.any { activity.type == it.name }
    }

    fun StravaDetailedActivity.buildActivityEfforts(): List<ActivityEffort> {

        // Filter segment efforts to keep only the ones with a climbCategory > 2 (hard climbs),
        // or starred segments, or top PR ranks
        val activityEffortsFromSegmentSegments = this.segmentEfforts
            .filter { segmentEffort ->
                segmentEffort.segment.climbCategory > 2
                        || segmentEffort.segment.starred
                        || segmentEffort.prRank != null && segmentEffort.prRank <= 3
            }
            .map { segmentEffort: StravaSegmentEffort ->
                segmentEffort.toActivityEffort(this)
            }

        val activityEfforts = listOfNotNull(
            this.calculateBestTimeForDistance(1000.0),
            this.calculateBestTimeForDistance(5000.0),
            this.calculateBestTimeForDistance(10000.0),
            this.calculateBestDistanceForTime(60 * 60),
            this.calculateBestElevationForDistance(500.0),
            this.calculateBestElevationForDistance(1000.0),
            this.calculateBestElevationForDistance(10000.0)
        )

        val slopes = this.stream?.listSlopes()?.filter { slope -> slope.type == SlopeType.ASCENT } ?: emptyList()
        val slopesEfforts = slopes.mapIndexed { index, slope ->
            ActivityEffort(
                distance = slope.distance,
                seconds = slope.duration,
                deltaAltitude = slope.endAltitude - slope.startAltitude,
                idxStart = slope.startIndex,
                idxEnd = slope.endIndex,
                averagePower = 0,
                label = "Slope: $index - max gradient ${String.format("%.1f", slope.maxGrade)} %",
                activityShort = ActivityShort(id = this.id, name = this.name, type = this.sportType)
            )
        }

        return slopesEfforts + activityEfforts + activityEffortsFromSegmentSegments
    }

    /**
     * Group activities by month
     * @param activities list of activities
     * @return a map with the month as a key and the list of activities as a value
     * @see StravaActivity
     */
    fun groupActivitiesByMonth(activities: List<StravaActivity>): Map<String, List<StravaActivity>> {
        val activitiesByMonth =
            activities.groupBy { activity -> activity.startDateLocal.subSequence(5, 7).toString() }.toMutableMap()

        // Add months without activities
        for (month in (1..12)) {
            if (!activitiesByMonth.contains("$month".padStart(2, '0'))) {
                activitiesByMonth["$month".padStart(2, '0')] = emptyList()
            }
        }

        return activitiesByMonth.toSortedMap().mapKeys { (key, _) ->
            Month.of(key.toInt()).getDisplayName(TextStyle.FULL_STANDALONE, Locale.getDefault())
        }.toMap()
    }

    /**
     * Group activities by week
     * @param activities list of activities
     * @return a map with the week as a key and the list of activities as value
     * @see StravaActivity
     */
    fun groupActivitiesByWeek(activities: List<StravaActivity>): Map<String, List<StravaActivity>> {

        val activitiesByWeek = activities.groupBy { activity ->
            val week = LocalDateTime.parse(activity.startDateLocal, inDateTimeFormatter)
                .get(WeekFields.of(Locale.getDefault()).weekOfYear())
            "$week".padStart(2, '0')
        }.toMutableMap()

        // Add weeks without activities (use 1..53 to cover years with 53 weeks)
        for (week in (1..53)) {
            if (!activitiesByWeek.contains("$week".padStart(2, '0'))) {
                activitiesByWeek["$week".padStart(2, '0')] = emptyList()
            }
        }

        return activitiesByWeek.toSortedMap()
    }

    /**
     * Group activities by day
     * @param activities list of activities
     * @return a map with the day as a key and the list of activities as a value
     * @see StravaActivity
     */
    fun groupActivitiesByDay(activities: List<StravaActivity>, year: Int): Map<String, List<StravaActivity>> {
        val activitiesByDay =
            activities.groupBy { activity -> activity.startDateLocal.subSequence(5, 10).toString() }.toMutableMap()

        // Add days without activities: iterate every day of the given year
        val daysInYear = if (LocalDate.of(year, 1, 1).isLeapYear) 366 else 365
        for (dayOfYear in 1..daysInYear) {
            val date = LocalDate.ofYearDay(year, dayOfYear)
            val dayString = "${date.monthValue}".padStart(2, '0') + "-" + "${date.dayOfMonth}".padStart(2, '0')
            if (!activitiesByDay.containsKey(dayString)) {
                activitiesByDay[dayString] = emptyList()
            }
        }

        return activitiesByDay.toSortedMap()
    }
}

fun StravaActivity.toStravaDetailedActivity(): StravaDetailedActivity {
    return StravaDetailedActivity(
        achievementCount = 0,
        athlete = MetaActivity(id = 0),
        athleteCount = 1,
        averageCadence = this.averageCadence,
        averageHeartrate = this.averageHeartrate,
        averageSpeed = this.averageSpeed,
        averageTemp = 0,
        averageWatts = this.averageWatts.toDouble(),
        calories = 0.0,
        commentCount = 0,
        commute = this.commute,
        description = "",
        deviceName = null,
        deviceWatts = this.deviceWatts,
        distance = this.distance.toInt(),
        elapsedTime = this.elapsedTime,
        elevHigh = this.elevHigh,
        elevLow = 0.0,
        embedToken = "",
        endLatLng = listOf(),
        externalId = "",
        flagged = false,
        fromAcceptedTag = false,
        gear = Gear(
            id = "",
            distance = 0,
            convertedDistance = 0.0,
            name = "",
            nickname = "",
            primary = false,
            retired = false
        ),
        gearId = "",
        hasHeartRate = true,
        hasKudoed = false,
        hideFromHome = false,
        id = this.id,
        kilojoules = this.kilojoules,
        kudosCount = 0,
        leaderboardOptOut = false,
        map = null,
        manual = false,
        maxHeartrate = this.maxHeartrate,
        maxSpeed = this.maxSpeed.toDouble(),
        maxWatts = 0,
        movingTime = this.movingTime,
        name = this.name,
        prCount = 0,
        isPrivate = false,
        resourceState = 0,
        segmentEfforts = listOf(),
        segmentLeaderboardOptOut = false,
        splitsMetric = listOf(),
        sportType = this.type,
        startDate = this.startDate,
        startDateLocal = this.startDateLocal,
        startLatLng = this.startLatlng ?: listOf(),
        sufferScore = null,
        timezone = "",
        totalElevationGain = this.totalElevationGain.toInt(),
        totalPhotoCount = 0,
        trainer = false,
        type = this.type,
        uploadId = this.uploadId,
        utcOffset = 0,
        weightedAverageWatts = this.weightedAverageWatts,
        workoutType = 0,
        stream = this.stream
    )
}


private fun StravaSegmentEffort.toActivityEffort(activity: StravaDetailedActivity): ActivityEffort {
    val direction = resolveSegmentEffortDirection(this, activity)
    val label = directionAwareSegmentLabel(this.segment.name, direction)
    val deltaAltitude = resolveSegmentEffortDeltaAltitude(this, activity, direction)

    return ActivityEffort(
        distance = this.distance,
        seconds = this.elapsedTime,
        deltaAltitude = deltaAltitude,
        idxStart = this.startIndex,
        idxEnd = this.endIndex,
        averagePower = this.averageWatts.takeIf { it.toInt() != 0 }?.toInt(),
        label = label,
        activityShort = ActivityShort(
            id = this.id,
            name = label,
            type = this.segment.activityType
        )
    )
}

private const val SEGMENT_DIRECTION_MIN_ALTITUDE_DELTA_M = 3.0
private const val SEGMENT_DIRECTION_MIN_GRADE_PERCENT = 0.5

private enum class SegmentEffortDirection {
    ASCENT,
    DESCENT,
    UNKNOWN,
}

private val ascentDirectionKeywords = listOf(
    "montee",
    "ascent",
    "climb",
    "uphill",
)

private val descentDirectionKeywords = listOf(
    "descente",
    "descent",
    "downhill",
)

private fun resolveSegmentEffortDirection(
    effort: StravaSegmentEffort,
    activity: StravaDetailedActivity,
): SegmentEffortDirection {
    return resolveDirectionFromAltitudeStream(activity, effort)
        ?: resolveDirectionFromLabels(effort.name, effort.segment.name)
        ?: resolveDirectionFromAverageGrade(effort.segment.averageGrade)
}

private fun resolveDirectionFromAltitudeStream(
    activity: StravaDetailedActivity,
    effort: StravaSegmentEffort,
): SegmentEffortDirection? {
    val altitude = activity.stream?.altitude?.data ?: return null
    if (altitude.isEmpty()) return null
    if (effort.startIndex < 0 || effort.endIndex < 0) return null
    if (effort.startIndex >= altitude.size || effort.endIndex >= altitude.size || effort.startIndex == effort.endIndex) {
        return null
    }

    val altitudeDelta = altitude[effort.endIndex] - altitude[effort.startIndex]
    if (!altitudeDelta.isFinite()) {
        return null
    }
    if (abs(altitudeDelta) < SEGMENT_DIRECTION_MIN_ALTITUDE_DELTA_M) {
        return null
    }
    return if (altitudeDelta > 0.0) SegmentEffortDirection.ASCENT else SegmentEffortDirection.DESCENT
}

private fun resolveDirectionFromLabels(vararg labels: String): SegmentEffortDirection? {
    labels.forEach { rawLabel ->
        val label = normalizeDirectionLabel(rawLabel)
        if (label.isBlank()) {
            return@forEach
        }
        if (descentDirectionKeywords.any { keyword -> label.contains(keyword) }) {
            return SegmentEffortDirection.DESCENT
        }
        if (ascentDirectionKeywords.any { keyword -> label.contains(keyword) }) {
            return SegmentEffortDirection.ASCENT
        }
    }
    return null
}

private fun normalizeDirectionLabel(label: String): String {
    if (label.isBlank()) {
        return ""
    }

    return label
        .lowercase(Locale.getDefault())
        .replace("é", "e")
        .replace("è", "e")
        .replace("ê", "e")
        .replace("ë", "e")
        .replace("à", "a")
        .replace("â", "a")
        .replace("ä", "a")
        .replace("î", "i")
        .replace("ï", "i")
        .replace("ô", "o")
        .replace("ö", "o")
        .replace("ù", "u")
        .replace("û", "u")
        .replace("ü", "u")
        .replace("ç", "c")
        .replace("’", "'")
        .replace("-", " ")
        .replace(Regex("\\s+"), " ")
        .trim()
}

private fun resolveDirectionFromAverageGrade(averageGrade: Double): SegmentEffortDirection {
    if (!averageGrade.isFinite()) {
        return SegmentEffortDirection.UNKNOWN
    }
    if (abs(averageGrade) < SEGMENT_DIRECTION_MIN_GRADE_PERCENT) {
        return SegmentEffortDirection.UNKNOWN
    }
    return if (averageGrade > 0.0) SegmentEffortDirection.ASCENT else SegmentEffortDirection.DESCENT
}

private fun resolveSegmentEffortDeltaAltitude(
    effort: StravaSegmentEffort,
    activity: StravaDetailedActivity,
    direction: SegmentEffortDirection,
): Double {
    val altitude = activity.stream?.altitude?.data
    if (altitude != null
        && effort.startIndex >= 0
        && effort.endIndex >= 0
        && effort.startIndex < altitude.size
        && effort.endIndex < altitude.size
    ) {
        val altitudeDelta = altitude[effort.endIndex] - altitude[effort.startIndex]
        if (altitudeDelta.isFinite()) {
            return altitudeDelta
        }
    }

    val segmentDelta = effort.segment.elevationHigh - effort.segment.elevationLow
    if (!segmentDelta.isFinite()) {
        return 0.0
    }
    return when (direction) {
        SegmentEffortDirection.ASCENT -> abs(segmentDelta)
        SegmentEffortDirection.DESCENT -> -abs(segmentDelta)
        SegmentEffortDirection.UNKNOWN -> segmentDelta
    }
}

private fun directionAwareSegmentLabel(
    baseLabel: String,
    direction: SegmentEffortDirection,
): String {
    val normalized = baseLabel.lowercase(Locale.getDefault())
    return when (direction) {
        SegmentEffortDirection.ASCENT -> if (normalized.contains("(ascent)")) baseLabel else "$baseLabel (ascent)"
        SegmentEffortDirection.DESCENT -> if (normalized.contains("(descent)")) baseLabel else "$baseLabel (descent)"
        SegmentEffortDirection.UNKNOWN -> baseLabel
    }
}
