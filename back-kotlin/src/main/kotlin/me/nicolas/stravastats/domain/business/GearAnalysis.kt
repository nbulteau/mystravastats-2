package me.nicolas.stravastats.domain.business

enum class GearKind {
    BIKE,
    SHOE,
    UNKNOWN,
}

data class GearAnalysis(
    val items: List<GearAnalysisItem>,
    val unassigned: GearAnalysisSummary,
    val coverage: GearAnalysisCoverage,
)

data class GearAnalysisItem(
    val id: String,
    val name: String,
    val kind: GearKind,
    val retired: Boolean,
    val primary: Boolean,
    val maintenanceStatus: String,
    val maintenanceLabel: String,
    val maintenanceTasks: List<GearMaintenanceTask> = emptyList(),
    val maintenanceHistory: List<GearMaintenanceRecord> = emptyList(),
    val distance: Double,
    val movingTime: Int,
    val elevationGain: Double,
    val activities: Int,
    val averageSpeed: Double,
    val firstUsed: String,
    val lastUsed: String,
    val longestActivity: ActivityShort?,
    val biggestElevationActivity: ActivityShort?,
    val fastestActivity: ActivityShort?,
    val monthlyDistance: List<GearAnalysisPeriodPoint>,
)

data class GearAnalysisSummary(
    val distance: Double,
    val movingTime: Int,
    val elevationGain: Double,
    val activities: Int,
    val averageSpeed: Double,
)

data class GearAnalysisCoverage(
    val totalActivities: Int,
    val assignedActivities: Int,
    val unassignedActivities: Int,
)

data class GearAnalysisPeriodPoint(
    val periodKey: String,
    val value: Double,
    val activityCount: Int,
)

data class GearMaintenanceRecord(
    val id: String,
    val gearId: String,
    val gearName: String,
    val component: String,
    val componentLabel: String,
    val operation: String,
    val date: String,
    val distance: Double,
    val note: String? = null,
    val createdAt: String,
    val updatedAt: String,
)

data class GearMaintenanceRecordRequest(
    val gearId: String = "",
    val component: String = "",
    val operation: String = "",
    val date: String = "",
    val distance: Double = 0.0,
    val note: String? = null,
)

data class GearMaintenanceTask(
    val component: String,
    val componentLabel: String,
    val intervalDistance: Double,
    val intervalMonths: Int,
    val status: String,
    val statusLabel: String,
    val distanceSince: Double,
    val distanceRemaining: Double,
    val nextDueDistance: Double,
    val monthsSince: Int,
    val monthsRemaining: Int,
    val lastMaintenance: GearMaintenanceRecord? = null,
)
