package me.nicolas.stravastats.api.controllers

import com.ninjasquad.springmockk.MockkBean
import io.mockk.every
import me.nicolas.stravastats.domain.business.Period
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
@WebMvcTest(ChartsController::class)
class ChartsControllerTest {

    @Autowired
    private lateinit var mockMvc: MockMvc

    @MockkBean
    private lateinit var chartsService: IChartsService

    @Test
    fun `get distance by period returns distances when valid activity type, year, and period`() {
        // GIVEN
        val activityTypes = setOf(ActivityType.Run)
        val year = 2022
        val period = Period.MONTHS

        every { chartsService.getDistanceByPeriodByActivityTypeByYear(activityTypes, year, period) } returns listOf(Pair("January", 100.0))

        // WHEN
        mockMvc.perform(
            get("/charts/distance-by-period")
                .param("activityType", activityTypes.joinToString("_"))
                .param("year", year.toString())
                .param("period", period.name)
                .accept(MediaType.APPLICATION_JSON)
        )
            // THEN
            .andExpect(status().isOk)
            .andExpect(content().contentType(MediaType.APPLICATION_JSON))
            .andExpect(jsonPath("$[0].January").value(100.0))
    }

    @Test
    fun `get distance by period returns empty list when no distances found`() {
        // GIVEN
        val activityTypes = setOf(ActivityType.Run)
        val year = 2022
        val period = Period.MONTHS

        every { chartsService.getDistanceByPeriodByActivityTypeByYear(activityTypes, year, period) } returns emptyList()

        // WHEN
        mockMvc.perform(
            get("/charts/distance-by-period")
                .param("activityType", activityTypes.joinToString("_"))
                .param("year", year.toString())
                .param("period", period.name)
                .accept(MediaType.APPLICATION_JSON)
        )
            // THEN
            .andExpect(status().isOk)
            .andExpect(content().contentType(MediaType.APPLICATION_JSON))
            .andExpect(jsonPath("$").isEmpty)
    }

    @Test
    fun `get distance by period returns bad request when activity type is invalid`() {
        // WHEN
        mockMvc.perform(
            get("/charts/distance-by-period")
                .param("activityType", "InvalidType")
                .param("year", "2022")
                .param("period", Period.MONTHS.name)
                .accept(MediaType.APPLICATION_JSON)
        )
            // THEN
            .andExpect(status().isBadRequest)
            .andExpect(content().contentType(MediaType.APPLICATION_JSON))
            .andExpect(jsonPath("$.message").value("Illegal argument"))
    }

    @Test
    fun `get distance by period returns bad request when year is invalid`() {
        // WHEN
        mockMvc.perform(
            get("/charts/distance-by-period")
                .param("activityType", ActivityType.Run.name)
                .param("year", "invalidYear")
                .param("period", Period.MONTHS.name)
                .accept(MediaType.APPLICATION_JSON)
        )
            // THEN
            .andExpect(status().isBadRequest)
            .andExpect(content().contentType(MediaType.APPLICATION_JSON))
            .andExpect(jsonPath("$.message").value("Invalid year value"))
    }

    @Test
    fun `get distance by period returns bad request when period is invalid`() {
        // WHEN
        mockMvc.perform(
            get("/charts/distance-by-period")
                .param("activityType", ActivityType.Run.name)
                .param("year", "2022")
                .param("period", "InvalidPeriod")
                .accept(MediaType.APPLICATION_JSON)
        )
            // THEN
            .andExpect(status().isBadRequest)
            .andExpect(content().contentType(MediaType.APPLICATION_JSON))
            .andExpect(jsonPath("$.message").value("Invalid period value"))
    }

    @Test
    fun `get elevation by period returns elevations when valid activity type, year, and period`() {
        // GIVEN
        val activityTypes = setOf(ActivityType.Run)
        val year = 2022
        val period = Period.MONTHS

        every { chartsService.getElevationByPeriodByActivityTypeByYear(activityTypes, year, period) } returns listOf(Pair("January", 500.0))

        // WHEN
        mockMvc.perform(
            get("/charts/elevation-by-period")
                .param("activityType", activityTypes.joinToString("_"))
                .param("year", year.toString())
                .param("period", period.name)
                .accept(MediaType.APPLICATION_JSON)
        )
            // THEN
            .andExpect(status().isOk)
            .andExpect(content().contentType(MediaType.APPLICATION_JSON))
            .andExpect(jsonPath("$[0].January").value(500.0))
    }

    @Test
    fun `get elevation by period returns empty list when no elevations found`() {
        // GIVEN
        val activityTypes = setOf(ActivityType.Run)
        val year = 2022
        val period = Period.MONTHS

        every { chartsService.getElevationByPeriodByActivityTypeByYear(activityTypes, year, period) } returns emptyList()

        // WHEN
        mockMvc.perform(
            get("/charts/elevation-by-period")
                .param("activityType", activityTypes.joinToString("_"))
                .param("year", year.toString())
                .param("period", period.name)
                .accept(MediaType.APPLICATION_JSON)
        )
            // THEN
            .andExpect(status().isOk)
            .andExpect(content().contentType(MediaType.APPLICATION_JSON))
            .andExpect(jsonPath("$").isEmpty)
    }

    @Test
    fun `get elevation by period returns bad request when activity type is invalid`() {
        // WHEN
        mockMvc.perform(
            get("/charts/elevation-by-period")
                .param("activityType", "InvalidType")
                .param("year", "2022")
                .param("period", Period.MONTHS.name)
                .accept(MediaType.APPLICATION_JSON)
        )
            // THEN
            .andExpect(status().isBadRequest)
            .andExpect(content().contentType(MediaType.APPLICATION_JSON))
            .andExpect(jsonPath("$.message").value("Illegal argument"))
    }

    @Test
    fun `get elevation by period returns bad request when year is invalid`() {
        // WHEN
        mockMvc.perform(
            get("/charts/elevation-by-period")
                .param("activityType", ActivityType.Run.name)
                .param("year", "invalidYear")
                .param("period", Period.MONTHS.name)
                .accept(MediaType.APPLICATION_JSON)
        )
            // THEN
            .andExpect(status().isBadRequest)
            .andExpect(content().contentType(MediaType.APPLICATION_JSON))
            .andExpect(jsonPath("$.message").value("Invalid year value"))
    }

    @Test
    fun `get elevation by period returns bad request when period is invalid`() {
        // WHEN
        mockMvc.perform(
            get("/charts/elevation-by-period")
                .param("activityType", ActivityType.Run.name)
                .param("year", "2022")
                .param("period", "InvalidPeriod")
                .accept(MediaType.APPLICATION_JSON)
        )
            // THEN
            .andExpect(status().isBadRequest)
            .andExpect(content().contentType(MediaType.APPLICATION_JSON))
            .andExpect(jsonPath("$.message").value("Invalid period value"))
    }
}