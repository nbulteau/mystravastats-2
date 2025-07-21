package me.nicolas.stravastats.api.controllers

import io.swagger.v3.oas.annotations.Operation
import io.swagger.v3.oas.annotations.media.Content
import io.swagger.v3.oas.annotations.media.Schema
import io.swagger.v3.oas.annotations.responses.ApiResponse
import me.nicolas.stravastats.api.dto.*
import me.nicolas.stravastats.domain.services.IDashboardService
import org.springframework.http.MediaType
import org.springframework.web.bind.annotation.GetMapping
import org.springframework.web.bind.annotation.RequestMapping
import org.springframework.web.bind.annotation.RestController

@RestController
@RequestMapping("/dashboard", produces = [MediaType.APPLICATION_JSON_VALUE])
@Schema(description = "Dashboard controller", name = "DashboardController")
class DashboardController(
    private val dashboardService: IDashboardService,
) {
    @Operation(
        description = "Get the dashboard data for a specific activity type",
        summary = "Get the dashboard data for a specific activity type",
        responses = [ApiResponse(
            responseCode = "200", description = "Dashboard data found",
            content = [Content(
                mediaType = MediaType.APPLICATION_JSON_VALUE,
                schema = Schema(implementation = DashboardDataDto::class)
            )]
        ), ApiResponse(
            responseCode = "404", description = "Dashboard data not found",
            content = [Content(
                mediaType = MediaType.APPLICATION_JSON_VALUE,
                schema = Schema(implementation = ErrorResponseMessageDto::class)
            )]
        )],
    )
    @GetMapping
    fun getDashboardData(
        activityType: String,
    ): DashboardDataDto {
        val activityTypes = activityType.convertToActivityTypeSet()

        val dashboardData = dashboardService.getDashboardData(activityTypes)

        return dashboardData.toDto()
    }

    @Operation(
        description = "Get the Eddington number for a specific activity type",
        summary = "Get the Eddington number for a specific activity type",
        responses = [ApiResponse(
            responseCode = "200", description = "Eddington number found",
            content = [Content(
                mediaType = MediaType.APPLICATION_JSON_VALUE,
                schema = Schema(implementation = EddingtonNumberDto::class)
            )]
        ), ApiResponse(
            responseCode = "404", description = "Eddington number not found",
            content = [Content(
                mediaType = MediaType.APPLICATION_JSON_VALUE,
                schema = Schema(implementation = ErrorResponseMessageDto::class)
            )]
        )],
    )
    @GetMapping("/eddington-number")
    fun getEddingtonNumber(
        activityType: String,
    ): EddingtonNumberDto {
        val activityTypes = activityType.convertToActivityTypeSet()

        return dashboardService.getEddingtonNumber(activityTypes).toDto()
    }

    @Operation(
        description = "Get the cumulative data for a year",
        summary = "Get the cumulative data for a year for a specific activity type",
        responses = [ApiResponse(
            responseCode = "200", description = "Cumulative data by months found",
            content = [Content(
                mediaType = MediaType.APPLICATION_JSON_VALUE,
                schema = Schema(implementation = Map::class)
            )]
        ), ApiResponse(
            responseCode = "404", description = "Cumulative data by months not found",
            content = [Content(
                mediaType = MediaType.APPLICATION_JSON_VALUE,
                schema = Schema(implementation = ErrorResponseMessageDto::class)
            )]
        )],
    )
    @GetMapping("/cumulative-data-per-year")
    fun getCumulativeDataPerYear(
        activityType: String,
    ): CumulativeDataPerYearDto {
        val activityTypes = activityType.convertToActivityTypeSet()

        return CumulativeDataPerYearDto(
            distance = dashboardService.getCumulativeDistancePerYear(activityTypes),
            elevation = dashboardService.getCumulativeElevationPerYear(activityTypes)
        )
    }
}