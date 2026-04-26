package me.nicolas.stravastats.domain.services

import me.nicolas.stravastats.domain.business.ActivityShort
import me.nicolas.stravastats.domain.business.ActivityType
import me.nicolas.stravastats.domain.business.GearAnalysis
import me.nicolas.stravastats.domain.business.GearAnalysisCoverage
import me.nicolas.stravastats.domain.business.GearAnalysisItem
import me.nicolas.stravastats.domain.business.GearAnalysisPeriodPoint
import me.nicolas.stravastats.domain.business.GearAnalysisSummary
import me.nicolas.stravastats.domain.business.GearKind
import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.business.strava.StravaAthlete
import me.nicolas.stravastats.domain.services.activityproviders.IActivityProvider
import org.springframework.stereotype.Service
import kotlin.math.floor

private data class GearMetadata(
    val name: String,
    val kind: GearKind,
    val retired: Boolean,
    val primary: Boolean,
)

interface IGearAnalysisService {
    fun getGearAnalysis(activityTypes: Set<ActivityType>, year: Int?): GearAnalysis
}

@Service
internal class GearAnalysisService(
    activityProvider: IActivityProvider,
) : IGearAnalysisService, AbstractStravaService(activityProvider) {

    override fun getGearAnalysis(activityTypes: Set<ActivityType>, year: Int?): GearAnalysis {
        val activities = activityProvider.getActivitiesByActivityTypeAndYear(activityTypes, year)
        return buildGearAnalysis(activities, activityProvider.athlete())
    }

    private fun buildGearAnalysis(activities: List<StravaActivity>, athlete: StravaAthlete): GearAnalysis {
        val metadataById = buildGearMetadata(athlete)
        val itemsById = linkedMapOf<String, GearAccumulator>()
        val unassigned = GearSummaryAccumulator()
        var assignedActivities = 0
        var unassignedActivities = 0

        activities.forEach { activity ->
            val gearId = activity.gearId?.trim().orEmpty()
            if (gearId.isBlank()) {
                unassignedActivities++
                unassigned.add(activity)
                return@forEach
            }

            assignedActivities++
            val accumulator = itemsById.getOrPut(gearId) {
                val metadata = metadataById[gearId] ?: GearMetadata(
                    name = gearDisplayName(gearId, null),
                    kind = inferGearKind(gearId),
                    retired = false,
                    primary = false,
                )
                GearAccumulator(
                    id = gearId,
                    name = gearDisplayName(gearId, metadata),
                    kind = metadata.kind,
                    retired = metadata.retired,
                    primary = metadata.primary,
                )
            }
            accumulator.add(activity)
        }

        val items = itemsById.values
            .map { it.toItem() }
            .sortedWith(
                compareByDescending<GearAnalysisItem> { it.distance }
                    .thenBy { it.name.lowercase() }
            )

        return GearAnalysis(
            items = items,
            unassigned = unassigned.toSummary(),
            coverage = GearAnalysisCoverage(
                totalActivities = activities.size,
                assignedActivities = assignedActivities,
                unassignedActivities = unassignedActivities,
            )
        )
    }

    private fun buildGearMetadata(athlete: StravaAthlete): Map<String, GearMetadata> {
        val metadata = mutableMapOf<String, GearMetadata>()
        athlete.bikes.orEmpty().forEach { bike ->
            metadata[bike.id] = GearMetadata(
                name = bike.nickname?.takeIf { it.isNotBlank() } ?: bike.name,
                kind = GearKind.BIKE,
                retired = bike.retired ?: false,
                primary = bike.primary,
            )
        }
        athlete.shoes.orEmpty().forEach { shoe ->
            metadata[shoe.id] = GearMetadata(
                name = shoe.nickname?.takeIf { it.isNotBlank() } ?: shoe.name,
                kind = GearKind.SHOE,
                retired = shoe.retired ?: false,
                primary = shoe.primary,
            )
        }
        return metadata
    }

    private class GearAccumulator(
        private val id: String,
        private val name: String,
        private val kind: GearKind,
        private val retired: Boolean,
        private val primary: Boolean,
    ) {
        private val monthly = mutableMapOf<String, GearAnalysisPeriodPointAccumulator>()
        private var distance = 0.0
        private var movingTime = 0
        private var elevationGain = 0.0
        private var activities = 0
        private var firstUsed = ""
        private var lastUsed = ""
        private var longestDistance = 0.0
        private var biggestElevationGain = 0.0
        private var fastestSpeed = 0.0
        private var longestActivity: ActivityShort? = null
        private var biggestElevationActivity: ActivityShort? = null
        private var fastestActivity: ActivityShort? = null

        fun add(activity: StravaActivity) {
            distance += activity.distance.finiteOrZero()
            movingTime += activity.movingTime
            elevationGain += activity.totalElevationGain.finiteOrZero()
            activities++

            val date = activityDate(activity)
            if (date.isNotBlank()) {
                if (firstUsed.isBlank() || date < firstUsed) firstUsed = date
                if (lastUsed.isBlank() || date > lastUsed) lastUsed = date

                val month = activityMonth(date)
                if (month.isNotBlank()) {
                    monthly.getOrPut(month) { GearAnalysisPeriodPointAccumulator(month) }
                        .add(activity.distance.finiteOrZero())
                }
            }

            if (activity.distance > longestDistance || longestActivity == null) {
                longestDistance = activity.distance
                longestActivity = activity.toActivityShort()
            }
            if (activity.totalElevationGain > biggestElevationGain || biggestElevationActivity == null) {
                biggestElevationGain = activity.totalElevationGain
                biggestElevationActivity = activity.toActivityShort()
            }
            val speed = activitySpeed(activity)
            if (speed > fastestSpeed || fastestActivity == null) {
                fastestSpeed = speed
                fastestActivity = activity.toActivityShort()
            }
        }

        fun toItem(): GearAnalysisItem {
            return GearAnalysisItem(
                id = id,
                name = name,
                kind = kind,
                retired = retired,
                primary = primary,
                maintenanceStatus = gearMaintenance(kind, distance).first,
                maintenanceLabel = gearMaintenance(kind, distance).second,
                distance = distance.roundGearValue(),
                movingTime = movingTime,
                elevationGain = elevationGain.roundGearValue(),
                activities = activities,
                averageSpeed = if (movingTime > 0) (distance / movingTime).roundGearValue() else 0.0,
                firstUsed = firstUsed,
                lastUsed = lastUsed,
                longestActivity = longestActivity,
                biggestElevationActivity = biggestElevationActivity,
                fastestActivity = fastestActivity,
                monthlyDistance = monthly.values.map { it.toPoint() }.sortedBy { it.periodKey },
            )
        }
    }

    private class GearSummaryAccumulator {
        private var distance = 0.0
        private var movingTime = 0
        private var elevationGain = 0.0
        private var activities = 0

        fun add(activity: StravaActivity) {
            distance += activity.distance.finiteOrZero()
            movingTime += activity.movingTime
            elevationGain += activity.totalElevationGain.finiteOrZero()
            activities++
        }

        fun toSummary(): GearAnalysisSummary {
            return GearAnalysisSummary(
                distance = distance.roundGearValue(),
                movingTime = movingTime,
                elevationGain = elevationGain.roundGearValue(),
                activities = activities,
                averageSpeed = if (movingTime > 0) (distance / movingTime).roundGearValue() else 0.0,
            )
        }
    }

    private class GearAnalysisPeriodPointAccumulator(
        private val periodKey: String,
    ) {
        private var value = 0.0
        private var activityCount = 0

        fun add(distance: Double) {
            value += distance
            activityCount++
        }

        fun toPoint(): GearAnalysisPeriodPoint {
            return GearAnalysisPeriodPoint(
                periodKey = periodKey,
                value = value.roundGearValue(),
                activityCount = activityCount,
            )
        }
    }
}

