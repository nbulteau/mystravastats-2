package me.nicolas.stravastats.api.controllers

import io.swagger.v3.oas.annotations.Operation
import io.swagger.v3.oas.annotations.media.Content
import io.swagger.v3.oas.annotations.media.Schema
import io.swagger.v3.oas.annotations.responses.ApiResponse
import me.nicolas.stravastats.api.dto.CumulativeDataPerYearDto
import me.nicolas.stravastats.api.dto.EddingtonNumberDto
import me.nicolas.stravastats.api.dto.ErrorResponseMessageDto
import me.nicolas.stravastats.api.dto.toDto
import me.nicolas.stravastats.domain.business.ActivityType
import me.nicolas.stravastats.domain.services.IChartsService
import org.springframework.http.MediaType
import org.springframework.web.bind.annotation.GetMapping
import org.springframework.web.bind.annotation.RequestMapping
import org.springframework.web.bind.annotation.RestController

@RestController
@RequestMapping("/dashboard", produces = [MediaType.APPLICATION_JSON_VALUE])
@Schema(description = "Dashboard controller", name = "DashboardController")
class DashboardController(
    private val chartsService: IChartsService,
) {
    @GetMapping
    fun getDashboardData(): List<String> {
        return emptyList()
    }

    @Operation(
        description = "Get the Eddington number for a specific stravaActivity type",
        summary = "Get the Eddington number for a specific stravaActivity type",
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
        activityType: ActivityType,
    ): EddingtonNumberDto {

        return chartsService.getEddingtonNumber(activityType).toDto()
    }

    @Operation(
        description = "Get the cumulative data for a year",
        summary = "Get the cumulative data for a year for a specific stravaActivity type",
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
        activityType: ActivityType,
    ): CumulativeDataPerYearDto {
        return CumulativeDataPerYearDto(
            distance = chartsService.getCumulativeDistancePerYear(activityType),
            elevation = chartsService.getCumulativeElevationPerYear(activityType)
        )
    }
}