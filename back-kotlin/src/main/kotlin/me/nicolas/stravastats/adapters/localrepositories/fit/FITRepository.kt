package me.nicolas.stravastats.adapters.localrepositories.fit

import com.garmin.fit.*
import me.nicolas.stravastats.domain.business.strava.AthleteRef
import me.nicolas.stravastats.domain.business.strava.stream.PowerStream
import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.business.strava.stream.*
import me.nicolas.stravastats.domain.utils.inDateTimeFormatter
import org.slf4j.LoggerFactory
import java.io.File
import java.time.LocalDateTime
import java.time.ZoneId
import java.time.ZoneOffset
import java.util.*
import kotlin.math.absoluteValue

class FITRepository(fitDirectory: String) {

    private val logger = LoggerFactory.getLogger(FITRepository::class.java)

    private val cacheDirectory = File(fitDirectory)

    private val fitDecoder = FitDecoder()

    fun loadActivitiesFromCache(year: Int): List<StravaActivity> {

        val yearActivitiesDirectory = File(cacheDirectory, "$year")
        val fitFiles = yearActivitiesDirectory.listFiles { file ->
            file.extension.lowercase(Locale.getDefault()) == "fit"
        }
        val activities: List<StravaActivity> = fitFiles?.mapNotNull { fitFile ->
            try {
                val fitMessages = fitDecoder.decode(fitFile.inputStream())
                fitMessages.toActivity()
            } catch (exception: Exception) {
                logger.error("Something wrong during FIT conversion: ${exception.message}")
                null
            }
        }?.toList() ?: emptyList()

        return activities
    }

    /**
     * Convert a FIT stravaActivity to a Strava stravaActivity
     */
    private fun FitMessages.toActivity(): StravaActivity {

        val sessionMesg = this.sessionMesgs.first()

        val stream: Stream = this.recordMesgs.buildStream()

        // StravaAthlete
        val athlete = AthleteRef(0)
        // The stravaActivity's average speed, in meters per second
        val averageSpeed: Double = sessionMesg?.avgSpeed?.toDouble() ?: 0.0
        // The effort's average cadence
        val averageCadence: Double = sessionMesg?.avgCadence?.toDouble() ?: 0.0
        // The heart rate of the stravaAthlete during this effort
        val averageHeartRate: Double = sessionMesg?.avgHeartRate?.toDouble() ?: 0.0
        // The maximum heart rate of the stravaAthlete during this effort
        val maxHeartRate: Int = sessionMesg?.maxHeartRate?.toInt() ?: 0
        //The average wattage of this effort
        val averageWatts: Int = sessionMesg?.avgPower ?: 0 // TODO : Calculate ?
        // Whether this stravaActivity is a commute
        val commute = false
        // The stravaActivity's distance, in meters
        val distance: Double = sessionMesg?.totalDistance?.toDouble() ?: 0.0
        // The stravaActivity's elapsed time, in seconds
        val elapsedTime: Int = sessionMesg?.totalElapsedTime?.toInt() ?: 0
        // The stravaActivity's highest elevation, in meters
        val extractedElevHigh: Double = extractElevHigh(sessionMesg)
        val elevHigh: Double = if (extractedElevHigh != 0.0) {
            extractedElevHigh
        } else if (stream.altitude != null) {
            stream.altitude.data.maxOf { it }
        } else {
            0.0
        }
        // The total work done in kilojoules during this stravaActivity. Rides only
        val kilojoules = 0.8604 * averageWatts * elapsedTime / 1000
        // The stravaActivity's max speed, in meters per second
        val maxSpeed: Float = sessionMesg?.maxSpeed?.toFloat() ?: 0.0F
        // The stravaActivity's moving time, in seconds
        val movingTime: Int = sessionMesg?.timestamp?.timestamp?.minus(sessionMesg.startTime?.timestamp!!)?.toInt()!!
        // The time at which the stravaActivity was started.
        val startDate: String = extractDate(sessionMesg.startTime?.timestamp!!)
        // The time at which the stravaActivity was started in the local timezone.
        val startDateLocal: String = extractDateLocal(sessionMesg.startTime?.timestamp!!)
        // StravaActivity name
        val name = "${extractActivityType(sessionMesg.sport!!)} - $startDateLocal"
        // The unique identifier of the stravaActivity
        val id: Long = name.hashCode().toLong().absoluteValue
        // Latitude /longitude of the start point
        val extractedStartLatLng = extractLatLng(sessionMesg.startPositionLat, sessionMesg.startPositionLong)
        val startLatlng: List<Double>? = extractedStartLatLng.ifEmpty {
            stream.latlng?.data?.first()
        }
        // Total elevation gain
        val deltas = if (stream.altitude != null) {
            stream.altitude.data.zipWithNext { a, b -> b - a }
        } else {
            null
        }
        val sum = deltas?.filter { it > 0 }?.sumOf { it } ?: 0.0
        val totalElevationGain: Double = sessionMesg.totalAscent?.toDouble() ?: sum

        // StravaActivity type (i.e. Ride, Run ...)
        val type: String = extractActivityType(sessionMesg.sport!!)

        return StravaActivity(
            athlete = athlete,
            averageSpeed = averageSpeed,
            averageCadence = averageCadence,
            averageHeartrate = averageHeartRate,
            maxHeartrate = maxHeartRate,
            averageWatts = averageWatts,
            commute = commute,
            distance = distance,
            elapsedTime = elapsedTime,
            elevHigh = elevHigh,
            id = id,
            kilojoules = kilojoules,
            maxSpeed = maxSpeed,
            movingTime = movingTime,
            name = name,
            startDate = startDate,
            startDateLocal = startDateLocal,
            startLatlng = startLatlng,
            totalElevationGain = totalElevationGain,
            type = type,
            uploadId = 0,
            weightedAverageWatts = sessionMesg.avgPower?.toInt() ?: 0,
            stream = stream
        )
    }