private fun gearDisplayName(id: String, metadata: GearMetadata?): String {
    val name = metadata?.name?.trim().orEmpty()
    if (name.isNotBlank()) return name
    return when {
        id.startsWith("b") -> "Bike $id"
        id.startsWith("g") -> "Shoes $id"
        else -> "Gear $id"
    }
}

private fun inferGearKind(id: String): GearKind {
    return when {
        id.startsWith("b") -> GearKind.BIKE
        id.startsWith("g") -> GearKind.SHOE
        else -> GearKind.UNKNOWN
    }
}

private fun StravaActivity.toActivityShort(): ActivityShort {
    val resolvedType = if (commute) {
        ActivityType.Commute
    } else {
        ActivityType.entries.firstOrNull { it.name == sportType }
            ?: ActivityType.entries.firstOrNull { it.name == type }
            ?: ActivityType.Ride
    }
    return ActivityShort(id = id, name = name, type = resolvedType)
}

private fun activityDate(activity: StravaActivity): String {
    return activity.startDateLocal.ifBlank { activity.startDate }
}

private fun activityMonth(date: String): String {
    val trimmed = date.trim()
    return if (trimmed.length >= 7) trimmed.substring(0, 7) else ""
}

private fun activitySpeed(activity: StravaActivity): Double {
    if (activity.averageSpeed > 0) return activity.averageSpeed
    return if (activity.movingTime > 0) activity.distance / activity.movingTime else 0.0
}

private fun Double.finiteOrZero(): Double = if (isFinite()) this else 0.0

private fun Double.roundGearValue(): Double = floor(finiteOrZero() * 10.0 + 0.5) / 10.0

private fun gearMaintenance(kind: GearKind, distanceMeters: Double): Pair<String, String> {
    val distanceKm = distanceMeters / 1000.0
    return when (kind) {
        GearKind.SHOE -> when {
            distanceKm >= 800 -> "REVIEW" to "800+ km"
            distanceKm >= 600 -> "WATCH" to "600+ km"
            else -> "OK" to "OK"
        }

        GearKind.BIKE -> when {
            distanceKm >= 5000 -> "REVIEW" to "5000+ km"
            distanceKm >= 3000 -> "WATCH" to "3000+ km"
            else -> "OK" to "OK"
        }

        GearKind.UNKNOWN -> "OK" to "OK"
    }
}
