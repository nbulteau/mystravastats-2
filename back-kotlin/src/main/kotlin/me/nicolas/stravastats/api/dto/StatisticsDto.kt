package me.nicolas.stravastats.api.dto

import com.fasterxml.jackson.annotation.JsonInclude
import io.swagger.v3.oas.annotations.media.Schema
import me.nicolas.stravastats.domain.business.ActivityShort
import me.nicolas.stravastats.domain.services.statistics.ActivityStatistic
import me.nicolas.stravastats.domain.services.statistics.Statistic

@Schema(description = "Statistics object", name = "Statistics")
@JsonInclude(JsonInclude.Include.NON_NULL)
data class StatisticsDto(
    @param:Schema(description = "Label of the statistic")
    val label: String,
    @param:Schema(description = "Value of the statistic")
    val value: String,
    @param:Schema(description = "StravaActivity related to the statistic")
    val activity: ActivityShortDto? = null,
)

fun Statistic.toDto(): StatisticsDto {
    return when (this) {
        is ActivityStatistic -> StatisticsDto(
            label = name,
            value = value,
            activity = activity?.toDto()
        )

        else -> StatisticsDto(name, value)
    }
}

data class ActivityShortDto(
    val id: Long,
    val name: String,
    val type: String,
)

fun ActivityShort.toDto(): ActivityShortDto {
    return ActivityShortDto(
        id = id,
        name = name,
        type = type.name
    )
}




