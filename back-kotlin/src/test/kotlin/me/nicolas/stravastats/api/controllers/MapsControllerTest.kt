package me.nicolas.stravastats.api.controllers

import com.ninjasquad.springmockk.MockkBean
import io.mockk.every
import me.nicolas.stravastats.TestHelper
import me.nicolas.stravastats.domain.business.ActivityType

import me.nicolas.stravastats.domain.services.activityproviders.IActivityProvider
import me.nicolas.stravastats.domain.business.strava.stream.LatLngStream
import me.nicolas.stravastats.domain.business.strava.stream.Stream
import me.nicolas.stravastats.domain.business.strava.stream.TimeStream
import me.nicolas.stravastats.domain.business.strava.stream.DistanceStream
import org.junit.jupiter.api.Test
import org.junit.jupiter.api.extension.ExtendWith
import org.springframework.beans.factory.annotation.Autowired
import org.springframework.boot.webmvc.test.autoconfigure.WebMvcTest
import org.springframework.http.MediaType
import org.springframework.test.context.junit.jupiter.SpringExtension
import org.springframework.test.web.servlet.MockMvc
import org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get
import org.springframework.test.web.servlet.result.MockMvcResultMatchers.*

@ExtendWith(SpringExtension::class)
@WebMvcTest(MapsController::class)
class MapsControllerTest{

    @Autowired
    private lateinit var mockMvc: MockMvc

    @MockkBean
    private lateinit var stravaProxy: IActivityProvider

    @Test
    fun `get GPX returns GPX coordinates when valid activity type and year`() {
        // GIVEN
        val activityTypes = setOf(ActivityType.Run)
        val year = 2022
        val activity = TestHelper.stravaActivity.copy(
            stream = Stream(
                distance = DistanceStream(listOf(0.0, 100.0, 200.0), 3, "high", "distance"),
                time = TimeStream(listOf(0, 10, 20), 3, "high", "time"),
                latlng = LatLngStream(
                    data = listOf(
                        listOf(48.8566, 2.3522),
                        listOf(48.8570, 2.3530),
                        listOf(48.8573, 2.3536),
                    ),
                    originalSize = 3,
                    resolution = "high",
                    seriesType = "distance",
                ),
            ),
        )
        val activities = listOf(activity)

        every { stravaProxy.getActivitiesByActivityTypeAndYear(activityTypes, year) } returns activities

        // WHEN
        mockMvc.perform(
            get("/api/maps/gpx")
                .param("activityType", activityTypes.joinToString("_"))
                .param("year", year.toString())
                .accept(MediaType.APPLICATION_JSON)
        )
            // THEN
            .andExpect(status().isOk)
            .andExpect(content().contentType(MediaType.APPLICATION_JSON))
            .andExpect(jsonPath("$[0].activityId").value(activity.id))
            .andExpect(jsonPath("$[0].activityName").value(activity.name))
            .andExpect(jsonPath("$[0].coordinates").isArray)
            .andExpect(jsonPath("$[0].coordinates[0][0]").value(48.8566))
    }

    @Test
    fun `get GPX returns empty list when no activities found`() {
        // GIVEN
        val activityTypes = setOf(ActivityType.Run)
        val year = 2022

        every { stravaProxy.getActivitiesByActivityTypeAndYear(activityTypes, year) } returns emptyList()

        // WHEN
        mockMvc.perform(
            get("/api/maps/gpx")
                .param("activityType", activityTypes.joinToString("_"))
                .param("year", year.toString())
                .accept(MediaType.APPLICATION_JSON)
        )
            // THEN
            .andExpect(status().isOk)
            .andExpect(content().contentType(MediaType.APPLICATION_JSON))
            .andExpect(jsonPath("$").isEmpty)
    }

