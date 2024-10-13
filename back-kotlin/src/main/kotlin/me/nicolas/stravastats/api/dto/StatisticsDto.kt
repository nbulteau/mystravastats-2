package me.nicolas.stravastats.api.dto

import com.fasterxml.jackson.annotation.JsonInclude
import io.swagger.v3.oas.annotations.media.Schema
import me.nicolas.stravastats.domain.services.statistics.ActivityStatistic
import me.nicolas.stravastats.domain.services.statistics.Statistic

@Schema(description = "Statistics object", name = "Statistics")
@JsonInclude(JsonInclude.Include.NON_NULL)
data class StatisticsDto(
    @Schema(description = "Label of the statistic")
    val label: String,
    @Schema(description = "Value of the statistic")
    val value: String,
    @Schema(description = "StravaActivity related to the statistic")
    val stravaActivity: ActivityDto? = null,
)

fun Statistic.toDto(): StatisticsDto {
    return when (this) {
        is ActivityStatistic -> StatisticsDto(
            label = name,
            value = value,
            stravaActivity = stravaActivity?.toDto()
        )

        else -> StatisticsDto(name, value)
    }
}



