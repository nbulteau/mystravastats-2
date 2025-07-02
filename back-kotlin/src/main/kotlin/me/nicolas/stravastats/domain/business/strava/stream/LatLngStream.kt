package me.nicolas.stravastats.domain.business.strava.stream

import com.fasterxml.jackson.annotation.JsonProperty
import kotlin.math.atan2
import kotlin.math.cos
import kotlin.math.sin
import kotlin.math.sqrt

data class LatLngStream(
    // The sequence of lat/long values for this stream
    @param:JsonProperty("data")
    val `data`: List<List<Double>>,
    // The number of data points in this stream
    @param:JsonProperty("original_size")
    var originalSize: Int,
    @param:JsonProperty("resolution")
    val resolution: String,
    @param:JsonProperty("series_type")
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
        val earthRadius = 6371e3 // Earth radius in meters
        val lat1InRadian = Math.toRadians(lat1) // Latitude in radians
        val lat2InRadian = Math.toRadians(lat2) // Latitude in radians
        val differenceInLatitude = Math.toRadians(lat2 - lat1) // Difference in latitude
        val differenceInLongitude = Math.toRadians(lon2 - lon1) // Difference in longitude

        val a = sin(differenceInLatitude / 2) * sin(differenceInLatitude / 2) +
                cos(lat1InRadian) * cos(lat2InRadian) *
                sin(differenceInLongitude / 2) * sin(differenceInLongitude / 2)
        val c = 2 * atan2(sqrt(a), sqrt(1 - a))

        return earthRadius * c // in meters
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