package me.nicolas.stravastats.domain.utils

import java.time.LocalDateTime
import java.time.temporal.ChronoUnit

data class GPXPoint(val latitude: Double, val longitude: Double, val elevation: Double, val time: LocalDateTime)

fun interpolateGPX(points: List<GPXPoint>): List<GPXPoint> {
    val interpolatedPoints = mutableListOf<GPXPoint>()
    for (i in 0 until points.size - 1) {
        val start = points[i]
        val end = points[i + 1]
        interpolatedPoints.add(start)

        val timeDiff = ChronoUnit.SECONDS.between(start.time, end.time)
        if (timeDiff > 1) {
            val latDiff = (end.latitude - start.latitude) / timeDiff
            val lonDiff = (end.longitude - start.longitude) / timeDiff
            val eleDiff = (end.elevation - start.elevation) / timeDiff

            for (j in 1 until timeDiff) {
                val interpolatedPoint = GPXPoint(
                    latitude = start.latitude + latDiff * j,
                    longitude = start.longitude + lonDiff * j,
                    elevation = start.elevation + eleDiff * j,
                    time = start.time.plusSeconds(j)
                )
                interpolatedPoints.add(interpolatedPoint)
            }
        }
    }
    interpolatedPoints.add(points.last())
    return interpolatedPoints
}

fun main() {
    val gpxPoints = listOf(
        GPXPoint(48.858844, 2.294351, 35.0, LocalDateTime.parse("2023-10-01T12:00:00")),
        GPXPoint(48.858860, 2.294370, 37.0, LocalDateTime.parse("2023-10-01T12:02:00"))
    )

    val correctedGPXPoints = interpolateGPX(gpxPoints)

    correctedGPXPoints.forEach { point ->
        println("Lat: ${point.latitude}, Lon: ${point.longitude}, Ele: ${point.elevation}, Time: ${point.time}")
    }
}