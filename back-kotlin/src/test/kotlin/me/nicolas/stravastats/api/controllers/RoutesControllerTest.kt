package me.nicolas.stravastats.api.controllers

import com.ninjasquad.springmockk.MockkBean
import io.mockk.every
import io.mockk.verify
import me.nicolas.stravastats.domain.business.ActivityShort
import me.nicolas.stravastats.domain.business.ActivityType
import me.nicolas.stravastats.domain.business.RouteExplorerRequest
import me.nicolas.stravastats.domain.business.RouteExplorerResult
import me.nicolas.stravastats.domain.business.RouteRecommendation
import me.nicolas.stravastats.domain.business.RouteVariantType
import me.nicolas.stravastats.domain.services.IRouteExplorerService
import org.junit.jupiter.api.Test
import org.junit.jupiter.api.extension.ExtendWith
import org.springframework.beans.factory.annotation.Autowired
import org.springframework.boot.webmvc.test.autoconfigure.WebMvcTest
import org.springframework.http.MediaType
import org.springframework.test.context.junit.jupiter.SpringExtension
import org.springframework.test.web.servlet.MockMvc
import org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get
import org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post
import org.springframework.test.web.servlet.result.MockMvcResultMatchers.content
import org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath
import org.springframework.test.web.servlet.result.MockMvcResultMatchers.status
import java.nio.file.Files
import java.nio.file.Path
import java.nio.file.Paths
import tools.jackson.databind.json.JsonMapper
import tools.jackson.module.kotlin.KotlinModule
import tools.jackson.module.kotlin.readValue
import kotlin.test.fail

@ExtendWith(SpringExtension::class)
@WebMvcTest(RoutesController::class)
class RoutesControllerTest {

    @Autowired
    private lateinit var mockMvc: MockMvc

    @MockkBean
    private lateinit var routeExplorerService: IRouteExplorerService

    @Test
    fun `generate shape routes returns unified routes payload`() {
        // GIVEN
        every {
            routeExplorerService.getRouteExplorer(
                any(),
                any(),
                match { request ->
                    request.startPoint?.let { startPoint ->
                        startPoint.lat == 45.19 && startPoint.lng == 5.73
                    } == true &&
                        request.distanceTargetKm == null &&
                        request.elevationTargetM == null
                }
            )
        } returns RouteExplorerResult(
            closestLoops = emptyList(),
            variants = emptyList(),
            seasonal = emptyList(),
            roadGraphLoops = emptyList(),
            shapeMatches = listOf(
                RouteRecommendation(
                    routeId = "generated-loop-kt",
                    activity = ActivityShort(12L, "Generated loop", ActivityType.Ride),
                    activityDate = "2025-01-01",
                    distanceKm = 41.3,
                    elevationGainM = 850.0,
                    durationSec = 7100,
                    isLoop = true,
                    start = null,
                    end = null,
                    startArea = "Grenoble",
                    season = "SPRING",
                    variantType = RouteVariantType.SHAPE_MATCH,
                    matchScore = 92.1,
                    reasons = listOf(
                        "Generated with OSM road graph (OSRM)",
                        "Shape similarity: 90%",
                        "Shape mode: projected waypoints",
                    ),
                    previewLatLng = listOf(listOf(45.18, 5.72), listOf(45.20, 5.75), listOf(45.18, 5.72)),
                    shape = "CUSTOM_SHAPE",
                    shapeScore = 0.9,
                    experimental = true,
                )
            ),
            shapeRemixes = emptyList(),
        )

        // WHEN
        mockMvc.perform(
            post("/api/routes/generate/shape")
                .param("activityType", "Ride")
                .param("year", "2025")
                .contentType(MediaType.APPLICATION_JSON)
                .content(
                    """
                    {
                      "shapeInputType": "draw",
                      "shapeData": "[[45.18,5.72],[45.20,5.75],[45.18,5.72]]",
                      "startPoint": {"lat": 45.19, "lng": 5.73},
                      "routeType": "RIDE",
                      "variantCount": 3
                    }
                    """.trimIndent()
                )
        )
            // THEN
            .andExpect(status().isOk)
            .andExpect(content().contentType(MediaType.APPLICATION_JSON))
            .andExpect(jsonPath("$.routes[0].routeId").value("generated-loop-kt"))
            .andExpect(jsonPath("$.routes[0].isRoadGraphGenerated").value(true))
            .andExpect(jsonPath("$.routes[0].score.global").value(92.1))
    }

