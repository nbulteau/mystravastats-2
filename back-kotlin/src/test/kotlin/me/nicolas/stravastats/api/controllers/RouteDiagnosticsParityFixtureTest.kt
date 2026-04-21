package me.nicolas.stravastats.api.controllers

import io.mockk.mockk
import me.nicolas.stravastats.api.dto.GeneratedRouteDto
import me.nicolas.stravastats.api.dto.RouteGenerationDiagnosticDto
import me.nicolas.stravastats.api.dto.RouteGenerationScoreDto
import me.nicolas.stravastats.domain.services.IRouteExplorerService
import org.junit.jupiter.api.Test
import java.nio.file.Files
import java.nio.file.Path
import java.nio.file.Paths
import tools.jackson.databind.json.JsonMapper
import tools.jackson.module.kotlin.KotlinModule
import tools.jackson.module.kotlin.readValue
import kotlin.test.assertEquals
import kotlin.test.fail

class RouteDiagnosticsParityFixtureTest {

    @Test
    fun `target diagnostics parity fixture maps reasons to expected codes`() {
        // GIVEN
        val fixture = loadDiagnosticsParityFixture()
        val controller = RoutesController(
            routeExplorerService = mockk<IRouteExplorerService>(relaxed = true),
        )

        // WHEN + THEN
        fixture.cases.forEach { parityCase ->
            val routes = listOf(
                buildGeneratedRoute(reasons = parityCase.reasons),
            )
            val diagnostics = invokeBuildSuccessfulTargetDiagnostics(controller, routes)
            val gotCodes = diagnostics.map { diagnostic -> diagnostic.code }

            assertEquals(
                parityCase.expectedCodes,
                gotCodes,
                "diagnostic code mismatch for case ${parityCase.name}",
            )
        }
    }

    private fun buildGeneratedRoute(reasons: List<String>): GeneratedRouteDto {
        return GeneratedRouteDto(
            routeId = "generated-parity-route",
            title = "Generated parity route",
            variantType = "ROAD_GRAPH",
            routeType = "RIDE",
            startDirection = "N",
            distanceKm = 42.0,
            elevationGainM = 600.0,
            durationSec = 7200,
            estimatedDurationSec = 7200,
            score = RouteGenerationScoreDto(
                global = 80.0,
                distance = 80.0,
                elevation = 80.0,
                duration = 80.0,
                direction = 80.0,
                shape = 80.0,
                roadFitness = 80.0,
            ),
            reasons = reasons,
            previewLatLng = emptyList(),
            start = null,
            end = null,
            activityId = null,
            isRoadGraphGenerated = true,
        )
    }

    @Suppress("UNCHECKED_CAST")
    private fun invokeBuildSuccessfulTargetDiagnostics(
        controller: RoutesController,
        routes: List<GeneratedRouteDto>,
    ): List<RouteGenerationDiagnosticDto> {
        val method = controller.javaClass.getDeclaredMethod(
            "buildSuccessfulTargetDiagnostics",
            List::class.java,
        )
        method.isAccessible = true
        return method.invoke(controller, routes) as List<RouteGenerationDiagnosticDto>
    }

    private fun loadDiagnosticsParityFixture(): DiagnosticsParityFixture {
        val mapper = JsonMapper.builder()
            .addModule(KotlinModule.Builder().build())
            .build()
        val relative = Paths.get("test-fixtures", "routes", "target-diagnostics-parity.json")
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
        fail("failed to locate shared diagnostics parity fixture file: $relative")
    }

    private data class DiagnosticsParityFixture(
        val cases: List<DiagnosticsParityCase> = emptyList(),
    )

    private data class DiagnosticsParityCase(
        val name: String,
        val reasons: List<String>,
        val expectedCodes: List<String>,
    )
}
