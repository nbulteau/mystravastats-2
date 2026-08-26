package me.nicolas.stravastats.adapters.localrepositories.fit

import com.garmin.fit.*
import me.nicolas.stravastats.domain.business.ActivityType
import me.nicolas.stravastats.domain.business.strava.AthleteRef
import me.nicolas.stravastats.domain.business.strava.stream.PowerStream
import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.business.strava.stream.*
import me.nicolas.stravastats.domain.interfaces.IYearActivityStorageProvider
import org.slf4j.LoggerFactory
import java.io.File
import java.nio.charset.StandardCharsets
import java.security.MessageDigest
import java.time.Instant
import java.time.ZoneId
import java.time.format.DateTimeFormatter
import java.util.*
import kotlin.math.max
import kotlin.math.roundToInt

class FITRepository(fitDirectory: String) : IYearActivityStorageProvider {

    private val logger = LoggerFactory.getLogger(FITRepository::class.java)

    private val cacheDirectory = File(fitDirectory)

    private val fitDecoder = FitDecoder()

    override fun loadActivitiesFromCache(year: Int): List<StravaActivity> {

        val yearActivitiesDirectory = File(cacheDirectory, "$year")
        val fitFiles = yearActivitiesDirectory.listFiles { file ->
            file.extension.lowercase(Locale.getDefault()) == "fit"
        }
        val activities: List<StravaActivity> = fitFiles?.mapNotNull { fitFile ->
            decodeActivity(fitFile)
        }?.toList() ?: emptyList()

        return activities
    }

    fun decodeActivity(fitFile: File): StravaActivity? {
        return try {
            val fitMessages = fitDecoder.decode(fitFile.inputStream())
            fitMessages.toActivity(fitFile)
        } catch (exception: Exception) {
            logger.error("Something wrong during FIT conversion: ${exception.message}")
            null
        }
    }

    /**
     * Convert a FIT stravaActivity to a Strava stravaActivity
     */
    private fun FitMessages.toActivity(fitFile: File): StravaActivity {

        val sessionMesg = this.sessionMesgs.firstOrNull()
            ?: throw IllegalArgumentException("FIT file has no session message")

        val stream: Stream? = this.recordMesgs.buildStream()

        // StravaAthlete
        val athlete = AthleteRef(0)
        // The stravaActivity's average speed, in meters per second
        val averageSpeed: Double = sessionMesg.avgSpeed?.toDouble() ?: 0.0
        // The effort's average cadence
        val averageCadence: Double = sessionMesg.avgCadence?.toDouble() ?: 0.0
        // The heart rate of the stravaAthlete during this effort
        val averageHeartRate: Double = sessionMesg.avgHeartRate?.toDouble() ?: 0.0
        // The maximum heart rate of the stravaAthlete during this effort
        val maxHeartRate: Int = sessionMesg.maxHeartRate?.toInt() ?: 0
        // Whether this stravaActivity is a commute
        val classification = extractFITActivityClassification(sessionMesg.sport, sessionMesg.subSport)
        val commute = classification.commute
        // The stravaActivity's distance, in meters
        val distance: Double = sessionMesg.totalDistance?.toDouble() ?: 0.0
        // The stravaActivity's elapsed time, in seconds
        val elapsedTime: Int = sessionMesg.totalElapsedTime?.toInt() ?: 0
        val powerMetrics = computeFitPowerMetrics(sessionMesg.avgPower, stream ?: emptyStream(), elapsedTime)
        // The stravaActivity's highest elevation, in meters
        val extractedElevHigh: Double = extractElevHigh(sessionMesg)
        val elevHigh: Double = if (extractedElevHigh != 0.0) {
            extractedElevHigh
        } else if (stream?.altitude != null) {
            stream.altitude.data.maxOf { it }
        } else {
            0.0
        }
        // The stravaActivity's max speed, in meters per second
        val maxSpeed: Float = sessionMesg.maxSpeed ?: 0.0F
        // The stravaActivity's moving time, in seconds
        val movingTime: Int = resolveMovingTime(sessionMesg, stream, elapsedTime)
        // The time at which the stravaActivity was started.
        val startTimestamp = sessionMesg.startTime?.timestamp
            ?: throw IllegalArgumentException("FIT file has no session start time")
        val startInstant = fitTimestampToInstant(startTimestamp)
        val startDate: String = extractDate(startInstant)
        // The time at which the stravaActivity was started in the local timezone.
        val startDateLocal: String = extractDateLocal(startInstant)
        // StravaActivity name
        val name = "${classification.type} - $startDateLocal"
        // The unique identifier of the stravaActivity
        val id: Long = fitActivityID(fitFile, startInstant, classification.type, distance)
        // Latitude /longitude of the start point
        val startLatlng: List<Double>? = extractLatLng(sessionMesg.startPositionLat, sessionMesg.startPositionLong)
            ?: stream?.latlng?.data?.firstOrNull()
        // Total elevation gain
        val deltas = stream?.altitude?.data?.zipWithNext { a, b -> b - a }
        val sum = deltas?.filter { it > 0 }?.sumOf { it } ?: 0.0
        val totalElevationGain: Double = sessionMesg.totalAscent?.toDouble() ?: sum

        // StravaActivity type (i.e. Ride, Run ...)
        val type: String = classification.type

        return StravaActivity(
            athlete = athlete,
            averageSpeed = averageSpeed,
            averageCadence = averageCadence,
            averageHeartrate = averageHeartRate,
            maxHeartrate = maxHeartRate,
            averageWatts = powerMetrics.averageWatts,
            commute = commute,
            distance = distance,
            deviceWatts = powerMetrics.hasDeviceWatts,
            elapsedTime = elapsedTime,
            elevHigh = elevHigh,
            id = id,
            kilojoules = powerMetrics.kilojoules,
            maxSpeed = maxSpeed,
            movingTime = movingTime,
            name = name,
            _sportType = classification.sportType,
            startDate = startDate,
            startDateLocal = startDateLocal,
            startLatlng = startLatlng,
            totalElevationGain = totalElevationGain,
            type = type,
            uploadId = 0,
            weightedAverageWatts = powerMetrics.weightedAverageWatts,
            stream = stream
        )
    }

