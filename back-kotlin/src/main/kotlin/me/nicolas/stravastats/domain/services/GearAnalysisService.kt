package me.nicolas.stravastats.domain.services

import me.nicolas.stravastats.domain.business.ActivityShort
import me.nicolas.stravastats.domain.business.ActivityType
import me.nicolas.stravastats.domain.business.GearAnalysis
import me.nicolas.stravastats.domain.business.GearAnalysisCoverage
import me.nicolas.stravastats.domain.business.GearAnalysisItem
import me.nicolas.stravastats.domain.business.GearAnalysisPeriodPoint
import me.nicolas.stravastats.domain.business.GearAnalysisSummary
import me.nicolas.stravastats.domain.business.GearKind
import me.nicolas.stravastats.domain.business.GearMaintenanceRecord
import me.nicolas.stravastats.domain.business.GearMaintenanceRecordRequest
import me.nicolas.stravastats.domain.business.GearMaintenanceTask
import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.business.strava.StravaAthlete
import me.nicolas.stravastats.domain.services.activityproviders.IActivityProvider
import org.springframework.stereotype.Service
import tools.jackson.databind.DeserializationFeature
import tools.jackson.databind.json.JsonMapper
import tools.jackson.module.kotlin.KotlinModule
import java.io.File
import java.time.Instant
import java.time.LocalDate
import java.time.temporal.ChronoUnit
import java.util.Locale
import kotlin.math.floor
import kotlin.math.max

private data class GearMetadata(
    val name: String,
    val kind: GearKind,
    val retired: Boolean,
    val primary: Boolean,
)

interface IGearAnalysisService {
    fun getGearAnalysis(activityTypes: Set<ActivityType>, year: Int?): GearAnalysis
    fun saveMaintenanceRecord(request: GearMaintenanceRecordRequest): GearMaintenanceRecord
    fun deleteMaintenanceRecord(recordId: String)
}

