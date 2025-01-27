package me.nicolas.stravastats.domain.business.strava.stream

import com.fasterxml.jackson.annotation.JsonIgnoreProperties
import com.fasterxml.jackson.annotation.JsonProperty

@JsonIgnoreProperties(ignoreUnknown = true)
data class Stream(
    val distance: DistanceStream,
    val time: TimeStream,
    val latlng: LatLngStream? = null,
    val cadence: CadenceStream? = null,
    val heartrate: HeartRateStream? = null,
    val moving: MovingStream? = null,
    val altitude: AltitudeStream? = null,
    val watts: PowerStream? = null,
    @JsonProperty("velocity_smooth")
    val velocitySmooth: SmoothVelocityStream? = null,
    @JsonProperty("grade_smooth")
    val gradeSmooth: SmoothGradeStream? = null,
) {
    fun hasLatLngStream() = latlng != null
    fun hasPowerStream() = watts != null
    fun hasAltitudeStream() = altitude != null
    fun hasMovingStream() = moving != null
    fun hasVelocitySmoothStream() = velocitySmooth != null
    fun hasGradeSmoothStream() = gradeSmooth != null
    fun hasHeartRateStream() = heartrate != null
    fun hasCadenceStream() = cadence != null
}