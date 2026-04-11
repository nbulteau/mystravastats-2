package me.nicolas.stravastats.api.controllers

import io.swagger.v3.oas.annotations.Operation
import io.swagger.v3.oas.annotations.media.Content
import io.swagger.v3.oas.annotations.media.Schema
import io.swagger.v3.oas.annotations.responses.ApiResponse
import me.nicolas.stravastats.api.dto.AthleteDto
import me.nicolas.stravastats.api.dto.ErrorResponseMessageDto
import me.nicolas.stravastats.api.dto.HeartRateZoneSettingsDto
import me.nicolas.stravastats.api.dto.toDto
import me.nicolas.stravastats.api.dto.toDomain
import me.nicolas.stravastats.domain.services.activityproviders.IActivityProvider
import me.nicolas.stravastats.domain.services.IHeartRateZoneService
import org.springframework.http.MediaType
import org.springframework.web.bind.annotation.GetMapping
import org.springframework.web.bind.annotation.PutMapping
import org.springframework.web.bind.annotation.RequestBody
import org.springframework.web.bind.annotation.RequestMapping
import org.springframework.web.bind.annotation.RestController


@RestController
@RequestMapping("/athletes")
@Schema(description = "User controller", name = "UserController")
class AthleteController(
    private val stravaProxy: IActivityProvider,
    private val heartRateZoneService: IHeartRateZoneService,
) {
    @Operation(
        description = "Get the authenticated user",
        summary = "Get the authenticated user",
        responses = [
            ApiResponse(
                responseCode = "200",
                description = "StravaAthlete found",
                content = [Content(
                    mediaType = MediaType.APPLICATION_JSON_VALUE,
                    schema = Schema(implementation = AthleteDto::class)
                )]
            ),
            ApiResponse(
                responseCode = "404",
                description = "StravaAthlete not found",
                content = [Content(
                    mediaType = MediaType.APPLICATION_JSON_VALUE,
                    schema = Schema(implementation = ErrorResponseMessageDto::class)
                )]
            )
        ],
    )
    @GetMapping("/me")
    fun getAthlete(): AthleteDto {
        return stravaProxy.athlete().toDto()
    }

    @GetMapping("/me/heart-rate-zones")
    fun getHeartRateZoneSettings(): HeartRateZoneSettingsDto {
        return heartRateZoneService.getSettings().toDto()
    }

    @PutMapping("/me/heart-rate-zones")
    fun updateHeartRateZoneSettings(@RequestBody settings: HeartRateZoneSettingsDto): HeartRateZoneSettingsDto {
        return heartRateZoneService.updateSettings(settings.toDomain()).toDto()
    }
}

