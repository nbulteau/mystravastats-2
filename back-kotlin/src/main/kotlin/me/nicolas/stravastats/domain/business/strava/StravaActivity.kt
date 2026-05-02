package me.nicolas.stravastats.domain.business.strava


import com.fasterxml.jackson.annotation.JsonIgnoreProperties
import com.fasterxml.jackson.annotation.JsonProperty
import me.nicolas.stravastats.domain.business.strava.stream.AltitudeStream
import me.nicolas.stravastats.domain.business.strava.stream.Stream
import me.nicolas.stravastats.domain.utils.formatDate
import me.nicolas.stravastats.domain.utils.formatSeconds
import java.util.Locale
import kotlin.math.abs

@JsonIgnoreProperties(ignoreUnknown = true)
data class StravaActivity(
    val athlete: AthleteRef,
    @param:JsonProperty("average_speed")
    val averageSpeed: Double,
    @param:JsonProperty("average_cadence")
    val averageCadence: Double = 0.0,
    @param:JsonProperty("average_heartrate")
    val averageHeartrate: Double = 0.0,
    @param:JsonProperty("max_heartrate")
    val maxHeartrate: Int = 0,
    @param:JsonProperty("average_watts")
    val averageWatts: Int = 0,
    val commute: Boolean,
    val distance: Double,
    @param:JsonProperty("device_watts")
    val deviceWatts: Boolean = false,
    @param:JsonProperty("elapsed_time")
    val elapsedTime: Int,
    @param:JsonProperty("elev_high")
    val elevHigh: Double = 0.0,
    val id: Long,
    val kilojoules: Double = 0.0,
    @param:JsonProperty("max_speed")
    val maxSpeed: Float,
    @param:JsonProperty("moving_time")
    val movingTime: Int,
    val name: String,
    @param:JsonProperty("sport_type")
    private val _sportType: String? = null,
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
    val weightedAverageWatts: Int = 0,
    @param:JsonProperty("gear_id")
    val gearId: String? = null,

    val stream: Stream? = null,
) {
    val sportType: String
        get() = _sportType ?: type

    override fun toString() = "${name.trim()} (${startDateLocal.formatDate()})"

    fun processAverageSpeed(): String {
        return if (type.endsWith("Run")) {
            (elapsedTime * 1000 / distance).formatSeconds()
        } else {
            "%.02f".format(Locale.FRANCE, distance / elapsedTime * 3600 / 1000)
        }
    }

    fun calculateTotalAscentGain(): Double {
        val altitudeData = stream?.altitude?.data ?: return 0.0
        val deltas = altitudeData.zipWithNext { a, b -> b - a }
        return abs(deltas.filter { it <= 0 }.sumOf { it })
    }

    fun calculateTotalDescentGain(): Double {
        val altitudeData = stream?.altitude?.data ?: return 0.0
        val deltas = altitudeData.zipWithNext { a, b -> b - a }
        return abs(deltas.filter { it >= 0 }.sumOf { it })
    }

    fun setStreamAltitude(altitude: AltitudeStream): StravaActivity {
        val updatedStream = this.stream?.copy(altitude = altitude)

        // Recompute total elevation gain from altitude deltas
        val deltas = altitude.data.zipWithNext { a, b -> b - a }
        val updatedTotalElevationGain = deltas.filter { it > 0 }.sumOf { it }

        // Recompute highest point
        val newElevHigh = altitude.data.maxOrNull() ?: 0.0

        return this.copy(stream = updatedStream, totalElevationGain = updatedTotalElevationGain, elevHigh = newElevHigh)
    }
}

@JsonIgnoreProperties(ignoreUnknown = true)
data class AthleteRef(
    val id: Int,
)
