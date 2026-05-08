package me.nicolas.stravastats.domain.services

import io.mockk.every
import io.mockk.mockk
import me.nicolas.stravastats.domain.business.ActivityType
import me.nicolas.stravastats.domain.business.DataQualityCorrectionImpact
import me.nicolas.stravastats.domain.business.DataQualityCorrectionPreview
import me.nicolas.stravastats.domain.business.DataQualityIssue
import me.nicolas.stravastats.domain.business.DataQualityReport
import me.nicolas.stravastats.domain.business.DataQualitySummary
import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.services.activityproviders.IActivityProvider
import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Test
import tools.jackson.module.kotlin.jacksonTypeRef
import tools.jackson.databind.json.JsonMapper
import tools.jackson.module.kotlin.KotlinModule
import java.nio.file.Path
import kotlin.math.round

class DataQualityParityFixtureTest {

    private val objectMapper = JsonMapper.builder()
        .addModule(KotlinModule.Builder().build())
        .build()

    @Test
    fun `local data quality fixture matches shared parity snapshot`() {
        val fixture = readFixture<DataQualityParityFixture>("local-activity-anomalies.json")
        val expected = readFixture<DataQualityParitySnapshot>("expected-local-activity-anomalies.snapshot.json")

        val actual = DataQualityParitySnapshot(
            cases = fixture.cases.map { fixtureCase ->
                val service = DataQualityService(providerFor(fixtureCase))
                val report = service.getReport()
                val preview = service.previewSafeCorrections()
                DataQualityParityCaseSnapshot(
                    source = fixtureCase.source,
                    summary = report.summary.toSnapshot(),
                    issues = report.issues.toIssueSnapshots(),
                    safeCorrectionPreview = preview.toCorrectionSnapshotSet(),
                )
            }
        )

        assertEquals(expected, actual)
    }

    private fun providerFor(fixtureCase: DataQualityParityCase): IActivityProvider {
        val provider = mockk<IActivityProvider>()
        val diagnostics = when (fixtureCase.source) {
            "fit" -> mapOf("provider" to "fit", "fitDirectory" to fixtureCase.sourcePath)
            "gpx" -> mapOf("provider" to "gpx", "gpxDirectory" to fixtureCase.sourcePath)
            else -> mapOf("provider" to fixtureCase.source)
        }
        every { provider.getCacheDiagnostics() } returns diagnostics
        every { provider.cacheIdentity() } returns null
        every { provider.getActivitiesByActivityTypeAndYear(ActivityType.values().toSet(), null) } returns fixtureCase.activities
        return provider
    }

    private inline fun <reified T> readFixture(name: String): T {
        val path = Path.of("..", "test-fixtures", "data-quality", name)
        return objectMapper.readValue(path.toFile(), jacksonTypeRef<T>())
    }
}

private data class DataQualityParityFixture(
    val cases: List<DataQualityParityCase>,
)

private data class DataQualityParityCase(
    val source: String,
    val sourcePath: String,
    val activities: List<StravaActivity>,
)

private data class DataQualityParitySnapshot(
    val cases: List<DataQualityParityCaseSnapshot>,
)

private data class DataQualityParityCaseSnapshot(
    val source: String,
    val summary: DataQualitySummarySnapshot,
    val issues: List<DataQualityIssueSnapshot>,
    val safeCorrectionPreview: DataQualityCorrectionSnapshotSet,
)

private data class DataQualitySummarySnapshot(
    val status: String,
    val issueCount: Int,
    val impactedActivities: Int,
    val excludedActivities: Int,
    val correctionCount: Int,
    val safeCorrectionCount: Int,
    val manualReviewCount: Int,
    val bySeverity: Map<String, Int>,
    val byCategory: Map<String, Int>,
)

private data class DataQualityIssueSnapshot(
    val id: String,
    val activityId: Long,
    val severity: String,
    val category: String,
    val field: String,
    val correctionAvailable: Boolean,
    val correctionSafety: String? = null,
    val correctionType: String? = null,
)

private data class DataQualityCorrectionSnapshotSet(
    val summary: DataQualityCorrectionSummarySnapshot,
    val corrections: List<DataQualityCorrectionSnapshot>,
)

