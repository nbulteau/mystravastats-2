package me.nicolas.stravastats.api.controllers

import io.swagger.v3.oas.annotations.Operation
import io.swagger.v3.oas.annotations.media.Content
import io.swagger.v3.oas.annotations.media.Schema
import io.swagger.v3.oas.annotations.responses.ApiResponse
import me.nicolas.stravastats.api.dto.ErrorResponseMessageDto
import me.nicolas.stravastats.api.dto.StatisticsDto
import me.nicolas.stravastats.api.dto.toDto
import me.nicolas.stravastats.domain.services.IStatisticsService
import org.springframework.http.MediaType
import org.springframework.web.bind.annotation.GetMapping
import org.springframework.web.bind.annotation.RequestMapping
import org.springframework.web.bind.annotation.RequestParam
import org.springframework.web.bind.annotation.RestController

@RestController
@RequestMapping("/statistics")
@Schema(description = "Statistics controller", name = "StatisticsController")
class StatisticsController(
    private val statisticsService: IStatisticsService,
) {

    @Operation(
        description = "Get the statistics for a specific stravaActivity type and year",
        summary = "Get the statistics for a specific stravaActivity type and year",
        responses = [
            ApiResponse(
                responseCode = "200",
                description = "Statistics found",
                content = [Content(
                    mediaType = MediaType.APPLICATION_JSON_VALUE,
                    schema = Schema(implementation = StatisticsDto::class)
                )]
            ),
            ApiResponse(
                responseCode = "404",
                description = "Statistics not found",
                content = [Content(
                    mediaType = MediaType.APPLICATION_JSON_VALUE,
                    schema = Schema(implementation = ErrorResponseMessageDto::class)
                )]
            )
        ]
    )
    @GetMapping
    fun getStatistics(
        @RequestParam(required = true) activityType: String,
        @RequestParam(required = false) year: Int?,
    ): List<StatisticsDto> {
        val activityTypes = activityType.convertToActivityTypeSet()

        return statisticsService.getStatistics(activityTypes, year)
            .map { activityStatistic -> activityStatistic.toDto() }
    }
}

