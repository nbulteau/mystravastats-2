package me.nicolas.stravastats.domain.business.strava

import com.fasterxml.jackson.annotation.JsonIgnoreProperties
import com.fasterxml.jackson.annotation.JsonProperty
import me.nicolas.stravastats.domain.business.strava.stream.Stream

@JsonIgnoreProperties(ignoreUnknown = true)
data class StravaDetailedActivity(
    @param:JsonProperty("achievement_count")
    val achievementCount: Int,
    val athlete: MetaActivity,
    @param:JsonProperty("athlete_count")
    val athleteCount: Int,
    @param:JsonProperty("average_cadence")
    val averageCadence: Double,
    @param:JsonProperty("average_heartrate")
    val averageHeartrate: Double,
    @param:JsonProperty("average_speed")
    val averageSpeed: Double,
    @param:JsonProperty("average_temp")
    val averageTemp: Int,
    @param:JsonProperty("average_watts")
    val averageWatts: Double,
    val calories: Double,
    @param:JsonProperty("comment_count")
    val commentCount: Int,
    val commute: Boolean,
    val description: String?,
    @param:JsonProperty("device_name")
    val deviceName: String?,
    @param:JsonProperty("device_watts")
    val deviceWatts: Boolean,
    val distance: Int,
    @param:JsonProperty("elapsed_time")
    val elapsedTime: Int,
    @param:JsonProperty("elev_high")
    val elevHigh: Double,
    @param:JsonProperty("elev_low")
    val elevLow: Double,
    @param:JsonProperty("embed_token")
    val embedToken: String,
    @param:JsonProperty("end_latlng")
    val endLatLng: List<Double>,
    @param:JsonProperty("external_id")
    val externalId: String,
    val flagged: Boolean,
    @param:JsonProperty("from_accepted_tag")
    val fromAcceptedTag: Boolean,
    val gear: Gear?,
    @param:JsonProperty("gear_id")
    val gearId: String?,
    @param:JsonProperty("has_heartrate")
    val hasHeartRate: Boolean,
    @param:JsonProperty("has_kudoed")
    val hasKudoed: Boolean,
    @param:JsonProperty("hide_from_home")
    val hideFromHome: Boolean,
    val id: Long,
    val kilojoules: Double,
    @param:JsonProperty("kudos_count")
    val kudosCount: Int,
    @param:JsonProperty("leaderboard_opt_out")
    val leaderboardOptOut: Boolean,
    @param:JsonProperty("map")
    val map: GeoMap?,
    val manual: Boolean,
    @param:JsonProperty("max_heartrate")
    val maxHeartrate: Int,
    @param:JsonProperty("max_speed")
    val maxSpeed: Double,
    @param:JsonProperty("max_watts")
    val maxWatts: Int,
    @param:JsonProperty("moving_time")
    val movingTime: Int,
    val name: String,
    @param:JsonProperty("pr_count")
    val prCount: Int,
    @param:JsonProperty("private")
    val isPrivate: Boolean,
    @param:JsonProperty("resource_state")
    val resourceState: Int,
    @param:JsonProperty("segment_efforts")
    val segmentEfforts: List<StravaSegmentEffort>,
    @param:JsonProperty("segment_leaderboard_opt_out")
    val segmentLeaderboardOptOut: Boolean,
    @param:JsonProperty("splits_metric")
    val splitsMetric: List<SplitsMetric>,
    @param:JsonProperty("sport_type")
    val sportType: String,
    @param:JsonProperty("start_date")
    val startDate: String,
    @param:JsonProperty("start_date_local")
    val startDateLocal: String,
    @param:JsonProperty("start_latlng")
    val startLatLng: List<Double>,
    @param:JsonProperty("suffer_score")
    val sufferScore: Double?,
    val timezone: String,
    @param:JsonProperty("total_elevation_gain")
    val totalElevationGain: Int,
    @param:JsonProperty("total_photo_count")
    val totalPhotoCount: Int,
    val trainer: Boolean,
    val type: String,
    @param:JsonProperty("upload_id")
    val uploadId: Long,
    @param:JsonProperty("utc_offset")
    val utcOffset: Int,
    @param:JsonProperty("weighted_average_watts")
    val weightedAverageWatts: Int,
    @param:JsonProperty("workout_type")
    val workoutType: Int,

    var stream: Stream? = null
)