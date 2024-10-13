package me.nicolas.stravastats.api.controllers

import com.ninjasquad.springmockk.MockkBean
import io.mockk.every
import me.nicolas.stravastats.TestHelper
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
@WebMvcTest(AthleteController::class)
class StravaAthleteControllerTest {

    @Autowired
    private lateinit var mockMvc: MockMvc

    @MockkBean
    private lateinit var stravaProxy: IActivityProvider

    @Test
    fun `get athlete returns athlete when athlete is found`() {
        // GIVEN
        val athlete = TestHelper.stravaAthlete
        every { stravaProxy.athlete() } returns athlete

        // WHEN
        val result = mockMvc.perform(
            get("/athletes/me")
                .accept(MediaType.APPLICATION_JSON)
        )

        // THEN
        result.andExpect(status().isOk)
            .andExpect(content().contentType(MediaType.APPLICATION_JSON))
            .andExpect(jsonPath("$.id").value(123456))
            .andExpect(jsonPath("$.username").value("john.doe"))
    }
}