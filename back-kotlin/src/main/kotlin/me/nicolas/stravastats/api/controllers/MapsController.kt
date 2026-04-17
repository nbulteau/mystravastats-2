package me.nicolas.stravastats.api.controllers

import io.swagger.v3.oas.annotations.Operation
import io.swagger.v3.oas.annotations.media.Content
import io.swagger.v3.oas.annotations.media.Schema
import io.swagger.v3.oas.annotations.responses.ApiResponse
import io.swagger.v3.oas.annotations.tags.Tag
import me.nicolas.stravastats.api.dto.ErrorResponseMessageDto
import me.nicolas.stravastats.api.dto.MapTrackDto
import me.nicolas.stravastats.domain.business.ActivityType
import me.nicolas.stravastats.domain.business.strava.StravaActivity
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
    ): List<MapTrackDto> {
        val activityTypes = activityType.convertToActivityTypeSet()

        val activities = stravaProxy.getActivitiesByActivityTypeAndYear(activityTypes, year)

        // Keep enough points to render smooth tracks while avoiding huge payloads.
        val step = year?.let { 10 } ?: 100

        return activities.mapNotNull { activity ->
            val coordinates = sampleCoordinates(activity.stream?.latlng?.data, step)
            if (coordinates.size < 2) {
                return@mapNotNull null
            }

            MapTrackDto(
                activityId = activity.id,
                activityName = activity.name,
                activityDate = activity.startDateLocal,
                activityType = resolveMapTrackActivityType(activity),
                distanceKm = activity.distance / 1000.0,
                elevationGainM = activity.totalElevationGain,
                coordinates = coordinates,
            )
        }
    }

    private fun sampleCoordinates(
        latlng: List<List<Double>>?,
        step: Int,
    ): List<List<Double>> {
        if (latlng.isNullOrEmpty()) {
            return emptyList()
        }

        val sampled = latlng
            .filter { pair ->
                pair.size >= 2
                    && pair[0].isFinite()
                    && pair[1].isFinite()
            }
            .filterIndexed { index, _ -> index % step == 0 }
            .map { pair -> listOf(pair[0], pair[1]) }
            .toMutableList()

        val last = latlng.lastOrNull()
        if (
            last != null
            && last.size >= 2
            && last[0].isFinite()
            && last[1].isFinite()
        ) {
            val shouldAppendLast = sampled.isEmpty()
                || sampled.last()[0] != last[0]
                || sampled.last()[1] != last[1]
            if (shouldAppendLast) {
                sampled.add(listOf(last[0], last[1]))
            }
        }
        return sampled
    }

    private fun resolveMapTrackActivityType(activity: StravaActivity): String {
        if (activity.commute) {
            return ActivityType.Commute.name
        }

        return activity.sportType.takeIf { it.isNotBlank() }
            ?: activity.type.takeIf { it.isNotBlank() }
            ?: ActivityType.Ride.name
    }
}
