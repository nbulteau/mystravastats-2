package me.nicolas.stravastats.adapters.localrepositories.gpx

import io.jenetics.jpx.GPX
import io.jenetics.jpx.Length
import io.jenetics.jpx.Track
import io.jenetics.jpx.TrackSegment
import me.nicolas.stravastats.domain.business.ActivityType
import me.nicolas.stravastats.domain.business.strava.*
import org.slf4j.LoggerFactory
import org.w3c.dom.Document
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

        var name = "Unknown"

        var previousPoint = gpx.tracks()
            .flatMap(Track::segments)
            .flatMap(TrackSegment::points)
            .findFirst().get()

        var totalDistance = 0.0
        var totalElevationGain = 0.0
        gpx.tracks()
            .flatMap(Track::segments)
            .flatMap(TrackSegment::points)
            .forEach { point ->
                val deltaDistance = point.distance(previousPoint).to(Length.Unit.METER)
                totalDistance += deltaDistance

                val elevation = point.elevation.get().to(Length.Unit.METER)
                val previousElevation = previousPoint.elevation.get().to(Length.Unit.METER)
                totalElevationGain += if (elevation > previousElevation) elevation - previousElevation else 0.0

                latitudeLongitude.add(listOf(point.latitude.toDouble(), point.longitude.toDouble()))

                time.add((point.time.get().epochSecond).toInt())

                altitude.add(point.elevation.get().to(Length.Unit.METER))

                distance.add(totalDistance)

                moving.add(deltaDistance > 0)

                previousPoint = point
            }

        val startTime = time.first()
        val totalElapsedTime = time.last() - startTime

        var totalCadence = 0.0
        var totalHeartrate = 0.0
        var cadenceCount = 0
        var heartrateCount = 0

        var type = "Ride"
        val extensions: Optional<Document> = gpx.extensions
        extensions.ifPresent { document ->
            val cadenceNodes = document.getElementsByTagName("cadence")
            for (i in 0 until cadenceNodes.length) {
                totalCadence += cadenceNodes.item(i).textContent.toDouble()
                cadenceCount++
            }

            val heartrateNodes = document.getElementsByTagName("heartrate")
            for (i in 0 until heartrateNodes.length) {
                totalHeartrate += heartrateNodes.item(i).textContent.toDouble()
                heartrateCount++
            }

            type = document.getElementsByTagName("type").item(0).textContent.toActivityType()
        }

        val averageCadence = if (cadenceCount > 0) totalCadence / cadenceCount else 0.0
        val averageHeartrate = if (heartrateCount > 0) totalHeartrate / heartrateCount else 0.0

        val stravaActivity = StravaActivity(
            athlete = AthleteRef(id = athleteId),
            averageSpeed = totalDistance / totalElapsedTime,
            averageCadence = averageCadence,
            averageHeartrate = averageHeartrate,
            maxHeartrate = 0.0,
            averageWatts = 0.0,
            commute = false,
            distance = totalDistance,
            deviceWatts = false,
            elapsedTime = totalElapsedTime,
            elevHigh = totalElevationGain,
            id = 0L,
            kilojoules = 0.0,
            maxSpeed = 0.0,
            movingTime = totalElapsedTime,
            name = name,
            startDate = ZonedDateTime.of(LocalDateTime.ofEpochSecond(startTime.toLong(), 0, ZoneOffset.UTC), ZoneOffset.UTC).toString(),
            startDateLocal = ZonedDateTime.of(LocalDateTime.ofEpochSecond(startTime.toLong(), 0, ZoneOffset.UTC), ZoneOffset.UTC).toString(),
            startLatlng = listOf(),
            totalElevationGain = totalElevationGain,
            type = type,
            uploadId = 0L,
            weightedAverageWatts = 0,
        )
        stravaActivity.stream = Stream(
            latitudeLongitude = LatitudeLongitude(
                data = latitudeLongitude,
                originalSize = 0,
                resolution = "",
                seriesType = "",
            ),
            time = Time(
                data = time,
                originalSize = 0,
                resolution = "",
                seriesType = "",
            ),
            distance = Distance(
                data = distance,
                originalSize = 0,
                resolution = "",
                seriesType = "",
            ),
            altitude = Altitude(
                data = altitude,
                originalSize = 0,
                resolution = "",
                seriesType = "",
            ),
            moving = Moving(
                data = moving,
                originalSize = 0,
                resolution = "",
                seriesType = "",
            ),
            watts = PowerStream(
                data = watts,
                originalSize = 0,
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
