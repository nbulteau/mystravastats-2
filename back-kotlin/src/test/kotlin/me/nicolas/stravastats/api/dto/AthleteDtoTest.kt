package me.nicolas.stravastats.api.dto

import me.nicolas.stravastats.domain.business.strava.StravaAthlete
import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Test

class AthleteDtoTest {
    @Test
    fun `athlete dto exposes ftp when present`() {
        val athlete = StravaAthlete(
            id = 123L,
            ftp = 250.0,
        )

        val dto = athlete.toDto()

        assertEquals(250, dto.ftp)
    }
}
