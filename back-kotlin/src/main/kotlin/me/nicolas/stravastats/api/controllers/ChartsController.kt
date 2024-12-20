package me.nicolas.stravastats.api.controllers

import io.swagger.v3.oas.annotations.Operation
import io.swagger.v3.oas.annotations.media.ArraySchema
import io.swagger.v3.oas.annotations.media.Content
import io.swagger.v3.oas.annotations.media.Schema
import io.swagger.v3.oas.annotations.responses.ApiResponse
import me.nicolas.stravastats.api.dto.ErrorResponseMessageDto
import me.nicolas.stravastats.domain.business.ActivityType
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
        description = "Get the distance by months for a specific stravaActivity type and year",
        summary = "Get the distance by months for a specific stravaActivity type and year",
        responses = [ApiResponse(
            responseCode = "200", description = "DistanceStream by months found",
            content = [Content(
                mediaType = MediaType.APPLICATION_JSON_VALUE,
                array = ArraySchema(schema = Schema(implementation = Map::class))
            )]
        ), ApiResponse(
            responseCode = "404", description = "DistanceStream by months not found",
            content = [Content(
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
        description = "Get the elevation by months for a specific stravaActivity type and year",
        summary = "Get the elevation by months for a specific stravaActivity type and year",
        responses = [ApiResponse(
            responseCode = "200", description = "Elevation by months found",
            content = [Content(
                mediaType = MediaType.APPLICATION_JSON_VALUE,
                array = ArraySchema(schema = Schema(implementation = Map::class))
            )]
        ), ApiResponse(
            responseCode = "404", description = "Elevation by months not found",
            content = [Content(
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

    @Operation(
        description = "Get the average speed by months for a specific stravaActivity type and year",
        summary = "Get the average speed by months for a specific stravaActivity type and year",
        responses = [ApiResponse(
            responseCode = "200", description = "Average speed by months found",
            content = [Content(
                mediaType = MediaType.APPLICATION_JSON_VALUE,
                array = ArraySchema(schema = Schema(implementation = Map::class))
            )]
        ), ApiResponse(
            responseCode = "404", description = "Average speed by months not found",
            content = [Content(
                mediaType = MediaType.APPLICATION_JSON_VALUE,
                schema = Schema(implementation = ErrorResponseMessageDto::class)
            )]
        )],
    )
    @GetMapping("/average-speed-by-period")
    fun getAverageSpeedByPeriod(
        activityType: ActivityType,
        year: Int,
        period: Period,
    ): List<Map<String, Double>> {
        return chartsService.getAverageSpeedByPeriodByActivityTypeByYear(activityType, year, period)
            .map { mapOf(it.first to it.second) }
    }


    @GetMapping("/average-cadence-by-period")
    fun getAverageCadenceByPeriod(
        activityType: ActivityType,
    ): List<List<Long>> {
        return chartsService.getAverageCadenceByPeriodByActivityTypeByYear(activityType)
            .map { (day, value) ->
                // Convert the day to seconds
                val seconds = LocalDate.parse(day).atStartOfDay(ZoneOffset.UTC).toEpochSecond()
                listOf(seconds, value.toLong())
            }
    }
}


