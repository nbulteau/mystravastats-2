package me.nicolas.stravastats.adapters.localrepositories.fit

import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Test

class FITMovingTimeTest {
    @Test
    fun `resolveFitMovingTime uses total moving time when present`() {
        val movingTime = resolveFitMovingTime(
            totalMovingTime = 12,
            totalTimerTime = 20,
            elapsedTime = 20,
            streamMovingTime = 10,
        )

        assertEquals(12, movingTime)
    }

    @Test
    fun `resolveFitMovingTime uses stream when it removes stop time from timer`() {
        val movingTime = resolveFitMovingTime(
            totalMovingTime = 0,
            totalTimerTime = 400,
            elapsedTime = 405,
            streamMovingTime = 300,
        )

        assertEquals(300, movingTime)
    }

    @Test
    fun `resolveFitMovingTime uses stream for Garmin timer with long stops`() {
        val movingTime = resolveFitMovingTime(
            totalMovingTime = 0,
            totalTimerTime = 16328,
            elapsedTime = 18581,
            streamMovingTime = 12701,
        )

        assertEquals(12701, movingTime)
    }

    @Test
    fun `resolveFitMovingTime keeps timer when timer already excludes stops`() {
        val movingTime = resolveFitMovingTime(
            totalMovingTime = 0,
            totalTimerTime = 220,
            elapsedTime = 900,
            streamMovingTime = 400,
        )

        assertEquals(220, movingTime)
    }
}
