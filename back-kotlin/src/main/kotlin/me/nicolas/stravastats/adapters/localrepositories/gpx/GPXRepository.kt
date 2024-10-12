package me.nicolas.stravastats.adapters.localrepositories.gpx

import me.nicolas.stravastats.adapters.srtm.SRTMProvider
import me.nicolas.stravastats.domain.business.strava.*
import me.nicolas.stravastats.domain.interfaces.ISRTMProvider
import org.slf4j.LoggerFactory
import org.w3c.dom.Document
import org.w3c.dom.Element
import java.io.File
import java.util.*
import javax.xml.bind.DatatypeConverter
import javax.xml.parsers.DocumentBuilderFactory

// WIP : GPXRepository
class GPXRepository(gpxDirectory: String) {

    private val logger = LoggerFactory.getLogger(GPXRepository::class.java)

    private val srtmProvider: ISRTMProvider = SRTMProvider()

    private val cacheDirectory = File(gpxDirectory)

    fun loadActivitiesFromCache(year: Int): List<Activity> {

        val yearActivitiesDirectory = File(cacheDirectory, "$year")
        val fitFiles = yearActivitiesDirectory.listFiles { file ->
            file.extension.lowercase(Locale.getDefault()) == "fit"
        }
        val activities: List<Activity> = fitFiles?.mapNotNull { fitFile ->
            try {
                convertGpxToActivity(fitFile, 0)
            } catch (exception: Exception) {
                logger.error("Something wrong during FIT conversion: ${exception.message}")
                null
            }
        }?.toList() ?: emptyList()

        return activities
    }

    private fun convertGpxToActivity(gpxFile: File, athleteId: Int): Activity {
        val document: Document = DocumentBuilderFactory.newInstance().newDocumentBuilder().parse(gpxFile)
        document.documentElement.normalize()

        val trk = document.getElementsByTagName("trk").item(0) as Element
        val name = trk.getElementsByTagName("name").item(0).textContent
        val trkseg = trk.getElementsByTagName("trkseg").item(0) as Element
        val trkpts = trkseg.getElementsByTagName("trkpt")

        var totalDistance = 0.0
        var totalElevationGain = 0.0
        var totalElapsedTime = 0
        var startTime = ""
        var endTime = ""
        var totalCadence = 0.0
        var totalHeartrate = 0.0
        var cadenceCount = 0
        var heartrateCount = 0

        for (i in 0 until trkpts.length) {
            val trkpt = trkpts.item(i) as Element
            val lat = trkpt.getAttribute("lat").toDouble()
            val lon = trkpt.getAttribute("lon").toDouble()
            val ele = trkpt.getElementsByTagName("ele").item(0).textContent.toDouble()
            val time = trkpt.getElementsByTagName("time").item(0).textContent

            if (i == 0) {
                startTime = time
            } else if (i == trkpts.length - 1) {
                endTime = time
            }

            if (i > 0) {
                val prevTrkpt = trkpts.item(i - 1) as Element
                val prevLat = prevTrkpt.getAttribute("lat").toDouble()
                val prevLon = prevTrkpt.getAttribute("lon").toDouble()
                val prevEle = prevTrkpt.getElementsByTagName("ele").item(0).textContent.toDouble()

                totalDistance += haversine(lat, lon, prevLat, prevLon)
                totalElevationGain += if (ele > prevEle) ele - prevEle else 0.0
            }

            // Extract cadence and heartrate if available
            val cadenceNode = trkpt.getElementsByTagName("cadence").item(0)
            if (cadenceNode != null) {
                totalCadence += cadenceNode.textContent.toDouble()
                cadenceCount++
            }

            val heartrateNode = trkpt.getElementsByTagName("heartrate").item(0)
            if (heartrateNode != null) {
                totalHeartrate += heartrateNode.textContent.toDouble()
                heartrateCount++
            }
        }

        totalElapsedTime = calculateElapsedTime(startTime, endTime)
        val averageCadence = if (cadenceCount > 0) totalCadence / cadenceCount else 0.0
        val averageHeartrate = if (heartrateCount > 0) totalHeartrate / heartrateCount else 0.0

        val activity = Activity(
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
            startDate = startTime,
            startDateLocal = startTime,
            startLatlng = listOf(),
            totalElevationGain = totalElevationGain,
            type = "Run",
            uploadId = 0L,
            weightedAverageWatts = 0,
        )
        activity.stream = Stream(
            latitudeLongitude = LatitudeLongitude(
                data = (0 until trkpts.length).map { i ->
                    val trkpt = trkpts.item(i) as Element
                    listOf(trkpt.getAttribute("lat").toDouble(), trkpt.getAttribute("lon").toDouble())
                },
                originalSize = 0,
                resolution = "",
                seriesType = "",
            ),
            time = Time(
                data = (0 until trkpts.length).map { i ->
                    val trkpt = trkpts.item(i) as Element
                    (DatatypeConverter.parseDateTime(
                        trkpt.getElementsByTagName("time").item(0).textContent
                    ).timeInMillis / 1000).toInt()
                }.toMutableList(),
                originalSize = 0,
                resolution = "",
                seriesType = "",
            ),
            distance = Distance(
                data = (0 until trkpts.length).map { i ->
                    val trkpt = trkpts.item(i) as Element
                    val lat = trkpt.getAttribute("lat").toDouble()
                    val lon = trkpt.getAttribute("lon").toDouble()
                    if (i > 0) {
                        val prevTrkpt = trkpts.item(i - 1) as Element
                        val prevLat = prevTrkpt.getAttribute("lat").toDouble()
                        val prevLon = prevTrkpt.getAttribute("lon").toDouble()
                        haversine(lat, lon, prevLat, prevLon)
                    } else {
                        0.0
                    }
                }.toMutableList(),
                originalSize = 0,
                resolution = "",
                seriesType = "",
            ),
            altitude = Altitude(
                data = (0 until trkpts.length).map { i ->
                    val trkpt = trkpts.item(i) as Element
                    trkpt.getElementsByTagName("ele").item(0).textContent.toDouble()
                }.toMutableList(),
                originalSize = 0,
                resolution = "",
                seriesType = "",
            ),
            moving = Moving(
                data = (0 until trkpts.length).map { i ->
                    true
                }.toMutableList(),
                originalSize = 0,
                resolution = "",
                seriesType = "",
            ),
            watts = PowerStream(
                data = (0 until trkpts.length).map { i ->
                    0
                }.toMutableList(),
                originalSize = 0,
                resolution = "",
                seriesType = "",
            ),
        )


        return activity
    }

    private fun haversine(lat1: Double, lon1: Double, lat2: Double, lon2: Double): Double {
        val R = 6371e3 // Earth radius in meters
        val phi1 = Math.toRadians(lat1)
        val phi2 = Math.toRadians(lat2)
        val deltaPhi = Math.toRadians(lat2 - lat1)
        val deltaLambda = Math.toRadians(lon2 - lon1)

        val a = Math.sin(deltaPhi / 2) * Math.sin(deltaPhi / 2) +
                Math.cos(phi1) * Math.cos(phi2) *
                Math.sin(deltaLambda / 2) * Math.sin(deltaLambda / 2)
        val c = 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1 - a))

        return R * c
    }

    private fun calculateElapsedTime(startTime: String, endTime: String): Int {
        val start = DatatypeConverter.parseDateTime(startTime).time
        val end = DatatypeConverter.parseDateTime(endTime).time
        return ((end.time - start.time) / 1000).toInt()
    }
}