    @Test
    fun `get passages counts activities instead of GPS points`() {
        // GIVEN
        val activityTypes = setOf(ActivityType.Ride)
        val year = 2026
        val sparseActivity = TestHelper.stravaActivity.copy(
            id = 1001,
            type = ActivityType.Ride.name,
            stream = Stream(
                distance = DistanceStream(listOf(0.0, 100.0, 200.0), 3, "high", "distance"),
                time = TimeStream(listOf(0, 10, 20), 3, "high", "time"),
                latlng = LatLngStream(
                    data = listOf(
                        listOf(48.0000, 2.0000),
                        listOf(48.0010, 2.0000),
                        listOf(48.0020, 2.0000),
                        listOf(48.0030, 2.0000),
                    ),
                    originalSize = 4,
                    resolution = "high",
                    seriesType = "distance",
                ),
            ),
        )
        val denseActivity = TestHelper.stravaActivity.copy(
            id = 1002,
            type = ActivityType.Ride.name,
            stream = Stream(
                distance = DistanceStream(listOf(0.0, 100.0, 200.0), 3, "high", "distance"),
                time = TimeStream(listOf(0, 10, 20), 3, "high", "time"),
                latlng = LatLngStream(
                    data = listOf(
                        listOf(48.0000, 2.0000),
                        listOf(48.0003, 2.0000),
                        listOf(48.0006, 2.0000),
                        listOf(48.0009, 2.0000),
                        listOf(48.0012, 2.0000),
                        listOf(48.0015, 2.0000),
                        listOf(48.0018, 2.0000),
                        listOf(48.0021, 2.0000),
                        listOf(48.0024, 2.0000),
                        listOf(48.0027, 2.0000),
                        listOf(48.0030, 2.0000),
                    ),
                    originalSize = 11,
                    resolution = "high",
                    seriesType = "distance",
                ),
            ),
        )

        every { stravaProxy.getActivitiesByActivityTypeAndYear(activityTypes, year) } returns listOf(sparseActivity, denseActivity)
        every { stravaProxy.cacheIdentity() } returns null

        // WHEN
        mockMvc.perform(
            get("/api/maps/passages")
                .param("activityType", activityTypes.joinToString("_"))
                .param("year", year.toString())
                .accept(MediaType.APPLICATION_JSON)
        )
            // THEN
            .andExpect(status().isOk)
            .andExpect(content().contentType(MediaType.APPLICATION_JSON))
            .andExpect(jsonPath("$.includedActivities").value(2))
            .andExpect(jsonPath("$.excludedActivities").value(0))
            .andExpect(jsonPath("$.resolutionMeters").value(120))
            .andExpect(jsonPath("$.minPassageCount").value(1))
            .andExpect(jsonPath("$.segments").isArray)
            .andExpect(jsonPath("$.segments[0].passageCount").value(2))
            .andExpect(jsonPath("$.segments[0].activityTypeCounts.Ride").value(2))
    }

    @Test
    fun `get passages in all years filters one-off corridors`() {
        // GIVEN
        val activityTypes = setOf(ActivityType.Ride)
        val repeatedA = TestHelper.stravaActivity.copy(
            id = 1101,
            type = ActivityType.Ride.name,
            stream = Stream(
                distance = DistanceStream(listOf(0.0, 100.0, 200.0), 3, "high", "distance"),
                time = TimeStream(listOf(0, 10, 20), 3, "high", "time"),
                latlng = LatLngStream(
                    data = listOf(
                        listOf(48.0000, 2.0000),
                        listOf(48.0010, 2.0000),
                        listOf(48.0020, 2.0000),
                    ),
                    originalSize = 3,
                    resolution = "high",
                    seriesType = "distance",
                ),
            ),
        )
        val repeatedB = repeatedA.copy(id = 1102)
        val oneOff = TestHelper.stravaActivity.copy(
            id = 1103,
            type = ActivityType.Ride.name,
            stream = Stream(
                distance = DistanceStream(listOf(0.0, 100.0, 200.0), 3, "high", "distance"),
                time = TimeStream(listOf(0, 10, 20), 3, "high", "time"),
                latlng = LatLngStream(
                    data = listOf(
                        listOf(49.0000, 3.0000),
                        listOf(49.0010, 3.0000),
                        listOf(49.0020, 3.0000),
                    ),
                    originalSize = 3,
                    resolution = "high",
                    seriesType = "distance",
                ),
            ),
        )

        every { stravaProxy.getActivitiesByActivityTypeAndYear(activityTypes, null) } returns listOf(repeatedA, repeatedB, oneOff)
        every { stravaProxy.cacheIdentity() } returns null

        // WHEN
        mockMvc.perform(
            get("/api/maps/passages")
                .param("activityType", activityTypes.joinToString("_"))
                .accept(MediaType.APPLICATION_JSON)
        )
            // THEN
            .andExpect(status().isOk)
            .andExpect(jsonPath("$.includedActivities").value(3))
            .andExpect(jsonPath("$.resolutionMeters").value(250))
            .andExpect(jsonPath("$.minPassageCount").value(2))
            .andExpect(jsonPath("$.omittedSegments").value(org.hamcrest.Matchers.greaterThan(0)))
            .andExpect(jsonPath("$.segments[0].passageCount").value(2))
    }

    @Test
    fun `get GPX returns bad request when activity type is invalid`() {
        // GIVEN
        // WHEN
        mockMvc.perform(
            get("/api/maps/gpx")
                .param("activityType", "InvalidType")
                .param("year", "2022")
                .accept(MediaType.APPLICATION_JSON)
        )
            // THEN
            .andExpect(status().isBadRequest)
            .andExpect(content().contentType(MediaType.APPLICATION_JSON))
            .andExpect(jsonPath("$.message").value("Illegal argument"))
    }

    @Test
    fun `get GPX returns bad request when year is invalid`() {
        // GIVEN
        // WHEN
        mockMvc.perform(
            get("/api/maps/gpx")
                .param("activityType", ActivityType.Run.name)
                .param("year", "invalidYear")
                .accept(MediaType.APPLICATION_JSON)
        )
            // THEN
            .andExpect(status().isBadRequest)
            .andExpect(content().contentType(MediaType.APPLICATION_JSON))
            .andExpect(jsonPath("$.message").value("Invalid year value"))
    }
}
