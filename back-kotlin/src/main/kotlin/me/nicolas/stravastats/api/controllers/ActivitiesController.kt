package me.nicolas.stravastats.api.controllers

import io.swagger.v3.oas.annotations.Operation
import io.swagger.v3.oas.annotations.media.ArraySchema
import io.swagger.v3.oas.annotations.media.Content
import io.swagger.v3.oas.annotations.media.Schema
import io.swagger.v3.oas.annotations.responses.ApiResponse
import io.swagger.v3.oas.annotations.tags.Tag
import me.nicolas.stravastats.api.dto.ActivityDto
import me.nicolas.stravastats.api.dto.DetailedActivityDto
import me.nicolas.stravastats.api.dto.ErrorResponseMessageDto
import me.nicolas.stravastats.api.dto.toDto
import me.nicolas.stravastats.domain.services.IActivityService
import org.springframework.data.rest.webmvc.ResourceNotFoundException
import org.springframework.http.HttpHeaders
import org.springframework.http.MediaType
import org.springframework.http.ResponseEntity
import org.springframework.web.bind.annotation.*

@RestController
@RequestMapping("/activities", produces = [MediaType.APPLICATION_JSON_VALUE])
@Schema(description = "Activities controller", name = "ActivitiesController")
@Tag(name = "Activities", description = "Activities endpoints")
class ActivitiesController(
    private val activityService: IActivityService,
) {

    @Operation(
        description = "Get activities by activity type and year. If year is null, all activities are returned.",
        summary = "Get activities by activity type",
        responses = [ApiResponse(
            responseCode = "200", description = "Activities found",
            content = [Content(
                mediaType = MediaType.APPLICATION_JSON_VALUE,
                array = ArraySchema(schema = Schema(implementation = ActivityDto::class))
            )]
        ), ApiResponse(
            responseCode = "404", description = "Activities not found",
            content = [Content(
                mediaType = MediaType.APPLICATION_JSON_VALUE,
                schema = Schema(implementation = ErrorResponseMessageDto::class)
            )]
        )],
    )

    @GetMapping
    fun getActivitiesByActivityType(
        @RequestParam(required = true) activityType: String,
        @RequestParam(required = false) year: Int?,
    ): List<ActivityDto> {
        val activityTypes = activityType.convertToActivityTypeSet()

        return activityService.getActivitiesByActivityTypeAndYear(activityTypes, year)
            .map { activity -> activity.toDto() }
    }

    @Operation(
        description = "Export the activities in CSV format",
        summary = "Export the activities in CSV format",
        responses = [
            ApiResponse(
                responseCode = "200",
                description = "CSV file",
                content = [Content(
                    mediaType = "text/csv",
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
    @GetMapping("/csv", produces = ["text/csv"])
    fun exportCSV(
        @RequestParam(required = true) activityType: String,
        @RequestParam(required = false) year: Int?,
    ): ResponseEntity<String> {
        val activityTypes = activityType.convertToActivityTypeSet()

        val csvContent = activityService.exportCSV(activityTypes, year)
        val fileSuffix = year?.toString() ?: "all-years"

        return ResponseEntity.ok()
            .contentType(MediaType.valueOf("text/csv"))
            .header(HttpHeaders.CONTENT_DISPOSITION, "attachment; filename=\"activities-$fileSuffix.csv\"")
            .body(csvContent)
    }

    @Operation(
        description = "Get a detailed stravaActivity",
        summary = "Get a detailed stravaActivity",
        responses = [
            ApiResponse(
                responseCode = "200",
                description = "StravaActivity found",
                content = [Content(
                    mediaType = MediaType.APPLICATION_JSON_VALUE,
                    schema = Schema(implementation = DetailedActivityDto::class)
                )]
            ),
            ApiResponse(
                responseCode = "404",
                description = "StravaActivity not found",
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
        return (activityService.getDetailedActivity(activityId)
            ?: throw ResourceNotFoundException("StravaActivity id $activityId not found")).toDto()
    }
}
