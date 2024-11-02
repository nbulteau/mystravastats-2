package me.nicolas.stravastats.api.dto

import io.swagger.v3.oas.annotations.media.Schema

/**
 * Cumulative data per year DTO.
 */
@Schema(description = "Cumulative data per year.", name = "CumulativeDataPerYear")
data class CumulativeDataPerYearDto (
    @Schema(description = "Distance by year and day (Map<Year, Map<Day, Value>>).", name = "Distance")
    val distance: Map<String, Map<String, Double>>,
    @Schema(description = "Elevation by year and day (Map<Year, Map<Day, Value>>).", name = "Elevation")
    val elevation: Map<String, Map<String, Int>>
)