    @Test
    fun `generate shape routes uses surface fitness reason for road fitness score`() {
        // GIVEN
        every {
            routeExplorerService.getRouteExplorer(any(), any(), any())
        } returns RouteExplorerResult(
            closestLoops = emptyList(),
            variants = emptyList(),
            seasonal = emptyList(),
            roadGraphLoops = emptyList(),
            shapeMatches = listOf(
                RouteRecommendation(
                    routeId = "generated-surface-kt",
                    activity = ActivityShort(42L, "Generated surface loop", ActivityType.GravelRide),
                    activityDate = "2025-01-01",
                    distanceKm = 38.6,
                    elevationGainM = 620.0,
                    durationSec = 6400,
                    isLoop = true,
                    start = null,
                    end = null,
                    startArea = "Grenoble",
                    season = "SPRING",
                    variantType = RouteVariantType.SHAPE_MATCH,
                    matchScore = 87.0,
                    reasons = listOf(
                        "Generated with OSM road graph (OSRM)",
                        "Shape similarity: 82%",
                        "Shape mode: projected waypoints",
                        "Surface mix: paved 38%, gravel 52%, trail 10%, unknown 0%",
                        "Surface fitness: 68%",
                    ),
                    previewLatLng = listOf(listOf(45.18, 5.72), listOf(45.20, 5.75), listOf(45.18, 5.72)),
                    shape = "CUSTOM_SHAPE",
                    shapeScore = 0.82,
                    experimental = true,
                )
            ),
            shapeRemixes = emptyList(),
        )

        // WHEN
        mockMvc.perform(
            post("/api/routes/generate/shape")
                .param("activityType", "Ride")
                .contentType(MediaType.APPLICATION_JSON)
                .content(
                    """
                    {
                      "shapeInputType": "draw",
                      "shapeData": "[[45.18,5.72],[45.20,5.75],[45.18,5.72]]",
                      "startPoint": {"lat": 45.19, "lng": 5.73},
                      "routeType": "GRAVEL"
                    }
                    """.trimIndent()
                )
        )
            // THEN
            .andExpect(status().isOk)
            .andExpect(jsonPath("$.routes[0].routeId").value("generated-surface-kt"))
            .andExpect(jsonPath("$.routes[0].score.roadFitness").value(68.0))
    }

    @Test
    fun `generate shape routes rejects invalid shapeInputType`() {
        // GIVEN
        every {
            routeExplorerService.getRouteExplorer(any(), any(), any())
        } returns RouteExplorerResult(
            closestLoops = emptyList(),
            variants = emptyList(),
            seasonal = emptyList(),
            roadGraphLoops = emptyList(),
            shapeMatches = emptyList(),
            shapeRemixes = emptyList(),
        )

        // WHEN
        mockMvc.perform(
            post("/api/routes/generate/shape")
                .param("activityType", "Ride")
                .contentType(MediaType.APPLICATION_JSON)
                .content(
                    """
                    {
                      "shapeInputType": "invalid",
                      "shapeData": "[[45.0,6.0],[45.1,6.1]]",
                      "routeType": "RIDE"
                    }
                    """.trimIndent()
                )
        )
            // THEN
            .andExpect(status().isBadRequest)
    }

    @Test
    fun `generate shape routes returns failure summary and propagates request id header`() {
        // GIVEN
        every {
            routeExplorerService.getRouteExplorer(any(), any(), any())
        } returns RouteExplorerResult(
            closestLoops = emptyList(),
            variants = emptyList(),
            seasonal = emptyList(),
            roadGraphLoops = emptyList(),
            shapeMatches = emptyList(),
            shapeRemixes = emptyList(),
        )

        // WHEN
        mockMvc.perform(
            post("/api/routes/generate/shape")
                .param("activityType", "Ride")
                .header("X-Request-Id", "req-shape-kt-2")
                .contentType(MediaType.APPLICATION_JSON)
                .content(
                    """
                    {
                      "shapeInputType": "draw",
                      "shapeData": "[[45.0,6.0],[45.1,6.1]]",
                      "routeType": "RIDE"
                    }
                    """.trimIndent()
                )
        )
            // THEN
            .andExpect(status().isOk)
            .andExpect(content().string(org.hamcrest.Matchers.containsString("FAILURE_SUMMARY")))
            .andExpect(content().string(org.hamcrest.Matchers.containsString("requestId=req-shape-kt-2")))
            .andExpect(org.springframework.test.web.servlet.result.MockMvcResultMatchers.header().string("X-Request-Id", "req-shape-kt-2"))
    }