@Service
internal class GearAnalysisService(
    activityProvider: IActivityProvider,
) : IGearAnalysisService, AbstractStravaService(activityProvider) {

    override fun getGearAnalysis(activityTypes: Set<ActivityType>, year: Int?): GearAnalysis {
        val activities = activityProvider.getActivitiesByActivityTypeAndYear(activityTypes, year)
            .withoutDataQualityExcludedStats(activityProvider)
        return buildGearAnalysis(activities, activityProvider.athlete(), GearMaintenanceStorage.load(activityProvider))
    }

    override fun saveMaintenanceRecord(request: GearMaintenanceRecordRequest): GearMaintenanceRecord {
        val normalized = request.normalize()
        val gearName = gearNameForMaintenance(activityProvider.athlete(), normalized.gearId)
            .ifBlank { gearDisplayName(normalized.gearId, null) }
        val now = Instant.now().toString()
        val record = GearMaintenanceRecord(
            id = "gm-${System.currentTimeMillis()}",
            gearId = normalized.gearId,
            gearName = gearName,
            component = normalized.component,
            componentLabel = gearMaintenanceComponentLabel(normalized.component),
            operation = normalized.operation.ifBlank { "${gearMaintenanceComponentLabel(normalized.component)} serviced" },
            date = normalized.date,
            distance = normalized.distance.roundGearValue(),
            note = normalized.note?.trim()?.takeIf { it.isNotBlank() },
            createdAt = now,
            updatedAt = now,
        )
        GearMaintenanceStorage.save(activityProvider, GearMaintenanceStorage.load(activityProvider) + record)
        return record
    }

    override fun deleteMaintenanceRecord(recordId: String) {
        val trimmedId = recordId.trim()
        require(trimmedId.isNotBlank()) { "recordId is required" }
        val records = GearMaintenanceStorage.load(activityProvider)
        val updated = records.filterNot { it.id == trimmedId }
        require(updated.size != records.size) { "maintenance record $trimmedId not found" }
        GearMaintenanceStorage.save(activityProvider, updated)
    }

    private fun buildGearAnalysis(
        activities: List<StravaActivity>,
        athlete: StravaAthlete,
        maintenanceRecords: List<GearMaintenanceRecord>,
    ): GearAnalysis {
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
            .withGearMaintenance(maintenanceRecords)
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

private data class GearMaintenanceRule(
    val component: String,
    val label: String,
    val intervalDistance: Double = 0.0,
    val intervalMonths: Int = 0,
)

private val bikeMaintenanceRules = listOf(
    GearMaintenanceRule(component = "CHAIN", label = "Chain", intervalDistance = 1500.0 * 1000.0),
    GearMaintenanceRule(component = "CASSETTE", label = "Cassette", intervalDistance = 5000.0 * 1000.0),
    GearMaintenanceRule(component = "BRAKE_PADS_FRONT", label = "Front brake pads", intervalDistance = 1800.0 * 1000.0),
    GearMaintenanceRule(component = "BRAKE_PADS_REAR", label = "Rear brake pads", intervalDistance = 1800.0 * 1000.0),
    GearMaintenanceRule(component = "BRAKE_BLEED", label = "Brake bleed", intervalMonths = 12),
    GearMaintenanceRule(component = "TIRES", label = "Tires", intervalDistance = 3500.0 * 1000.0),
    GearMaintenanceRule(component = "TUBELESS_FRONT", label = "Front tubeless sealant", intervalMonths = 4),
    GearMaintenanceRule(component = "TUBELESS_REAR", label = "Rear tubeless sealant", intervalMonths = 4),
    GearMaintenanceRule(component = "BOTTOM_BRACKET", label = "Bottom bracket", intervalDistance = 8000.0 * 1000.0),
    GearMaintenanceRule(component = "BEARINGS", label = "Bearings", intervalDistance = 6000.0 * 1000.0),
    GearMaintenanceRule(component = "DRIVETRAIN", label = "Drivetrain", intervalDistance = 5000.0 * 1000.0),
)

private fun List<GearAnalysisItem>.withGearMaintenance(records: List<GearMaintenanceRecord>): List<GearAnalysisItem> {
    val recordsByGear = records
        .groupBy { it.gearId }
        .mapValues { (_, gearRecords) ->
            gearRecords.sortedWith(compareByDescending<GearMaintenanceRecord> { it.date }.thenByDescending { it.createdAt })
        }
    val today = LocalDate.now()
    return map { item ->
        val history = recordsByGear[item.id].orEmpty()
        if (item.kind != GearKind.BIKE) {
            item.copy(maintenanceHistory = history)
        } else {
            val tasks = buildGearMaintenanceTasks(item, history, today)
            val summary = summarizeGearMaintenance(tasks)
            item.copy(
                maintenanceStatus = summary.first,
                maintenanceLabel = summary.second,
                maintenanceTasks = tasks,
                maintenanceHistory = history,
            )
        }
    }
}

private fun buildGearMaintenanceTasks(
    item: GearAnalysisItem,
    records: List<GearMaintenanceRecord>,
    today: LocalDate,
): List<GearMaintenanceTask> {
    val recordsByComponent = records.groupBy { it.component }
    return bikeMaintenanceRules.map { rule ->
        val last = recordsByComponent[rule.component]
            .orEmpty()
            .maxWithOrNull(compareBy<GearMaintenanceRecord> { it.date }.thenBy { it.createdAt })
        if (last == null) {
            return@map GearMaintenanceTask(
                component = rule.component,
                componentLabel = rule.label,
                intervalDistance = rule.intervalDistance,
                intervalMonths = rule.intervalMonths,
                status = "DUE",
                statusLabel = "No service logged",
                distanceSince = 0.0,
                distanceRemaining = 0.0,
                nextDueDistance = 0.0,
                monthsSince = 0,
                monthsRemaining = 0,
                lastMaintenance = null,
            )
        }

        var status = "OK"
        var statusLabel = "OK"
        var distanceSince = 0.0
        var distanceRemaining = 0.0
        var nextDueDistance = 0.0
        if (rule.intervalDistance > 0.0) {
            distanceSince = max(0.0, item.distance - last.distance).roundGearValue()
            nextDueDistance = (last.distance + rule.intervalDistance).roundGearValue()
            distanceRemaining = max(0.0, nextDueDistance - item.distance).roundGearValue()
            val ratio = distanceSince / rule.intervalDistance
            when {
                ratio >= 1.0 -> {
                    status = "OVERDUE"
                    statusLabel = "${kotlin.math.ceil((distanceSince - rule.intervalDistance) / 1000.0).toInt()} km overdue"
                }
                ratio >= 0.85 -> {
                    status = "SOON"
                    statusLabel = "${kotlin.math.ceil(distanceRemaining / 1000.0).toInt()} km left"
                }
            }
        }

        var monthsSince = 0
        var monthsRemaining = 0
        if (rule.intervalMonths > 0) {
            val lastDate = runCatching { LocalDate.parse(last.date.take(10)) }.getOrNull()
            if (lastDate != null) {
                monthsSince = ChronoUnit.MONTHS.between(lastDate, today).toInt().coerceAtLeast(0)
                monthsRemaining = (rule.intervalMonths - monthsSince).coerceAtLeast(0)
                val timeStatus = when {
                    monthsSince >= rule.intervalMonths -> "OVERDUE"
                    monthsSince.toDouble() / rule.intervalMonths.toDouble() >= 0.80 -> "SOON"
                    else -> "OK"
                }
                val timeLabel = when (timeStatus) {
                    "OVERDUE" -> "${monthsSince - rule.intervalMonths} months overdue"
                    "SOON" -> "$monthsRemaining months left"
                    else -> "OK"
                }
                if (maintenanceStatusRank(timeStatus) > maintenanceStatusRank(status)) {
                    status = timeStatus
                    statusLabel = timeLabel
                }
            }
        }

        GearMaintenanceTask(
            component = rule.component,
            componentLabel = rule.label,
            intervalDistance = rule.intervalDistance,
            intervalMonths = rule.intervalMonths,
            status = status,
            statusLabel = statusLabel,
            distanceSince = distanceSince,
            distanceRemaining = distanceRemaining,
            nextDueDistance = nextDueDistance,
            monthsSince = monthsSince,
            monthsRemaining = monthsRemaining,
            lastMaintenance = last,
        )
    }
}

private fun summarizeGearMaintenance(tasks: List<GearMaintenanceTask>): Pair<String, String> {
    if (tasks.isEmpty()) return "OK" to "OK"
    val counts = tasks.groupingBy { it.status }.eachCount()
    val worst = tasks.maxBy { maintenanceStatusRank(it.status) }.status
    return when (worst) {
        "OVERDUE" -> worst to "${counts[worst] ?: 0} overdue"
        "DUE" -> worst to "${counts[worst] ?: 0} due"
        "SOON" -> worst to "${counts[worst] ?: 0} soon"
        else -> "OK" to "OK"
    }
}

private fun GearMaintenanceRecordRequest.normalize(): GearMaintenanceRecordRequest {
    val normalizedComponent = normalizeGearMaintenanceComponent(component)
    require(gearId.trim().isNotBlank()) { "gearId is required" }
    require(normalizedComponent.isNotBlank()) { "component is required" }
    val normalizedDate = date.trim().take(10)
    require(runCatching { LocalDate.parse(normalizedDate) }.isSuccess) { "date must use YYYY-MM-DD" }
    require(distance >= 0.0) { "distance must be >= 0" }
    return GearMaintenanceRecordRequest(
        gearId = gearId.trim(),
        component = normalizedComponent,
        operation = operation.trim(),
        date = normalizedDate,
        distance = distance,
        note = note?.trim(),
    )
}

private fun normalizeGearMaintenanceComponent(value: String): String {
    val normalized = gearMaintenanceComponentKey(value)
    if (normalized.isBlank()) return ""
    return bikeMaintenanceRules.firstOrNull {
        it.component == normalized || gearMaintenanceComponentKey(it.label) == normalized
    }?.component ?: normalized
}

private fun gearMaintenanceComponentLabel(component: String): String {
    return bikeMaintenanceRules.firstOrNull { it.component == component }?.label ?: gearMaintenanceHumanLabel(component)
}

private fun gearMaintenanceComponentKey(value: String): String {
    val builder = StringBuilder()
    var lastWasSeparator = true
    value.trim().uppercase(Locale.ROOT).forEach { char ->
        if (char.isLetterOrDigit()) {
            builder.append(char)
            lastWasSeparator = false
        } else if (!lastWasSeparator) {
            builder.append('_')
            lastWasSeparator = true
        }
    }
    return builder.toString().trim('_')
}

private fun gearMaintenanceHumanLabel(component: String): String {
    return component.trim()
        .lowercase(Locale.ROOT)
        .replace("_", " ")
        .split(" ")
        .filter { it.isNotBlank() }
        .joinToString(" ") { word -> word.replaceFirstChar { char -> char.titlecase(Locale.ROOT) } }
}

private fun gearNameForMaintenance(athlete: StravaAthlete, gearId: String): String {
    return athlete.bikes.orEmpty()
        .firstOrNull { it.id.trim() == gearId }
        ?.let { bike -> bike.nickname?.takeIf { it.isNotBlank() } ?: bike.name }
        ?.trim()
        .orEmpty()
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
            distanceKm >= 800 -> "OVERDUE" to "800+ km"
            distanceKm >= 600 -> "SOON" to "600+ km"
            else -> "OK" to "OK"
        }

        GearKind.BIKE -> when {
            distanceKm >= 5000 -> "OVERDUE" to "5000+ km"
            distanceKm >= 3000 -> "SOON" to "3000+ km"
            else -> "OK" to "OK"
        }

        GearKind.UNKNOWN -> "OK" to "OK"
    }
}

