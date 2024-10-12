package me.nicolas.stravastats.api.controllers

import com.ninjasquad.springmockk.MockkBean
import io.mockk.every
import me.nicolas.stravastats.TestHelper
import me.nicolas.stravastats.domain.business.strava.ActivityType
import me.nicolas.stravastats.domain.services.activityproviders.IActivityProvider
import org.junit.jupiter.api.Test
import org.junit.jupiter.api.extension.ExtendWith
import org.springframework.beans.factory.annotation.Autowired
import org.springframework.boot.test.autoconfigure.web.servlet.WebMvcTest
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
        val activityType = ActivityType.Run
        val year = 2022
        val activities = listOf(TestHelper.activity)

        every { stravaProxy.getFilteredActivitiesByActivityTypeAndYear(activityType, year) } returns activities

        // WHEN
        mockMvc.perform(
            get("/maps/gpx")
                .param("activityType", activityType.name)
                .param("year", year.toString())
                .accept(MediaType.APPLICATION_JSON)
        )
            // THEN
            .andExpect(status().isOk)
            .andExpect(content().contentType(MediaType.APPLICATION_JSON))
            .andExpect(jsonPath("$[0]").isArray)
    }

    @Test
    fun `get GPX returns empty list when no activities found`() {
        // GIVEN
        val activityType = ActivityType.Run
        val year = 2022

        every { stravaProxy.getFilteredActivitiesByActivityTypeAndYear(activityType, year) } returns emptyList()

        // WHEN
        mockMvc.perform(
            get("/maps/gpx")
                .param("activityType", activityType.name)
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
        // WHEN
        mockMvc.perform(
            get("/maps/gpx")
                .param("activityType", "InvalidType")
                .param("year", "2022")
                .accept(MediaType.APPLICATION_JSON)
        )
            // THEN
            .andExpect(status().isBadRequest)
            .andExpect(content().contentType(MediaType.APPLICATION_JSON))
            .andExpect(jsonPath("$.message").value("Unknown activity type"))
    }

    @Test
    fun `get GPX returns bad request when year is invalid`() {
        // WHEN
        mockMvc.perform(
            get("/maps/gpx")
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