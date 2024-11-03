package me.nicolas.stravastats.api.controllers

import io.swagger.v3.oas.annotations.Operation
import io.swagger.v3.oas.annotations.media.ArraySchema
import io.swagger.v3.oas.annotations.media.Content
import io.swagger.v3.oas.annotations.media.Schema
import io.swagger.v3.oas.annotations.responses.ApiResponse
import io.swagger.v3.oas.annotations.tags.Tag
import me.nicolas.stravastats.api.dto.BadgeCheckResultDto
import me.nicolas.stravastats.api.dto.ErrorResponseMessageDto
import me.nicolas.stravastats.api.dto.toDto
import me.nicolas.stravastats.domain.business.ActivityType
import me.nicolas.stravastats.domain.business.badges.BadgeSetEnum
import me.nicolas.stravastats.domain.services.IBadgesService
import org.springframework.http.MediaType
import org.springframework.web.bind.annotation.GetMapping
import org.springframework.web.bind.annotation.RequestMapping
import org.springframework.web.bind.annotation.RequestParam
import org.springframework.web.bind.annotation.RestController

@RestController
@RequestMapping("/badges", produces = [MediaType.APPLICATION_JSON_VALUE])
@Schema(description = "Badges controller", name = "BadgesController")
@Tag(name = "Badges", description = "Badges endpoints")
class BadgesController(
    private val badgesService: IBadgesService,
) {
    @Operation(
        description = "Get the badges for a specific stravaActivity type and year",
        summary = "Get the badges for a specific stravaActivity type and year",
        responses = [ApiResponse(
            responseCode = "200", description = "Badges found",
            content = [Content(
                mediaType = MediaType.APPLICATION_JSON_VALUE,
                array = ArraySchema(schema = Schema(implementation = BadgeCheckResultDto::class))
            )]
        ), ApiResponse(
            responseCode = "404", description = "Badges not found",
            content = [Content(
                mediaType = MediaType.APPLICATION_JSON_VALUE,
                schema = Schema(implementation = ErrorResponseMessageDto::class)
            )]
        )],
    )
    @GetMapping
    fun getBadges(
        @RequestParam(required = true) activityType: ActivityType,
        @RequestParam(required = false) year: Int? = null,
        @RequestParam(required = false) badgeSet: BadgeSetEnum? = null,
    ): List<BadgeCheckResultDto> {

        return when (badgeSet) {
            BadgeSetEnum.GENERAL -> badgesService.getGeneralBadges(activityType, year)
            BadgeSetEnum.FAMOUS -> badgesService.getFamousBadges(activityType, year)
            else -> badgesService.getGeneralBadges(activityType, year) + badgesService.getFamousBadges(activityType, year)
        }.map { badgeCheckResult ->  badgeCheckResult.toDto(activityType) }
    }
}

