package me.nicolas.stravastats.domain.business


import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.business.strava.Stream
import me.nicolas.stravastats.domain.services.statistics.calculateBestDistanceForTime
import me.nicolas.stravastats.domain.services.statistics.calculateBestElevationForDistance
import me.nicolas.stravastats.domain.services.statistics.calculateBestTimeForDistance

data class DetailedActivity(
    val averageSpeed: Double,
    val averageCadence: Double,
    val averageHeartrate: Double,
    val maxHeartrate: Double,
    val averageWatts: Double,
    val commute: Boolean,
    var distance: Double,
    val deviceWatts: Boolean = false,
    var elapsedTime: Int,
    val elevHigh: Double,
    val id: Long,
    val kilojoules: Double,
    val maxSpeed: Double,
    val movingTime: Int,
    val name: String,
    val startDate: String,
    val startDateLocal: String,
    val startLatlng: List<Double>?,
    val totalElevationGain: Double,
    val totalDescent: Double,
    val type: String,
    val weightedAverageWatts: Int,
    val stream: Stream? = null,
    val activityEfforts: Map<String, ActivityEffort?>
) {

    constructor(activity: StravaActivity) : this(
        averageSpeed = activity.averageSpeed,
        averageCadence = activity.averageCadence,
        averageHeartrate = activity.averageHeartrate,
        maxHeartrate = activity.maxHeartrate,
        averageWatts = activity.averageWatts,
        commute = activity.commute,
        distance = activity.distance,
        deviceWatts = activity.deviceWatts,
        elapsedTime = activity.elapsedTime,
        elevHigh = activity.elevHigh,
        id = activity.id,
        kilojoules = activity.kilojoules,
        maxSpeed = activity.maxSpeed,
        movingTime = activity.movingTime,
        name = activity.name,
        startDate = activity.startDate,
        startDateLocal = activity.startDateLocal,
        startLatlng = activity.startLatlng,
        totalElevationGain = activity.totalElevationGain,
        totalDescent = activity.calculateTotalDescentGain(),
        type = activity.type,
        weightedAverageWatts = activity.weightedAverageWatts,
        stream = activity.stream,
        activityEfforts = mapOf(
            "Best speed for 1 000m" to activity.calculateBestTimeForDistance(1000.0),
            "Best speed for 5 000m" to activity.calculateBestTimeForDistance(5000.0),
            "Best speed for 10 000m" to activity.calculateBestTimeForDistance(10000.0),
            "Best distance for 1h" to activity.calculateBestDistanceForTime(60 * 60),
            "Best elevation for 500m" to activity.calculateBestElevationForDistance(500.0),
            "Best elevation for 1 000m" to activity.calculateBestElevationForDistance(1000.0),
            "Best elevation for 10 000m" to activity.calculateBestElevationForDistance(10000.0)
        ),
    )
}