package me.nicolas.stravastats.api.dto

import io.swagger.v3.oas.annotations.media.Schema
import me.nicolas.stravastats.domain.business.AnnualGoalTargets
import me.nicolas.stravastats.domain.business.AnnualGoals

@Schema(description = "Annual goal targets", name = "AnnualGoalTargets")
data class AnnualGoalTargetsDto(
    val distanceKm: Double? = null,
    val elevationMeters: Int? = null,
    val movingTimeSeconds: Int? = null,
    val activities: Int? = null,
    val activeDays: Int? = null,
    val eddington: Int? = null,
)

@Schema(description = "Annual goal progress", name = "AnnualGoalProgress")
data class AnnualGoalProgressDto(
    val metric: String,
    val label: String,
    val unit: String,
    val current: Double,
    val target: Double,
    val progressPercent: Double,
    val expectedProgressPercent: Double,
    val projectedEndOfYear: Double,
    val requiredPace: Double,
    val requiredPaceUnit: String,
    val requiredWeeklyPace: Double,
    val last30Days: Double,
    val last30DaysWeeklyPace: Double,
    val weeklyPaceGap: Double,
    val suggestedTarget: Double?,
    val monthly: List<AnnualGoalMonthDto>,
    val status: String,
)

@Schema(description = "Monthly annual goal progress", name = "AnnualGoalMonth")
data class AnnualGoalMonthDto(
    val month: Int,
    val value: Double,
    val cumulative: Double,
    val expectedCumulative: Double,
)

@Schema(description = "Annual goals and projections", name = "AnnualGoals")
data class AnnualGoalsDto(
    val year: Int,
    val activityTypeKey: String,
    val targets: AnnualGoalTargetsDto,
    val progress: List<AnnualGoalProgressDto>,
)

fun AnnualGoals.toDto(): AnnualGoalsDto {
    return AnnualGoalsDto(
        year = year,
        activityTypeKey = activityTypeKey,
        targets = targets.toDto(),
        progress = progress.map { item ->
            AnnualGoalProgressDto(
                metric = item.metric.name,
                label = item.label,
                unit = item.unit,
                current = item.current,
                target = item.target,
                progressPercent = item.progressPercent,
                expectedProgressPercent = item.expectedProgressPercent,
                projectedEndOfYear = item.projectedEndOfYear,
                requiredPace = item.requiredPace,
                requiredPaceUnit = item.requiredPaceUnit,
                requiredWeeklyPace = item.requiredWeeklyPace,
                last30Days = item.last30Days,
                last30DaysWeeklyPace = item.last30DaysWeeklyPace,
                weeklyPaceGap = item.weeklyPaceGap,
                suggestedTarget = item.suggestedTarget,
                monthly = item.monthly.map { month ->
                    AnnualGoalMonthDto(
                        month = month.month,
                        value = month.value,
                        cumulative = month.cumulative,
                        expectedCumulative = month.expectedCumulative,
                    )
                },
                status = item.status.name,
            )
        },
    )
}

fun AnnualGoalTargets.toDto(): AnnualGoalTargetsDto {
    return AnnualGoalTargetsDto(
        distanceKm = distanceKm,
        elevationMeters = elevationMeters,
        movingTimeSeconds = movingTimeSeconds,
        activities = activities,
        activeDays = activeDays,
        eddington = eddington,
    )
}

fun AnnualGoalTargetsDto.toDomain(): AnnualGoalTargets {
    return AnnualGoalTargets(
        distanceKm = distanceKm,
        elevationMeters = elevationMeters,
        movingTimeSeconds = movingTimeSeconds,
        activities = activities,
        activeDays = activeDays,
        eddington = eddington,
    )
}