    private fun resolveMovingTime(sessionMesg: SessionMesg, stream: Stream?, elapsedTime: Int): Int {
        val totalMovingTime = sessionMesg.totalMovingTime?.roundToInt() ?: 0
        val totalTimerTime = sessionMesg.totalTimerTime?.roundToInt() ?: 0
        val streamMovingTime = stream?.movingTimeSeconds() ?: 0
        return resolveFitMovingTime(totalMovingTime, totalTimerTime, elapsedTime, streamMovingTime)
    }

    private fun Stream.movingTimeSeconds(): Int {
        val movingData = moving?.data.orEmpty()
        val timeData = time.data
        if (movingData.isEmpty() || timeData.size < 2) return 0

        var movingTime = 0
        val limit = minOf(timeData.size, movingData.size)
        for (index in 1 until limit) {
            val delta = timeData[index] - timeData[index - 1]
            if (delta > 0 && movingData[index]) {
                movingTime += delta
            }
        }
        return movingTime
    }

    /**
     * Build Strava Stream structure using the GPS records
     */
    private fun List<RecordMesg>.buildStream(): Stream? {
        if (this.isEmpty()) {
            return null
        }

        // distance
        val dataDistance = this.map { recordMesg -> recordMesg.distance?.toDouble() ?: 0.0 }
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
        val dataLatitudeLongitude = normalizeCoordinates(this.map { recordMesg ->
            extractLatLng(recordMesg.positionLat, recordMesg.positionLong)
        })
        val streamLatitudeLongitude = dataLatitudeLongitude?.let { coordinates ->
            LatLngStream(
                data = coordinates,
                originalSize = coordinates.size,
                resolution = "high",
                seriesType = "distance"
            )
        }

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
            (recordMesg.speed ?: 0.0F) > 0.1F
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
            recordMesg.cadence?.toInt() ?: 0
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
            recordMesg.heartRate?.toInt() ?: 0
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
            recordMesg.speed ?: 0.0F
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
            recordMesg.grade ?: 0.0F
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

    private fun extractLatLng(lat: Int?, lng: Int?): List<Double>? {
        return if (lat != null && lng != null) {
            // 11930465 = (2^32 / 360)
            listOf(lat.toDouble() / 11930465, lng.toDouble() / 11930465)
                .takeIf { coordinates -> validLatLng(coordinates) }
        } else {
            null
        }
    }

    private fun extractDateLocal(value: Instant): String {
        return value.atZone(ZoneId.systemDefault()).format(DateTimeFormatter.ISO_OFFSET_DATE_TIME)
    }

    private fun extractDate(value: Instant): String {
        return DateTimeFormatter.ISO_INSTANT.format(value)
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

    private fun normalizeCoordinates(rawCoordinates: List<List<Double>?>): List<List<Double>>? {
        if (rawCoordinates.none { coordinates -> validLatLng(coordinates) }) {
            return null
        }

        return rawCoordinates.mapIndexed { index, coordinates ->
            if (validLatLng(coordinates)) {
                coordinates!!
            } else {
                val previous = rawCoordinates.take(index).lastOrNull { candidate -> validLatLng(candidate) }
                val next = rawCoordinates.drop(index + 1).firstOrNull { candidate -> validLatLng(candidate) }
                when {
                    previous != null && next != null -> listOf((previous[0] + next[0]) / 2, (previous[1] + next[1]) / 2)
                    previous != null -> previous
                    next != null -> next
                    else -> listOf(0.0, 0.0)
                }
            }
        }
    }

    private fun validLatLng(value: List<Double>?): Boolean {
        return value != null &&
            value.size >= 2 &&
            value[0].isFinite() &&
            value[1].isFinite() &&
            value[0] in -90.0..90.0 &&
            value[1] in -180.0..180.0 &&
            !(value[0] == 0.0 && value[1] == 0.0)
    }

    private fun fitTimestampToInstant(value: Long): Instant {
        return FIT_EPOCH.plusSeconds(max(value, 0L))
    }

    private fun fitActivityID(fitFile: File, startDate: Instant, sportType: String, distanceMeters: Double): Long {
        val raw = "${fitFile.canonicalPath}|$startDate|$sportType|${distanceMeters.roundToInt()}"
        val digest = MessageDigest.getInstance("SHA-256").digest(raw.toByteArray(StandardCharsets.UTF_8))
        var value = 0L
        for (index in 0 until 8) {
            value = (value shl 8) or (digest[index].toLong() and 0xff)
        }
        val safeValue = (value and Long.MAX_VALUE) % MAX_SAFE_JS_INTEGER
        return if (safeValue == 0L) 1L else safeValue
    }

    private fun emptyStream(): Stream {
        return Stream(
            distance = DistanceStream(emptyList(), 0, "high", "distance"),
            time = TimeStream(emptyList(), 0, "high", "time"),
        )
    }

    companion object {
        private val FIT_EPOCH: Instant = Instant.parse("1989-12-31T00:00:00Z")
        private const val MAX_SAFE_JS_INTEGER: Long = 9_007_199_254_740_991L
    }
}

internal data class FITActivityClassification(
    val type: String,
    val sportType: String = type,
    val commute: Boolean = false,
)

internal fun extractFITActivityType(sport: Sport?, subSport: SubSport?): String {
    return extractFITActivityClassification(sport, subSport).type
}

internal fun extractFITActivityClassification(sport: Sport?, subSport: SubSport?): FITActivityClassification {
    return when (sport) {
        Sport.CYCLING -> when (subSport) {
            SubSport.COMMUTING -> fitCommuteClassification(ActivityType.Ride.name)
            SubSport.MOUNTAIN, SubSport.E_BIKE_MOUNTAIN -> FITActivityClassification(ActivityType.MountainBikeRide.name)
            SubSport.GRAVEL_CYCLING, SubSport.MIXED_SURFACE -> FITActivityClassification(ActivityType.GravelRide.name)
            SubSport.VIRTUAL_ACTIVITY, SubSport.INDOOR_CYCLING -> FITActivityClassification(ActivityType.VirtualRide.name)
            else -> FITActivityClassification(ActivityType.Ride.name)
        }
        Sport.RUNNING -> when (subSport) {
            SubSport.TRAIL -> FITActivityClassification(ActivityType.TrailRun.name)
            else -> FITActivityClassification(ActivityType.Run.name)
        }
        Sport.FITNESS_EQUIPMENT -> when (subSport) {
            SubSport.INDOOR_CYCLING -> FITActivityClassification(ActivityType.VirtualRide.name)
            SubSport.TREADMILL, SubSport.INDOOR_RUNNING -> FITActivityClassification(ActivityType.Run.name)
            SubSport.INDOOR_WALKING -> FITActivityClassification(ActivityType.Walk.name)
            else -> FITActivityClassification(ActivityType.Ride.name)
        }
        Sport.WALKING -> FITActivityClassification(ActivityType.Walk.name)
        Sport.HIKING, Sport.MOUNTAINEERING -> FITActivityClassification(ActivityType.Hike.name)
        Sport.ALPINE_SKIING -> FITActivityClassification(ActivityType.AlpineSki.name)
        Sport.INLINE_SKATING -> FITActivityClassification(ActivityType.InlineSkate.name)
        Sport.E_BIKING -> when (subSport) {
            SubSport.COMMUTING -> fitCommuteClassification(ActivityType.Ride.name)
            SubSport.E_BIKE_MOUNTAIN -> FITActivityClassification(ActivityType.MountainBikeRide.name)
            SubSport.VIRTUAL_ACTIVITY, SubSport.INDOOR_CYCLING -> FITActivityClassification(ActivityType.VirtualRide.name)
            else -> FITActivityClassification(ActivityType.Ride.name)
        }
        else -> FITActivityClassification(ActivityType.Ride.name)
    }
}

private fun fitCommuteClassification(sportType: String): FITActivityClassification {
    return FITActivityClassification(
        type = ActivityType.Commute.name,
        sportType = sportType,
        commute = true,
    )
}

internal fun resolveFitMovingTime(
    totalMovingTime: Int,
    totalTimerTime: Int,
    elapsedTime: Int,
    streamMovingTime: Int,
): Int {
    if (totalMovingTime > 0) return totalMovingTime
    if (totalTimerTime > 0) {
        if (shouldUseStreamMovingTimeFallback(totalTimerTime, streamMovingTime)) {
            return streamMovingTime
        }
        return totalTimerTime
    }
    if (streamMovingTime > 0) return streamMovingTime
    return elapsedTime
}

private fun shouldUseStreamMovingTimeFallback(totalTimerTime: Int, streamMovingTime: Int): Boolean {
    if (totalTimerTime <= 0 || streamMovingTime <= 0 || streamMovingTime >= totalTimerTime) {
        return false
    }

    val streamRemovesMeaningfulStopTime = (totalTimerTime - streamMovingTime).toDouble() > max(60.0, totalTimerTime.toDouble() * 0.02)
    return streamRemovesMeaningfulStopTime
}