private data class DataQualityCorrectionSummarySnapshot(
    val safeCorrectionCount: Int,
    val manualReviewCount: Int,
    val unsupportedIssueCount: Int,
    val activityCount: Int,
    val distanceDeltaMeters: Double,
    val elevationDeltaMeters: Double,
    val modifiedFields: List<String>,
    val potentiallyImpactsRecords: Boolean,
)

private data class DataQualityCorrectionSnapshot(
    val id: String,
    val issueId: String,
    val activityId: Long,
    val type: String,
    val safety: String,
    val pointIndexes: List<Int> = emptyList(),
    val modifiedFields: List<String>,
    val impact: DataQualityCorrectionImpactSnapshot,
)

private data class DataQualityCorrectionImpactSnapshot(
    val distanceMetersBefore: Double,
    val distanceMetersAfter: Double,
    val elevationMetersBefore: Double,
    val elevationMetersAfter: Double,
    val maxSpeedBefore: Double,
    val maxSpeedAfter: Double,
    val distanceDeltaMeters: Double,
    val elevationDeltaMeters: Double,
)

private fun DataQualitySummary.toSnapshot(): DataQualitySummarySnapshot =
    DataQualitySummarySnapshot(
        status = status,
        issueCount = issueCount,
        impactedActivities = impactedActivities,
        excludedActivities = excludedActivities,
        correctionCount = correctionCount,
        safeCorrectionCount = safeCorrectionCount,
        manualReviewCount = manualReviewCount,
        bySeverity = bySeverity,
        byCategory = byCategory,
    )

private fun List<DataQualityIssue>.toIssueSnapshots(): List<DataQualityIssueSnapshot> =
    map { issue ->
        val correction = issue.correction?.takeIf { it.available }
        DataQualityIssueSnapshot(
            id = issue.id,
            activityId = issue.activityId ?: 0,
            severity = issue.severity,
            category = issue.category,
            field = issue.field,
            correctionAvailable = correction != null,
            correctionSafety = correction?.safety,
            correctionType = correction?.type,
        )
    }.sortedWith(
        compareBy<DataQualityIssueSnapshot> { it.activityId }
            .thenBy { it.category }
            .thenBy { it.field }
            .thenBy { it.id }
    )

private fun DataQualityCorrectionPreview.toCorrectionSnapshotSet(): DataQualityCorrectionSnapshotSet =
    DataQualityCorrectionSnapshotSet(
        summary = DataQualityCorrectionSummarySnapshot(
            safeCorrectionCount = summary.safeCorrectionCount,
            manualReviewCount = summary.manualReviewCount,
            unsupportedIssueCount = summary.unsupportedIssueCount,
            activityCount = summary.activityCount,
            distanceDeltaMeters = summary.distanceDeltaMeters.roundDataQuality(),
            elevationDeltaMeters = summary.elevationDeltaMeters.roundDataQuality(),
            modifiedFields = summary.modifiedFields,
            potentiallyImpactsRecords = summary.potentiallyImpactsRecords,
        ),
        corrections = corrections.map { correction ->
            DataQualityCorrectionSnapshot(
                id = correction.id,
                issueId = correction.issueId,
                activityId = correction.activityId,
                type = correction.type,
                safety = correction.safety,
                pointIndexes = correction.pointIndexes,
                modifiedFields = correction.modifiedFields,
                impact = correction.impact.toSnapshot(),
            )
        },
    )

private fun DataQualityCorrectionImpact.toSnapshot(): DataQualityCorrectionImpactSnapshot =
    DataQualityCorrectionImpactSnapshot(
        distanceMetersBefore = distanceMetersBefore.roundDataQuality(),
        distanceMetersAfter = distanceMetersAfter.roundDataQuality(),
        elevationMetersBefore = elevationMetersBefore.roundDataQuality(),
        elevationMetersAfter = elevationMetersAfter.roundDataQuality(),
        maxSpeedBefore = maxSpeedBefore.roundDataQuality(),
        maxSpeedAfter = maxSpeedAfter.roundDataQuality(),
        distanceDeltaMeters = distanceDeltaMeters.roundDataQuality(),
        elevationDeltaMeters = elevationDeltaMeters.roundDataQuality(),
    )

private fun Double.roundDataQuality(): Double =
    round(this * 1000.0) / 1000.0