    /**
     * Build Strava Stream structure using the GPS records
     */
    private fun List<RecordMesg>.buildStream(): Stream {
        // distance
        val dataDistance = this.map { recordMesg -> recordMesg.distance.toDouble() }
        val streamDistance = DistanceStream(
            data = dataDistance.toMutableList(),
            originalSize = dataDistance.size,
            resolution = "high",
            seriesType = "distance"
        )

        //  time
        val startTime = this.first().timestamp.timestamp
        val dataTime = this.map { recordMesg ->
            (recordMesg.timestamp.timestamp - startTime).toInt()
        }
        val streamTime = TimeStream(
            data = dataTime.toMutableList(),
            originalSize = dataTime.size,
            resolution = "high",
            seriesType = "distance"
        )

        // latitude/longitude
        val dataLatitude = this.map { recordMesg ->
            if (recordMesg.positionLat == null) {
                0
            } else {
                recordMesg.positionLat
            }
        }.toMutableList()
        dataLatitude.fixCoordinate()

        val dataLongitude = this.map { recordMesg ->
            if (recordMesg.positionLong == null) {
                0
            } else {
                recordMesg.positionLong
            }
        }.toMutableList()
        dataLongitude.fixCoordinate()

        val dataLatitudeLongitude = dataLatitude.zip(dataLongitude) { lat, long -> extractLatLng(lat, long) }
        val streamLatitudeLongitude = LatLngStream(
            data = dataLatitudeLongitude,
            originalSize = dataLatitudeLongitude.size,
            resolution = "high",
            seriesType = "distance"
        )

        // altitude
        val dataAltitude = this.mapNotNull { recordMesg ->
            recordMesg.altitude?.toDouble()
        }
        val streamAltitude = if (dataAltitude.isNotEmpty()) {
            AltitudeStream(
                data = dataAltitude.toMutableList(),
                originalSize = dataAltitude.size,
                resolution = "high",
                seriesType = "distance"
            )
        } else {
            null
        }

        // moving
        val dataMoving = this.map { recordMesg ->
            recordMesg.speed > 0.0
        }
        val streamMoving = if (dataMoving.isNotEmpty()) {
            MovingStream(
                data = dataMoving.toMutableList(),
                originalSize = dataMoving.size,
                resolution = "high",
                seriesType = "distance"
            )
        } else {
            null
        }

        // power
        val dataPower = this.mapNotNull { recordMesg ->
            recordMesg.power
        }
        val streamPower = if (dataPower.isNotEmpty()) {
            PowerStream(
                data = dataPower.toMutableList(),
                originalSize = dataPower.size,
                resolution = "high",
                seriesType = "distance"
            )
        } else {
            null
        }

        // cadence
        val dataCadence = this.map { recordMesg ->
            recordMesg.cadence.toInt()
        }
        val streamCadence = if (dataCadence.isNotEmpty()) {
            CadenceStream(
                data = dataCadence.toMutableList(),
                originalSize = dataCadence.size,
                resolution = "high",
                seriesType = "distance"
            )
        } else {
            null
        }

        // heart rate
        val dataHeartRate = this.map { recordMesg ->
            recordMesg.heartRate.toInt()
        }
        val streamHeartRate = if (dataHeartRate.isNotEmpty()) {
            HeartRateStream(
                data = dataHeartRate.toMutableList(),
                originalSize = dataHeartRate.size,
                resolution = "high",
                seriesType = "distance"
            )
        } else {
            null
        }

        // velocity smooth
        val dataVelocitySmooth = this.map { recordMesg ->
            recordMesg.speed
        }
        val streamVelocitySmooth = if (dataVelocitySmooth.isNotEmpty()) {
            SmoothVelocityStream(
                data = dataVelocitySmooth.toMutableList(),
                originalSize = dataVelocitySmooth.size,
                resolution = "high",
                seriesType = "distance"
            )
        } else {
            null
        }

        // grade smooth
        val dataGradeSmooth = this.map { recordMesg ->
            recordMesg.grade
        }
        val streamGradeSmooth = if (dataGradeSmooth.isNotEmpty()) {
            SmoothGradeStream(
                data = dataGradeSmooth.toMutableList(),
                originalSize = dataGradeSmooth.size,
                resolution = "high",
                seriesType = "distance"
            )
        } else {
            null
        }

        return Stream(
            streamDistance,
            streamTime,
            streamLatitudeLongitude,
            streamCadence,
            streamHeartRate,
            streamMoving,
            streamAltitude,
            streamPower,
            streamVelocitySmooth,
            streamGradeSmooth
        )
    }

