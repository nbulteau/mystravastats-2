package me.nicolas.stravastats.domain.services.statistics

import me.nicolas.stravastats.domain.business.strava.AthleteRef
import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.business.strava.stream.AltitudeStream
import me.nicolas.stravastats.domain.business.strava.stream.DistanceStream
import me.nicolas.stravastats.domain.business.strava.stream.Stream
import me.nicolas.stravastats.domain.business.strava.stream.TimeStream

internal object StatisticsFixtures {
    fun syntheticRideActivity(
        id: Long,
        startDateLocal: String = "2025-01-01T10:00:00Z",
        stream: Stream? = defaultStream()
    ): StravaActivity {
        return StravaActivity(
            athlete = AthleteRef(id = 1),
            averageSpeed = 0.0,
            averageCadence = 0.0,
            averageHeartrate = 0.0,
            maxHeartrate = 0,
            averageWatts = 0,
            commute = false,
            distance = stream?.distance?.data?.lastOrNull() ?: 0.0,
            deviceWatts = false,
            elapsedTime = stream?.time?.data?.lastOrNull() ?: 0,
            elevHigh = stream?.altitude?.data?.maxOrNull() ?: 0.0,
            id = id,
            kilojoules = 0.0,
            maxSpeed = 0f,
            movingTime = stream?.time?.data?.lastOrNull() ?: 0,
            name = "Synthetic ride $id",
            startDate = startDateLocal,
            startDateLocal = startDateLocal,
            startLatlng = null,
            totalElevationGain = 0.0,
            type = "Ride",
            uploadId = id + 1000,
            weightedAverageWatts = 0,
            stream = stream
        )
    }

    fun defaultStream(
        distances: List<Double> = listOf(0.0, 100.0, 200.0, 300.0, 400.0),
        times: List<Int> = listOf(0, 10, 20, 35, 50),
        altitudes: List<Double>? = listOf(100.0, 105.0, 115.0, 118.0, 130.0),
    ): Stream {
        val originalSize = distances.size
        return Stream(
            distance = DistanceStream(
                data = distances,
                originalSize = originalSize,
                resolution = "high",
                seriesType = "distance",
            ),
            time = TimeStream(
                data = times,
                originalSize = times.size,
                resolution = "high",
                seriesType = "time",
            ),
            altitude = altitudes?.let {
                AltitudeStream(
                    data = it,
                    originalSize = it.size,
                    resolution = "high",
                    seriesType = "distance",
                )
            },
        )
    }
}
