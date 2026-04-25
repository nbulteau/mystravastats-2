package me.nicolas.stravastats.api.controllers

import com.ninjasquad.springmockk.MockkBean
import io.mockk.every
import me.nicolas.stravastats.domain.business.AnnualGoalTargets
import me.nicolas.stravastats.domain.business.AnnualGoals
import me.nicolas.stravastats.domain.business.ActivityType
import me.nicolas.stravastats.domain.services.IDashboardService
import org.junit.jupiter.api.Test
import org.junit.jupiter.api.extension.ExtendWith
import org.springframework.beans.factory.annotation.Autowired
import org.springframework.boot.webmvc.test.autoconfigure.WebMvcTest
import org.springframework.http.MediaType
import org.springframework.test.context.junit.jupiter.SpringExtension
import org.springframework.test.web.servlet.MockMvc
import org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get
import org.springframework.test.web.servlet.request.MockMvcRequestBuilders.put
import org.springframework.test.web.servlet.result.MockMvcResultMatchers.*

@ExtendWith(SpringExtension::class)
@WebMvcTest(DashboardController::class)
class DashboardControllerTest{

    @Autowired
    private lateinit var mockMvc: MockMvc

    @MockkBean
    private lateinit var dashboardService: IDashboardService

    @Test
    fun `get cumulative distance per year returns distances when valid activity type`() {
        // GIVEN
        val activityTypes = setOf(ActivityType.Run)
        val cumulativeDistances = mapOf(
            "2021" to mapOf("January" to 100.0, "February" to 150.0),
            "2022" to mapOf("January" to 200.0, "February" to 250.0)
        )

        val cumulativeElevations = mapOf(
            "2021" to mapOf("January" to 500, "February" to 1500),
            "2022" to mapOf("January" to 300, "February" to 2500)
        )

        every { dashboardService.getCumulativeDistancePerYear(activityTypes) } returns cumulativeDistances
        every { dashboardService.getCumulativeElevationPerYear(activityTypes) } returns cumulativeElevations

        // WHEN
        mockMvc.perform(
            get("/api/dashboard/cumulative-data-per-year")
                .param("activityType", activityTypes.joinToString("_"))
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
        val activityTypes = setOf(ActivityType.Run)

        every { dashboardService.getCumulativeDistancePerYear(activityTypes) } returns emptyMap()
        every { dashboardService.getCumulativeElevationPerYear(activityTypes) } returns emptyMap()

        // WHEN
        mockMvc.perform(
            get("/api/dashboard/cumulative-data-per-year")
                .param("activityType", activityTypes.joinToString("_"))
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
        // GIVEN
        // WHEN
        mockMvc.perform(
            get("/api/dashboard/cumulative-data-per-year")
                .param("activityType", "InvalidType")
                .accept(MediaType.APPLICATION_JSON)
        )
            // THEN
            .andExpect(status().isBadRequest)
            .andExpect(content().contentType(MediaType.APPLICATION_JSON))
            .andExpect(jsonPath("$.message").value("Illegal argument"))
    }

    @Test
    fun `get annual goals returns goals when valid activity type and year`() {
        // GIVEN
        val activityTypes = setOf(ActivityType.Ride)
        every { dashboardService.getAnnualGoals(2026, activityTypes) } returns AnnualGoals(
            year = 2026,
            activityTypeKey = "Ride",
            targets = AnnualGoalTargets(distanceKm = 5000.0),
            progress = emptyList(),
        )

        // WHEN
        mockMvc.perform(
            get("/api/dashboard/annual-goals")
                .param("activityType", "Ride")
                .param("year", "2026")
                .accept(MediaType.APPLICATION_JSON)
        )
            // THEN
            .andExpect(status().isOk)
            .andExpect(content().contentType(MediaType.APPLICATION_JSON))
            .andExpect(jsonPath("$.year").value(2026))
            .andExpect(jsonPath("$.targets.distanceKm").value(5000.0))
    }

    @Test
    fun `put annual goals saves goals when request is valid`() {
        // GIVEN
        val activityTypes = setOf(ActivityType.Ride)
        val targets = AnnualGoalTargets(distanceKm = 5000.0, activities = 120)
        every { dashboardService.saveAnnualGoals(2026, activityTypes, targets) } returns AnnualGoals(
            year = 2026,
            activityTypeKey = "Ride",
            targets = targets,
            progress = emptyList(),
        )

        // WHEN
        mockMvc.perform(
            put("/api/dashboard/annual-goals")
                .param("activityType", "Ride")
                .param("year", "2026")
                .contentType(MediaType.APPLICATION_JSON)
                .content("""{"distanceKm":5000.0,"activities":120}""")
                .accept(MediaType.APPLICATION_JSON)
        )
            // THEN
            .andExpect(status().isOk)
            .andExpect(content().contentType(MediaType.APPLICATION_JSON))
            .andExpect(jsonPath("$.targets.distanceKm").value(5000.0))
            .andExpect(jsonPath("$.targets.activities").value(120))
    }
}
