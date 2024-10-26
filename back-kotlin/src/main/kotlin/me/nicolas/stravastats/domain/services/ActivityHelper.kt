package me.nicolas.stravastats.domain.services

import me.nicolas.stravastats.domain.business.ActivityEffort
import me.nicolas.stravastats.domain.business.ActivityShort
import me.nicolas.stravastats.domain.business.ActivityType
import me.nicolas.stravastats.domain.business.strava.Gear
import me.nicolas.stravastats.domain.business.strava.MetaActivity
import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.business.strava.StravaDetailedActivity
import me.nicolas.stravastats.domain.business.strava.StravaSegmentEffort
import me.nicolas.stravastats.domain.services.statistics.calculateBestDistanceForTime
import me.nicolas.stravastats.domain.services.statistics.calculateBestElevationForDistance
import me.nicolas.stravastats.domain.services.statistics.calculateBestTimeForDistance

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
     * Remove activities that are not in the list of stravaActivity types to consider (i.e. Run, Ride, Hike, etc.)
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

        val activityEfforts = listOf(
            this.calculateBestTimeForDistance(1000.0),
            this.calculateBestTimeForDistance(5000.0),
            this.calculateBestTimeForDistance(10000.0),
            this.calculateBestDistanceForTime(60 * 60),
            this.calculateBestElevationForDistance(500.0),
            this.calculateBestElevationForDistance(1000.0),
            this.calculateBestElevationForDistance(10000.0)
        )
            .filterNotNull()

        return activityEfforts + activityEffortsFromSegmentSegments
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
        gear = Gear(id = "", distance = 0, convertedDistance = 0.0, name = "", nickname = "", primary = false, retired = false),
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

