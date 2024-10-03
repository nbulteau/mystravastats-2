package me.nicolas.stravastats.api.controllers

import com.ninjasquad.springmockk.MockkBean
import io.mockk.every
import me.nicolas.stravastats.TestHelper
import me.nicolas.stravastats.domain.business.strava.ActivityType
import me.nicolas.stravastats.domain.services.IActivityService
import org.junit.jupiter.api.Test
import org.junit.jupiter.api.extension.ExtendWith
import org.springframework.beans.factory.annotation.Autowired
import org.springframework.boot.test.autoconfigure.web.servlet.WebMvcTest
import org.springframework.data.domain.PageImpl
import org.springframework.data.domain.PageRequest
import org.springframework.data.domain.Sort
import org.springframework.http.MediaType
import org.springframework.test.context.junit.jupiter.SpringExtension
import org.springframework.test.web.servlet.MockMvc
import org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get
import org.springframework.test.web.servlet.result.MockMvcResultMatchers.*

@ExtendWith(SpringExtension::class)
@WebMvcTest(ActivitiesController::class)
class ActivitiesControllerTest {

    @Autowired
    private lateinit var mockMvc: MockMvc

    @MockkBean
    private lateinit var activityService: IActivityService

    @Test
    fun `list activities with pageable returns activities when valid pageable`() {
        // GIVEN
        val pageable = PageRequest.of(0, 10, Sort.by("averageSpeed"))

        val activity = TestHelper.activity
        val activities = listOf(activity)

        val page = PageImpl(activities, pageable, activities.size.toLong())

        every { activityService.listActivitiesPaginated(pageable) } returns page

        // WHEN
        mockMvc.perform(
            get("/activities/by-page")
                .param("page", "0")
                .param("size", "10")
                .param("sort", "averageSpeed")
                .accept(MediaType.APPLICATION_JSON)
        )
            // THEN
            .andExpect(status().isOk)
            .andExpect(content().contentType(MediaType.APPLICATION_JSON))
            .andExpect(jsonPath("$.content[0].name").value("Morning Run"))
    }

    @Test
    fun `list activities with pageable returns not found when page number is invalid`() {
        // GIVEN
        val pageable = PageRequest.of(10, 10, Sort.by("averageSpeed"))

        every { activityService.listActivitiesPaginated(pageable) } returns PageImpl(emptyList())

        // WHEN
        mockMvc.perform(
            get("/activities/by-page")
                .param("page", "10")
                .param("size", "10")
                .param("sort", "averageSpeed")
                .accept(MediaType.APPLICATION_JSON)
        )
            // THEN
            .andExpect(status().isNotFound)
            .andExpect(content().contentType(MediaType.APPLICATION_JSON))
    }

    @Test
    fun `list activities with pageable returns bad request when sort property is invalid`() {
        // GIVEN

        // WHEN
        mockMvc.perform(
            get("/activities/by-page")
                .param("page", "0")
                .param("size", "10")
                .param("sort", "invalidProperty")
                .accept(MediaType.APPLICATION_JSON)
        )
            // THEN
            .andExpect(status().isBadRequest)
            .andExpect(content().contentType(MediaType.APPLICATION_JSON))
            .andExpect(jsonPath("$.message").value("Illegal argument"))
    }

    @Test
    fun `get activities by activity type returns activities when valid activity type`() {
        // GIVEN
        val activityType = ActivityType.Run

        every { activityService.getActivitiesByActivityTypeGroupByActiveDays(activityType) } returns mapOf("2022-01-01" to 1)

        // WHEN
        mockMvc.perform(
            get("/activities/active-days")
                .param("activityType", activityType.name)
                .accept(MediaType.APPLICATION_JSON)
        )
            // THEN
            .andExpect(status().isOk)
            .andExpect(content().contentType(MediaType.APPLICATION_JSON))
            .andExpect(jsonPath("$.['2022-01-01']").value(1))
    }

    @Test
    fun `get activities by activity type returns empty map when no activities found`() {
        // GIVEN
        val activityType = ActivityType.Run

        every { activityService.getActivitiesByActivityTypeGroupByActiveDays(activityType) } returns emptyMap()

        // WHEN
        mockMvc.perform(
            get("/activities/active-days")
                .param("activityType", activityType.name)
                .accept(MediaType.APPLICATION_JSON)
        )
            // THEN
            .andExpect(status().isOk)
            .andExpect(content().contentType(MediaType.APPLICATION_JSON))
            .andExpect(jsonPath("$").isEmpty)
    }

    @Test
    fun `get activities by activity type returns bad request when activity type is invalid`() {
        // WHEN
        mockMvc.perform(
            get("/activities/active-days")
                .param("activityType", "InvalidType")
                .accept(MediaType.APPLICATION_JSON)
        )
            // THEN
            .andExpect(status().isBadRequest)
            .andExpect(content().contentType(MediaType.APPLICATION_JSON))
            .andExpect(jsonPath("$.message").value("Unknown activity type"))
    }

}