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
