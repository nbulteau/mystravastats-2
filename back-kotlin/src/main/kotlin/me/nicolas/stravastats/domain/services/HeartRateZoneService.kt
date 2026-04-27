package me.nicolas.stravastats.domain.services

import me.nicolas.stravastats.domain.business.ActivityShort
import me.nicolas.stravastats.domain.business.ActivityType
import me.nicolas.stravastats.domain.business.HeartRateZoneActivitySummary
import me.nicolas.stravastats.domain.business.HeartRateZoneAnalysis
import me.nicolas.stravastats.domain.business.HeartRateZoneDistribution
import me.nicolas.stravastats.domain.business.HeartRateZoneMethod
import me.nicolas.stravastats.domain.business.HeartRateZonePeriodSummary
import me.nicolas.stravastats.domain.business.HeartRateZoneSettings
import me.nicolas.stravastats.domain.business.HeartRateZoneSource
import me.nicolas.stravastats.domain.business.ResolvedHeartRateZoneSettings
import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.services.activityproviders.IActivityProvider
import org.springframework.stereotype.Service
import kotlin.math.min
import kotlin.math.roundToInt

interface IHeartRateZoneService {
    fun getSettings(): HeartRateZoneSettings
    fun updateSettings(settings: HeartRateZoneSettings): HeartRateZoneSettings
    fun getAnalysis(activityTypes: Set<ActivityType>, year: Int?): HeartRateZoneAnalysis
}

