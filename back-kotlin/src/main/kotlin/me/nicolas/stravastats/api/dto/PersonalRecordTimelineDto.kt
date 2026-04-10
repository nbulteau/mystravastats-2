package me.nicolas.stravastats.api.dto

import io.swagger.v3.oas.annotations.media.Schema
import me.nicolas.stravastats.domain.business.PersonalRecordTimelineEntry

@Schema(description = "Personal record timeline event", name = "PersonalRecordTimelineEvent")
data class PersonalRecordTimelineDto(
    @param:Schema(description = "Metric key")
    val metricKey: String,
    @param:Schema(description = "Metric label")
    val metricLabel: String,
    @param:Schema(description = "Date of the PR event")
    val activityDate: String,
    @param:Schema(description = "PR value reached on this date")
    val value: String,
    @param:Schema(description = "Previous PR value before this event")
    val previousValue: String? = null,
    @param:Schema(description = "Improvement compared with previous PR")
    val improvement: String? = null,
    @param:Schema(description = "Activity that set the PR")
    val activity: ActivityShortDto,
)

fun PersonalRecordTimelineEntry.toDto() = PersonalRecordTimelineDto(
    metricKey = metricKey,
    metricLabel = metricLabel,
    activityDate = activityDate,
    value = value,
    previousValue = previousValue,
    improvement = improvement,
    activity = activity.toDto()
)
