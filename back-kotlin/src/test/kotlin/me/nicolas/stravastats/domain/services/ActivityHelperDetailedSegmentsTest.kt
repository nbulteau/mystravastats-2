package me.nicolas.stravastats.domain.services

import me.nicolas.stravastats.domain.business.strava.MetaActivity
import me.nicolas.stravastats.domain.business.strava.MetaAthlete
import me.nicolas.stravastats.domain.business.strava.Segment
import me.nicolas.stravastats.domain.business.strava.StravaDetailedActivity
import me.nicolas.stravastats.domain.business.strava.StravaSegmentEffort
import me.nicolas.stravastats.domain.services.statistics.StatisticsFixtures
import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Assertions.assertNotNull
import org.junit.jupiter.api.Assertions.assertTrue
import org.junit.jupiter.api.Test

class ActivityHelperDetailedSegmentsTest {

    @Test
    fun `buildActivityEfforts separates ascent and descent labels for segment efforts`() {
        // GIVEN
        val detailedActivity = buildSyntheticDetailedActivity()

        // WHEN
        val efforts = with(ActivityHelper) { detailedActivity.buildActivityEfforts() }
        val segmentEfforts = efforts.filter { effort -> effort.label.contains("MURAILLE DE CHINE") }

        // THEN
        assertEquals(2, segmentEfforts.size)
        val ascent = segmentEfforts.firstOrNull { effort -> effort.label.contains("(ascent)") }
        val descent = segmentEfforts.firstOrNull { effort -> effort.label.contains("(descent)") }
        assertNotNull(ascent)
        assertNotNull(descent)
        assertTrue(ascent!!.deltaAltitude > 0.0)
        assertTrue(descent!!.deltaAltitude < 0.0)
    }

    @Test
    fun `buildActivityEfforts keeps finite delta altitude when stream contains NaN`() {
        // GIVEN
        val streamWithNaN = StatisticsFixtures.defaultStream(
            distances = listOf(0.0, 100.0, 200.0),
            times = listOf(0, 10, 20),
            altitudes = listOf(100.0, Double.NaN, 110.0),
        )
        val activity = StatisticsFixtures.syntheticRideActivity(id = 77L, stream = streamWithNaN)
            .toStravaDetailedActivity()
            .copy(
                stream = streamWithNaN,
                segmentEfforts = listOf(
                    buildSegmentEffort(
                        id = 2001L,
                        name = "NaN climb",
                        startIndex = 0,
                        endIndex = 2,
                        averageGrade = 5.0
                    )
                )
            )

        // WHEN
        val effort = with(ActivityHelper) { activity.buildActivityEfforts() }
            .first { built -> built.label.contains("MURAILLE DE CHINE") }

        // THEN
        assertTrue(!effort.deltaAltitude.isNaN() && !effort.deltaAltitude.isInfinite())
        assertTrue(effort.deltaAltitude > 0.0)
    }

    @Test
    fun `buildActivityEfforts labels detected climbs explicitly`() {
        // GIVEN
        val stream = StatisticsFixtures.defaultStream(
            distances = listOf(
                0.0, 100.0, 200.0, 300.0, 400.0, 500.0, 600.0, 700.0, 800.0, 900.0,
                1000.0, 1100.0, 1200.0, 1300.0, 1400.0, 1500.0, 1600.0, 1700.0, 1800.0,
            ),
            times = listOf(
                0, 40, 80, 120, 160, 200, 240, 280, 320, 360,
                400, 440, 480, 520, 560, 600, 640, 680, 720,
            ),
            altitudes = listOf(
                100.0, 106.0, 112.0, 118.0, 124.0, 126.0, 125.0, 127.0, 133.0, 139.0,
                145.0, 151.0, 157.0, 163.0, 169.0, 175.0, 181.0, 187.0, 193.0,
            ),
        )
        val activity = StatisticsFixtures.syntheticRideActivity(id = 78L, stream = stream)
            .toStravaDetailedActivity()
            .copy(stream = stream, segmentEfforts = emptyList())

        // WHEN
        val climb = with(ActivityHelper) { activity.buildActivityEfforts() }
            .firstOrNull { effort -> effort.label.startsWith("Climb 1 -") }

        // THEN
        assertNotNull(climb)
        assertTrue(!climb!!.label.contains("Slope"))
        assertTrue(climb.label.contains("D+"))
        assertTrue(climb.deltaAltitude > 0.0)
    }

    private fun buildSyntheticDetailedActivity(): StravaDetailedActivity {
        val stream = StatisticsFixtures.defaultStream(
            distances = listOf(0.0, 100.0, 200.0, 300.0, 400.0, 500.0, 600.0),
            times = listOf(0, 10, 20, 30, 40, 50, 60),
            altitudes = listOf(100.0, 102.0, 105.0, 108.0, 106.0, 104.0, 102.0),
        )
        val baseActivity = StatisticsFixtures.syntheticRideActivity(id = 99L, stream = stream)

        return baseActivity.toStravaDetailedActivity().copy(
            stream = stream,
            segmentEfforts = listOf(
                buildSegmentEffort(id = 1001L, name = "Muraille montee", startIndex = 0, endIndex = 3, averageGrade = 8.0),
                buildSegmentEffort(id = 1002L, name = "Muraille descente", startIndex = 3, endIndex = 6, averageGrade = -8.0),
            )
        )
    }

    private fun buildSegmentEffort(
        id: Long,
        name: String,
        startIndex: Int,
        endIndex: Int,
        averageGrade: Double,
    ): StravaSegmentEffort {
        return StravaSegmentEffort(
            achievements = emptyList(),
            activity = MetaActivity(99L),
            athlete = MetaAthlete(1L),
            averageCadence = 0.0,
            averageHeartRate = 0.0,
            averageWatts = 180.0,
            deviceWatts = false,
            distance = 300.0,
            elapsedTime = 30,
            endIndex = endIndex,
            hidden = false,
            id = id,
            komRank = null,
            maxHeartRate = 0.0,
            movingTime = 30,
            name = name,
            prRank = 1,
            resourceState = 2,
            segment = Segment(
                activityType = "Ride",
                averageGrade = averageGrade,
                city = null,
                climbCategory = 4,
                country = null,
                distance = 300.0,
                elevationHigh = 108.0,
                elevationLow = 100.0,
                endLatLng = listOf(45.0, 6.0),
                hazardous = false,
                id = 9000L + id,
                maximumGrade = 10.0,
                name = "MURAILLE DE CHINE <Alpe d'Huez>",
                isPrivate = false,
                resourceState = 2,
                starred = true,
                startLatLng = listOf(45.0, 6.0),
                state = null,
            ),
            startDate = "2025-07-30T08:00:00Z",
            startDateLocal = "2025-07-30T10:00:00Z",
            startIndex = startIndex,
            visibility = null,
        )
    }
}
