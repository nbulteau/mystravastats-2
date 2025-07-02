package me.nicolas.stravastats.domain.business.strava


import com.fasterxml.jackson.annotation.JsonIgnoreProperties
import com.fasterxml.jackson.annotation.JsonProperty
import me.nicolas.stravastats.domain.business.ActivityType
import me.nicolas.stravastats.domain.business.strava.stream.AltitudeStream
import me.nicolas.stravastats.domain.business.strava.stream.Stream
import me.nicolas.stravastats.domain.utils.formatDate
import me.nicolas.stravastats.domain.utils.formatSeconds
import kotlin.math.abs

@JsonIgnoreProperties(ignoreUnknown = true)
data class StravaActivity(
    val athlete: AthleteRef,
    @param:JsonProperty("average_speed")
    val averageSpeed: Double,
    @param:JsonProperty("average_cadence")
    val averageCadence: Double,
    @param:JsonProperty("average_heartrate")
    val averageHeartrate: Double,
    @param:JsonProperty("max_heartrate")
    val maxHeartrate: Int,
    @param:JsonProperty("average_watts")
    val averageWatts: Int,
    val commute: Boolean,
    val distance: Double,
    @param:JsonProperty("device_watts")
    val deviceWatts: Boolean = false,
    @param:JsonProperty("elapsed_time")
    val elapsedTime: Int,
    @param:JsonProperty("elev_high")
    val elevHigh: Double,
    val id: Long,
    val kilojoules: Double,
    @param:JsonProperty("max_speed")
    val maxSpeed: Float,
    @param:JsonProperty("moving_time")
    val movingTime: Int,
    val name: String,
    @param:JsonProperty("start_date")
    val startDate: String,
    @param:JsonProperty("start_date_local")
    val startDateLocal: String,
    @param:JsonProperty("start_latlng")
    val startLatlng: List<Double>?,
    @param:JsonProperty("total_elevation_gain")
    val totalElevationGain: Double,
    val type: String,
    @param:JsonProperty("upload_id")
    val uploadId: Long,
    @param:JsonProperty("weighted_average_watts")
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
            val deltas = stream?.altitude?.data?.zipWithNext { a, b -> b - a }
            return abs(deltas?.filter { it <= 0 }?.sumOf { it }!!)
        }
        return 0.0
    }

    fun calculateTotalDescentGain(): Double {
        if (stream?.altitude?.data != null) {
            val deltas = stream?.altitude?.data?.zipWithNext { a, b -> b - a }
            return abs(deltas?.filter { it >= 0 }?.sumOf { it }!!)
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