package me.nicolas.stravastats.api.dto

import com.fasterxml.jackson.annotation.JsonIgnoreProperties
import io.swagger.v3.oas.annotations.media.Schema

/**
 * Class that manages error response message for exception handling.
 */
@JsonIgnoreProperties(ignoreUnknown = true)
@Schema(description = "The error message description.", name = "ErrorResponseMessage")
data class ErrorResponseMessageDto(
    @Schema(description = "A short localized string that describes the error.", required = true)
    val message: String,
    @Schema(
        description = "A long localized error description if needed. It can contain precise information about which parameter is missing, or what are the identifier acceptable values.",
        required = true
    )
    val description: String,
    @Schema(
        description = "An integer coding the error type. This is given to caller so he can translate them if required.",
        required = true
    )
    val code: Int,
)