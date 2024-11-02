package me.nicolas.stravastats.api.controllers

import com.ninjasquad.springmockk.MockkBean
import io.mockk.every
import me.nicolas.stravastats.domain.business.ActivityType
import me.nicolas.stravastats.domain.services.IChartsService
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
@WebMvcTest(DashboardController::class)
class DashboardControllerTest{

    @Autowired
    private lateinit var mockMvc: MockMvc

    @MockkBean
    private lateinit var chartsService: IChartsService

    @Test
    fun `get cumulative distance per year returns distances when valid activity type`() {
        // GIVEN
        val activityType = ActivityType.Run
        val cumulativeDistances = mapOf(
            "2021" to mapOf("January" to 100.0, "February" to 150.0),
            "2022" to mapOf("January" to 200.0, "February" to 250.0)
        )

        val cumulativeElevations = mapOf(
            "2021" to mapOf("January" to 500, "February" to 1500),
            "2022" to mapOf("January" to 300, "February" to 2500)
        )

        every { chartsService.getCumulativeDistancePerYear(activityType) } returns cumulativeDistances
        every { chartsService.getCumulativeElevationPerYear(activityType) } returns cumulativeElevations

        // WHEN
        mockMvc.perform(
            get("/dashboard/cumulative-data-per-year")
                .param("activityType", activityType.name)
                .accept(MediaType.APPLICATION_JSON)
        )
            // THEN
            .andExpect(status().isOk)
            .andExpect(content().contentType(MediaType.APPLICATION_JSON))
            .andExpect(jsonPath("$.distance['2021'].January").value(100.0))
            .andExpect(jsonPath("$.distance['2021'].February").value(150.0))
            .andExpect(jsonPath("$.elevation['2021'].January").value(500))
            .andExpect(jsonPath("$.elevation['2022'].February").value(2500))
    }

    @Test
    fun `get cumulative distance per year returns empty map when no distances found`() {
        // GIVEN
        val activityType = ActivityType.Run

        every { chartsService.getCumulativeDistancePerYear(activityType) } returns emptyMap()
        every { chartsService.getCumulativeElevationPerYear(activityType) } returns emptyMap()

        // WHEN
        mockMvc.perform(
            get("/dashboard/cumulative-data-per-year")
                .param("activityType", activityType.name)
                .accept(MediaType.APPLICATION_JSON)
        )
            // THEN
            .andExpect(status().isOk)
            .andExpect(content().contentType(MediaType.APPLICATION_JSON))
            .andExpect(jsonPath("$.distance").isEmpty)
            .andExpect(jsonPath("$.elevation").isEmpty)

    }

    @Test
    fun `get cumulative distance per year returns bad request when activity type is invalid`() {
        // WHEN
        mockMvc.perform(
            get("/dashboard/cumulative-data-per-year")
                .param("activityType", "InvalidType")
                .accept(MediaType.APPLICATION_JSON)
        )
            // THEN
            .andExpect(status().isBadRequest)
            .andExpect(content().contentType(MediaType.APPLICATION_JSON))
            .andExpect(jsonPath("$.message").value("Unknown stravaActivity type"))
    }
}
