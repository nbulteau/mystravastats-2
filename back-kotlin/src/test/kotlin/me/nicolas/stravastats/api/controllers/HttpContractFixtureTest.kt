package me.nicolas.stravastats.api.controllers

import com.ninjasquad.springmockk.MockkBean
import io.mockk.every
import me.nicolas.stravastats.domain.business.SourceSyncResult
import me.nicolas.stravastats.domain.services.IActivityService
import me.nicolas.stravastats.domain.services.IChartsService
import me.nicolas.stravastats.domain.services.ILocalDataBackupService
import me.nicolas.stravastats.domain.services.ISourceSyncService
import org.junit.jupiter.api.Assertions.*
import org.junit.jupiter.api.Test
import org.springframework.beans.factory.annotation.Autowired
import org.springframework.boot.webmvc.test.autoconfigure.WebMvcTest
import org.springframework.http.HttpMethod
import org.springframework.http.MediaType
import org.springframework.test.web.servlet.MockMvc
import org.springframework.test.web.servlet.request.MockMvcRequestBuilders.request
import tools.jackson.databind.json.JsonMapper
import java.io.File
import java.net.URI

@WebMvcTest(ActivitiesController::class, ChartsController::class, LocalDataController::class, SourceSyncController::class)
class HttpContractFixtureTest {
    @Autowired private lateinit var mockMvc: MockMvc
    @MockkBean private lateinit var activities: IActivityService
    @MockkBean private lateinit var charts: IChartsService
    @MockkBean private lateinit var backup: ILocalDataBackupService
    @MockkBean private lateinit var sync: ISourceSyncService

    @Test
    fun `shared HTTP contract`() {
        val mapper = JsonMapper.builder().build()
        val cases = mapper.readTree(File("../test-fixtures/api/http-contract.json"))
        for (case in cases) {
            val name = case["name"].asText()
            every { activities.getActivitiesByActivityTypeAndYear(any(), any()) } returns emptyList()
            if (case["scenario"]?.asText() == "read-failure") {
                every { activities.getDetailedActivity(any(), any()) } throws IllegalStateException("private storage failure")
            } else {
                every { activities.getDetailedActivity(any(), any()) } returns null
            }
            every { sync.synchronize("manual") } returns SourceSyncResult(
                status = case["bodyStatus"]?.asText() ?: "success", reason = "manual", message = "Test result",
            )
            val requestId = case["requestId"]?.asText() ?: "shared-http-contract"
            val response = mockMvc.perform(request(HttpMethod.valueOf(case["method"].asText()), URI(case["path"].asText()))
                .contentType(MediaType.APPLICATION_JSON)
                .header("X-Request-Id", requestId)
                .content(case["body"]?.asText() ?: ""))
                .andReturn().response
            assertEquals(case["status"].asInt(), response.status, "$name: ${response.contentAsString}")
            assertTrue(response.contentType?.startsWith("application/json") == true, name)
            val id = response.getHeader("X-Request-Id")
            assertFalse(id.isNullOrBlank(), name)
            if (case["replaceRequestId"]?.asBoolean() == true) assertNotEquals(requestId, id, name)
            else if (requestId.isNotEmpty()) assertEquals(requestId, id, name)
            case["allow"]?.forEach { assertTrue(response.getHeader("Allow")?.contains(it.asText()) == true, name) }
            if (case["emptyBody"]?.asBoolean() == true) {
                // Servlet 6.1 delegates HEAD body suppression to the container.
                // MockMvc has no container and retains the generated representation.
                assertEquals("HEAD", case["method"].asText(), name)
                continue
            }
            val body = mapper.readTree(response.contentAsString)
            if (case["bodyType"]?.asText() == "array") assertTrue(body.isArray, name)
            if (case["error"]?.asBoolean() == true) {
                listOf("message", "description", "requestId").forEach {
                    assertTrue(body[it]?.isString == true && body[it].asText().isNotBlank(), "$name: $it")
                }
                assertEquals(1, body["code"].asInt(), name)
                assertEquals(id, body["requestId"].asText(), name)
            }
            case["bodyStatus"]?.let { assertEquals(it.asText(), body["status"].asText(), name) }
            assertFalse(response.contentAsString.contains("private storage failure"), name)
        }
    }
}