    @Test
    fun `generate shape routes infers shape from encoded polyline`() {
        // GIVEN
        val encodedPolyline = "_p~iF~ps|U_ulLnnqC_mqNvxq`@"
        every {
            routeExplorerService.getRouteExplorer(any(), any(), any())
        } returns RouteExplorerResult(
            closestLoops = emptyList(),
            variants = emptyList(),
            seasonal = emptyList(),
            roadGraphLoops = emptyList(),
            shapeMatches = emptyList(),
            shapeRemixes = emptyList(),
        )

        // WHEN
        mockMvc.perform(
            post("/api/routes/generate/shape")
                .param("activityType", "Ride")
                .contentType(MediaType.APPLICATION_JSON)
                .content(
                    """
                    {
                      "shapeInputType": "polyline",
                      "shapeData": "_p~iF~ps|U_ulLnnqC_mqNvxq`@",
                      "routeType": "RIDE"
                    }
                    """.trimIndent()
                )
        )
            // THEN
            .andExpect(status().isOk)

        verify {
            routeExplorerService.getRouteExplorer(
                any(),
                any(),
                match { request: RouteExplorerRequest ->
                    request.shape == "POINT_TO_POINT" && request.shapePolyline == encodedPolyline
                }
            )
        }
    }

    @Test
    fun `generate shape routes infers shape from gpx payload`() {
        // GIVEN
        val gpxData = "<gpx><trk><trkseg><trkpt lat=\"48.1000\" lon=\"-1.6000\"/><trkpt lat=\"48.1200\" lon=\"-1.6200\"/><trkpt lat=\"48.1300\" lon=\"-1.6300\"/></trkseg></trk></gpx>"
        every {
            routeExplorerService.getRouteExplorer(any(), any(), any())
        } returns RouteExplorerResult(
            closestLoops = emptyList(),
            variants = emptyList(),
            seasonal = emptyList(),
            roadGraphLoops = emptyList(),
            shapeMatches = emptyList(),
            shapeRemixes = emptyList(),
        )

        // WHEN
        mockMvc.perform(
            post("/api/routes/generate/shape")
                .param("activityType", "Ride")
                .contentType(MediaType.APPLICATION_JSON)
                .content(
                    """
                    {
                      "shapeInputType": "gpx",
                      "shapeData": "<gpx><trk><trkseg><trkpt lat=\"48.1000\" lon=\"-1.6000\"/><trkpt lat=\"48.1200\" lon=\"-1.6200\"/><trkpt lat=\"48.1300\" lon=\"-1.6300\"/></trkseg></trk></gpx>",
                      "routeType": "RIDE"
                    }
                    """.trimIndent()
                )
        )
            // THEN
            .andExpect(status().isOk)

        verify {
            routeExplorerService.getRouteExplorer(
                any(),
                any(),
                match { request: RouteExplorerRequest ->
                    request.shape == "POINT_TO_POINT" && request.shapePolyline == gpxData
                }
            )
        }
    }

    @Test
    fun `Strava Art smoke generates route and exports gpx`() {
        // GIVEN
        val fixture = loadStravaArtSmokeFixture()
        every {
            routeExplorerService.getRouteExplorer(any(), any(), any())
        } returns RouteExplorerResult(
            closestLoops = emptyList(),
            variants = emptyList(),
            seasonal = emptyList(),
            roadGraphLoops = emptyList(),
            shapeMatches = listOf(
                RouteRecommendation(
                    routeId = fixture.generatedRouteId,
                    activity = ActivityShort(34L, fixture.generatedRouteName, ActivityType.Ride),
                    activityDate = "2025-01-01",
                    distanceKm = 30.0,
                    elevationGainM = 450.0,
                    durationSec = 4500,
                    isLoop = true,
                    start = null,
                    end = null,
                    startArea = "Grenoble",
                    season = "SUMMER",
                    variantType = RouteVariantType.SHAPE_MATCH,
                    matchScore = 88.0,
                    reasons = listOf(
                        "Generated with OSM road graph (OSRM)",
                        "Shape similarity: 85%",
                        "Shape mode: projected waypoints",
                    ),
                    previewLatLng = fixture.generatedPreviewLatLng,
                    shape = "CUSTOM_SHAPE",
                    shapeScore = 0.85,
                    experimental = true,
                )
            ),
            shapeRemixes = emptyList(),
        )

        mockMvc.perform(
            post("/api/routes/generate/shape")
                .param("activityType", "Ride")
                .contentType(MediaType.APPLICATION_JSON)
                .content(
                    """
                    {
                      "shapeInputType": "${fixture.shapeInputType}",
                      "shapeData": "${fixture.shapeData}",
                      "startPoint": {"lat": ${fixture.startPoint.lat}, "lng": ${fixture.startPoint.lng}},
                      "routeType": "${fixture.routeType}",
                      "variantCount": ${fixture.variantCount}
                    }
                    """.trimIndent()
                )
        ).andExpect(status().isOk)
            .andExpect(jsonPath("$.routes[0].routeId").value(fixture.generatedRouteId))

        // WHEN
        mockMvc.perform(get("/api/routes/${fixture.generatedRouteId}/gpx"))
            // THEN
            .andExpect(status().isOk)
            .andExpect(content().contentType("application/gpx+xml"))
            .andExpect(content().string(org.hamcrest.Matchers.containsString("<gpx")))
    }

