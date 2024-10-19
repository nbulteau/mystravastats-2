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
                convertGpxToActivity(gpxFile, 0)
            } catch (exception: Exception) {
                logger.error("Something wrong during GPX conversion: ${exception.message}")
                null
            }
        }?.toList() ?: emptyList()

        return activities
    }

    private fun convertGpxToActivity(gpxFile: Path, athleteId: Int): StravaActivity {

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

        var totalDistance = 0.0
        var totalElevationGain = 0.0
        var totalCadence = 0.0
        var totalHeartrate = 0.0
        var maxHeartrate = 0.0
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

                time.add((point.time.get().epochSecond).toInt())

                altitude.add(point.elevation.get().to(Length.Unit.METER))

                distance.add(totalDistance)

                if(deltaDistance > 0) {
                    movingTime += point.time.get().toEpochMilli() - previousPoint.time.get().toEpochMilli()
                    moving.add(true)
                } else {
                    moving.add(false)
                }

                point.extensions.ifPresent { extensions ->
                    val cadence = extensions.getElementsByTagName("gpxtpx:cad")
                    totalCadence += if (cadence.length > 0) {
                        cadence.item(0).textContent.toDouble()
                    } else {
                        0.0
                    }

                    val heartrate = extensions.getElementsByTagName("gpxtpx:hr")
                    totalHeartrate += if (heartrate.length > 0) {
                        maxHeartrate = maxOf(maxHeartrate, heartrate.item(0).textContent.toDouble())
                        heartrate.item(0).textContent.toDouble()
                    } else {
                        0.0
                    }
                }

                previousPoint = point
            }

        val startTime = time.first()
        val totalElapsedTime = time.last() - startTime

        val nbPoints = time.size

        val averageCadence = if (totalCadence > 0) totalCadence / nbPoints else 0.0
        val averageHeartbeat = if (totalHeartrate > 0) totalHeartrate / nbPoints else 0.0

        val stravaActivity = StravaActivity(
            athlete = AthleteRef(id = athleteId),
            averageSpeed = totalDistance / totalElapsedTime,
            averageCadence = averageCadence,
            averageHeartrate = averageHeartbeat,
            maxHeartrate = maxHeartrate,
            averageWatts = 0.0,
            commute = false,
            distance = totalDistance,
            deviceWatts = false,
            elapsedTime = totalElapsedTime,
            elevHigh = totalElevationGain,
            id = 0L,
            kilojoules = 0.0,
            maxSpeed = maxSpeed,
            movingTime = movingTime.toInt() / 1000,
            name = name,
            startDate = ZonedDateTime.of(
                LocalDateTime.ofEpochSecond(startTime.toLong(), 0, ZoneOffset.UTC),
                ZoneOffset.UTC
            ).toString(),
            startDateLocal = ZonedDateTime.of(
                LocalDateTime.ofEpochSecond(startTime.toLong(), 0, ZoneOffset.UTC),
                ZoneOffset.UTC
            ).toString(),
            startLatlng = listOf(),
            totalElevationGain = totalElevationGain,
            type = type,
            uploadId = 0L,
            weightedAverageWatts = 0,
        )
        stravaActivity.stream = Stream(
            latitudeLongitude = LatitudeLongitude(
                data = latitudeLongitude,
                originalSize = latitudeLongitude.size,
                resolution = "",
                seriesType = "",
            ),
            time = Time(
                data = time,
                originalSize = time.size,
                resolution = "",
                seriesType = "",
            ),
            distance = Distance(
                data = distance,
                originalSize = distance.size,
                resolution = "",
                seriesType = "",
            ),
            altitude = Altitude(
                data = altitude,
                originalSize = altitude.size,
                resolution = "",
                seriesType = "",
            ),
            moving = Moving(
                data = moving,
                originalSize = moving.size,
                resolution = "",
                seriesType = "",
            ),
            watts = PowerStream(
                data = watts,
                originalSize = watts.size,
                resolution = "",
                seriesType = "",
            ),
        )

        return stravaActivity
    }
}

private fun String.toActivityType(): String {
    return when (this) {
        "cycling" -> ActivityType.Ride.name
        else -> ActivityType.Ride.name
    }

}