    private fun extractLatLng(lat: Int?, lng: Int?): List<Double> {
        return if (lat != null && lng != null) {
            // 11930465 = (2^32 / 360)
            listOf(lat.toDouble() / 11930465, lng.toDouble() / 11930465)
        } else {
            emptyList()
        }
    }

    private fun extractActivityType(sport: Sport): String {
        return when (sport) {
            Sport.CYCLING -> "Ride"
            Sport.RUNNING -> "Run"
            Sport.INLINE_SKATING -> "InlineSkate"
            Sport.ALPINE_SKIING -> "AlpineSki"
            Sport.HIKING -> "Hike"
            else -> "Unknown"
        }
    }

    private fun extractDateLocal(value: Long): String {
        var localDateTime = LocalDateTime.of(1989, 12, 31, 0, 0, 0, 0)
        if (value >= 0L) {
            localDateTime = localDateTime.plusSeconds(value)
        }
        return localDateTime
            .atZone(ZoneOffset.UTC).withZoneSameInstant(ZoneId.systemDefault())
            .toLocalDateTime()
            .format(inDateTimeFormatter)
    }

    private fun extractDate(value: Long): String {
        var localDateTime = LocalDateTime.of(1989, 12, 31, 0, 0, 0, 0)
        if (value >= 0L) {
            localDateTime = localDateTime.plusSeconds(value)
        }

        return localDateTime.format(inDateTimeFormatter)
    }

    private fun extractElevHigh(sessionMesg: SessionMesg): Double {
        return if (sessionMesg.maxAltitude != null) {
            sessionMesg.maxAltitude.toDouble()
        } else if (sessionMesg.enhancedMaxAltitude != null) {
            sessionMesg.enhancedMaxAltitude.toDouble()
        } else {
            0.0
        }
    }

    /**
     * Fix missing coordinates
     */
    private fun MutableList<Int>.fixCoordinate() {
        var index = 0

        // if start with 0 : get the first valid value
        if (this.first() == 0) {
            val firstValidValue: Int = try {
                this.first { it != 0 }
            } catch (noSuchElementException: NoSuchElementException) {
                0
            }
            while (index < this.size && this[index] == 0) {
                this[index] = firstValidValue
                index++
            }
        }

        // if a value is missing set average value
        while (index < this.size) {
            if (this[index] == 0) {
                val lastValidValue: Int = this[index - 1]
                val firstValidValue: Int = try {
                    this.drop(index).first { it != 0 }
                } catch (noSuchElementException: NoSuchElementException) {
                    lastValidValue
                }
                while (this[index] == 0) {
                    this[index] = (lastValidValue + firstValidValue) / 2
                    index++
                }
            }
            index++
        }
    }
}