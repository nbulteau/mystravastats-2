package me.nicolas.stravastats.domain.services.statistics

import java.io.File
import me.nicolas.stravastats.domain.business.strava.stream.PowerTimeline
import org.junit.jupiter.api.Assertions.*
import org.junit.jupiter.api.Test
import tools.jackson.databind.json.JsonMapper

class PowerTimingTest {
    @Test fun `shared time weighted power and coverage`() {
        val cases=JsonMapper.builder().build().readTree(File("../test-fixtures/api/power-timing.json"))
        for(case in cases) {
            val times=case["times"].toList().map { it.asInt() }
            val watts=case["watts"].toList().map { if(it.isNull) null else it.asInt() }
            val timeline=PowerTimeline(watts,times)
            val average=if(times.isEmpty()) null else timeline.average(times.first().toDouble(),times.last().toDouble())
            val best=timeline.best(case["seconds"].asInt())?.average
            val activity=StatisticsFixtures.syntheticRideActivity(id=995,stream=StatisticsFixtures.defaultStream(
                distances=times.map { it*2.0 },times=times,altitudes=times.map { 100.0 },watts=watts,
            ))
            val effort=activity.calculateBestPowerForTime(case["seconds"].asInt())
            assertEquals(best?.toInt(),effort?.averagePower,case["name"].asText())
            if(effort!=null) {
                assertEquals(case["seconds"].asInt(),effort.seconds)
                assertEquals(case["seconds"].asInt()*2.0,effort.distance,1e-7)
            }
            for((name,value) in listOf("average" to average,"best" to best, "normalized" to timeline.normalized())) {
                val expected=case[name]
                if(expected==null || expected.isNull) assertNull(value,case["name"].asText())
                else {assertNotNull(value,case["name"].asText());assertEquals(expected.asDouble(),value!!,1e-7,case["name"].asText())}
            }
            val covered=(0 until times.lastIndex).filter { timeline.intervalValid(it) }.sumOf { times[it+1]-times[it] }
            assertEquals(case["covered"].asInt(),covered,case["name"].asText())
        }
    }
}
