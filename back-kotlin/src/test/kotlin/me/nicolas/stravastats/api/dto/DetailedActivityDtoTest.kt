package me.nicolas.stravastats.api.dto

import me.nicolas.stravastats.TestHelper
import me.nicolas.stravastats.domain.business.ActivityEffort
import me.nicolas.stravastats.domain.business.ActivityShort
import me.nicolas.stravastats.domain.business.strava.stream.AltitudeStream
import me.nicolas.stravastats.domain.business.strava.stream.DistanceStream
import me.nicolas.stravastats.domain.business.strava.stream.LatLngStream
import me.nicolas.stravastats.domain.business.strava.stream.PowerStream
import me.nicolas.stravastats.domain.business.strava.stream.SmoothVelocityStream
import me.nicolas.stravastats.domain.business.strava.stream.Stream
import me.nicolas.stravastats.domain.business.strava.stream.TimeStream
import me.nicolas.stravastats.domain.services.toStravaDetailedActivity
import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Assertions.assertNotEquals
import org.junit.jupiter.api.Assertions.assertNull
import org.junit.jupiter.api.Test

class DetailedActivityDtoTest {

    @Test
    fun `ActivityEffort dto id stays unique for same label with different indexes`() {
        // GIVEN
        val first = ActivityEffort(
            distance = 300.0,
            seconds = 30,
            deltaAltitude = 20.0,
            idxStart = 10,
            idxEnd = 40,
            label = "MURAILLE DE CHINE <Alpe d'Huez>",
            activityShort = ActivityShort(id = 1L, name = "A", type = "Ride"),
        )
        val second = ActivityEffort(
            distance = 300.0,
            seconds = 30,
            deltaAltitude = -20.0,
            idxStart = 50,
            idxEnd = 80,
            label = "MURAILLE DE CHINE <Alpe d'Huez>",
            activityShort = ActivityShort(id = 1L, name = "A", type = "Ride"),
        )

        // WHEN
        val firstDto = first.toDto()
        val secondDto = second.toDto()

        // THEN
        assertNotEquals(firstDto.id, secondDto.id)
    }

    @Test
    fun `detailed activity dto sanitizes non finite values`() {
        // GIVEN
        @Suppress("UNCHECKED_CAST")
        val distanceDataWithNull = listOf<Double?>(0.0, null, Double.NaN) as List<Double>
        @Suppress("UNCHECKED_CAST")
        val coordinateDataWithNull = listOf(
            listOf(45.0, 6.0),
            listOf<Double?>(null, Double.POSITIVE_INFINITY) as List<Double>,
            listOf(Double.NaN, 6.2),
        )
        @Suppress("UNCHECKED_CAST")
        val altitudeDataWithNull = listOf<Double?>(100.0, null, Double.POSITIVE_INFINITY) as List<Double>
        @Suppress("UNCHECKED_CAST")
        val velocityDataWithNull = listOf<Float?>(4.0f, null, Float.NaN) as List<Float>
        val stream = Stream(
            distance = DistanceStream(
                data = distanceDataWithNull,
                originalSize = 3,
                resolution = "high",
                seriesType = "distance",
            ),
            time = TimeStream(
                data = listOf(0, 10, 20),
                originalSize = 3,
                resolution = "high",
                seriesType = "time",
            ),
            latlng = LatLngStream(
                data = coordinateDataWithNull,
                originalSize = 3,
                resolution = "high",
                seriesType = "distance",
            ),
            altitude = AltitudeStream(
                data = altitudeDataWithNull,
                originalSize = 3,
                resolution = "high",
                seriesType = "distance",
            ),
            watts = PowerStream(
                data = listOf(200, null),
                originalSize = 2,
                resolution = "high",
                seriesType = "distance",
            ),
            velocitySmooth = SmoothVelocityStream(
                data = velocityDataWithNull,
                originalSize = 3,
                resolution = "high",
                seriesType = "distance",
            ),
        )
        val activity = TestHelper.stravaActivity
            .toStravaDetailedActivity()
            .copy(
                averageCadence = Double.NaN,
                averageHeartrate = Double.POSITIVE_INFINITY,
                averageSpeed = Double.NEGATIVE_INFINITY,
                averageWatts = Double.NaN,
                calories = Double.POSITIVE_INFINITY,
                elevHigh = Double.NaN,
                elevLow = Double.POSITIVE_INFINITY,
                kilojoules = Double.NEGATIVE_INFINITY,
                maxSpeed = Double.POSITIVE_INFINITY,
                startLatLng = listOf(Double.NaN, Double.POSITIVE_INFINITY),
                sufferScore = Double.NaN,
                stream = stream,
            )

        // WHEN
        val dto = activity.toDto()

        // THEN
        assertEquals(0, dto.averageCadence)
        assertEquals(0, dto.averageHeartrate)
        assertEquals(0, dto.averageWatts)
        assertEquals(0f, dto.averageSpeed)
        assertEquals(0.0, dto.calories)
        assertEquals(0.0, dto.elevHigh)
        assertEquals(0.0, dto.kilojoules)
        assertEquals(0f, dto.maxSpeed)
        assertEquals(listOf(0.0, 0.0), dto.startLatlng)
        assertNull(dto.sufferScore)
        assertEquals(0.0, dto.totalDescent)
        assertEquals(0.0, dto.stream?.distance?.get(1))
        assertEquals(0.0, dto.stream?.distance?.get(2))
        assertEquals(0.0, dto.stream?.latlng?.get(1)?.get(0))
        assertEquals(0.0, dto.stream?.latlng?.get(1)?.get(1))
        assertEquals(0.0, dto.stream?.altitude?.get(1))
        assertEquals(0.0, dto.stream?.altitude?.get(2))
        assertEquals(0.0, dto.stream?.velocitySmooth?.get(1))
        assertEquals(0.0, dto.stream?.velocitySmooth?.get(2))
    }
}
