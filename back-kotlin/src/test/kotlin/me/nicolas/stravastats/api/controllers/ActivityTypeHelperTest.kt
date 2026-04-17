package me.nicolas.stravastats.api.controllers

import me.nicolas.stravastats.domain.business.ActivityType
import org.assertj.core.api.Assertions.assertThat
import org.junit.jupiter.api.Test

class ActivityTypeHelperTest {

    @Test
    fun `convertToActivityTypeSet parses walk`() {
        // GIVEN
        val raw = "Walk"

        // WHEN
        val result = raw.convertToActivityTypeSet()

        // THEN
        assertThat(result).containsExactly(ActivityType.Walk)
    }

    @Test
    fun `convertToActivityTypeSet parses mixed walk and run`() {
        // GIVEN
        val raw = "Run_Walk"

        // WHEN
        val result = raw.convertToActivityTypeSet()

        // THEN
        assertThat(result).containsExactlyInAnyOrder(ActivityType.Run, ActivityType.Walk)
    }
}

