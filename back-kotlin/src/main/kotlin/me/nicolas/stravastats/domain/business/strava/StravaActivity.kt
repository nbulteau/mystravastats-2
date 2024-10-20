package me.nicolas.stravastats.domain.business.strava


import com.fasterxml.jackson.annotation.JsonIgnoreProperties
import com.fasterxml.jackson.annotation.JsonProperty
import me.nicolas.stravastats.domain.business.ActivityType
import me.nicolas.stravastats.domain.business.strava.stream.AltitudeStream
import me.nicolas.stravastats.domain.business.strava.stream.Stream
import me.nicolas.stravastats.domain.services.ActivityHelper.smooth
import me.nicolas.stravastats.domain.utils.formatDate
import me.nicolas.stravastats.domain.utils.formatSeconds

@JsonIgnoreProperties(ignoreUnknown = true)
data class StravaActivity(
    val athlete: AthleteRef,
    @JsonProperty("average_speed")
    val averageSpeed: Double,
    @JsonProperty("average_cadence")
    val averageCadence: Double,
    @JsonProperty("average_heartrate")
    val averageHeartrate: Double,
    @JsonProperty("max_heartrate")
    val maxHeartrate: Int,
    @JsonProperty("average_watts")
    val averageWatts: Int,
    val commute: Boolean,
    val distance: Double,
    @JsonProperty("device_watts")
    val deviceWatts: Boolean = false,
    @JsonProperty("elapsed_time")
    val elapsedTime: Int,
    @JsonProperty("elev_high")
    val elevHigh: Double,
    val id: Long,
    val kilojoules: Double,
    @JsonProperty("max_speed")
    val maxSpeed: Float,
    @JsonProperty("moving_time")
    val movingTime: Int,
    val name: String,
    @JsonProperty("start_date")
    val startDate: String,
    @JsonProperty("start_date_local")
    val startDateLocal: String,
    @JsonProperty("start_latlng")
    val startLatlng: List<Double>?,
    @JsonProperty("total_elevation_gain")
    val totalElevationGain: Double,
    val type: String,
    @JsonProperty("upload_id")
    val uploadId: Long,
    @JsonProperty("weighted_average_watts")
    val weightedAverageWatts: Int,

    var stream: Stream? = null
) {

    override fun toString() = "${name.trim()} (${startDateLocal.formatDate()})"

    fun processAverageSpeed(): String {
        return if (type == ActivityType.Run.name) {
            (elapsedTime * 1000 / distance).formatSeconds()
        } else {
            "%.02f".format(distance / elapsedTime * 3600 / 1000)
        }
    }

    fun calculateTotalAscentGain(): Double {
        if (stream?.altitude?.data != null) {
            val deltas = stream?.altitude?.data?.smooth()?.zipWithNext { a, b -> b - a }
            return deltas?.filter { it <= 0 }?.sumOf { it }!!
        }
        return 0.0
    }

    fun calculateTotalDescentGain(): Double {
        if (stream?.altitude?.data != null) {
            val deltas = stream?.altitude?.data?.smooth()?.zipWithNext { a, b -> b - a }
            return deltas?.filter { it >= 0 }?.sumOf { it }!!
        }
        return 0.0
    }

    fun setStreamAltitude(altitude: AltitudeStream): StravaActivity {
        val updatedStream = this.stream?.copy(altitude = altitude)

        // totalElevationGain
        val deltas = altitude.data.zipWithNext { a, b -> b - a }
        val updatedTotalElevationGain = deltas.filter { it > 0 }.sumOf { it }

        // elevHigh
        val elevHigh = altitude.data.maxOrNull() ?: 0.0
        val updatedElevHigh = elevHigh

        return this.copy(stream = updatedStream, totalElevationGain = updatedTotalElevationGain, elevHigh = updatedElevHigh)
    }
}

@JsonIgnoreProperties(ignoreUnknown = true)
data class AthleteRef(
    val id: Int,
)