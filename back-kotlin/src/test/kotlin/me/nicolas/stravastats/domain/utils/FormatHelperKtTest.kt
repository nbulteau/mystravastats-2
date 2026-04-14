package me.nicolas.stravastats.domain.utils

import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Test


internal class FormatHelperTest {

    @Test
    fun `Int formatSeconds`() {
        // GIVEN
        val testCases = mapOf(
            0 to "00s",
            59 to "59s",
            60 to "01m 00s",
            120 to "02m 00s",
            3599 to "59m 59s",
        )

        testCases.forEach { (input, expected) ->
            // WHEN
            val result = input.formatSeconds()

            // THEN
            assertEquals(expected, result)
        }
    }

    @Test
    fun `Double formatSeconds`() {
        // GIVEN
        val testCases = mapOf(
            0.0 to "0'00",
            59.0 to "0'59",
            60.0 to "1'00",
            120.0 to "2'00",
            3599.0 to "59'59",
            120.99 to "2'00",
            120.994 to "2'00",
            120.995 to "2'01",
            3599.994 to "59'59",
        )

        testCases.forEach { (input, expected) ->
            // WHEN
            val result = input.formatSeconds()

            // THEN
            assertEquals(expected, result)
        }
    }
}