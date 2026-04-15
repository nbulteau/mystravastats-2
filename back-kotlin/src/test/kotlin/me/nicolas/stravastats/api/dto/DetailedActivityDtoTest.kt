package me.nicolas.stravastats.api.dto

import me.nicolas.stravastats.domain.business.ActivityEffort
import me.nicolas.stravastats.domain.business.ActivityShort
import org.junit.jupiter.api.Assertions.assertNotEquals
import org.junit.jupiter.api.Test

class DetailedActivityDtoTest {

    @Test
    fun `ActivityEffort dto id stays unique for same label with different indexes`() {
        // GIVEN
        val first = ActivityEffort(
            distance = 300.0,
            seconds = 30,
            deltaAltitude = 20.0,
            idxStart = 10,
            idxEnd = 40,
            label = "MURAILLE DE CHINE <Alpe d'Huez>",
            activityShort = ActivityShort(id = 1L, name = "A", type = "Ride"),
        )
        val second = ActivityEffort(
            distance = 300.0,
            seconds = 30,
            deltaAltitude = -20.0,
            idxStart = 50,
            idxEnd = 80,
            label = "MURAILLE DE CHINE <Alpe d'Huez>",
            activityShort = ActivityShort(id = 1L, name = "A", type = "Ride"),
        )

        // WHEN
        val firstDto = first.toDto()
        val secondDto = second.toDto()

        // THEN
        assertNotEquals(firstDto.id, secondDto.id)
    }
}
