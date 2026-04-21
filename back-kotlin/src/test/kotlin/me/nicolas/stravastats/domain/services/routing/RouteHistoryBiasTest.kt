package me.nicolas.stravastats.domain.services.routing

import me.nicolas.stravastats.domain.business.Coordinates
import org.junit.jupiter.api.Test
import java.util.Locale
import kotlin.test.assertEquals
import kotlin.test.assertFalse
import kotlin.test.assertTrue

class RouteHistoryBiasTest {

    @Test
    fun `build history bias context is disabled when route types mismatch`() {
        // GIVEN
        val request = RoutingEngineRequest(
            startPoint = Coordinates(45.0, 6.0),
            distanceTargetKm = 40.0,
            elevationTargetM = null,
            startDirection = null,
            routeType = "RIDE",
            limit = 3,
            historyBiasEnabled = true,
            historyProfile = RoutingHistoryProfile(
                routeType = "RUN",
                halfLifeDays = 75,
                activityCount = 10,
                segmentCount = 100,
                axisScores = mapOf(axisKey(45.0, 6.0, 45.01, 6.01) to 100.0),
                zoneScores = emptyMap(),
                latestActivityEpochMs = 0,
            ),
        )

        // WHEN
        val context = buildRoutingHistoryBiasContext(request)

        // THEN
        assertFalse(context.enabled)
    }

    @Test
    fun `sort anchors by history reuse prioritizes most used anchors`() {
        // GIVEN
        val start = Coordinates(45.10, 6.10)
        val highReuseAnchor = Coordinates(45.30, 6.30)
        val lowReuseAnchor = Coordinates(45.32, 6.32)
        val context = RoutingHistoryBiasContext(
            enabled = true,
            normalizedRouteType = "RIDE",
            zoneScores = mapOf(
                zoneKey(highReuseAnchor.lat, highReuseAnchor.lng) to 10_000.0,
                zoneKey(lowReuseAnchor.lat, lowReuseAnchor.lng) to 150.0,
            ),
            maxZoneScore = 10_000.0,
        )

        // WHEN
        val sorted = sortAnchorsByHistoryReuse(listOf(highReuseAnchor, lowReuseAnchor), start, context)

        // THEN
        assertEquals(2, sorted.size)
        assertEquals(highReuseAnchor, sorted.first())
    }

    @Test
    fun `compute history reuse score is higher on known corridors`() {
        // GIVEN
        val knownPoints = listOf(
            listOf(45.0000, 6.0000),
            listOf(45.0200, 6.0000),
            listOf(45.0000, 6.0000),
        )
        val freshPoints = listOf(
            listOf(45.5000, 6.5000),
            listOf(45.5200, 6.5200),
            listOf(45.5000, 6.5000),
        )
        val knownAxisA = axisKey(45.0000, 6.0000, 45.0200, 6.0000)
        val knownAxisB = axisKey(45.0200, 6.0000, 45.0000, 6.0000)
        val knownZoneA = zoneKey((45.0000 + 45.0200) / 2.0, 6.0000)
        val knownZoneB = zoneKey((45.0200 + 45.0000) / 2.0, 6.0000)
        val context = RoutingHistoryBiasContext(
            enabled = true,
            normalizedRouteType = "RIDE",
            axisScores = mapOf(knownAxisA to 8_000.0, knownAxisB to 8_000.0),
            zoneScores = mapOf(knownZoneA to 6_000.0, knownZoneB to 6_000.0),
            maxAxisScore = 8_000.0,
            maxZoneScore = 6_000.0,
        )

        // WHEN
        val knownReuse = computeHistoryReuseScore(knownPoints, context)
        val freshReuse = computeHistoryReuseScore(freshPoints, context)

        // THEN
        assertTrue(knownReuse > freshReuse)
        assertTrue(knownReuse > 0.0)
    }

    private fun axisKey(lat1: Double, lng1: Double, lat2: Double, lng2: Double): String {
        val left = nodeKey(lat1, lng1, 4)
        val right = nodeKey(lat2, lng2, 4)
        return if (left <= right) "$left|$right" else "$right|$left"
    }

    private fun zoneKey(lat: Double, lng: Double): String {
        return nodeKey(lat, lng, 2)
    }

    private fun nodeKey(lat: Double, lng: Double, precision: Int): String {
        return "%.${precision}f:%.${precision}f".format(Locale.US, lat, lng)
    }
}
