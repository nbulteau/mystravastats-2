package me.nicolas.stravastats.api.controllers

import com.ninjasquad.springmockk.MockkBean
import io.mockk.every
import io.mockk.verify
import me.nicolas.stravastats.TestHelper
import me.nicolas.stravastats.api.dto.toDto
import me.nicolas.stravastats.domain.business.badges.BadgeCheckResult
import me.nicolas.stravastats.domain.business.badges.BadgeSetEnum
import me.nicolas.stravastats.domain.business.badges.DistanceBadge
import me.nicolas.stravastats.domain.business.badges.FamousClimbBadge
import me.nicolas.stravastats.domain.business.ActivityType

import me.nicolas.stravastats.domain.business.strava.GeoCoordinate
import me.nicolas.stravastats.domain.services.IBadgesService
import org.junit.jupiter.api.Test
import org.junit.jupiter.api.extension.ExtendWith
import org.springframework.beans.factory.annotation.Autowired
import org.springframework.boot.webmvc.test.autoconfigure.WebMvcTest
import org.springframework.http.MediaType
import org.springframework.test.context.junit.jupiter.SpringExtension
import org.springframework.test.web.servlet.MockMvc
import org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get
import org.springframework.test.web.servlet.result.MockMvcResultMatchers.content
import org.springframework.test.web.servlet.result.MockMvcResultMatchers.status
import tools.jackson.module.kotlin.jacksonObjectMapper

@ExtendWith(SpringExtension::class)
@WebMvcTest(BadgesController::class)
class BadgesControllerTest {

    @Autowired
    private lateinit var mockMvc: MockMvc

    @MockkBean
    private lateinit var badgesService: IBadgesService

    private val objectMapper = jacksonObjectMapper()

    @Test
    fun `get badges return general badges when badgeset is general`() {
        // GIVEN
        val activityTypes = setOf(ActivityType.Run)
        val year = 2021
        val badgeCheckResults = listOf(
            BadgeCheckResult(
                badge = DistanceBadge("badge1", 10),
                activities = listOf(TestHelper.stravaActivity),
                isCompleted = true,
            ),
        )
        val badgeCheckResultDtos = badgeCheckResults.map { badgeCheckResult -> badgeCheckResult.toDto(activityTypes) }

        every { badgesService.getGeneralBadges(activityTypes, year) } returns badgeCheckResults

        // WHEN
        mockMvc.perform(
            get("/badges")
                .param("activityType", activityTypes.joinToString("_"))
                .param("year", year.toString())
                .param("badgeSet", BadgeSetEnum.GENERAL.name)
        )
            .andExpect(status().isOk)
            .andExpect(content().contentType(MediaType.APPLICATION_JSON))
            // THEN
            .andExpect(content().json(objectMapper.writeValueAsString(badgeCheckResultDtos)))

        verify { badgesService.getGeneralBadges(activityTypes, year) }
    }

    @Test
    fun `get badges return famous badges when badgeset is famous`() {
        // GIVEN
        val activityTypes = setOf(ActivityType.Run)
        val year = 2021
        val badgeCheckResults = listOf(
            BadgeCheckResult(
                badge = FamousClimbBadge(
                    "badge1", "Alpe d'Huez", 1850,
                    start = GeoCoordinate(
                        latitude = 45.092401,
                        longitude = 6.0699443
                    ),
                    end = GeoCoordinate(
                        latitude = 45.0642762,
                        longitude = 6.0390149
                    ), 13.9, 1118, 8.0, 994
                ),
                activities = emptyList(),
                isCompleted = false,
            ),
        )
        val badgeCheckResultDtos = badgeCheckResults.map { it.toDto(activityTypes) }

        every { badgesService.getFamousBadges(activityTypes, year) } returns badgeCheckResults

        // WHEN
        mockMvc.perform(
            get("/badges")
                .param("activityType", activityTypes.joinToString("_"))
                .param("year", year.toString())
                .param("badgeSet", BadgeSetEnum.FAMOUS.name)
        )
            .andExpect(status().isOk)
            .andExpect(content().contentType(MediaType.APPLICATION_JSON))
            // THEN
            .andExpect(content().json(objectMapper.writeValueAsString(badgeCheckResultDtos)))

        verify { badgesService.getFamousBadges(activityTypes, year) }
    }
}
