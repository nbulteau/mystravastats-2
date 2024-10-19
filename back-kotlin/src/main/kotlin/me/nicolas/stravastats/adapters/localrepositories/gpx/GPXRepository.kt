package me.nicolas.stravastats.adapters.localrepositories.gpx

import io.jenetics.jpx.*
import me.nicolas.stravastats.domain.business.ActivityType
import me.nicolas.stravastats.domain.business.strava.*
import org.slf4j.LoggerFactory
import java.io.File
import java.nio.file.Path
import java.time.LocalDateTime
import java.time.ZoneOffset
import java.time.ZonedDateTime
import java.util.*
import kotlin.math.absoluteValue

// WIP : GPXRepository
class GPXRepository(gpxDirectory: String) {

    private val logger = LoggerFactory.getLogger(GPXRepository::class.java)

    private val cacheDirectory = File(gpxDirectory)

    fun loadActivitiesFromCache(year: Int): List<StravaActivity> {

        val yearActivitiesDirectory = File(cacheDirectory, "$year")

        val gpxFiles = yearActivitiesDirectory
            .listFiles { _, name -> name.lowercase(Locale.getDefault()).endsWith(".gpx") }
        val gpxFilesPath = gpxFiles?.map { file -> file.toPath() }

        val activities: List<StravaActivity> = gpxFilesPath?.mapNotNull { gpxFile ->
            try {
                convertGpxToActivity(gpxFile)
            } catch (exception: Exception) {
                logger.error("Something wrong during GPX conversion: ${exception.message}")
                null
            }
        }?.toList() ?: emptyList()

        return activities
    }

    private fun convertGpxToActivity(gpxFile: Path): StravaActivity {

        val latitudeLongitude = mutableListOf<List<Double>>()
        val time = mutableListOf<Int>()
        val distance = mutableListOf<Double>()
        val altitude = mutableListOf<Double>()
        val moving = mutableListOf<Boolean>()
        val watts = mutableListOf<Int>()

        val gpx = GPX.read(gpxFile)
        val firstTrack = gpx.tracks.first()

        val name: String = firstTrack.name.orElse("Unknown")
        val type: String = firstTrack.type.orElse("cycling").toActivityType()

        var previousPoint: WayPoint = firstTrack.segments.first().points.first()
        val startPointTime = previousPoint.time.get().epochSecond

        var totalDistance = 0.0
        var totalElevationGain = 0.0
        var totalCadence = 0.0
        var totalHeartRate = 0.0
        var maxHeartRate = 0.0
        var maxSpeed = 0.0
        var movingTime = 0.0

        firstTrack.segments
            .flatMap { trackSegment -> trackSegment.points }
            .forEach { point: WayPoint ->
                val deltaDistance = point.distance(previousPoint).to(Length.Unit.METER)
                totalDistance += deltaDistance

                if(point != previousPoint) {
                    val speed = deltaDistance / (point.time.get().epochSecond - previousPoint.time.get().epochSecond)
                    maxSpeed = maxOf(maxSpeed, speed)
                }

                val elevation = point.elevation.get().to(Length.Unit.METER)
                val previousElevation = previousPoint.elevation.get().to(Length.Unit.METER)
                totalElevationGain += if (elevation > previousElevation) elevation - previousElevation else 0.0

                latitudeLongitude.add(listOf(point.latitude.toDouble(), point.longitude.toDouble()))

                time.add((point.time.get().epochSecond - startPointTime).toInt())

                altitude.add(point.elevation.get().to(Length.Unit.METER))

                distance.add(totalDistance)

                if(deltaDistance > 0) {
                    movingTime += point.time.get().toEpochMilli() - previousPoint.time.get().toEpochMilli()
                    moving.add(true)
                } else {
                    moving.add(false)
                }

                point.extensions.ifPresent { extensions ->
                    //
                    val power = extensions.getElementsByTagName("power")
                    watts.add(if (power.length > 0) power.item(0).textContent.toInt() else 0)

                    val cadence = extensions.getElementsByTagName("gpxtpx:cad")
                    totalCadence += if (cadence.length > 0) {
                        cadence.item(0).textContent.toDouble()
                    } else {
                        0.0
                    }

                    val heartRate = extensions.getElementsByTagName("gpxtpx:hr")
                    totalHeartRate += if (heartRate.length > 0) {
                        maxHeartRate = maxOf(maxHeartRate, heartRate.item(0).textContent.toDouble())
                        heartRate.item(0).textContent.toDouble()
                    } else {
                        0.0
                    }
                }

                previousPoint = point
            }

        val totalElapsedTime = time.last()

        val nbPoints = time.size

        val averageCadence = if (totalCadence > 0) totalCadence / nbPoints else 0.0
        val averageHeartbeat = if (totalHeartRate > 0) totalHeartRate / nbPoints else 0.0
        val averageWatts = watts.sum() / nbPoints

        val kilojoules = 0.8604 * averageWatts * totalElapsedTime / 1000

        val stream = buildStream(latitudeLongitude, time, distance, altitude, moving, watts)

        return StravaActivity(
            athlete = AthleteRef(id = 0),
            averageSpeed = totalDistance / totalElapsedTime,
            averageCadence = averageCadence,
            averageHeartrate = averageHeartbeat,
            maxHeartrate = maxHeartRate,
            averageWatts = averageWatts,
            commute = false,
            distance = totalDistance,
            deviceWatts = watts.isNotEmpty(),
            elapsedTime = totalElapsedTime,
            elevHigh = totalElevationGain,
            id = name.hashCode().toLong().absoluteValue,
            kilojoules = kilojoules,
            maxSpeed = maxSpeed,
            movingTime = movingTime.toInt() / 1000,
            name = name,
            startDate = ZonedDateTime.of(
                LocalDateTime.ofEpochSecond(startPointTime, 0, ZoneOffset.UTC),
                ZoneOffset.UTC
            ).toString(),
            startDateLocal = ZonedDateTime.of(
                LocalDateTime.ofEpochSecond(startPointTime, 0, ZoneOffset.UTC),
                ZoneOffset.UTC
            ).toString(),
            startLatlng = listOf(),
            totalElevationGain = totalElevationGain,
            type = type,
            uploadId = 0L,
            weightedAverageWatts = 0,
            stream = stream
        )
    }

    private fun buildStream(
        latitudeLongitude: MutableList<List<Double>>,
        time: MutableList<Int>,
        distance: MutableList<Double>,
        altitude: MutableList<Double>,
        moving: MutableList<Boolean>,
        watts: MutableList<Int>
    ): Stream {
        val stream = Stream(
            latitudeLongitude = LatitudeLongitude(
                data = latitudeLongitude,
                originalSize = latitudeLongitude.size,
                resolution = "high",
                seriesType = "distance",
            ),
            time = Time(
                data = time,
                originalSize = time.size,
                resolution = "high",
                seriesType = "distance",
            ),
            distance = Distance(
                data = distance,
                originalSize = distance.size,
                resolution = "high",
                seriesType = "distance",
            ),
            altitude = Altitude(
                data = altitude,
                originalSize = altitude.size,
                resolution = "high",
                seriesType = "distance",
            ),
            moving = Moving(
                data = moving,
                originalSize = moving.size,
                resolution = "high",
                seriesType = "distance",
            ),
            watts = PowerStream(
                data = watts,
                originalSize = watts.size,
                resolution = "high",
                seriesType = "distance",
            ),
        )
        return stream
    }
}

private fun String.toActivityType(): String {
    return when (this) {
        "cycling" -> ActivityType.Ride.name
        "running" -> ActivityType.Run.name
        else ->this
    }

}
