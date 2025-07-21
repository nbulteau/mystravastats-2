package me.nicolas.stravastats.api.controllers

import io.swagger.v3.oas.annotations.Operation
import io.swagger.v3.oas.annotations.media.Content
import io.swagger.v3.oas.annotations.media.Schema
import io.swagger.v3.oas.annotations.responses.ApiResponse
import io.swagger.v3.oas.annotations.tags.Tag
import me.nicolas.stravastats.api.dto.ErrorResponseMessageDto
import me.nicolas.stravastats.domain.services.activityproviders.IActivityProvider
import org.springframework.http.MediaType
import org.springframework.web.bind.annotation.GetMapping
import org.springframework.web.bind.annotation.RequestMapping
import org.springframework.web.bind.annotation.RequestParam
import org.springframework.web.bind.annotation.RestController

@RestController
@RequestMapping("/maps")
@Schema(description = "Maps controller", name = "MapsController")
@Tag(name = "Maps", description = "Maps endpoints")
class MapsController(
    private val stravaProxy: IActivityProvider,
) {
    @Operation(
        description = "Get the GPX coordinates for a specific stravaActivity type and year",
        summary = "Get the GPX coordinates for a specific stravaActivity type and year",
        responses = [ApiResponse(
            responseCode = "200", description = "GPX coordinates found", content = [Content(
                mediaType = MediaType.APPLICATION_JSON_VALUE, schema = Schema(implementation = List::class)
            )]
        ), ApiResponse(
            responseCode = "404", description = "GPX coordinates not found", content = [Content(
                mediaType = MediaType.APPLICATION_JSON_VALUE,
                schema = Schema(implementation = ErrorResponseMessageDto::class)
            )]
        )],
    )
    @GetMapping("/gpx")
    fun getGPX(
        @RequestParam(required = true) activityType: String,
        @RequestParam(required = false) year: Int?,
    ): List<List<List<Number>>> {
        val activityTypes = activityType.convertToActivityTypeSet()

        val activities = stravaProxy.getActivitiesByActivityTypeAndYear(activityTypes, year)

        // Take 1 out 100 points for this map to avoid too many points
        val step = year?.let { 10 } ?: 100

        return activities.map { activity ->
            // Take 1 out 100 points for this map
            val coordinates = activity.stream?.latlng?.data?.windowed(1, step)?.flatten()
            coordinates?.map { pair ->
                listOf(pair.first(), pair.last())
            } ?: emptyList()
        }
    }
}