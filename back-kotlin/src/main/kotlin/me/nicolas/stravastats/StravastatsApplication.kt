package me.nicolas.stravastats

import io.swagger.v3.oas.annotations.Operation
import io.swagger.v3.oas.annotations.media.ArraySchema
import io.swagger.v3.oas.annotations.media.Content
import io.swagger.v3.oas.annotations.media.Schema
import io.swagger.v3.oas.annotations.responses.ApiResponse
import io.swagger.v3.oas.annotations.tags.Tag
import me.nicolas.stravastats.api.dto.PersonalRecordTimelineDto
import me.nicolas.stravastats.api.dto.AnnualGoalMonthDto
import me.nicolas.stravastats.api.dto.AnnualGoalProgressDto
import me.nicolas.stravastats.api.dto.AnnualGoalTargetsDto
import me.nicolas.stravastats.api.dto.AnnualGoalsDto
import me.nicolas.stravastats.api.dto.GearAnalysisCoverageDto
import me.nicolas.stravastats.api.dto.GearAnalysisDto
import me.nicolas.stravastats.api.dto.GearAnalysisItemDto
import me.nicolas.stravastats.api.dto.GearAnalysisPeriodPointDto
import me.nicolas.stravastats.api.dto.GearAnalysisSummaryDto
import me.nicolas.stravastats.api.dto.GearMaintenanceRecordDto
import me.nicolas.stravastats.api.dto.GearMaintenanceRecordRequestDto
import me.nicolas.stravastats.api.dto.GearMaintenanceTaskDto
import me.nicolas.stravastats.api.dto.HeartRateZoneActivitySummaryDto
import me.nicolas.stravastats.api.dto.HeartRateZoneAnalysisDto
import me.nicolas.stravastats.api.dto.HeartRateZoneDistributionDto
import me.nicolas.stravastats.api.dto.HeartRateZonePeriodSummaryDto
import me.nicolas.stravastats.api.dto.HeartRateZoneSettingsDto
import me.nicolas.stravastats.api.dto.ResolvedHeartRateZoneSettingsDto
import me.nicolas.stravastats.domain.business.PersonalRecordTimelineEntry
import me.nicolas.stravastats.domain.business.AnnualGoalMonth
import me.nicolas.stravastats.domain.business.AnnualGoalProgress
import me.nicolas.stravastats.domain.business.AnnualGoalTargets
import me.nicolas.stravastats.domain.business.AnnualGoals
import me.nicolas.stravastats.domain.business.DataQualityExclusion
import me.nicolas.stravastats.domain.business.DataQualityExclusionRequest
import me.nicolas.stravastats.domain.business.DataQualityCorrection
import me.nicolas.stravastats.domain.business.DataQualityCorrectionBatchSummary
import me.nicolas.stravastats.domain.business.DataQualityCorrectionImpact
import me.nicolas.stravastats.domain.business.DataQualityCorrectionPreview
import me.nicolas.stravastats.domain.business.DataQualityCorrectionSuggestion
import me.nicolas.stravastats.domain.business.DataQualityIssue
import me.nicolas.stravastats.domain.business.DataQualityReport
import me.nicolas.stravastats.domain.business.DataQualitySummary
import me.nicolas.stravastats.domain.business.GearAnalysis
import me.nicolas.stravastats.domain.business.GearAnalysisCoverage
import me.nicolas.stravastats.domain.business.GearAnalysisItem
import me.nicolas.stravastats.domain.business.GearAnalysisPeriodPoint
import me.nicolas.stravastats.domain.business.GearAnalysisSummary
import me.nicolas.stravastats.domain.business.GearMaintenanceRecord
import me.nicolas.stravastats.domain.business.GearMaintenanceRecordRequest
import me.nicolas.stravastats.domain.business.GearMaintenanceTask
import me.nicolas.stravastats.domain.business.SourceModeEnvironmentVariable
import me.nicolas.stravastats.domain.business.SourceModePreview
import me.nicolas.stravastats.domain.business.SourceModePreviewError
import me.nicolas.stravastats.domain.business.SourceModePreviewRequest
import me.nicolas.stravastats.domain.business.SourceModeYearPreview
import me.nicolas.stravastats.domain.business.StravaOAuthStartRequest
import me.nicolas.stravastats.domain.business.StravaOAuthStartResult
import me.nicolas.stravastats.domain.business.StravaOAuthStatus
import me.nicolas.stravastats.domain.business.HeartRateZoneActivitySummary
import me.nicolas.stravastats.domain.business.HeartRateZoneAnalysis
import me.nicolas.stravastats.domain.business.HeartRateZoneDistribution
import me.nicolas.stravastats.domain.business.HeartRateZonePeriodSummary
import me.nicolas.stravastats.domain.business.HeartRateZoneSettings
import me.nicolas.stravastats.domain.business.ResolvedHeartRateZoneSettings
import me.nicolas.stravastats.domain.business.strava.Achievement
import me.nicolas.stravastats.domain.business.strava.AthleteRef
import me.nicolas.stravastats.domain.business.strava.Bike
import me.nicolas.stravastats.domain.business.strava.Gear
import me.nicolas.stravastats.domain.business.strava.GeoCoordinate
import me.nicolas.stravastats.domain.business.strava.GeoMap
import me.nicolas.stravastats.domain.business.strava.MetaActivity
import me.nicolas.stravastats.domain.business.strava.MetaAthlete
import me.nicolas.stravastats.domain.business.strava.Segment
import me.nicolas.stravastats.domain.business.strava.Shoe
import me.nicolas.stravastats.domain.business.strava.SplitsMetric
import me.nicolas.stravastats.domain.business.strava.StravaActivity
import me.nicolas.stravastats.domain.business.strava.StravaAthlete
import me.nicolas.stravastats.domain.business.strava.StravaDetailedActivity
import me.nicolas.stravastats.domain.business.strava.StravaSegmentEffort
import me.nicolas.stravastats.domain.business.strava.stream.AltitudeStream
import me.nicolas.stravastats.domain.business.strava.stream.CadenceStream
import me.nicolas.stravastats.domain.business.strava.stream.DistanceStream
import me.nicolas.stravastats.domain.business.strava.stream.HeartRateStream
import me.nicolas.stravastats.domain.business.strava.stream.LatLngStream
import me.nicolas.stravastats.domain.business.strava.stream.MovingStream
import me.nicolas.stravastats.domain.business.strava.stream.PowerStream
import me.nicolas.stravastats.domain.business.strava.stream.SmoothGradeStream
import me.nicolas.stravastats.domain.business.strava.stream.SmoothVelocityStream
import me.nicolas.stravastats.domain.business.strava.stream.Stream
import me.nicolas.stravastats.domain.business.strava.stream.TimeStream
import me.nicolas.stravastats.domain.business.badges.Alternative
import me.nicolas.stravastats.domain.business.badges.FamousClimb
import org.springframework.aot.hint.annotation.RegisterReflectionForBinding
import org.springframework.boot.autoconfigure.SpringBootApplication
import org.springframework.boot.runApplication

