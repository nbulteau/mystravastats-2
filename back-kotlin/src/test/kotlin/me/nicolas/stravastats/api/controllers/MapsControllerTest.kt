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
