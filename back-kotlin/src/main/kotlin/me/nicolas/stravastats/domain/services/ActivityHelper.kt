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

object ActivityHelper {
    /**
     * Smooth a list of doubles
     * @param size the size of the smoothing window
     * @return a list of smoothed doubles
     */
    fun List<Double>.smooth(size: Int = 5): List<Double> {
        val smooth = DoubleArray(this.size)
        for (i in 0 until size) {
            smooth[i] = this[i]
        }
        for (i in size until this.size - size) {
            smooth[i] = this.subList(i - size, i + size).sum() / (2 * size + 1)
        }
        for (i in this.size - size until this.size) {
            smooth[i] = this[i]
        }

        return smooth.toList()
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

        // Filter segments efforts to keep only the ones with a climbCategory > 0
        val activityEffortsFromSegmentSegments = this.segmentEfforts
            .filter { segmentEffort ->
                segmentEffort.segment.climbCategory > 2
                        || segmentEffort.segment.starred
                        || segmentEffort.prRank != null && segmentEffort.prRank <= 3
            }
            .map { segmentEffort: StravaSegmentEffort ->
                segmentEffort.toActivityEffort()
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

        // Add weeks without activities
        for (week in (1..52)) {
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

        // Add days without activities
        var currentDate = LocalDate.ofYearDay(year, 1)
        (0..(365 + if (currentDate.isLeapYear) 1 else 0)).forEach { _ ->
            currentDate = currentDate.plusDays(1L)
            val dayString =
                "${currentDate.monthValue}".padStart(2, '0') + "-" + "${currentDate.dayOfMonth}".padStart(2, '0')
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


private fun StravaSegmentEffort.toActivityEffort(): ActivityEffort {
    return ActivityEffort(
        distance = this.distance,
        seconds = this.elapsedTime,
        deltaAltitude = this.segment.elevationHigh - this.segment.elevationLow,
        idxStart = this.startIndex,
        idxEnd = this.endIndex,
        averagePower = this.averageWatts.takeIf { it.toInt() != 0 }?.toInt(),
        label = this.segment.name,
        activityShort = ActivityShort(
            id = this.id,
            name = this.segment.name,
            type = this.segment.activityType
        )
    )
}

