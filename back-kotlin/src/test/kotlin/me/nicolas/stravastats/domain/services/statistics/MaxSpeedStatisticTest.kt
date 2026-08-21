package me.nicolas.stravastats.domain.services.statistics

import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Test

class MaxSpeedStatisticTest {

    @Test
    fun `statistic ignores impossible activity max speed`() {
        val invalid = StatisticsFixtures.syntheticRideActivity(id = 21).copy(maxSpeed = 200f)
        val valid = StatisticsFixtures.syntheticRideActivity(id = 22).copy(maxSpeed = 20f)

        val statistic = MaxSpeedStatistic(listOf(invalid, valid))

        assertEquals(valid.id, statistic.activity?.id)
        assertEquals("72,00 km/h", statistic.value)
    }
}
