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

@ExtendWith(SpringExtension::class)
@WebMvcTest(RoutesController::class)
class RoutesControllerTest {

    @Autowired
    private lateinit var mockMvc: MockMvc

    @MockkBean
    private lateinit var routeExplorerService: IRouteExplorerService

    @Test
    fun `generate target routes returns unified routes payload`() {
        // GIVEN
        every {
            routeExplorerService.getRouteExplorer(
                any(),
                any(),
                match { request ->
                    request.startPoint?.let { startPoint ->
                        startPoint.lat == 45.19 && startPoint.lng == 5.73
                    } == true
                }
            )
        } returns RouteExplorerResult(
            closestLoops = emptyList(),
            variants = emptyList(),
            seasonal = emptyList(),
            roadGraphLoops = listOf(
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
                    variantType = RouteVariantType.ROAD_GRAPH,
                    matchScore = 92.1,
                    reasons = listOf("Road-graph generated loop"),
                    previewLatLng = listOf(listOf(45.18, 5.72), listOf(45.20, 5.75), listOf(45.18, 5.72)),
                    shape = "LOOP",
                    shapeScore = 0.9,
                    experimental = true,
                )
            ),
            shapeMatches = emptyList(),
            shapeRemixes = emptyList(),
        )

        // WHEN
        mockMvc.perform(
            post("/api/routes/generate/target")
                .param("activityType", "Ride")
                .param("year", "2025")
                .contentType(MediaType.APPLICATION_JSON)
                .content(
                    """
                    {
                      "startPoint": {"lat": 45.19, "lng": 5.73},
                      "routeType": "RIDE",
                      "startDirection": "N",
                      "distanceTargetKm": 42.0,
                      "elevationTargetM": 900.0,
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
    fun `generate target routes uses surface fitness reason for road fitness score`() {
        // GIVEN
        every {
            routeExplorerService.getRouteExplorer(any(), any(), any())
        } returns RouteExplorerResult(
            closestLoops = emptyList(),
            variants = emptyList(),
            seasonal = emptyList(),
            roadGraphLoops = listOf(
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
                    variantType = RouteVariantType.ROAD_GRAPH,
                    matchScore = 87.0,
                    reasons = listOf(
                        "Generated with OSM road graph (OSRM)",
                        "Surface mix: paved 38%, gravel 52%, trail 10%, unknown 0%",
                        "Surface fitness: 68%",
                    ),
                    previewLatLng = listOf(listOf(45.18, 5.72), listOf(45.20, 5.75), listOf(45.18, 5.72)),
                    shape = "LOOP",
                    shapeScore = 0.82,
                    experimental = true,
                )
            ),
            shapeMatches = emptyList(),
            shapeRemixes = emptyList(),
        )

        // WHEN
        mockMvc.perform(
            post("/api/routes/generate/target")
                .param("activityType", "Ride")
                .contentType(MediaType.APPLICATION_JSON)
                .content(
                    """
                    {
                      "startPoint": {"lat": 45.19, "lng": 5.73},
                      "routeType": "GRAVEL",
                      "distanceTargetKm": 40.0
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
    fun `generate target routes infers strictDirection when startDirection is undefined`() {
        // GIVEN
        every {
            routeExplorerService.getRouteExplorer(
                any(),
                any(),
                any(),
            )
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
            post("/api/routes/generate/target")
                .param("activityType", "Ride")
                .contentType(MediaType.APPLICATION_JSON)
                .content(
                    """
                    {
                      "startPoint": {"lat": 45.19, "lng": 5.73},
                      "generationMode": "AUTOMATIC",
                      "routeType": "RIDE",
                      "startDirection": "UNDEFINED",
                      "distanceTargetKm": 42.0
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
                    request.strictDirection && request.startDirection == null
                }
            )
        }
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
    fun `generate target routes rejects custom mode without waypoints`() {
        // GIVEN
        // WHEN
        mockMvc.perform(
            post("/api/routes/generate/target")
                .param("activityType", "Ride")
                .contentType(MediaType.APPLICATION_JSON)
                .content(
                    """
                    {
                      "startPoint": {"lat": 45.19, "lng": 5.73},
                      "generationMode": "CUSTOM",
                      "routeType": "RIDE",
                      "distanceTargetKm": 42.0
                    }
                    """.trimIndent()
                )
        )
            // THEN
            .andExpect(status().isBadRequest)
    }

    @Test
    fun `generated route gpx endpoint returns file after generate`() {
        // GIVEN
        every {
            routeExplorerService.getRouteExplorer(any(), any(), any())
        } returns RouteExplorerResult(
            closestLoops = emptyList(),
            variants = emptyList(),
            seasonal = emptyList(),
            roadGraphLoops = listOf(
                RouteRecommendation(
                    routeId = "generated-cache-kt",
                    activity = ActivityShort(34L, "Generated loop cache", ActivityType.Ride),
                    activityDate = "2025-01-01",
                    distanceKm = 30.0,
                    elevationGainM = 450.0,
                    durationSec = 4500,
                    isLoop = true,
                    start = null,
                    end = null,
                    startArea = "Grenoble",
                    season = "SUMMER",
                    variantType = RouteVariantType.ROAD_GRAPH,
                    matchScore = 88.0,
                    reasons = listOf("Generated route"),
                    previewLatLng = listOf(listOf(45.18, 5.72), listOf(45.19, 5.73), listOf(45.18, 5.72)),
                    shape = "LOOP",
                    shapeScore = 0.85,
                    experimental = true,
                )
            ),
            shapeMatches = emptyList(),
            shapeRemixes = emptyList(),
        )

        mockMvc.perform(
            post("/api/routes/generate/target")
                .param("activityType", "Ride")
                .contentType(MediaType.APPLICATION_JSON)
                .content(
                    """
                    {
                      "startPoint": {"lat": 45.19, "lng": 5.73},
                      "routeType": "RIDE",
                      "distanceTargetKm": 30.0
                    }
                    """.trimIndent()
                )
        ).andExpect(status().isOk)

        // WHEN
        mockMvc.perform(get("/api/routes/generated-cache-kt/gpx"))
            // THEN
            .andExpect(status().isOk)
            .andExpect(content().contentType("application/gpx+xml"))
            .andExpect(content().string(org.hamcrest.Matchers.containsString("<gpx")))
    }

    @Test
    fun `generate target routes falls back to historical routes when road graph is unavailable`() {
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
            roadGraphLoops = emptyList(),
            shapeMatches = emptyList(),
            shapeRemixes = emptyList(),
        )

        // WHEN
        mockMvc.perform(
            post("/api/routes/generate/target")
                .param("activityType", "Ride")
                .contentType(MediaType.APPLICATION_JSON)
                .content(
                    """
                    {
                      "startPoint": {"lat": 45.19, "lng": 5.73},
                      "routeType": "RIDE",
                      "startDirection": "N",
                      "distanceTargetKm": 42.0
                    }
                    """.trimIndent()
                )
        )
            // THEN
            .andExpect(status().isOk)
            .andExpect(jsonPath("$.routes.length()").value(1))
            .andExpect(jsonPath("$.routes[0].routeId").value("legacy-route-kt"))
            .andExpect(jsonPath("$.diagnostics[0].code").value("ENGINE_CACHE_FALLBACK"))
    }

    @Test
    fun `generate target routes returns fallback diagnostics when route is relaxed`() {
        // GIVEN
        every {
            routeExplorerService.getRouteExplorer(any(), any(), any())
        } returns RouteExplorerResult(
            closestLoops = emptyList(),
            variants = emptyList(),
            seasonal = emptyList(),
            roadGraphLoops = listOf(
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
                    variantType = RouteVariantType.ROAD_GRAPH,
                    matchScore = 88.2,
                    reasons = listOf(
                        "Direction relaxed: no route found with requested heading",
                        "Selection profile: directional-best-effort",
                    ),
                    previewLatLng = listOf(listOf(45.18, 5.72), listOf(45.22, 5.76), listOf(45.18, 5.72)),
                    shape = "LOOP",
                    shapeScore = 0.85,
                    experimental = true,
                )
            ),
            shapeMatches = emptyList(),
            shapeRemixes = emptyList(),
        )

        // WHEN
        mockMvc.perform(
            post("/api/routes/generate/target")
                .param("activityType", "Ride")
                .contentType(MediaType.APPLICATION_JSON)
                .content(
                    """
                    {
                      "startPoint": {"lat": 45.19, "lng": 5.73},
                      "routeType": "RIDE",
                      "startDirection": "N",
                      "distanceTargetKm": 40.0
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
    fun `generate target routes defaults include walk when activityType is missing`() {
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
            post("/api/routes/generate/target")
                .contentType(MediaType.APPLICATION_JSON)
                .content(
                    """
                    {
                      "startPoint": {"lat": 45.19, "lng": 5.73},
                      "routeType": "RIDE",
                      "distanceTargetKm": 30.0
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
}
