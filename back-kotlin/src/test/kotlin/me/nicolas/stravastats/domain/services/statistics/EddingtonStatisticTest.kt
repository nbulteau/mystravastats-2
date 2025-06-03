package me.nicolas.stravastats.domain.services.statistics

import me.nicolas.stravastats.TestHelper
import me.nicolas.stravastats.domain.business.strava.AthleteRef
import me.nicolas.stravastats.domain.business.strava.StravaActivity
import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Test
import java.time.LocalDateTime
import java.time.format.DateTimeFormatter

class EddingtonStatisticTest {

    @Test
    fun `should calculate correct Eddington number`() {
        // GIVEN
        val activities = createTestActivities()

        // WHEN
        val eddingtonStatistic = EddingtonStatistic(activities)

        // THEN
        assertEquals("3 km", eddingtonStatistic.value)
        assertEquals(listOf(5, 4, 3, 2, 1), eddingtonStatistic.nbDaysDistanceIsReached)
    }

    @Test
    fun `should return zero when no activities`() {
        // GIVEN
        val activities = emptyList<StravaActivity>()

        // WHEN
        val eddingtonStatistic = EddingtonStatistic(activities)

        // THEN
        assertEquals("0 km", eddingtonStatistic.value)
        assertEquals(emptyList<Int>(), eddingtonStatistic.nbDaysDistanceIsReached)
    }

    @Test
    fun `should handle activities on same day`() {
        // GIVEN
        val activities = listOf(
            createActivity("2021-06-12T08:00:00Z", 2000.0),
            createActivity("2021-06-12T18:00:00Z", 3000.0)
        )

        // WHEN
        val eddingtonStatistic = EddingtonStatistic(activities)

        // THEN
        assertEquals("1 km", eddingtonStatistic.value)
        assertEquals(listOf(1, 1, 1, 1, 1), eddingtonStatistic.nbDaysDistanceIsReached)
    }

    private fun createTestActivities(): List<StravaActivity> {
        return listOf(
            createActivity("2021-06-10T08:00:00Z", 1000.0),  // 1km
            createActivity("2021-06-11T08:00:00Z", 2000.0),  // 2km
            createActivity("2021-06-12T08:00:00Z", 3000.0),  // 3km
            createActivity("2021-06-13T08:00:00Z", 4000.0),  // 4km
            createActivity("2021-06-14T08:00:00Z", 5000.0)   // 5km
        )
    }

    private fun createActivity(startDateLocal: String, distance: Double): StravaActivity {
        return StravaActivity(
            id = 0,
            name = "Test Activity",
            distance = distance,
            movingTime = 0,
            elapsedTime = 0,
            totalElevationGain = 0.0,
            elevHigh = 0.0,
            type = "Run",
            startDate = startDateLocal,
            startDateLocal = startDateLocal,
            startLatlng = null,
            averageSpeed = 0.0,
            maxSpeed = 0.0f,
            averageHeartrate = 0.0,
            maxHeartrate = 0,
            athlete = AthleteRef(id = 0),
            averageCadence = 0.0,
            averageWatts = 0,
            commute = false,
            kilojoules = 0.0,
            uploadId = 0L,
            weightedAverageWatts = 0,
            deviceWatts = false
        )
    }
}