private fun maintenanceStatusRank(status: String): Int {
    return when (status) {
        "OVERDUE" -> 3
        "DUE" -> 2
        "SOON" -> 1
        else -> 0
    }
}

private data class GearMaintenanceFile(
    val records: List<GearMaintenanceRecord> = emptyList(),
)

private object GearMaintenanceStorage {
    private val objectMapper = JsonMapper.builder()
        .addModule(KotlinModule.Builder().build())
        .disable(DeserializationFeature.FAIL_ON_NULL_FOR_PRIMITIVES)
        .build()

    fun load(activityProvider: IActivityProvider): List<GearMaintenanceRecord> {
        val file = file(activityProvider) ?: return emptyList()
        if (!file.exists()) return emptyList()
        return runCatching {
            normalize(objectMapper.readValue(file, GearMaintenanceFile::class.java).records)
        }.getOrElse {
            emptyList()
        }
    }

    fun save(activityProvider: IActivityProvider, records: List<GearMaintenanceRecord>) {
        val file = file(activityProvider) ?: return
        file.parentFile?.mkdirs()
        objectMapper.writerWithDefaultPrettyPrinter().writeValue(file, GearMaintenanceFile(normalize(records)))
    }

    private fun file(activityProvider: IActivityProvider): File? {
        val identity = runCatching { activityProvider.cacheIdentity() }.getOrNull() ?: return null
        val athleteDirectory = File(identity.cacheRoot, "strava-${identity.athleteId}")
        return File(athleteDirectory, "gear-maintenance-${identity.athleteId}.json")
    }

    private fun normalize(records: List<GearMaintenanceRecord>): List<GearMaintenanceRecord> {
        return records.mapNotNull { record ->
            val id = record.id.trim()
            val gearId = record.gearId.trim()
            val component = normalizeGearMaintenanceComponent(record.component)
            if (id.isBlank() || gearId.isBlank() || component.isBlank()) {
                return@mapNotNull null
            }
            record.copy(
                id = id,
                gearId = gearId,
                gearName = record.gearName.trim(),
                component = component,
                componentLabel = gearMaintenanceComponentLabel(component),
                operation = record.operation.trim(),
                date = record.date.trim().take(10),
                distance = record.distance.roundGearValue(),
                note = record.note?.trim()?.takeIf { it.isNotBlank() },
            )
        }.sortedWith(compareBy<GearMaintenanceRecord> { it.gearId }.thenByDescending { it.date }.thenByDescending { it.createdAt })
    }
}
