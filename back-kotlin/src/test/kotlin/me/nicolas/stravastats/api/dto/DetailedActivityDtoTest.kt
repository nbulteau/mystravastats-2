package me.nicolas.stravastats.api.dto

import me.nicolas.stravastats.TestHelper
import me.nicolas.stravastats.domain.business.ActivityEffort
import me.nicolas.stravastats.domain.business.ActivityShort
import me.nicolas.stravastats.domain.business.strava.Achievement
import me.nicolas.stravastats.domain.business.strava.MetaActivity
import me.nicolas.stravastats.domain.business.strava.MetaAthlete
import me.nicolas.stravastats.domain.business.strava.Segment
import me.nicolas.stravastats.domain.business.strava.StravaSegmentEffort
import me.nicolas.stravastats.domain.business.strava.stream.AltitudeStream
import me.nicolas.stravastats.domain.business.strava.stream.CadenceStream
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
    fun `detailed activity dto exposes Strava segment efforts`() {
        // GIVEN
        val activity = TestHelper.stravaActivity
            .toStravaDetailedActivity()
            .copy(
                type = "Ride",
                sportType = "GravelRide",
                segmentEfforts = listOf(
                    stravaSegmentEffort(
                        id = 1001,
                        name = "Local sprint",
                        segmentName = "Local sprint segment",
                    )
                )
            )

        // WHEN
        val dto = activity.toDto()

        // THEN
        assertEquals(1, dto.stravaSegmentEfforts.size)
        assertEquals("Ride", dto.type)
        assertEquals("GravelRide", dto.sportType)
        val effort = dto.stravaSegmentEfforts.first()
        assertEquals("Local sprint", effort.name)
        assertEquals("Local sprint segment", effort.segment.name)
        assertEquals(10, effort.startIndex)
        assertEquals(85, effort.endIndex)
        assertEquals(315.7, effort.averageWatts)
        assertEquals(2, effort.prRank)
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
            cadence = CadenceStream(
                data = listOf(82, 84, 86),
                originalSize = 3,
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
        assertEquals(listOf(82, 84, 86), dto.stream?.cadence)
        assertEquals(0.0, dto.stream?.velocitySmooth?.get(1))
        assertEquals(0.0, dto.stream?.velocitySmooth?.get(2))
    }

    private fun stravaSegmentEffort(
        id: Long,
        name: String,
        segmentName: String,
    ): StravaSegmentEffort =
        StravaSegmentEffort(
            achievements = emptyList<Achievement>(),
            activity = MetaActivity(1),
            athlete = MetaAthlete(1),
            averageCadence = 82.0,
            averageHeartRate = 165.0,
            averageWatts = 315.7,
            deviceWatts = true,
            distance = 520.5,
            elapsedTime = 75,
            endIndex = 85,
            hidden = false,
            id = id,
            komRank = null,
            maxHeartRate = 172.0,
            movingTime = 74,
            name = name,
            prRank = 2,
            resourceState = 2,
            segment = Segment(
                activityType = "Ride",
                averageGrade = 4.2,
                city = null,
                climbCategory = 0,
                country = null,
                distance = 520.5,
                elevationHigh = 120.0,
                elevationLow = 98.0,
                endLatLng = emptyList(),
                hazardous = false,
                id = id,
                maximumGrade = 8.0,
                name = segmentName,
                isPrivate = false,
                resourceState = 2,
                starred = true,
                startLatLng = emptyList(),
                state = null,
            ),
            startDate = "2026-05-31T07:14:44Z",
            startDateLocal = "2026-05-31T09:14:44Z",
            startIndex = 10,
            visibility = null,
        )
}
