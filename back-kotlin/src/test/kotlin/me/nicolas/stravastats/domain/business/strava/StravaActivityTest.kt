package me.nicolas.stravastats.domain.business.strava

import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Test


internal class StravaActivityTest {

    @Test
    fun processAverageSpeed() {
        // GIVEN
        val colAgnelActivity = loadColAgnelActivity()

        // WHEN
        val result = colAgnelActivity.processAverageSpeed()

        // THEN
        assertEquals("15,48", result)
    }

    @Test
    fun getTotalElevationGain() {
        // GIVEN
        val colAgnelActivity = loadColAgnelActivity()

        // WHEN
        val result = colAgnelActivity.totalElevationGain

        // THEN
        assertEquals(2090, result.toInt())
    }

    @Test
    fun getTotalAscentGain() {
        // GIVEN
        val colAgnelActivity = loadColAgnelActivity()

        // WHEN
        val result = colAgnelActivity.calculateTotalAscentGain()

        // THEN
        assertEquals(2107, result.toInt())
    }

    @Test
    fun getTotalDescentGain() {
        // GIVEN
        val colAgnelActivity = loadColAgnelActivity()

        // WHEN
        val result = colAgnelActivity.calculateTotalDescentGain()

        // THEN
        assertEquals(2131, result.toInt())
    }
}