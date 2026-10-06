package me.nicolas.stravastats.api.controllers

import com.ninjasquad.springmockk.MockkBean
import io.mockk.every
import me.nicolas.stravastats.domain.business.*
import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.business.strava.StravaDetailedActivity
import me.nicolas.stravastats.domain.services.*
import me.nicolas.stravastats.domain.services.activityproviders.IActivityProvider
import me.nicolas.stravastats.domain.services.statistics.ActivityStatistic
import me.nicolas.stravastats.domain.services.statistics.Statistic
import org.junit.jupiter.api.Assertions.*
import org.junit.jupiter.api.Test
import org.springframework.beans.factory.annotation.Autowired
import org.springframework.boot.webmvc.test.autoconfigure.WebMvcTest
import org.springframework.http.HttpMethod
import org.springframework.http.MediaType
import org.springframework.test.web.servlet.MockMvc
import org.springframework.test.web.servlet.request.MockMvcRequestBuilders.request
import tools.jackson.databind.JsonNode
import tools.jackson.databind.json.JsonMapper
import tools.jackson.module.kotlin.KotlinModule
import java.io.File
import java.net.URI

@WebMvcTest(ActivitiesController::class, AthleteController::class, StatisticsController::class)
class BusinessResponseFixtureTest {
    @Autowired private lateinit var mvc: MockMvc
    @MockkBean private lateinit var activities: IActivityService
    @MockkBean private lateinit var provider: IActivityProvider
    @MockkBean private lateinit var performance: IAthletePerformanceSettingsService
    @MockkBean private lateinit var statistics: IStatisticsService
    @MockkBean private lateinit var heartRate: IHeartRateZoneService
    @MockkBean private lateinit var segments: ISegmentProgressionService
    private val mapper = JsonMapper.builder().addModule(KotlinModule.Builder().build()).build()

    @Test
    fun `shared business responses pass through the real controllers and converters`() {
        val cases = mapper.readTree(File("../test-fixtures/api/business-responses.json"))
        val outputs = mutableListOf<Map<String, Any>>()
        for (case in cases) {
            val name = case["name"].asText()
            val rawActivities = case["activities"]?.toList()?.map { mapper.treeToValue(it, StravaActivity::class.java) } ?: emptyList()
            every { activities.getActivitiesByActivityTypeAndYear(any(), any()) } returns rawActivities
            every { provider.getActivitiesByActivityTypeAndYear(any(), any()) } returns rawActivities
            val detailed = case["activity"]?.let { mapper.treeToValue(it, StravaDetailedActivity::class.java) }
            every { activities.getDetailedActivity(any(), any()) } returns detailed
            every { activities.getActivityComparison(any()) } returns null
            var stored = case["settings"]?.let { mapper.treeToValue(it, AthletePerformanceSettings::class.java) }
                ?: AthletePerformanceSettings()
            every { provider.getPerformanceSettings() } answers { stored }
            every { provider.savePerformanceSettings(any()) } answers { firstArg<AthletePerformanceSettings>().also { stored = it } }
            // Exercise actual normalization and FTP availability, not precomputed DTOs.
            val service = AthletePerformanceSettingsService(provider)
            every { performance.getSettings() } answers { service.getSettings() }
            every { performance.updateSettings(any()) } answers { service.updateSettings(firstArg()) }
            every { performance.estimateFtp(any(), any()) } answers { service.estimateFtp(firstArg(), secondArg()) }
            val values = case["statistics"]?.toList()?.map { item ->
                object : ActivityStatistic(item["label"].asText(), emptyList()) {
                    override val value = item["value"].asText()
                }.apply {
                    activity = item["activity"]?.let { mapper.treeToValue(it, ActivityShort::class.java) }
                } as Statistic
            } ?: emptyList()
            every { statistics.getStatistics(any(), any()) } returns values
            every { statistics.getPersonalRecordsTimeline(any(), any(), any()) } returns
                (case["timeline"]?.toList()?.map { mapper.treeToValue(it, PersonalRecordTimelineEntry::class.java) } ?: emptyList())
            case["heartRate"]?.let {
                every { heartRate.getAnalysis(any(), any()) } returns mapper.treeToValue(it, HeartRateZoneAnalysis::class.java)
            }
            val response = mvc.perform(request(HttpMethod.valueOf(case["method"].asText()), URI(case["path"].asText()))
                .contentType(MediaType.APPLICATION_JSON)
                .content(case["requestBody"]?.toString() ?: ""))
                .andReturn().response
            assertEquals(case["status"].asInt(), response.status, "$name: ${response.contentAsString}")
            val body = mapper.readTree(response.contentAsString)
            compareJSON(case["expected"], body, name)
            outputs += mapOf("name" to name, "operationId" to case["operationId"].asText(), "status" to response.status, "body" to body)
            if (case["method"].asText() == "PUT") compareJSON(case["expected"], mapper.valueToTree(stored), "$name persisted")
        }
        File("../test-results/api-kotlin-responses.json").apply {
            parentFile.mkdirs()
            writeText(mapper.writerWithDefaultPrettyPrinter().writeValueAsString(outputs))
        }
    }

    // OpenAPI validation separately checks required nullable properties on the raw JSON.
    private fun compareJSON(expected: JsonNode?, actual: JsonNode?, path: String) {
        if (expected == null || expected.isNull) {
            assertTrue(actual == null || actual.isNull, "$path: expected unavailable, got $actual")
        } else if (expected.isObject) {
            assertTrue(actual?.isObject == true, path)
            expected.properties().forEach { (key, value) -> compareJSON(value, actual!![key], "$path.$key") }
            actual!!.properties().forEach { (key, value) ->
                if (!expected.has(key)) assertTrue(value.isNull, "$path: unexpected $key=$value")
            }
        } else if (expected.isArray) {
            assertTrue(actual?.isArray == true, path)
            assertEquals(expected.size(), actual!!.size(), path)
            expected.forEachIndexed { index, value -> compareJSON(value, actual[index], "$path[$index]") }
        } else if (expected.isNumber) {
            assertTrue(actual?.isNumber == true, path)
            assertEquals(expected.asDouble(), actual!!.asDouble(), 1e-6, path)
        } else assertEquals(expected, actual, path)
    }
}
