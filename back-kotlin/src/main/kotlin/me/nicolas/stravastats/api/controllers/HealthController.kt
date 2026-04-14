package me.nicolas.stravastats.api.controllers

import io.swagger.v3.oas.annotations.Operation
import io.swagger.v3.oas.annotations.media.Content
import io.swagger.v3.oas.annotations.media.Schema
import io.swagger.v3.oas.annotations.responses.ApiResponse
import me.nicolas.stravastats.api.dto.ErrorResponseMessageDto
import me.nicolas.stravastats.domain.services.activityproviders.IActivityProvider
import org.springframework.http.MediaType
import org.springframework.web.bind.annotation.GetMapping
import org.springframework.web.bind.annotation.RequestMapping
import org.springframework.web.bind.annotation.RestController

@RestController
@RequestMapping("/health", produces = [MediaType.APPLICATION_JSON_VALUE])
class HealthController(
    private val activityProvider: IActivityProvider,
) {

    @Operation(
        summary = "Get cache health details",
        description = "Returns cache diagnostics including manifest, warmup status and best-effort cache details.",
        responses = [
            ApiResponse(
                responseCode = "200",
                description = "Cache diagnostics",
                content = [Content(mediaType = MediaType.APPLICATION_JSON_VALUE)]
            ),
            ApiResponse(
                responseCode = "500",
                description = "Unexpected error",
                content = [Content(
                    mediaType = MediaType.APPLICATION_JSON_VALUE,
                    schema = Schema(implementation = ErrorResponseMessageDto::class)
                )]
            )
        ]
    )
    @GetMapping("/details")
    fun getHealthDetails(): Map<String, Any?> {
        return activityProvider.getCacheDiagnostics()
    }
}