    @Test
    fun `generate shape routes ignores historical route candidates`() {
        // GIVEN
        every {
            routeExplorerService.getRouteExplorer(any(), any(), any())
        } returns RouteExplorerResult(
            closestLoops = listOf(
                RouteRecommendation(
                    routeId = "legacy-route-kt",
                    activity = ActivityShort(90L, "Already done ride", ActivityType.Ride),
                    activityDate = "2025-01-01",
                    distanceKm = 40.0,
                    elevationGainM = 700.0,
                    durationSec = 6800,
                    isLoop = true,
                    start = null,
                    end = null,
                    startArea = "Grenoble",
                    season = "SPRING",
                    variantType = RouteVariantType.CLOSE_MATCH,
                    matchScore = 85.0,
                    reasons = listOf("Historical match"),
                    previewLatLng = listOf(listOf(45.18, 5.72), listOf(45.19, 5.73)),
                    shape = "LOOP",
                    shapeScore = 0.8,
                    experimental = false,
                )
            ),
            variants = emptyList(),
            seasonal = emptyList(),
            roadGraphLoops = listOf(
                RouteRecommendation(
                    routeId = "cache-road-graph-kt",
                    activity = ActivityShort(0L, "Generated from local cache", ActivityType.Ride),
                    activityDate = "2025-01-01",
                    distanceKm = 42.0,
                    elevationGainM = 720.0,
                    durationSec = 7000,
                    isLoop = true,
                    start = null,
                    end = null,
                    startArea = "Grenoble",
                    season = "SPRING",
                    variantType = RouteVariantType.ROAD_GRAPH,
                    matchScore = 82.0,
                    reasons = listOf("Generated on cache road-graph (beta)", "Built from local road-network connectivity"),
                    previewLatLng = listOf(listOf(45.18, 5.72), listOf(45.20, 5.75)),
                    shape = "LOOP",
                    shapeScore = 0.78,
                    experimental = true,
                )
            ),
            shapeMatches = listOf(
                RouteRecommendation(
                    routeId = "historical-shape-kt",
                    activity = ActivityShort(91L, "Already done shape", ActivityType.Ride),
                    activityDate = "2025-01-01",
                    distanceKm = 39.0,
                    elevationGainM = 690.0,
                    durationSec = 6600,
                    isLoop = true,
                    start = null,
                    end = null,
                    startArea = "Grenoble",
                    season = "SPRING",
                    variantType = RouteVariantType.SHAPE_MATCH,
                    matchScore = 86.0,
                    reasons = listOf("Shape match: loop", "Route geometry confidence 81%"),
                    previewLatLng = listOf(listOf(45.18, 5.72), listOf(45.19, 5.73)),
                    shape = "LOOP",
                    shapeScore = 0.81,
                    experimental = false,
                )
            ),
            shapeRemixes = emptyList(),
        )

        // WHEN
        mockMvc.perform(
            post("/api/routes/generate/shape")
                .param("activityType", "Ride")
                .contentType(MediaType.APPLICATION_JSON)
                .content(
                    """
                    {
                      "shapeInputType": "draw",
                      "shapeData": "[[45.18,5.72],[45.19,5.73],[45.18,5.72]]",
                      "startPoint": {"lat": 45.19, "lng": 5.73},
                      "routeType": "RIDE"
                    }
                    """.trimIndent()
                )
        )
            // THEN
            .andExpect(status().isOk)
            .andExpect(jsonPath("$.routes.length()").value(0))
            .andExpect(content().string(org.hamcrest.Matchers.containsString("NON_SHAPE_CANDIDATES_IGNORED")))
    }

