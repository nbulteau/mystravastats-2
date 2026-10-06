package me.nicolas.stravastats.api.controllers

import com.ninjasquad.springmockk.MockkBean
import io.mockk.every
import me.nicolas.stravastats.domain.business.AthletePerformanceSettings
import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.services.TrainingLoadService
import me.nicolas.stravastats.domain.services.activityproviders.IActivityProvider
import org.junit.jupiter.api.Assertions.*
import org.junit.jupiter.api.Test
import org.springframework.beans.factory.annotation.Autowired
import org.springframework.boot.webmvc.test.autoconfigure.WebMvcTest
import org.springframework.context.annotation.Import
import org.springframework.test.web.servlet.MockMvc
import org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get
import tools.jackson.databind.JsonNode
import tools.jackson.databind.json.JsonMapper
import tools.jackson.module.kotlin.KotlinModule
import java.io.File

@WebMvcTest(TrainingLoadController::class)
@Import(TrainingLoadService::class)
class TrainingLoadControllerTest {
    @Autowired private lateinit var mvc: MockMvc
    @MockkBean private lateinit var provider: IActivityProvider
    private val mapper=JsonMapper.builder().addModule(KotlinModule.Builder().build()).build()
    @Test fun `shared weekly reports through real HTTP controller`() {
        val fixtures=mapper.readTree(File("../test-fixtures/api/training-load.json"))
        val outputs=mutableListOf<Map<String,Any>>()
        every { provider.cacheIdentity() } returns null
        for(case in fixtures) {
            val activities=case["activities"].toList().map { mapper.treeToValue(it,StravaActivity::class.java) }
            every { provider.getActivitiesByActivityTypeAndYear(any(),any()) } returns activities
            every { provider.getPerformanceSettings() } returns mapper.treeToValue(case["settings"],AthletePerformanceSettings::class.java)
            val response=mvc.perform(get("/api/statistics/training-load").param("activityType","Ride_Run").param("week",case["week"].asText()).header("X-Request-Id","training-fixture")).andReturn().response
            assertEquals(200,response.status,response.contentAsString)
            assertEquals("training-fixture",response.getHeader("X-Request-Id"))
            val body=mapper.readTree(response.contentAsString)
            assertEquals(5,body["weeks"].size())
            body["weeks"].forEach { assertEquals(7,it["days"].size()) }
            subset(case["expectedSummary"],body["weeks"][0]["summary"])
            assertEquals(case["undatedActivities"].asInt(),body["undatedActivities"].asInt())
            for(entry in case["expectedActivities"].properties()) {
                val row=body["activities"].firstOrNull { it["activityId"].asText()==entry.key }
                assertNotNull(row,"Missing activity ${entry.key}")
                subset(entry.value,row!!)
            }
            outputs.add(mapOf("name" to case["name"].asText(),"operationId" to "getTrainingLoad","status" to 200,"body" to body))
        }
        File("../test-results").mkdirs()
        mapper.writerWithDefaultPrettyPrinter().writeValue(File("../test-results/training-kotlin-responses.json"),outputs)
    }
    private fun subset(expected: JsonNode,actual: JsonNode) {
        for(entry in expected.properties()) {
            val value=actual[entry.key]
            assertNotNull(value,"missing ${entry.key}")
            if(entry.value.isNumber) assertEquals(entry.value.asDouble(),value.asDouble(),1e-6,entry.key)
            else assertEquals(entry.value,value,entry.key)
        }
    }
    @Test fun `invalid parameters are rejected`() {
        for(query in listOf("activityType=Ride","activityType=Ride&week=2026-02-30","activityType=Unknown&week=2026-10-05","week=2026-10-05")) {
            val response=mvc.perform(get("/api/statistics/training-load?$query")).andReturn().response
            assertEquals(400,response.status,response.contentAsString)
        }
    }
}
