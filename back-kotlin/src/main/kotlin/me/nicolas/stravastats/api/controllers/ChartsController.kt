package me.nicolas.stravastats.api.controllers

import io.swagger.v3.oas.annotations.Operation
import io.swagger.v3.oas.annotations.media.Content
import io.swagger.v3.oas.annotations.media.Schema
import io.swagger.v3.oas.annotations.responses.ApiResponse
import me.nicolas.stravastats.api.dto.EddingtonNumberDto
import me.nicolas.stravastats.api.dto.ErrorResponseMessageDto
import me.nicolas.stravastats.api.dto.toDto
import me.nicolas.stravastats.domain.business.Period
import me.nicolas.stravastats.domain.business.strava.ActivityType
import me.nicolas.stravastats.domain.services.IChartsService
import org.springframework.http.MediaType
import org.springframework.web.bind.annotation.GetMapping
import org.springframework.web.bind.annotation.RequestMapping
import org.springframework.web.bind.annotation.RestController

@RestController
@RequestMapping("/charts")
@Schema(description = "Charts controller", name = "ChartsController")
class ChartsController(
    private val chartsService: IChartsService,
) {

    @Operation(
        description = "Get the distance by months for a specific activity type and year",
        summary = "Get the distance by months for a specific activity type and year",
        responses = [ApiResponse(
            responseCode = "200", description = "Distance by months found", content = [Content(
                mediaType = MediaType.APPLICATION_JSON_VALUE, schema = Schema(implementation = Map::class)
            )]
        ), ApiResponse(
            responseCode = "404", description = "Distance by months not found", content = [Content(
                mediaType = MediaType.APPLICATION_JSON_VALUE,
                schema = Schema(implementation = ErrorResponseMessageDto::class)
            )]
        )],
    )
    @GetMapping("/distance-by-period")
    fun getDistanceByPeriod(
        activityType: ActivityType,
        year: Int,
        period: Period,
    ): List<Map<String, Double>> {
        return chartsService.getDistanceByPeriodByActivityTypeByYear(activityType, year, period)
            .map { mapOf(it.first to it.second) }
    }

    @Operation(
        description = "Get the elevation by months for a specific activity type and year",
        summary = "Get the elevation by months for a specific activity type and year",
        responses = [ApiResponse(
            responseCode = "200", description = "Elevation by months found", content = [Content(
                mediaType = MediaType.APPLICATION_JSON_VALUE, schema = Schema(implementation = Map::class)
            )]
        ), ApiResponse(
            responseCode = "404", description = "Elevation by months not found", content = [Content(
                mediaType = MediaType.APPLICATION_JSON_VALUE,
                schema = Schema(implementation = ErrorResponseMessageDto::class)
            )]
        )],
    )
    @GetMapping("/elevation-by-period")
    fun getElevationByPeriod(
        activityType: ActivityType,
        year: Int,
        period: Period,
    ): List<Map<String, Double>> {
        return chartsService.getElevationByPeriodByActivityTypeByYear(activityType, year, period)
            .map { mapOf(it.first to it.second) }
    }

    @GetMapping("/cumulative-distance-per-year")
    fun getCumulativeDistancePerYear(
        activityType: ActivityType,
    ): Map<String, Map<String, Double>> {

        return chartsService.getCumulativeDistancePerYear(activityType)
    }

    @GetMapping("/eddington-number")
    fun getEddingtonNumber(
        activityType: ActivityType,
    ): EddingtonNumberDto {

        return chartsService.getEddingtonNumber(activityType).toDto()
    }
}


