package me.nicolas.stravastats.api.controllers

import io.swagger.v3.oas.annotations.Operation
import io.swagger.v3.oas.annotations.media.Content
import io.swagger.v3.oas.annotations.media.Schema
import io.swagger.v3.oas.annotations.responses.ApiResponse
import me.nicolas.stravastats.api.dto.ErrorResponseMessageDto
import me.nicolas.stravastats.api.dto.SegmentClimbAttemptDto
import me.nicolas.stravastats.api.dto.SegmentClimbTargetSummaryDto
import me.nicolas.stravastats.api.dto.SegmentSummaryDto
import me.nicolas.stravastats.api.dto.toDto
import me.nicolas.stravastats.domain.services.IStatisticsService
import org.springframework.data.rest.webmvc.ResourceNotFoundException
import org.springframework.http.MediaType
import org.springframework.web.bind.annotation.GetMapping
import org.springframework.web.bind.annotation.PathVariable
import org.springframework.web.bind.annotation.RequestMapping
import org.springframework.web.bind.annotation.RequestParam
import org.springframework.web.bind.annotation.RestController
import java.time.LocalDate

@RestController
@RequestMapping("/segments", produces = [MediaType.APPLICATION_JSON_VALUE])
@Schema(description = "Segments analysis controller", name = "SegmentsController")
class SegmentsController(
    private val statisticsService: IStatisticsService,
) {

    @Operation(
        description = "List repeated segments/climbs with summary statistics",
        summary = "List segment analysis targets",
        responses = [
            ApiResponse(
                responseCode = "200",
                description = "Segment list found",
                content = [Content(
                    mediaType = MediaType.APPLICATION_JSON_VALUE,
                    schema = Schema(implementation = SegmentClimbTargetSummaryDto::class)
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
    @GetMapping
    fun getSegments(
        @RequestParam(required = true) activityType: String,
        @RequestParam(required = false) year: Int?,
        @RequestParam(required = false) metric: String?,
        @RequestParam(required = false) query: String?,
        @RequestParam(required = false) from: String?,
        @RequestParam(required = false) to: String?,
    ): List<SegmentClimbTargetSummaryDto> {
        val activityTypes = activityType.convertToActivityTypeSet()
        val fromDate = validateDate(from, "from")
        val toDate = validateDate(to, "to")

        return statisticsService
            .listSegments(activityTypes, year, metric, query, fromDate, toDate)
            .map { segment -> segment.toDto() }
    }

    @GetMapping("/{segmentId}/efforts")
    fun getSegmentEfforts(
        @PathVariable segmentId: Long,
        @RequestParam(required = true) activityType: String,
        @RequestParam(required = false) year: Int?,
        @RequestParam(required = false) metric: String?,
        @RequestParam(required = false) from: String?,
        @RequestParam(required = false) to: String?,
    ): List<SegmentClimbAttemptDto> {
        val activityTypes = activityType.convertToActivityTypeSet()
        val fromDate = validateDate(from, "from")
        val toDate = validateDate(to, "to")

        return statisticsService
            .getSegmentEfforts(activityTypes, year, metric, segmentId, fromDate, toDate)
            .map { effort -> effort.toDto() }
    }

    @GetMapping("/{segmentId}/summary")
    fun getSegmentSummary(
        @PathVariable segmentId: Long,
        @RequestParam(required = true) activityType: String,
        @RequestParam(required = false) year: Int?,
        @RequestParam(required = false) metric: String?,
        @RequestParam(required = false) from: String?,
        @RequestParam(required = false) to: String?,
    ): SegmentSummaryDto {
        val activityTypes = activityType.convertToActivityTypeSet()
        val fromDate = validateDate(from, "from")
        val toDate = validateDate(to, "to")

        val summary = statisticsService.getSegmentSummary(activityTypes, year, metric, segmentId, fromDate, toDate)
            ?: throw ResourceNotFoundException("No segment efforts found for segmentId=$segmentId")

        return SegmentSummaryDto(
            metric = summary.metric,
            segment = summary.segment.toDto(),
            personalRecord = summary.personalRecord?.toDto(),
            topEfforts = summary.topEfforts.map { effort -> effort.toDto() }
        )
    }

    private fun validateDate(value: String?, paramName: String): String? {
        if (value.isNullOrBlank()) {
            return null
        }
        val normalized = value.trim()
        runCatching { LocalDate.parse(normalized) }.getOrElse {
            throw IllegalArgumentException("Invalid $paramName date '$value' (expected YYYY-MM-DD)")
        }
        return normalized
    }
}