@Service
internal class HeartRateZoneService(
    activityProvider: IActivityProvider,
) : IHeartRateZoneService, AbstractStravaService(activityProvider) {

    private val zoneCodes = listOf("Z1", "Z2", "Z3", "Z4", "Z5")
    private val zoneLabels = listOf("Recovery", "Endurance", "Tempo", "Threshold", "VO2 Max")
    private val easyZoneIndexes = setOf(0, 1)
    private val hardZoneIndexes = setOf(3, 4)

    override fun getSettings(): HeartRateZoneSettings {
        return activityProvider.getHeartRateZoneSettings()
    }

    override fun updateSettings(settings: HeartRateZoneSettings): HeartRateZoneSettings {
        return activityProvider.saveHeartRateZoneSettings(settings.normalize())
    }

    override fun getAnalysis(activityTypes: Set<ActivityType>, year: Int?): HeartRateZoneAnalysis {
        val settings = activityProvider.getHeartRateZoneSettings().normalize()
        val activities = activityProvider.getActivitiesByActivityTypeAndYear(activityTypes, year)
            .withoutDataQualityExcludedStats(activityProvider)
            .sortedBy { activity -> activity.startDateLocal }
        val resolvedSettings = resolveSettings(settings, activities) ?: return emptyAnalysis(settings, null)

        val activitySummaries = activities.mapNotNull { activity ->
            buildActivitySummary(activity, resolvedSettings)
        }

        if (activitySummaries.isEmpty()) {
            return emptyAnalysis(settings, resolvedSettings)
        }

        val globalDistribution = aggregateZoneDistribution(activitySummaries.flatMap { summary -> summary.zones })
        val totalTrackedSeconds = globalDistribution.sumOf { distribution -> distribution.seconds }
        val easySeconds = globalDistribution
            .filterIndexed { index, _ -> easyZoneIndexes.contains(index) }
            .sumOf { distribution -> distribution.seconds }
        val hardSeconds = globalDistribution
            .filterIndexed { index, _ -> hardZoneIndexes.contains(index) }
            .sumOf { distribution -> distribution.seconds }

        return HeartRateZoneAnalysis(
            settings = settings,
            resolvedSettings = resolvedSettings,
            hasHeartRateData = true,
            totalTrackedSeconds = totalTrackedSeconds,
            easyHardRatio = calculateEasyHardRatio(easySeconds, hardSeconds),
            zones = toDistributions(globalDistribution.map { distribution -> distribution.seconds }, totalTrackedSeconds),
            activities = activitySummaries,
            byMonth = summarizeByPeriod(activitySummaries) { summary -> summary.activityDate.take(7) },
            byYear = summarizeByPeriod(activitySummaries) { summary -> summary.activityDate.take(4) },
        )
    }

    private fun summarizeByPeriod(
        activitySummaries: List<HeartRateZoneActivitySummary>,
        keySelector: (HeartRateZoneActivitySummary) -> String,
    ): List<HeartRateZonePeriodSummary> {
        return activitySummaries
            .groupBy(keySelector)
            .toSortedMap()
            .map { (period, summaries) ->
                val zoneTotals = IntArray(zoneCodes.size)
                var totalTrackedSeconds = 0
                var easySeconds = 0
                var hardSeconds = 0

                summaries.forEach { summary ->
                    summary.zones.forEachIndexed { index, zone ->
                        zoneTotals[index] += zone.seconds
                    }
                    totalTrackedSeconds += summary.totalTrackedSeconds
                    easySeconds += summary.easySeconds
                    hardSeconds += summary.hardSeconds
                }

                HeartRateZonePeriodSummary(
                    period = period,
                    totalTrackedSeconds = totalTrackedSeconds,
                    easySeconds = easySeconds,
                    hardSeconds = hardSeconds,
                    easyHardRatio = calculateEasyHardRatio(easySeconds, hardSeconds),
                    zones = toDistributions(zoneTotals.toList(), totalTrackedSeconds),
                )
            }
    }

    private fun buildActivitySummary(
        activity: StravaActivity,
        resolvedSettings: ResolvedHeartRateZoneSettings,
    ): HeartRateZoneActivitySummary? {
        val stream = activity.stream ?: return null
        val heartrate = stream.heartrate?.data ?: return null
        val time = stream.time.data
        val sampleSize = min(heartrate.size, time.size)
        if (sampleSize < 2) {
            return null
        }

        val zoneTotals = IntArray(zoneCodes.size)
        var trackedSeconds = 0

        for (index in 0 until sampleSize - 1) {
            val hr = heartrate[index]
            val delta = time[index + 1] - time[index]
            if (hr <= 0 || delta <= 0) {
                continue
            }

            val zoneIndex = resolveZoneIndex(hr, resolvedSettings)
            zoneTotals[zoneIndex] += delta
            trackedSeconds += delta
        }

        if (trackedSeconds <= 0) {
            return null
        }

        val easySeconds = easyZoneIndexes.sumOf { zoneTotals[it] }
        val hardSeconds = hardZoneIndexes.sumOf { zoneTotals[it] }

        return HeartRateZoneActivitySummary(
            activity = ActivityShort(activity.id, activity.name, activity.type),
            activityDate = activity.startDateLocal,
            totalTrackedSeconds = trackedSeconds,
            easySeconds = easySeconds,
            hardSeconds = hardSeconds,
            easyHardRatio = calculateEasyHardRatio(easySeconds, hardSeconds),
            zones = toDistributions(zoneTotals.toList(), trackedSeconds),
        )
    }

    private fun resolveSettings(
        settings: HeartRateZoneSettings,
        activities: List<StravaActivity>,
    ): ResolvedHeartRateZoneSettings? {
        val maxHr = settings.maxHr
        val thresholdHr = settings.thresholdHr
        val reserveHr = settings.reserveHr

        if (thresholdHr != null) {
            val derivedMax = if (maxHr == null) deriveMaxHr(activities) else null
            val resolvedMax = maxHr ?: derivedMax ?: thresholdHr
            val source = if (maxHr == null && derivedMax != null) {
                HeartRateZoneSource.DERIVED_FROM_DATA
            } else {
                HeartRateZoneSource.ATHLETE_SETTINGS
            }
            return ResolvedHeartRateZoneSettings(
                maxHr = resolvedMax,
                thresholdHr = thresholdHr,
                reserveHr = reserveHr,
                method = HeartRateZoneMethod.THRESHOLD,
                source = source,
            )
        }

        if (maxHr != null && reserveHr != null && reserveHr in 1 until maxHr) {
            return ResolvedHeartRateZoneSettings(
                maxHr = maxHr,
                thresholdHr = null,
                reserveHr = reserveHr,
                method = HeartRateZoneMethod.RESERVE,
                source = HeartRateZoneSource.ATHLETE_SETTINGS,
            )
        }

        if (maxHr != null) {
            return ResolvedHeartRateZoneSettings(
                maxHr = maxHr,
                thresholdHr = null,
                reserveHr = reserveHr,
                method = HeartRateZoneMethod.MAX,
                source = HeartRateZoneSource.ATHLETE_SETTINGS,
            )
        }

        val derivedMax = deriveMaxHr(activities) ?: return null
        return ResolvedHeartRateZoneSettings(
            maxHr = derivedMax,
            thresholdHr = null,
            reserveHr = null,
            method = HeartRateZoneMethod.MAX,
            source = HeartRateZoneSource.DERIVED_FROM_DATA,
        )
    }

    private fun deriveMaxHr(activities: List<StravaActivity>): Int? {
        return activities.maxOfOrNull { activity -> activity.maxHeartrate }?.takeIf { hr -> hr > 0 }
    }

    private fun resolveZoneIndex(hr: Int, settings: ResolvedHeartRateZoneSettings): Int {
        val z1Upper: Double
        val z2Upper: Double
        val z3Upper: Double
        val z4Upper: Double

        when (settings.method) {
            HeartRateZoneMethod.THRESHOLD -> {
                val threshold = settings.thresholdHr?.toDouble() ?: return 0
                z1Upper = threshold * 0.81
                z2Upper = threshold * 0.89
                z3Upper = threshold * 0.93
                z4Upper = threshold * 0.99
            }

            HeartRateZoneMethod.RESERVE -> {
                val reserve = settings.reserveHr?.toDouble() ?: return 0
                val resting = (settings.maxHr.toDouble() - reserve).coerceAtLeast(35.0)
                z1Upper = resting + reserve * 0.60
                z2Upper = resting + reserve * 0.70
                z3Upper = resting + reserve * 0.80
                z4Upper = resting + reserve * 0.90
            }

            HeartRateZoneMethod.MAX -> {
                val max = settings.maxHr.toDouble()
                z1Upper = max * 0.60
                z2Upper = max * 0.70
                z3Upper = max * 0.80
                z4Upper = max * 0.90
            }
        }

        return when {
            hr <= z1Upper -> 0
            hr <= z2Upper -> 1
            hr <= z3Upper -> 2
            hr <= z4Upper -> 3
            else -> 4
        }
    }

    private fun calculateEasyHardRatio(easySeconds: Int, hardSeconds: Int): Double? {
        if (easySeconds <= 0 || hardSeconds <= 0) {
            return null
        }

        return (easySeconds.toDouble() / hardSeconds.toDouble() * 100.0).roundToInt() / 100.0
    }

    private fun aggregateZoneDistribution(distributions: List<HeartRateZoneDistribution>): List<HeartRateZoneDistribution> {
        if (distributions.isEmpty()) {
            return zoneCodes.mapIndexed { index, zone ->
                HeartRateZoneDistribution(zone, zoneLabels[index], 0, 0.0)
            }
        }

        val byZone = distributions.groupBy { distribution -> distribution.zone }
        return zoneCodes.mapIndexed { index, zone ->
            HeartRateZoneDistribution(
                zone = zone,
                label = zoneLabels[index],
                seconds = byZone[zone].orEmpty().sumOf { distribution -> distribution.seconds },
                percentage = 0.0,
            )
        }
    }

    private fun toDistributions(zoneSeconds: List<Int>, totalTrackedSeconds: Int): List<HeartRateZoneDistribution> {
        return zoneCodes.mapIndexed { index, zone ->
            val seconds = zoneSeconds.getOrElse(index) { 0 }
            val percentage = if (totalTrackedSeconds <= 0) {
                0.0
            } else {
                (seconds.toDouble() / totalTrackedSeconds.toDouble() * 10000.0).roundToInt() / 100.0
            }
            HeartRateZoneDistribution(
                zone = zone,
                label = zoneLabels[index],
                seconds = seconds,
                percentage = percentage,
            )
        }
    }

    private fun HeartRateZoneSettings.normalize(): HeartRateZoneSettings {
        val normalizedMax = maxHr?.takeIf { value -> value > 0 }
        val normalizedThreshold = thresholdHr?.takeIf { value -> value > 0 }
        val normalizedReserve = reserveHr?.takeIf { value -> value > 0 }

        return HeartRateZoneSettings(
            maxHr = normalizedMax,
            thresholdHr = normalizedThreshold,
            reserveHr = normalizedReserve,
        )
    }

    private fun emptyAnalysis(
        settings: HeartRateZoneSettings,
        resolvedSettings: ResolvedHeartRateZoneSettings?,
    ): HeartRateZoneAnalysis {
        return HeartRateZoneAnalysis(
            settings = settings,
            resolvedSettings = resolvedSettings,
            hasHeartRateData = false,
            totalTrackedSeconds = 0,
            easyHardRatio = null,
            zones = toDistributions(zoneCodes.map { 0 }, 0),
            activities = emptyList(),
            byMonth = emptyList(),
            byYear = emptyList(),
        )
    }
}
