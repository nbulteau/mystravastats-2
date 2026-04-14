package me.nicolas.stravastats.domain.business.strava


import org.junit.jupiter.api.Assertions
import org.junit.jupiter.api.Test

internal class GeoCoordinateTest {

    companion object {
        val COL_AGNEL = GeoCoordinate(44.6839194, 6.9795741)
        val COL_D_IZOARD = GeoCoordinate(44.8200267, 6.7350408)
    }

    @Test
    fun haversineInM1() {
        // GIVEN
        // COL_AGNEL and COL_D_IZOARD are defined in companion object

        // WHEN
        val result = COL_AGNEL.haversineInM(
            COL_D_IZOARD.latitude,
            COL_D_IZOARD.longitude
        )

        // THEN
        Assertions.assertEquals(24561, result)
    }

    @Test
    fun haversineInM2() {
        // GIVEN
        // COL_AGNEL defined in companion object

        // WHEN
        val result = COL_AGNEL.haversineInM(COL_AGNEL.latitude, COL_AGNEL.longitude)

        // THEN
        Assertions.assertEquals(0, result)
    }
}