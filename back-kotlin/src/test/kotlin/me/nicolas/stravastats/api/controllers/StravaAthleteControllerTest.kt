package me.nicolas.stravastats.api.controllers

import com.ninjasquad.springmockk.MockkBean
import io.mockk.every
import me.nicolas.stravastats.TestHelper
import me.nicolas.stravastats.domain.business.ActivityType
import me.nicolas.stravastats.domain.business.AthleteFtpSetting
import me.nicolas.stravastats.domain.business.AthletePerformanceSettings
import me.nicolas.stravastats.domain.business.FtpEstimate
import me.nicolas.stravastats.domain.services.IAthletePerformanceSettingsService
import me.nicolas.stravastats.domain.services.IHeartRateZoneService
import me.nicolas.stravastats.domain.services.activityproviders.IActivityProvider
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
@WebMvcTest(AthleteController::class)
class StravaAthleteControllerTest {

    @Autowired
    private lateinit var mockMvc: MockMvc

    @MockkBean
    private lateinit var stravaProxy: IActivityProvider

    @MockkBean
    private lateinit var heartRateZoneService: IHeartRateZoneService

    @MockkBean
    private lateinit var performanceSettingsService: IAthletePerformanceSettingsService

    @Test
    fun `get athlete returns athlete when athlete is found`() {
        // GIVEN
        val athlete = TestHelper.stravaAthlete
        every { stravaProxy.athlete() } returns athlete

        // WHEN
        val result = mockMvc.perform(
            get("/api/athletes/me")
                .accept(MediaType.APPLICATION_JSON)
        )

        // THEN
        result.andExpect(status().isOk)
            .andExpect(content().contentType(MediaType.APPLICATION_JSON))
            .andExpect(jsonPath("$.id").value(123456))
            .andExpect(jsonPath("$.username").value("john.doe"))
    }

    @Test
    fun `get performance settings returns persisted FTP history`() {
        // GIVEN
        every { performanceSettingsService.getSettings() } returns AthletePerformanceSettings(
            ftpHistory = listOf(AthleteFtpSetting(effectiveFrom = "2026-05-01", ftp = 160)),
            weightKg = 72.5,
        )

        // WHEN
        val result = mockMvc.perform(
            get("/api/athletes/me/performance-settings")
                .accept(MediaType.APPLICATION_JSON)
        )

        // THEN
        result.andExpect(status().isOk)
            .andExpect(content().contentType(MediaType.APPLICATION_JSON))
            .andExpect(jsonPath("$.ftpHistory[0].effectiveFrom").value("2026-05-01"))
            .andExpect(jsonPath("$.ftpHistory[0].ftp").value(160))
            .andExpect(jsonPath("$.weightKg").value(72.5))
    }

    @Test
    fun `get FTP estimate returns derived estimate`() {
        // GIVEN
        every {
            performanceSettingsService.estimateFtp(setOf(ActivityType.Ride), 180)
        } returns FtpEstimate(
            available = true,
            ftp = 215,
            method = "best-60min",
            methodLabel = "Best 60 min power",
            bestPower = 215,
            multiplier = 1.0,
            basedOnSeconds = 3600,
            confidence = "high",
            source = "Power meter, last 180 days",
            sourceKind = "power-meter",
            activityId = 9901,
            activityName = "FTP test",
            activityType = "Ride",
            activityDate = "2026-06-20",
            windowDays = 180,
            activityCount = 1,
        )

        // WHEN
        val result = mockMvc.perform(
            get("/api/athletes/me/ftp-estimate?activityType=Ride&days=180")
                .accept(MediaType.APPLICATION_JSON)
        )

        // THEN
        result.andExpect(status().isOk)
            .andExpect(content().contentType(MediaType.APPLICATION_JSON))
            .andExpect(jsonPath("$.available").value(true))
            .andExpect(jsonPath("$.ftp").value(215))
            .andExpect(jsonPath("$.method").value("best-60min"))
            .andExpect(jsonPath("$.confidence").value("high"))
            .andExpect(jsonPath("$.activityDate").value("2026-06-20"))
    }

    @Test
    fun `update performance settings returns normalized settings`() {
        // GIVEN
        val settings = AthletePerformanceSettings(
            ftpHistory = listOf(AthleteFtpSetting(effectiveFrom = "2026-05-01", ftp = 160)),
            weightKg = 72.5,
        )
        every { performanceSettingsService.updateSettings(settings) } returns settings

        // WHEN
        val result = mockMvc.perform(
            put("/api/athletes/me/performance-settings")
                .contentType(MediaType.APPLICATION_JSON)
                .content("""{"ftpHistory":[{"effectiveFrom":"2026-05-01","ftp":160}],"weightKg":72.5}""")
                .accept(MediaType.APPLICATION_JSON)
        )

        // THEN
        result.andExpect(status().isOk)
            .andExpect(content().contentType(MediaType.APPLICATION_JSON))
            .andExpect(jsonPath("$.ftpHistory[0].ftp").value(160))
            .andExpect(jsonPath("$.weightKg").value(72.5))
    }
}
