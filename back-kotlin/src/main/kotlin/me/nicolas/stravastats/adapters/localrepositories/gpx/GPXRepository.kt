package me.nicolas.stravastats.adapters.localrepositories.gpx

import io.jenetics.jpx.GPX
import io.jenetics.jpx.Length
import io.jenetics.jpx.WayPoint
import me.nicolas.stravastats.domain.business.ActivityType
import me.nicolas.stravastats.domain.business.strava.AthleteRef
import me.nicolas.stravastats.domain.business.strava.stream.PowerStream
import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.business.strava.stream.*
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

        val gpx = GPX.read(gpxFile)

        val firstTrack = gpx.tracks.first()

        var previousPoint: WayPoint = firstTrack.segments.first().points.first()
        val startPointTime = previousPoint.time.get().epochSecond

        var totalDistance = 0.0
        var totalElevationGain = 0.0
        var movingTime = 0.0

        val distance = mutableListOf<Double>()
        val time = mutableListOf<Int>()
        val moving = mutableListOf<Boolean>()
        val altitude = mutableListOf<Double>()
        val latitudeLongitude = mutableListOf<List<Double>>()
        val watts = mutableListOf<Int>()
        val heartRate = mutableListOf<Int>()
        val cadence = mutableListOf<Int>()
        val velocitySmooth = mutableListOf<Float>()
        val gradeSmooth = mutableListOf<Float>()

        gpx.tracks
            .flatMap { track -> track.segments }
            .flatMap { segment -> segment.points }
            .forEach { point: WayPoint ->
                // DistanceStream
                val deltaDistance = point.distance(previousPoint).to(Length.Unit.METER)
                totalDistance += deltaDistance
                distance.add(totalDistance)

                // VelocitySmooth
                val instantSpeed = if (point != previousPoint) {
                    (deltaDistance / (point.time.get().epochSecond - previousPoint.time.get().epochSecond)).toFloat()
                } else {
                    0.0F
                }
                velocitySmooth.add(instantSpeed)

                // Latitude and Longitude
                latitudeLongitude.add(listOf(point.latitude.toDouble(), point.longitude.toDouble()))

                // TimeStream
                time.add((point.time.get().epochSecond - startPointTime).toInt())

                // Elevation
                val elevation = point.elevation.get().to(Length.Unit.METER)
                val previousElevation = previousPoint.elevation.get().to(Length.Unit.METER)
                totalElevationGain += if (elevation > previousElevation) elevation - previousElevation else 0.0
                altitude.add(point.elevation.get().to(Length.Unit.METER))

                // GradeSmooth
                val grade = if (deltaDistance > 0) {
                    ((elevation - previousElevation) / deltaDistance).toFloat()
                } else {
                    0.0F
                }
                gradeSmooth.add(grade)

                // MovingStream
                if (deltaDistance > 0) {
                    movingTime += point.time.get().toEpochMilli() - previousPoint.time.get().toEpochMilli()
                    moving.add(true)
                } else {
                    moving.add(false)
                }

                point.extensions.ifPresent { extensions ->
                    // Power
                    val power = extensions.getElementsByTagName("power")
                    watts.add(if (power.length > 0) power.item(0).textContent.toInt() else 0)

                    // Cadence
                    val cadenceNode = extensions.getElementsByTagName("gpxtpx:cad")
                    cadence.add(if (cadenceNode.length > 0) cadenceNode.item(0).textContent.toInt() else 0)

                    // Heart rate
                    val heartRateNode = extensions.getElementsByTagName("gpxtpx:hr")
                    heartRate.add(if (heartRateNode.length > 0) heartRateNode.item(0).textContent.toInt() else 0)
                }

                previousPoint = point
            }
        val stream = buildStream(
            latitudeLongitude,
            time,
            distance,
            altitude,
            moving,
            watts,
            heartRate,
            cadence,
            velocitySmooth,
            gradeSmooth
        )

        val name: String = firstTrack.name.orElse("Unknown")
        val type: String = firstTrack.type.orElse("cycling").toActivityType()
        val maxHeartRate = if (stream.hasHeartRateStream()) heartRate.max() else 0
        val maxSpeed = if (stream.hasVelocitySmoothStream()) velocitySmooth.max() else 0.0F
        val totalElapsedTime = stream.time.data.last()
        val nbPoints = stream.time.data.size

        val startDate = ZonedDateTime.of(
            LocalDateTime.ofEpochSecond(startPointTime, 0, ZoneOffset.UTC),
            ZoneOffset.UTC
        )
        val startDateLocal = startDate.toLocalDateTime()

        val averageCadence = if (stream.hasCadenceStream()) (stream.cadence!!.data.sum() / nbPoints).toDouble() else 0.0
        val averageHeartbeat =
            if (stream.hasHeartRateStream()) (stream.heartrate!!.data.sum() / nbPoints).toDouble() else 0.0
        val averageWatts = if (stream.hasPowerStream()) {
            val nonNullWatts = stream.watts!!.data.filterNotNull()
            val nbPointsNotNull = nonNullWatts.size
            nonNullWatts.sum() / nbPointsNotNull
        } else {
            0
        }

        val startLatlng = listOf(
            firstTrack.segments.first().points.first().latitude.toDouble(),
            firstTrack.segments.first().points.first().longitude.toDouble()
        )
        val kilojoules = 0.8604 * averageWatts * totalElapsedTime / 1000

        return StravaActivity(
            athlete = AthleteRef(id = 0),
            commute = false,
            averageSpeed = totalDistance / totalElapsedTime,
            averageCadence = averageCadence,
            averageHeartrate = averageHeartbeat,
            maxHeartrate = maxHeartRate,
            averageWatts = averageWatts,
            distance = totalDistance,
            deviceWatts = watts.isNotEmpty(),
            elapsedTime = totalElapsedTime,
            elevHigh = totalElevationGain,
            id = name.hashCode().toLong().absoluteValue,
            kilojoules = kilojoules,
            maxSpeed = maxSpeed,
            movingTime = movingTime.toInt() / 1000,
            name = name,
            startDate = startDate.toString(),
            startDateLocal = startDateLocal.toString(),
            startLatlng = startLatlng,
            totalElevationGain = totalElevationGain,
            type = type,
            uploadId = 0L,
            weightedAverageWatts = 0,
            stream = stream
        )
    }

    private fun buildStream(
        latitudeLongitude: List<List<Double>>,
        time: List<Int>,
        distance: List<Double>,
        altitude: List<Double>,
        moving: List<Boolean>,
        watts: List<Int>,
        heartRate: List<Int>,
        cadence: List<Int>,
        velocitySmooth: List<Float>,
        gradeSmooth: List<Float>,
    ): Stream {
        val altitudeStream = if (altitude.sum() > 0) {
            AltitudeStream(
                data = altitude,
                originalSize = altitude.size,
                resolution = "high",
                seriesType = "distance",
            )
        } else {
            null
        }

        val heartRateStream = if (heartRate.sum() > 0) {
            HeartRateStream(
                data = heartRate,
                originalSize = heartRate.size,
                resolution = "high",
                seriesType = "distance",
            )
        } else {
            null
        }

        val movingStream = if (moving.any { it }) {
            MovingStream(
                data = moving,
                originalSize = moving.size,
                resolution = "high",
                seriesType = "distance",
            )
        } else {
            null
        }

        val cadenceStream = if (cadence.sum() > 0) {
            CadenceStream(
                data = cadence,
                originalSize = cadence.size,
                resolution = "high",
                seriesType = "distance",
            )
        } else {
            null
        }

        val powerStream = if (watts.sum() > 0 ) {
            PowerStream(
                data = watts,
                originalSize = watts.size,
                resolution = "high",
                seriesType = "distance",
            )
        } else {
            null
        }

        val velocitySmoothStream = if (velocitySmooth.sum() > 0) {
            SmoothVelocityStream(
                data = velocitySmooth,
                originalSize = velocitySmooth.size,
                resolution = "high",
                seriesType = "distance",
            )
        } else {
            null
        }

        val gradeStream = if (gradeSmooth.sum() > 0) {
            SmoothGradeStream(
                data = gradeSmooth,
                originalSize = gradeSmooth.size,
                resolution = "high",
                seriesType = "distance",
            )
        } else {
            null
        }

        val stream = Stream(
            latlng = LatLngStream(
                data = latitudeLongitude,
                originalSize = latitudeLongitude.size,
                resolution = "high",
                seriesType = "distance",
            ),
            time = TimeStream(
                data = time,
                originalSize = time.size,
                resolution = "high",
                seriesType = "distance",
            ),
            distance = DistanceStream(
                data = distance,
                originalSize = distance.size,
                resolution = "high",
                seriesType = "distance",
            ),
            altitude = altitudeStream,
            heartrate = heartRateStream,
            moving = movingStream,
            cadence = cadenceStream,
            watts = powerStream,
            velocitySmooth = velocitySmoothStream,
            gradeSmooth = gradeStream,
        )
        return stream
    }
}

private fun String.toActivityType(): String {
    return when (this) {
        "cycling" -> ActivityType.Ride.name
        "running" -> ActivityType.Run.name
        else -> this
    }

}
