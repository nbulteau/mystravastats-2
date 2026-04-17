package me.nicolas.stravastats.api.controllers

import com.ninjasquad.springmockk.MockkBean
import io.mockk.every
import me.nicolas.stravastats.TestHelper
import me.nicolas.stravastats.domain.business.ActivityType
import me.nicolas.stravastats.domain.services.IActivityService
import org.hamcrest.Matchers.startsWith
import org.junit.jupiter.api.Test
import org.junit.jupiter.api.extension.ExtendWith
import org.springframework.beans.factory.annotation.Autowired
import org.springframework.boot.webmvc.test.autoconfigure.WebMvcTest
import org.springframework.http.MediaType
import org.springframework.test.context.junit.jupiter.SpringExtension
import org.springframework.test.web.servlet.MockMvc
import org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get
import org.springframework.test.web.servlet.result.MockMvcResultMatchers.content
import org.springframework.test.web.servlet.result.MockMvcResultMatchers.header
import org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath
import org.springframework.test.web.servlet.result.MockMvcResultMatchers.status

@ExtendWith(SpringExtension::class)
@WebMvcTest(ActivitiesController::class)
class ActivitiesControllerTest {

    @Autowired
    private lateinit var mockMvc: MockMvc

    @MockkBean
    private lateinit var activityService: IActivityService

    @Test
    fun `get activities returns activities when valid activity type`() {
        // GIVEN
        val activityTypes = setOf(ActivityType.Run)
        every { activityService.getActivitiesByActivityTypeAndYear(activityTypes, 2023) } returns listOf(TestHelper.stravaActivity)

        // WHEN
        mockMvc.perform(
            get("/api/activities")
                .param("activityType", "Run")
                .param("year", "2023")
                .accept(MediaType.APPLICATION_JSON)
        )
            // THEN
            .andExpect(status().isOk)
            .andExpect(content().contentType(MediaType.APPLICATION_JSON))
            .andExpect(jsonPath("$[0].name").value("Morning Run"))
    }

    @Test
    fun `get activities returns bad request when activity type is invalid`() {
        // WHEN
        mockMvc.perform(
            get("/api/activities")
                .param("activityType", "InvalidType")
                .accept(MediaType.APPLICATION_JSON)
        )
            // THEN
            .andExpect(status().isBadRequest)
            .andExpect(content().contentType(MediaType.APPLICATION_JSON))
            .andExpect(jsonPath("$.message").value("Illegal argument"))
            .andExpect(jsonPath("$.description").value(startsWith("Illegal argument : 'Unknown activity type: 'InvalidType'")))
    }

    @Test
    fun `export csv returns attachment when valid activity type and year`() {
        // GIVEN
        val activityTypes = setOf(ActivityType.Run)
        val csvContent = "name,distance\nMorning Run,10\n"
        every { activityService.exportCSV(activityTypes, 2023) } returns csvContent

        // WHEN
        mockMvc.perform(
            get("/api/activities/csv")
                .param("activityType", "Run")
                .param("year", "2023")
        )
            // THEN
            .andExpect(status().isOk)
            .andExpect(content().contentTypeCompatibleWith("text/csv"))
            .andExpect(header().string("Content-Disposition", "attachment; filename=\"activities-2023.csv\""))
            .andExpect(content().string(csvContent))
    }

    @Test
    fun `export csv uses all-years suffix when year is omitted`() {
        // GIVEN
        val activityTypes = setOf(ActivityType.Run)
        val csvContent = "name,distance\nMorning Run,10\n"
        every { activityService.exportCSV(activityTypes, null) } returns csvContent

        // WHEN
        mockMvc.perform(
            get("/api/activities/csv")
                .param("activityType", "Run")
        )
            // THEN
            .andExpect(status().isOk)
            .andExpect(content().contentTypeCompatibleWith("text/csv"))
            .andExpect(header().string("Content-Disposition", "attachment; filename=\"activities-all-years.csv\""))
            .andExpect(content().string(csvContent))
    }
}
