package me.nicolas.stravastats.api.controllers

import io.swagger.v3.oas.annotations.Operation
import io.swagger.v3.oas.annotations.media.Content
import io.swagger.v3.oas.annotations.media.Schema
import io.swagger.v3.oas.annotations.responses.ApiResponse
import me.nicolas.stravastats.api.dto.ErrorResponseMessageDto
import me.nicolas.stravastats.api.dto.RouteExplorerResultDto
import me.nicolas.stravastats.api.dto.toDto
import me.nicolas.stravastats.domain.business.RouteExplorerRequest
import me.nicolas.stravastats.domain.services.IRouteExplorerService
import org.springframework.http.MediaType
import org.springframework.web.bind.annotation.GetMapping
import org.springframework.web.bind.annotation.RequestMapping
import org.springframework.web.bind.annotation.RequestParam
import org.springframework.web.bind.annotation.RestController

@RestController
@RequestMapping("/routes", produces = [MediaType.APPLICATION_JSON_VALUE])
@Schema(description = "Routes explorer controller", name = "RoutesController")
class RoutesController(
    private val routeExplorerService: IRouteExplorerService,
) {

    @Operation(
        description = "Get route recommendations based on historical cached activities",
        summary = "Get route recommendations",
        responses = [
            ApiResponse(
                responseCode = "200",
                description = "Routes recommendations found",
                content = [Content(
                    mediaType = MediaType.APPLICATION_JSON_VALUE,
                    schema = Schema(implementation = RouteExplorerResultDto::class)
                )]
            ),
            ApiResponse(
                responseCode = "400",
                description = "Invalid parameters",
                content = [Content(
                    mediaType = MediaType.APPLICATION_JSON_VALUE,
                    schema = Schema(implementation = ErrorResponseMessageDto::class)
                )]
            )
        ]
    )
    @GetMapping("/recommendations")
    fun getRouteRecommendations(
        @RequestParam(required = true) activityType: String,
        @RequestParam(required = false) year: Int?,
        @RequestParam(required = false) distanceTargetKm: Double?,
        @RequestParam(required = false) elevationTargetM: Double?,
        @RequestParam(required = false) durationTargetMin: Int?,
        @RequestParam(required = false) startDirection: String?,
        @RequestParam(required = false) routeType: String?,
        @RequestParam(required = false) season: String?,
        @RequestParam(required = false) shape: String?,
        @RequestParam(required = false, defaultValue = "false") includeRemix: Boolean,
        @RequestParam(required = false, defaultValue = "6") limit: Int,
    ): RouteExplorerResultDto {
        val activityTypes = activityType.convertToActivityTypeSet()
        val request = RouteExplorerRequest(
            distanceTargetKm = distanceTargetKm,
            elevationTargetM = elevationTargetM,
            durationTargetMin = durationTargetMin,
            startDirection = startDirection,
            routeType = routeType,
            season = season,
            limit = limit,
            shape = shape,
            includeRemix = includeRemix,
        )
        return routeExplorerService.getRouteExplorer(activityTypes, year, request).toDto()
    }
}
