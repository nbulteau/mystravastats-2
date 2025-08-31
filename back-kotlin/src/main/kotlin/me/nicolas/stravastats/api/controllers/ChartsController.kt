package me.nicolas.stravastats.api.controllers

import io.swagger.v3.oas.annotations.Operation
import io.swagger.v3.oas.annotations.media.ArraySchema
import io.swagger.v3.oas.annotations.media.Content
import io.swagger.v3.oas.annotations.media.Schema
import io.swagger.v3.oas.annotations.responses.ApiResponse
import me.nicolas.stravastats.api.dto.ErrorResponseMessageDto
import me.nicolas.stravastats.domain.business.Period
import me.nicolas.stravastats.domain.services.IChartsService
import org.springframework.http.MediaType
import org.springframework.web.bind.annotation.GetMapping
import org.springframework.web.bind.annotation.RequestMapping
import org.springframework.web.bind.annotation.RestController
import java.time.LocalDate
import java.time.ZoneOffset

@RestController
@RequestMapping("/charts", produces = [MediaType.APPLICATION_JSON_VALUE])
@Schema(description = "Charts controller", name = "ChartsController")
class ChartsController(
    private val chartsService: IChartsService,
) {

    @Operation(
        description = "Get the distance for a specific stravaActivity type and year",
        summary = "Get the distance for a specific stravaActivity type and year",
        responses = [ApiResponse(
            responseCode = "200", description = "DistanceStream found",
            content = [Content(
                mediaType = MediaType.APPLICATION_JSON_VALUE,
                array = ArraySchema(schema = Schema(implementation = Map::class))
            )]
        ), ApiResponse(
            responseCode = "404", description = "DistanceStream not found",
            content = [Content(
                mediaType = MediaType.APPLICATION_JSON_VALUE,
                schema = Schema(implementation = ErrorResponseMessageDto::class)
            )]
        )],
    )
    @GetMapping("/distance-by-period")
    fun getDistanceByPeriod(
        activityType: String,
        year: Int,
        period: Period,
    ): List<Map<String, Double>> {
        val activityTypes = activityType.convertToActivityTypeSet()

        return chartsService.getDistanceByPeriodByActivityTypeByYear(activityTypes, year, period)
            .map { mapOf(it.first to it.second) }
    }

    @Operation(
        description = "Get the elevation for a specific stravaActivity type and year",
        summary = "Get the elevation for a specific stravaActivity type and year",
        responses = [ApiResponse(
            responseCode = "200", description = "Elevation found",
            content = [Content(
                mediaType = MediaType.APPLICATION_JSON_VALUE,
                array = ArraySchema(schema = Schema(implementation = Map::class))
            )]
        ), ApiResponse(
            responseCode = "404", description = "Elevation not found",
            content = [Content(
                mediaType = MediaType.APPLICATION_JSON_VALUE,
                schema = Schema(implementation = ErrorResponseMessageDto::class)
            )]
        )],
    )
    @GetMapping("/elevation-by-period")
    fun getElevationByPeriod(
        activityType: String,
        year: Int,
        period: Period,
    ): List<Map<String, Double>> {
        val activityTypes = activityType.convertToActivityTypeSet()

        return chartsService.getElevationByPeriodByActivityTypeByYear(activityTypes, year, period)
            .map { mapOf(it.first to it.second) }
    }

    @Operation(
        description = "Get the average speed for a specific stravaActivity type and year",
        summary = "Get the average speed for a specific stravaActivity type and year",
        responses = [ApiResponse(
            responseCode = "200", description = "Average speed found",
            content = [Content(
                mediaType = MediaType.APPLICATION_JSON_VALUE,
                array = ArraySchema(schema = Schema(implementation = Map::class))
            )]
        ), ApiResponse(
            responseCode = "404", description = "Average speed not found",
            content = [Content(
                mediaType = MediaType.APPLICATION_JSON_VALUE,
                schema = Schema(implementation = ErrorResponseMessageDto::class)
            )]
        )],
    )
    @GetMapping("/average-speed-by-period")
    fun getAverageSpeedByPeriod(
        activityType: String,
        year: Int,
        period: Period,
    ): List<Map<String, Double>> {
        val activityTypes = activityType.convertToActivityTypeSet()

        return chartsService.getAverageSpeedByPeriodByActivityTypeByYear(activityTypes, year, period)
            .map { mapOf(it.first to it.second) }
    }


    @GetMapping("/average-cadence-by-period")
    fun getAverageCadenceByPeriod(
        activityType: String,
        year: Int,
        period: Period,
    ): List<Map<String, Double>> {
        val activityTypes = activityType.convertToActivityTypeSet()

        return chartsService.getAverageCadenceByPeriodByActivityTypeByYear(activityTypes, year, period)
            .map { mapOf(it.first to it.second) }
    }
}


