package me.nicolas.stravastats.domain.business.strava

import com.fasterxml.jackson.annotation.JsonProperty
import kotlin.math.atan2
import kotlin.math.cos
import kotlin.math.sin
import kotlin.math.sqrt

data class LatitudeLongitude(
    // The sequence of altitude values for this stream, in meters
    @JsonProperty("data")
    val `data`: List<List<Double>>,
    // The number of data points in this stream
    @JsonProperty("original_size")
    var originalSize: Int,
    @JsonProperty("resolution")
    val resolution: String,
    @JsonProperty("series_type")
    val seriesType: String,
) {

    data class GpxPoint(val latitude: Double, val longitude: Double)

    private fun isValidPoint(previous: GpxPoint, current: GpxPoint, threshold: Double): Boolean {
        val distance = haversine(previous.latitude, previous.longitude, current.latitude, current.longitude)
        return distance <= threshold
    }

    /**
     * Calculate the distance between two points on the Earth's surface using the Haversine formula.
     */
    private fun haversine(lat1: Double, lon1: Double, lat2: Double, lon2: Double): Double {
        val R = 6371e3 // Earth radius in meters
        val φ1 = Math.toRadians(lat1)
        val φ2 = Math.toRadians(lat2)
        val Δφ = Math.toRadians(lat2 - lat1)
        val Δλ = Math.toRadians(lon2 - lon1)

        val a = sin(Δφ / 2) * sin(Δφ / 2) +
                cos(φ1) * cos(φ2) *
                sin(Δλ / 2) * sin(Δλ / 2)
        val c = 2 * atan2(sqrt(a), sqrt(1 - a))

        return R * c // in meters
    }

    fun correctInconsistentGpxPoints(threshold: Double): List<GpxPoint> {
        val correctedPoints = mutableListOf<GpxPoint>()
        val points = `data`.map { GpxPoint(it[0], it[1]) }
        correctedPoints.add(points.first())
        for (i in 1 until points.size) {
            val previous = correctedPoints.last()
            val current = points[i]
            if (isValidPoint(previous, current, threshold)) {
                correctedPoints.add(current)
            } else {
                // If the point is not valid, we need to correct it
                val correctedLatitude = (previous.latitude + current.latitude) / 2
                val correctedLongitude = (previous.longitude + current.longitude) / 2
                correctedPoints.add(GpxPoint(correctedLatitude, correctedLongitude))
            }
        }
        return correctedPoints
    }
}