package me.nicolas.stravastats.api.controllers

import com.ninjasquad.springmockk.MockkBean
import io.mockk.every
import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.business.ActivityType

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
        val activityTypes = setOf(ActivityType.Run)
        val year = 2023
        val activities = listOf<StravaActivity>()
        val statistic = GlobalStatistic("Nb activities", activities, "%d", List<StravaActivity>::size)
        val statistics = listOf<Statistic>(statistic)

        every { statisticsService.getStatistics(activityTypes, year) } returns (statistics)

        // WHEN
        mockMvc.perform(
            get("/statistics")
                .param("activityType", activityTypes.joinToString("_"))
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
        val activityTypes = setOf(ActivityType.Run)
        val activities = listOf<StravaActivity>()
        val statistic = GlobalStatistic("Nb activities", activities, "%d", List<StravaActivity>::size)
        val statistics = listOf<Statistic>(statistic)

        every { statisticsService.getStatistics(activityTypes, null) } returns statistics

        // WHEN
        mockMvc.perform(
            get("/statistics")
                .param("activityType", activityTypes.joinToString("_"))
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
            .andExpect(jsonPath("$.message").value("Illegal argument")) // replace with actual error message
    }

}