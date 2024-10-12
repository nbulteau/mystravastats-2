package me.nicolas.stravastats.api.controllers

import io.swagger.v3.oas.annotations.Operation
import io.swagger.v3.oas.annotations.media.ArraySchema
import io.swagger.v3.oas.annotations.media.Content
import io.swagger.v3.oas.annotations.media.Schema
import io.swagger.v3.oas.annotations.responses.ApiResponse
import io.swagger.v3.oas.annotations.tags.Tag
import me.nicolas.stravastats.api.dto.*
import me.nicolas.stravastats.domain.business.strava.Activity
import me.nicolas.stravastats.domain.business.strava.ActivityType
import me.nicolas.stravastats.domain.services.IActivityService
import org.springframework.cache.annotation.Cacheable
import org.springframework.data.domain.Page
import org.springframework.data.domain.Pageable
import org.springframework.data.rest.webmvc.ResourceNotFoundException
import org.springframework.data.web.PagedModel
import org.springframework.http.MediaType
import org.springframework.web.bind.annotation.*

@RestController
@RequestMapping("/activities", produces = [MediaType.APPLICATION_JSON_VALUE])
@Schema(description = "Activities controller", name = "ActivitiesController")
@Tag(name = "Activities", description = "Activities endpoints")
class ActivitiesController(
    private val activityService: IActivityService,
) {

    private val validActivitySortProperties = setOf(
        "averageSpeed",
        "averageCadence",
        "averageHeartrate",
        "maxHeartrate",
        "averageWatts",
        "distance",
        "elapsedTime",
        "elevHigh",
        "maxSpeed",
        "movingTime",
        "startDate",
        "totalElevationGain",
        "weightedAverageWatts"
    )

    @Operation(
        description = "Get all activities",
        summary = "Get all activities from the authenticated user",
        responses = [ApiResponse(
            responseCode = "200", description = "Activities found",
            content = [Content(
                mediaType = MediaType.APPLICATION_JSON_VALUE,
                schema = Schema(implementation = PagedModel::class)
            )]
        ), ApiResponse(
            responseCode = "404", description = "Activities not found",
            content = [Content(
                mediaType = MediaType.APPLICATION_JSON_VALUE,
                schema = Schema(implementation = ErrorResponseMessageDto::class)
            )]
        )],
    )
    @GetMapping(("/by-page"))
    fun listActivitiesWithPageable(
        pageable: Pageable,
    ): PagedModel<ActivityDto> {
        // Check if the pageable is valid
        pageable.pageNumber.takeIf { it >= 0 } ?: throw ResourceNotFoundException()
        pageable.pageSize.takeIf { it > 0 } ?: throw ResourceNotFoundException()

        if (pageable.sort.isSorted && pageable.sort.any { sort -> sort.property !in validActivitySortProperties }) {
            throw IllegalArgumentException("Invalid sort property : ${pageable.sort}")
        }

        val resultPage: Page<Activity> = activityService.listActivitiesPaginated(pageable)
        if (pageable.pageNumber > resultPage.totalPages) {
            throw ResourceNotFoundException("Page not found")
        }

        return PagedModel(resultPage.map { activity -> activity.toDto() })
    }

    @Operation(
        description = "Get activities by activity type and year. If year is null, all activities are returned. It return a map with the date as key and the cumulated distance in km as value.",
        summary = "Get the active days by activity type for a year",
        responses = [ApiResponse(
            responseCode = "200", description = "Active days found",
            content = [Content(
                mediaType = MediaType.APPLICATION_JSON_VALUE,
                array = ArraySchema(schema = Schema(implementation = ActivityDto::class))
            )]
        ), ApiResponse(
            responseCode = "404", description = "Active days not found",
            content = [Content(
                mediaType = MediaType.APPLICATION_JSON_VALUE,
                schema = Schema(implementation = ErrorResponseMessageDto::class)
            )]
        )],
    )

    @GetMapping
    @Cacheable("activities")
    fun getActivitiesByActivityType(
        @RequestParam(required = true) activityType: ActivityType,
        @RequestParam(required = false) year: Int?,
    ): List<ActivityDto> {
        return activityService.getActivitiesByActivityTypeAndYear(activityType, year)
            .map { activity -> activity.toDto() }
    }

    @Operation(
        description = "Get the active days by activity type",
        summary = "Get the active days by activity type",
        responses = [ApiResponse(
            responseCode = "200", description = "Active days found",
            content = [Content(
                mediaType = MediaType.APPLICATION_JSON_VALUE,
                schema = Schema(implementation = Map::class)
            )]
        ), ApiResponse(
            responseCode = "404", description = "Active days not found",
            content = [Content(
                mediaType = MediaType.APPLICATION_JSON_VALUE,
                schema = Schema(implementation = ErrorResponseMessageDto::class)
            )]
        )],
    )
    @GetMapping("/active-days")
    fun getActiveDaysByActivityType(
        activityType: ActivityType,
    ): Map<String, Int> {

        return activityService.getActivitiesByActivityTypeGroupByActiveDays(activityType)
    }

    @Operation(
        description = "Export the activities in CSV format",
        summary = "Export the activities in CSV format",
        responses = [
            ApiResponse(
                responseCode = "200",
                description = "CSV file",
                content = [Content(
                    mediaType = MediaType.TEXT_PLAIN_VALUE,
                    schema = Schema(implementation = String::class)
                )]
            ),
            ApiResponse(
                responseCode = "404",
                description = "CSV file not found",
                content = [Content(
                    mediaType = MediaType.APPLICATION_JSON_VALUE,
                    schema = Schema(implementation = ErrorResponseMessageDto::class)
                )]
            )
        ]
    )
    @GetMapping("/csv", produces = [MediaType.TEXT_PLAIN_VALUE])
    fun exportCSV(activityType: ActivityType, year: Int): String {

        return activityService.exportCSV(activityType, year)
    }

    @Operation(
        description = "Get a detailed activity",
        summary = "Get a detailed activity",
        responses = [
            ApiResponse(
                responseCode = "200",
                description = "Activity found",
                content = [Content(
                    mediaType = MediaType.APPLICATION_JSON_VALUE,
                    schema = Schema(implementation = DetailedActivityDto::class)
                )]
            ),
            ApiResponse(
                responseCode = "404",
                description = "Activity not found",
                content = [Content(
                    mediaType = MediaType.APPLICATION_JSON_VALUE,
                    schema = Schema(implementation = ErrorResponseMessageDto::class)
                )]
            )
        ]
    )
    @GetMapping("/{activityId}")
    fun getDetailedActivity(
        @PathVariable activityId: Long,
    ): DetailedActivityDto {

        return activityService.getActivity(activityId).orElseThrow {
            ResourceNotFoundException("Activity id $activityId not found")
        }.toDetailedActivityDto()
    }
}