    @Test
    fun `generate shape routes returns fallback diagnostics when route is relaxed`() {
        // GIVEN
        every {
            routeExplorerService.getRouteExplorer(any(), any(), any())
        } returns RouteExplorerResult(
            closestLoops = emptyList(),
            variants = emptyList(),
            seasonal = emptyList(),
            roadGraphLoops = emptyList(),
            shapeMatches = listOf(
                RouteRecommendation(
                    routeId = "generated-relaxed-kt",
                    activity = ActivityShort(101L, "Generated relaxed loop", ActivityType.Ride),
                    activityDate = "2025-01-01",
                    distanceKm = 39.9,
                    elevationGainM = 770.0,
                    durationSec = 6900,
                    isLoop = true,
                    start = null,
                    end = null,
                    startArea = "Grenoble",
                    season = "SPRING",
                    variantType = RouteVariantType.SHAPE_MATCH,
                    matchScore = 88.2,
                    reasons = listOf(
                        "Generated with OSM road graph (OSRM)",
                        "Shape similarity: 85%",
                        "Shape mode: projected waypoints",
                        "Direction relaxed: no route found with requested heading",
                        "Selection profile: directional-best-effort",
                    ),
                    previewLatLng = listOf(listOf(45.18, 5.72), listOf(45.22, 5.76), listOf(45.18, 5.72)),
                    shape = "CUSTOM_SHAPE",
                    shapeScore = 0.85,
                    experimental = true,
                )
            ),
            shapeRemixes = emptyList(),
        )

        // WHEN
        mockMvc.perform(
            post("/api/routes/generate/shape")
                .param("activityType", "Ride")
                .contentType(MediaType.APPLICATION_JSON)
                .content(
                    """
                    {
                      "shapeInputType": "draw",
                      "shapeData": "[[45.18,5.72],[45.22,5.76],[45.18,5.72]]",
                      "startPoint": {"lat": 45.19, "lng": 5.73},
                      "routeType": "RIDE"
                    }
                    """.trimIndent()
                )
        )
            // THEN
            .andExpect(status().isOk)
            .andExpect(jsonPath("$.routes.length()").value(1))
            .andExpect(jsonPath("$.diagnostics[0].code").value("DIRECTION_RELAXED"))
            .andExpect(jsonPath("$.diagnostics[1].code").value("DIRECTION_BEST_EFFORT"))
    }

    @Test
    fun `generate shape routes defaults include walk when activityType is missing`() {
        // GIVEN
        every {
            routeExplorerService.getRouteExplorer(any(), any(), any())
        } returns RouteExplorerResult(
            closestLoops = emptyList(),
            variants = emptyList(),
            seasonal = emptyList(),
            roadGraphLoops = emptyList(),
            shapeMatches = emptyList(),
            shapeRemixes = emptyList(),
        )

        // WHEN
        mockMvc.perform(
            post("/api/routes/generate/shape")
                .contentType(MediaType.APPLICATION_JSON)
                .content(
                    """
                    {
                      "shapeInputType": "draw",
                      "shapeData": "[[45.18,5.72],[45.19,5.73],[45.18,5.72]]",
                      "startPoint": {"lat": 45.19, "lng": 5.73},
                      "routeType": "RIDE"
                    }
                    """.trimIndent()
                )
        )
            // THEN
            .andExpect(status().isOk)

        verify {
            routeExplorerService.getRouteExplorer(
                match { activityTypes -> activityTypes.contains(ActivityType.Walk) },
                any(),
                any(),
            )
        }
    }

    private fun loadStravaArtSmokeFixture(): StravaArtSmokeFixture {
        val mapper = JsonMapper.builder()
            .addModule(KotlinModule.Builder().build())
            .build()
        val relative = Paths.get("test-fixtures", "routes", "strava-art-smoke.json")
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
        fail("failed to locate Strava Art smoke fixture file: $relative")
    }

    private data class StravaArtSmokeFixture(
        val shapeInputType: String,
        val shapeData: String,
        val startPoint: SmokeStartPoint,
        val routeType: String,
        val variantCount: Int,
        val generatedRouteId: String,
        val generatedRouteName: String,
        val generatedPreviewLatLng: List<List<Double>>,
    )

    private data class SmokeStartPoint(
        val lat: Double,
        val lng: Double,
    )
}
