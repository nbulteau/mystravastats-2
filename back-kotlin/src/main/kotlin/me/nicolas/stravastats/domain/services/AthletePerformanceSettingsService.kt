package me.nicolas.stravastats.domain.services

import me.nicolas.stravastats.domain.business.ActivityEffort
import me.nicolas.stravastats.domain.business.ActivityType
import me.nicolas.stravastats.domain.business.AthletePerformanceSettings
import me.nicolas.stravastats.domain.business.FtpEstimate
import me.nicolas.stravastats.domain.business.normalize
import me.nicolas.stravastats.domain.business.rideActivities
import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.services.activityproviders.IActivityProvider
import me.nicolas.stravastats.domain.services.statistics.calculateBestPowerForTime
import org.springframework.stereotype.Service
import java.time.LocalDate
import kotlin.math.roundToInt

interface IAthletePerformanceSettingsService {
    fun getSettings(): AthletePerformanceSettings
    fun updateSettings(settings: AthletePerformanceSettings): AthletePerformanceSettings
    fun estimateFtp(activityTypes: Set<ActivityType> = defaultFtpEstimateActivityTypes, windowDays: Int = 180): FtpEstimate
}

val defaultFtpEstimateActivityTypes: Set<ActivityType> = rideActivities

@Service
internal class AthletePerformanceSettingsService(
    activityProvider: IActivityProvider,
) : IAthletePerformanceSettingsService, AbstractStravaService(activityProvider) {

    override fun getSettings(): AthletePerformanceSettings {
        return activityProvider.getPerformanceSettings().normalize()
    }

    override fun updateSettings(settings: AthletePerformanceSettings): AthletePerformanceSettings {
        return activityProvider.savePerformanceSettings(settings.normalize())
    }

    override fun estimateFtp(activityTypes: Set<ActivityType>, windowDays: Int): FtpEstimate {
        val normalizedWindowDays = normalizeFtpEstimateWindowDays(windowDays)
        val selectedActivityTypes = activityTypes.ifEmpty { defaultFtpEstimateActivityTypes }
        val activities = activityProvider
            .getActivitiesByActivityTypeAndYear(selectedActivityTypes)
            .withoutDataQualityExcludedStats(activityProvider)

        if (activities.isEmpty()) {
            return unavailableFtpEstimate(normalizedWindowDays, 0, "No activities available")
        }

        val referenceDate = latestActivityDate(activities)
        val recentCutoff = referenceDate.minusDays(normalizedWindowDays.toLong())
        val candidateGroups = listOf(
            FtpEstimateCandidateGroup(
                activities = activities.filterFtpEstimateActivities(recentCutoff, deviceOnly = true),
                source = "Power meter, last $normalizedWindowDays days",
                sourceKind = "power-meter",
                recent = true,
                deviceWatts = true,
            ),
            FtpEstimateCandidateGroup(
                activities = activities.filterFtpEstimateActivities(cutoff = null, deviceOnly = true),
                source = "Power meter, all time",
                sourceKind = "power-meter",
                deviceWatts = true,
            ),
            FtpEstimateCandidateGroup(
                activities = activities.filterFtpEstimateActivities(recentCutoff, deviceOnly = false),
                source = "All power data, last $normalizedWindowDays days",
                sourceKind = "all-power",
                recent = true,
            ),
            FtpEstimateCandidateGroup(
                activities = activities.filterFtpEstimateActivities(cutoff = null, deviceOnly = false),
                source = "All power data, all time",
                sourceKind = "all-power",
            ),
        )

        for (group in candidateGroups) {
            val estimate = estimateFtpFromGroup(group, normalizedWindowDays)
            if (estimate != null) {
                return estimate
            }
        }

        return unavailableFtpEstimate(normalizedWindowDays, activities.size, "No usable power stream available")
    }

    private fun estimateFtpFromGroup(group: FtpEstimateCandidateGroup, windowDays: Int): FtpEstimate? {
        if (group.activities.isEmpty()) {
            return null
        }

        val best60MinuteEffort = bestPowerEffortByAveragePower(group.activities, seconds = 60 * 60)
        if (best60MinuteEffort?.averagePower != null) {
            val bestPower = best60MinuteEffort.averagePower
            return ftpEstimateFromEffort(
                group = group,
                effort = best60MinuteEffort,
                ftp = bestPower,
                bestPower = bestPower,
                multiplier = 1.0,
                method = "best-60min",
                methodLabel = "Best 60 min power",
                windowDays = windowDays,
            )
        }

        val best20MinuteEffort = bestPowerEffortByAveragePower(group.activities, seconds = 20 * 60)
        if (best20MinuteEffort?.averagePower != null) {
            val bestPower = best20MinuteEffort.averagePower
            return ftpEstimateFromEffort(
                group = group,
                effort = best20MinuteEffort,
                ftp = (bestPower * 0.95).roundToInt(),
                bestPower = bestPower,
                multiplier = 0.95,
                method = "95-percent-20min",
                methodLabel = "95% of best 20 min power",
                windowDays = windowDays,
            )
        }

        return null
    }

    private fun bestPowerEffortByAveragePower(activities: List<StravaActivity>, seconds: Int): ActivityEffort? {
        return activities
            .mapNotNull { activity -> activity.calculateBestPowerForTime(seconds) }
            .filter { effort -> effort.averagePower != null }
            .maxByOrNull { effort -> effort.averagePower ?: 0 }
    }

    private fun ftpEstimateFromEffort(
        group: FtpEstimateCandidateGroup,
        effort: ActivityEffort,
        ftp: Int,
        bestPower: Int,
        multiplier: Double,
        method: String,
        methodLabel: String,
        windowDays: Int,
    ): FtpEstimate {
        return FtpEstimate(
            available = ftp > 0,
            ftp = ftp,
            method = method,
            methodLabel = methodLabel,
            bestPower = bestPower,
            multiplier = multiplier,
            basedOnSeconds = effort.seconds,
            confidence = ftpEstimateConfidence(group, method),
            source = group.source,
            sourceKind = group.sourceKind,
            activityId = effort.activityShort.id,
            activityName = effort.activityShort.name,
            activityType = effort.activityShort.type.name,
            activityDate = ftpEstimateActivityDate(group.activities, effort.activityShort.id),
            windowDays = windowDays,
            activityCount = group.activities.size,
        )
    }

    private fun ftpEstimateConfidence(group: FtpEstimateCandidateGroup, method: String): String {
        if (group.recent && group.deviceWatts && method == "best-60min") {
            return "high"
        }
        if (group.deviceWatts && (group.recent || method == "best-60min")) {
            return "medium"
        }
        return "low"
    }

    private fun List<StravaActivity>.filterFtpEstimateActivities(cutoff: LocalDate?, deviceOnly: Boolean): List<StravaActivity> {
        return filter { activity ->
            (!deviceOnly || activity.deviceWatts) &&
                (cutoff == null || activityDate(activity)?.let { date -> !date.isBefore(cutoff) } == true)
        }
    }

    private fun latestActivityDate(activities: List<StravaActivity>): LocalDate {
        return activities
            .mapNotNull { activity -> activityDate(activity) }
            .maxOrNull()
            ?: LocalDate.now()
    }

    private fun activityDate(activity: StravaActivity): LocalDate? {
        return extractSortableDay(activity.startDateLocal)?.let(LocalDate::parse)
            ?: extractSortableDay(activity.startDate)?.let(LocalDate::parse)
    }

    private fun ftpEstimateActivityDate(activities: List<StravaActivity>, activityId: Long): String {
        val activity = activities.firstOrNull { candidate -> candidate.id == activityId } ?: return ""
        return extractSortableDay(activity.startDateLocal)
            ?: extractSortableDay(activity.startDate)
            ?: ""
    }

    private fun normalizeFtpEstimateWindowDays(windowDays: Int): Int =
        when {
            windowDays <= 0 -> 180
            windowDays < 30 -> 30
            windowDays > 730 -> 730
            else -> windowDays
        }

    private fun unavailableFtpEstimate(windowDays: Int, activityCount: Int, source: String): FtpEstimate {
        return FtpEstimate(
            available = false,
            confidence = "unavailable",
            source = source,
            sourceKind = "none",
            windowDays = windowDays,
            activityCount = activityCount,
        )
    }
}

private data class FtpEstimateCandidateGroup(
    val activities: List<StravaActivity>,
    val source: String,
    val sourceKind: String,
    val recent: Boolean = false,
    val deviceWatts: Boolean = false,
)
