package me.nicolas.stravastats.domain.services

import io.mockk.every
import io.mockk.mockk
import me.nicolas.stravastats.domain.business.ActivityType
import me.nicolas.stravastats.domain.business.Coordinates
import me.nicolas.stravastats.domain.business.RouteExplorerRequest
import me.nicolas.stravastats.domain.business.strava.AthleteRef
import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.business.strava.stream.DistanceStream
import me.nicolas.stravastats.domain.business.strava.stream.LatLngStream
import me.nicolas.stravastats.domain.business.strava.stream.Stream
import me.nicolas.stravastats.domain.business.strava.stream.TimeStream
import me.nicolas.stravastats.domain.services.activityproviders.IActivityProvider
import me.nicolas.stravastats.domain.services.routing.RoutingEnginePort
import me.nicolas.stravastats.domain.services.routing.RoutingEngineRequest
import org.junit.jupiter.api.Test
import java.nio.file.Files
import java.nio.file.Path
import java.nio.file.Paths
import tools.jackson.databind.json.JsonMapper
import tools.jackson.module.kotlin.KotlinModule
import tools.jackson.module.kotlin.readValue
import kotlin.test.assertEquals
import kotlin.test.assertTrue
import kotlin.test.fail

class RouteExplorerParityFixtureTest {

    @Test
    fun `route explorer parity fixture top closest loop matches expected`() {
        // GIVEN
        val fixture = loadParityFixture()
        assertTrue(fixture.cases.isNotEmpty(), "expected at least one parity fixture case")

        // WHEN + THEN
        fixture.cases.forEach { parityCase ->
            val activityProvider = mockk<IActivityProvider>()
            every {
                activityProvider.getActivitiesByActivityTypeAndYear(any(), any())
            } returns parityCase.activities.map { activity -> toStravaActivity(activity) }

            val routingEngine = object : RoutingEnginePort {
                override fun generateTargetLoops(request: RoutingEngineRequest) = emptyList<me.nicolas.stravastats.domain.business.RouteRecommendation>()
                override fun generateShapeLoops(request: RoutingEngineRequest) = emptyList<me.nicolas.stravastats.domain.business.RouteRecommendation>()
                override fun healthDetails(): Map<String, Any?> = mapOf("status" to "disabled")
            }
            val service = RouteExplorerService(activityProvider, routingEngine)
            val result = service.getRouteExplorer(
                activityTypes = setOf(ActivityType.Ride),
                year = null,
                request = toRouteExplorerRequest(parityCase.request),
            )

            assertTrue(
                result.closestLoops.isNotEmpty(),
                "expected at least one closest loop recommendation for case ${parityCase.name}",
            )
            assertEquals(
                parityCase.expect.topClosestLoopName,
                result.closestLoops.first().activity.name,
                "top closest loop mismatch for case ${parityCase.name}",
            )
        }
    }

    private fun toRouteExplorerRequest(request: ParityRequest): RouteExplorerRequest {
        return RouteExplorerRequest(
            distanceTargetKm = request.distanceTargetKm,
            elevationTargetM = request.elevationTargetM,
            durationTargetMin = request.durationTargetMin,
            startDirection = request.startDirection,
            startPoint = request.startPoint?.let { start -> Coordinates(lat = start.lat, lng = start.lng) },
            routeType = request.routeType,
            season = null,
            limit = request.limit,
            shape = null,
            includeRemix = false,
        )
    }

    private fun toStravaActivity(activity: ParityActivity): StravaActivity {
        val activityType = activity.type?.ifBlank { null } ?: "Ride"
        val sportType = activity.sportType?.ifBlank { null } ?: activityType
        return StravaActivity(
            athlete = AthleteRef(id = 1),
            averageSpeed = activity.distanceKm * 1000.0 / activity.durationSec.toDouble(),
            averageCadence = 80.0,
            averageHeartrate = 145.0,
            maxHeartrate = 175,
            averageWatts = 210,
            commute = false,
            distance = activity.distanceKm * 1000.0,
            deviceWatts = true,
            elapsedTime = activity.durationSec,
            elevHigh = 1900.0,
            id = activity.id,
            kilojoules = 500.0,
            maxSpeed = 15.0f,
            movingTime = activity.durationSec,
            name = activity.name,
            _sportType = sportType,
            startDate = activity.startDate,
            startDateLocal = activity.startDate,
            startLatlng = activity.start,
            totalElevationGain = activity.elevationM,
            type = activityType,
            uploadId = activity.id + 1000,
            weightedAverageWatts = 220,
            stream = Stream(
                distance = DistanceStream(
                    data = listOf(0.0, activity.distanceKm * 1000.0),
                    originalSize = 2,
                    resolution = "high",
                    seriesType = "distance",
                ),
                time = TimeStream(
                    data = listOf(0, activity.durationSec),
                    originalSize = 2,
                    resolution = "high",
                    seriesType = "time",
                ),
                latlng = LatLngStream(
                    data = activity.track,
                    originalSize = activity.track.size,
                    resolution = "high",
                    seriesType = "distance",
                ),
            ),
        )
    }

    private fun loadParityFixture(): ParityFixture {
        val mapper = JsonMapper.builder()
            .addModule(KotlinModule.Builder().build())
            .build()
        val relative = Paths.get("test-fixtures", "routes", "route-explorer-parity.json")
        val directCandidates = listOf(
            Paths.get("..").resolve(relative),
            relative,
        )
        directCandidates.forEach { candidate ->
            if (Files.exists(candidate)) {
                return mapper.readValue(Files.readString(candidate))
            }
        }

        var cursor: Path = Paths.get("").toAbsolutePath()
        repeat(8) {
            val candidate = cursor.resolve(relative)
            if (Files.exists(candidate)) {
                return mapper.readValue(Files.readString(candidate))
            }
            val parent = cursor.parent ?: return@repeat
            cursor = parent
        }
        fail("failed to locate shared parity fixture file: $relative")
    }

    private data class ParityFixture(
        val cases: List<ParityCase> = emptyList(),
    )

    private data class ParityCase(
        val name: String,
        val request: ParityRequest,
        val activities: List<ParityActivity>,
        val expect: ParityExpect,
    )

    private data class ParityRequest(
        val distanceTargetKm: Double?,
        val elevationTargetM: Double?,
        val durationTargetMin: Int?,
        val startDirection: String?,
        val startPoint: ParityPoint?,
        val routeType: String?,
        val limit: Int,
    )

    private data class ParityPoint(
        val lat: Double,
        val lng: Double,
    )

    private data class ParityActivity(
        val id: Long,
        val name: String,
        val startDate: String,
        val distanceKm: Double,
        val elevationM: Double,
        val durationSec: Int,
        val start: List<Double>,
        val track: List<List<Double>>,
        val type: String?,
        val sportType: String?,
    )

    private data class ParityExpect(
        val topClosestLoopName: String,
    )
}
