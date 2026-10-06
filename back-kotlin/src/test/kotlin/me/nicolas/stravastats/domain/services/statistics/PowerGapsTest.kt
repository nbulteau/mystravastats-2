package me.nicolas.stravastats.domain.services.statistics

import java.io.File
import me.nicolas.stravastats.domain.business.strava.stream.Stream
import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Test
import org.junit.jupiter.api.io.TempDir
import tools.jackson.databind.json.JsonMapper
import tools.jackson.module.kotlin.KotlinModule

class PowerGapsTest {
    @TempDir lateinit var directory: File
    private val mapper = JsonMapper.builder().addModule(KotlinModule.Builder().build()).build()

    @Test
    fun `shared gaps survive storage and invalidate cached complete windows`() {
        BestEffortCache.clear()
        val cases = mapper.readTree(File("../test-fixtures/api/power-gaps.json"))
        for (case in cases) {
            val watts = case["watts"].toList().map { if (it.isNull) null else it.asInt() }
            val expected = case["expected"].let { if (it.isNull) null else it.asInt() }
            val stream = StatisticsFixtures.defaultStream(
                distances = listOf(0.0, 10.0, 20.0, 30.0, 40.0, 50.0),
                times = listOf(0, 1, 2, 3, 4, 5),
                altitudes = List(6) { 100.0 }, watts = watts,
            )
            val file = File(directory, "stream.json")
            mapper.writeValue(file, stream)
            val restored = mapper.readValue(file, Stream::class.java)
            assertEquals(watts, restored.watts?.data)
            val activity = StatisticsFixtures.syntheticRideActivity(id = 991, stream = restored)
            assertEquals(expected, activity.calculateBestPowerForTime(2)?.averagePower)
            assertEquals(expected, activity.calculateBestPowerForDistance(20.0)?.averagePower)
        }
    }
}