@SpringBootApplication
@RegisterReflectionForBinding(
    classes = [
        Achievement::class,
        AthleteRef::class,
        Bike::class,
        Gear::class,
        GeoCoordinate::class,
        GeoMap::class,
        MetaActivity::class,
        MetaAthlete::class,
        Segment::class,
        Shoe::class,
        SplitsMetric::class,
        StravaActivity::class,
        StravaAthlete::class,
        StravaDetailedActivity::class,
        StravaSegmentEffort::class,
        Stream::class,
        AltitudeStream::class,
        CadenceStream::class,
        DistanceStream::class,
        HeartRateStream::class,
        LatLngStream::class,
        MovingStream::class,
        PowerStream::class,
        SmoothGradeStream::class,
        SmoothVelocityStream::class,
        TimeStream::class,
        FamousClimb::class,
        Alternative::class,
        Operation::class,
        ArraySchema::class,
        Content::class,
        Schema::class,
        Schema.AccessMode::class,
        Schema.AdditionalPropertiesValue::class,
        Schema.RequiredMode::class,
        Schema.SchemaResolution::class,
        ApiResponse::class,
        Tag::class,
        PersonalRecordTimelineEntry::class,
        PersonalRecordTimelineDto::class,
        AnnualGoalTargets::class,
        AnnualGoalMonth::class,
        AnnualGoalProgress::class,
        AnnualGoals::class,
        AnnualGoalTargetsDto::class,
        AnnualGoalMonthDto::class,
        AnnualGoalProgressDto::class,
        AnnualGoalsDto::class,
        DataQualityIssue::class,
        DataQualityExclusion::class,
        DataQualityExclusionRequest::class,
        DataQualityCorrection::class,
        DataQualityCorrectionBatchSummary::class,
        DataQualityCorrectionImpact::class,
        DataQualityCorrectionPreview::class,
        DataQualityCorrectionSuggestion::class,
        DataQualitySummary::class,
        DataQualityReport::class,
        GearAnalysis::class,
        GearAnalysisItem::class,
        GearAnalysisSummary::class,
        GearAnalysisCoverage::class,
        GearAnalysisPeriodPoint::class,
        GearMaintenanceRecord::class,
        GearMaintenanceRecordRequest::class,
        GearMaintenanceTask::class,
        GearAnalysisDto::class,
        GearAnalysisItemDto::class,
        GearAnalysisSummaryDto::class,
        GearAnalysisCoverageDto::class,
        GearAnalysisPeriodPointDto::class,
        GearMaintenanceRecordDto::class,
        GearMaintenanceRecordRequestDto::class,
        GearMaintenanceTaskDto::class,
        SourceModePreviewRequest::class,
        StravaOAuthStartRequest::class,
        StravaOAuthStartResult::class,
        SourceModeEnvironmentVariable::class,
        SourceModeYearPreview::class,
        SourceModePreviewError::class,
        StravaOAuthStatus::class,
        SourceModePreview::class,
        HeartRateZoneSettings::class,
        ResolvedHeartRateZoneSettings::class,
        HeartRateZoneDistribution::class,
        HeartRateZoneActivitySummary::class,
        HeartRateZonePeriodSummary::class,
        HeartRateZoneAnalysis::class,
        HeartRateZoneSettingsDto::class,
        ResolvedHeartRateZoneSettingsDto::class,
        HeartRateZoneDistributionDto::class,
        HeartRateZoneActivitySummaryDto::class,
        HeartRateZonePeriodSummaryDto::class,
        HeartRateZoneAnalysisDto::class,
    ]
)
class StravastatsApplication

fun main(args: Array<String>) {
    runApplication<StravastatsApplication>(*args)
}
