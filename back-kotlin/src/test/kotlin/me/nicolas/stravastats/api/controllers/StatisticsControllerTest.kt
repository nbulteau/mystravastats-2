package me.nicolas.stravastats.api.controllers

import com.ninjasquad.springmockk.MockkBean
import io.mockk.every
import me.nicolas.stravastats.domain.business.strava.Activity
import me.nicolas.stravastats.domain.business.strava.ActivityType
import me.nicolas.stravastats.domain.services.IStatisticsService
import me.nicolas.stravastats.domain.services.statistics.GlobalStatistic
import me.nicolas.stravastats.domain.services.statistics.Statistic
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
@WebMvcTest(StatisticsController::class)
class StatisticsControllerTest {

    @Autowired
    private lateinit var mockMvc: MockMvc

    @MockkBean
    private lateinit var statisticsService: IStatisticsService

    @Test
    fun getStatistics_returnsStatistics_whenStatisticsExist() {
        // GIVEN
        val activityType = ActivityType.Run
        val year = 2023
        val activities = listOf<Activity>()
        val statistic = GlobalStatistic("Nb activities", activities, "%d", List<Activity>::size)
        val statistics = listOf<Statistic>(statistic)

        every { statisticsService.getStatistics(activityType, year) } returns (statistics)

        // WHEN
        mockMvc.perform(
            get("/statistics")
                .param("activityType", activityType.name)
                .param("year", year.toString())
                .accept(MediaType.APPLICATION_JSON)
        )
            // THEN
            .andExpect(status().isOk)
            .andExpect(content().contentType(MediaType.APPLICATION_JSON))
            .andExpect(jsonPath("$[0].label").value(statistic.name))
            .andExpect(jsonPath("$[0].value").value(statistic.value))
    }

    @Test
    fun getStatistics_returnsBadRequest_whenYearIsMissing() {
        // GIVEN
        val activityType = ActivityType.Run
        val activities = listOf<Activity>()
        val statistic = GlobalStatistic("Nb activities", activities, "%d", List<Activity>::size)
        val statistics = listOf<Statistic>(statistic)

        every { statisticsService.getStatistics(activityType, null) } returns statistics

        // WHEN
        mockMvc.perform(
            get("/statistics")
                .param("activityType", activityType.name)
                .accept(MediaType.APPLICATION_JSON)
        )
            // THEN
            .andExpect(status().isOk)
            .andExpect(content().contentType(MediaType.APPLICATION_JSON))
            .andExpect(jsonPath("$[0].label").value(statistic.name))
            .andExpect(jsonPath("$[0].value").value(statistic.value))
    }

    @Test
    fun getStatistics_returnsNotFound_whenStatisticsDoNotExist() {
        // GIVEN
        val year = 2023

        // WHEN
        mockMvc.perform(
            get("/statistics")
                .param("activityType", "BadActivityType")
                .param("year", year.toString())
                .accept(MediaType.APPLICATION_JSON)
        )
            // THEN
            .andExpect(status().isBadRequest)
            .andExpect(content().contentType(MediaType.APPLICATION_JSON))
            .andExpect(jsonPath("$.message").value("Unknown activity type")) // replace with actual error message
    }

}