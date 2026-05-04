package dto

import (
	"fmt"
	"log"
	"math"
	"mystravastats/domain/badges"
	"mystravastats/domain/statistics"
	"mystravastats/internal/shared/domain/business"
	"mystravastats/internal/shared/domain/strava"
	"strconv"
	"strings"
	"time"
)

// calculateTotalDescent computes the total descent (in meters) from the altitude stream
// by summing all negative differences between consecutive altitude points.
// Returns 0 if the altitude stream is unavailable or has fewer than 2 points.
func calculateTotalDescent(stream *strava.Stream) float64 {
	if stream == nil || stream.Altitude == nil || len(stream.Altitude.Data) < 2 {
		return 0
	}
	totalDescent := 0.0
	data := stream.Altitude.Data
	for i := 1; i < len(data); i++ {
		diff := data[i] - data[i-1]
		if diff < 0 {
			totalDescent += math.Abs(diff)
		}
	}
	return totalDescent
}

func FormatSpeed(speed float64, activityType string) string {
	if strings.EqualFold(activityType, "Run") || strings.EqualFold(activityType, "TrailRun") {
		if speed <= 0 {
			return "-/km"
		}
		secondsPerKm := 1000.0 / speed
		return fmt.Sprintf("%s/km", formatSeconds(secondsPerKm))
	}
	return fmt.Sprintf("%.02f km/h", speed*3.6)
}

func FormatGradient(gradient float64) string {
	return fmt.Sprintf("%.02f", gradient)
}

func formatSeconds(totalSeconds float64) string {
	secs := int(math.Round(totalSeconds))
	if secs < 0 {
		secs = 0
	}
	h := secs / 3600
	m := (secs % 3600) / 60
	s := secs % 60

	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%d:%02d", m, s)
}

func ToActivityDto(activity strava.Activity) ActivityDto {
	bestPowerFor20Minutes := statistics.BestPowerForTime(activity, 20*60)
	bestPowerFor60Minutes := statistics.BestPowerForTime(activity, 60*60)

	var ftp = 0
	// FTP is defined as the highest power you can maintain in a quasi-steady state for approximately one hour without fatiguing.
	// If a 60-minute effort is not available, it is common practice to estimate FTP as 95% of a 20-minute effort.
	// https://www.trainingpeaks.com/blog/functional-threshold-power-ftp-what-is-it-and-how-to-test-it/
	// https://www.peakendurancesport.com/endurance-training/training-plans/functional-threshold-power-ftp/
	// https://www.cyclingnews.com/features/ask-cyclingnews-com-what-is-functional-threshold-power-ftp/
	// https://www.trainerroad.com/blog/what-is-functional-threshold-power/
	// https://www.strava.com/support/athlete/activities/what-is-functional-threshold-power-ftp
	if bestPowerFor60Minutes != nil {
		ftp = finiteEffortPower(bestPowerFor60Minutes)
	} else if bestPowerFor20Minutes != nil {
		ftp = finiteInt(finiteEffortPowerValue(bestPowerFor20Minutes) * 0.95)
	}

	link := ""
	if activity.UploadId != 0 {
		link = fmt.Sprintf("https://www.strava.com/activities/%d", activity.Id)
	}

	bestTimeForDistanceFor1000m := 0.0
	if bestTimeForDistance := statistics.BestTimeEffort(activity, 1000.0); bestTimeForDistance != nil {
		bestTimeForDistanceFor1000m = bestTimeForDistance.GetMSSpeed()
	}

	bestElevationForDistanceFor500m := 0.0
	if bestElevationForDistance := statistics.BestElevationEffort(activity, 500.0); bestElevationForDistance != nil {
		bestElevationForDistanceFor500m = bestElevationForDistance.GetGradient()
	}

	bestElevationForDistanceFor1000m := 0.0
	if bestElevationForDistance := statistics.BestElevationEffort(activity, 1000.0); bestElevationForDistance != nil {
		bestElevationForDistanceFor1000m = bestElevationForDistance.GetGradient()
	}

	return ActivityDto{
		Id:                               activity.Id,
		Name:                             activity.Name,
		Type:                             activity.Type,
		Commute:                          activity.Commute,
		Link:                             link,
		Distance:                         finiteInt(activity.Distance),
		ElapsedTime:                      activity.ElapsedTime,
		MovingTime:                       activity.MovingTime,
		TotalElevationGain:               finiteInt(activity.TotalElevationGain),
		AverageSpeed:                     finiteFloat64(activity.AverageSpeed), // in m/s
		AverageHeartrate:                 finiteInt(activity.AverageHeartrate),
		BestSpeedForDistanceFor1000m:     finiteFloat64(bestTimeForDistanceFor1000m), // in m/s
		BestElevationForDistanceFor500m:  finiteFloat64(bestElevationForDistanceFor500m),
		BestElevationForDistanceFor1000m: finiteFloat64(bestElevationForDistanceFor1000m),
		Date:                             activity.StartDateLocal,
		AverageWatts:                     finiteInt(activity.AverageWatts),
		WeightedAverageWatts:             activity.WeightedAverageWatts,
		BestPowerFor20Minutes:            finiteEffortPower(bestPowerFor20Minutes),
		BestPowerFor60Minutes:            finiteEffortPower(bestPowerFor60Minutes),
		FTP:                              ftp,
	}
}

func ToAnnualGoalsDto(goals business.AnnualGoals) AnnualGoalsDto {
	progress := make([]AnnualGoalProgressDto, 0, len(goals.Progress))
	for _, item := range goals.Progress {
		monthly := make([]AnnualGoalMonthDto, 0, len(item.Monthly))
		for _, month := range item.Monthly {
			monthly = append(monthly, AnnualGoalMonthDto{
				Month:              month.Month,
				Value:              month.Value,
				Cumulative:         month.Cumulative,
				ExpectedCumulative: month.ExpectedCumulative,
			})
		}
		progress = append(progress, AnnualGoalProgressDto{
			Metric:                  string(item.Metric),
			Label:                   item.Label,
			Unit:                    item.Unit,
			Current:                 item.Current,
			Target:                  item.Target,
			ProgressPercent:         item.ProgressPercent,
			ExpectedProgressPercent: item.ExpectedProgressPercent,
			ProjectedEndOfYear:      item.ProjectedEndOfYear,
			RequiredPace:            item.RequiredPace,
			RequiredPaceUnit:        item.RequiredPaceUnit,
			RequiredWeeklyPace:      item.RequiredWeeklyPace,
			Last30Days:              item.Last30Days,
			Last30DaysWeeklyPace:    item.Last30DaysWeeklyPace,
			WeeklyPaceGap:           item.WeeklyPaceGap,
			SuggestedTarget:         item.SuggestedTarget,
			Monthly:                 monthly,
			Status:                  string(item.Status),
		})
	}

	return AnnualGoalsDto{
		Year:            goals.Year,
		ActivityTypeKey: goals.ActivityTypeKey,
		Targets:         ToAnnualGoalTargetsDto(goals.Targets),
		Progress:        progress,
	}
}

func ToAnnualGoalTargetsDto(targets business.AnnualGoalTargets) AnnualGoalTargetsDto {
	return AnnualGoalTargetsDto{
		DistanceKm:      targets.DistanceKm,
		ElevationMeters: targets.ElevationMeters,
		Activities:      targets.Activities,
		ActiveDays:      targets.ActiveDays,
		Eddington:       targets.Eddington,
	}
}

func ToAnnualGoalTargets(targets AnnualGoalTargetsDto) business.AnnualGoalTargets {
	return business.AnnualGoalTargets{
		DistanceKm:      targets.DistanceKm,
		ElevationMeters: targets.ElevationMeters,
		Activities:      targets.Activities,
		ActiveDays:      targets.ActiveDays,
		Eddington:       targets.Eddington,
	}
}

func ToDetailedActivityDto(detailedActivity *strava.DetailedActivity) DetailedActivityDto {

	activityForDto := *detailedActivity
	activityForDto.Stream = sanitizeStreamForDto(detailedActivity.Stream)
	detailedActivity = &activityForDto

	activityEfforts := BuildActivityEfforts(detailedActivity)

	return DetailedActivityDto{
		AverageCadence:       finiteInt(detailedActivity.AverageCadence),
		AverageHeartrate:     finiteInt(detailedActivity.AverageHeartrate),
		AverageWatts:         finiteInt(detailedActivity.AverageWatts),
		AverageSpeed:         finiteFloat32(detailedActivity.AverageSpeed),
		Calories:             finiteFloat64(detailedActivity.Calories),
		Commute:              detailedActivity.Commute,
		DeviceWatts:          detailedActivity.DeviceWatts,
		Distance:             finiteFloat64(detailedActivity.Distance),
		ElapsedTime:          detailedActivity.ElapsedTime,
		ElevHigh:             finiteFloat64(detailedActivity.ElevHigh),
		ID:                   detailedActivity.Id,
		Kilojoules:           finiteFloat64(detailedActivity.Kilojoules),
		MaxHeartrate:         finiteInt(detailedActivity.MaxHeartrate),
		MaxSpeed:             finiteFloat32(detailedActivity.MaxSpeed),
		MaxWatts:             detailedActivity.MaxWatts,
		MovingTime:           detailedActivity.MovingTime,
		Name:                 detailedActivity.Name,
		ActivityEfforts:      toActivityEffortsDto(activityEfforts),
		StartDate:            parseTime(detailedActivity.StartDate),
		StartDateLocal:       parseTime(detailedActivity.StartDateLocal),
		StartLatlng:          finiteFloat64Slice(detailedActivity.StartLatLng),
		Stream:               toStreamDto(detailedActivity.Stream),
		SufferScore:          finiteFloat64Ptr(detailedActivity.SufferScore),
		TotalDescent:         finiteFloat64(calculateTotalDescent(detailedActivity.Stream)),
		TotalElevationGain:   finiteInt(detailedActivity.TotalElevationGain),
		Type:                 detailedActivity.Type,
		WeightedAverageWatts: detailedActivity.WeightedAverageWatts,
	}
}

func sanitizeStreamForDto(stream *strava.Stream) *strava.Stream {
	if stream == nil {
		return nil
	}

	sanitized := *stream
	sanitized.Distance.Data = finiteFloat64Slice(stream.Distance.Data)

	if stream.LatLng != nil {
		latlng := *stream.LatLng
		latlng.Data = finiteFloat64Grid(stream.LatLng.Data)
		sanitized.LatLng = &latlng
	}
	if stream.Altitude != nil {
		altitude := *stream.Altitude
		altitude.Data = finiteFloat64Slice(stream.Altitude.Data)
		sanitized.Altitude = &altitude
	}
	if stream.Watts != nil {
		watts := *stream.Watts
		watts.Data = finiteFloat64Slice(stream.Watts.Data)
		sanitized.Watts = &watts
	}
	if stream.VelocitySmooth != nil {
		velocitySmooth := *stream.VelocitySmooth
		velocitySmooth.Data = finiteFloat64Slice(stream.VelocitySmooth.Data)
		sanitized.VelocitySmooth = &velocitySmooth
	}
	if stream.GradeSmooth != nil {
		gradeSmooth := *stream.GradeSmooth
		gradeSmooth.Data = finiteFloat64Slice(stream.GradeSmooth.Data)
		sanitized.GradeSmooth = &gradeSmooth
	}

	return &sanitized
}

func parseTimePtr(value *string) time.Time {
	if value == nil {
		return time.Time{}
	}
	return parseTime(*value)
}

func toActivityEffortsDto(efforts []business.ActivityEffort) []ActivityEffortDto {
	var effortsDto []ActivityEffortDto
	for i, effort := range efforts {
		// Use a deterministic ID based on index and effort properties to ensure
		// stability across requests for the same activity data.
		id := fmt.Sprintf("%d_%d_%d", i, effort.IdxStart, effort.IdxEnd)
		effortsDto = append(effortsDto, ActivityEffortDto{
			ID:            id,
			Label:         effort.Label,
			Distance:      finiteFloat64(effort.Distance),
			Seconds:       effort.Seconds,
			DeltaAltitude: finiteFloat64(effort.DeltaAltitude),
			IdxStart:      effort.IdxStart,
			IdxEnd:        effort.IdxEnd,
			AveragePower:  finiteFloat64Ptr(effort.AveragePower),
			Description:   effort.GetDescription(),
		})
	}
	return effortsDto
}

func toStreamDto(stream *strava.Stream) *StreamDto {
	if stream == nil {
		return nil
	}

	// Latlng: copy [][]float64 directly — no pointer boxing needed.
	var latlng [][]float64
	if stream.LatLng != nil && stream.LatLng.Data != nil {
		latlng = make([][]float64, len(stream.LatLng.Data))
		for i, coords := range stream.LatLng.Data {
			if len(coords) >= 2 {
				latlng[i] = []float64{finiteFloat64(coords[0]), finiteFloat64(coords[1])}
			}
		}
	}

	// Moving: []bool — values are never null so no pointer boxing is needed.
	var moving []bool
	if stream.Moving != nil && stream.Moving.Data != nil {
		moving = make([]bool, len(stream.Moving.Data))
		copy(moving, stream.Moving.Data)
	}

	// Altitude, Watts, VelocitySmooth: []float64 — copy directly.
	var altitude []float64
	if stream.Altitude != nil && stream.Altitude.Data != nil {
		altitude = finiteFloat64Slice(stream.Altitude.Data)
	}

	var watts []float64
	if stream.Watts != nil && stream.Watts.Data != nil {
		watts = finiteFloat64Slice(stream.Watts.Data)
	}

	var velocitySmooth []float64
	if stream.VelocitySmooth != nil && stream.VelocitySmooth.Data != nil {
		velocitySmooth = finiteFloat64Slice(stream.VelocitySmooth.Data)
	}

	var heartrate []int
	if stream.HeartRate != nil {
		heartrate = make([]int, len(stream.HeartRate.Data))
		copy(heartrate, stream.HeartRate.Data)
	}

	return &StreamDto{
		Distance:       finiteFloat64Slice(stream.Distance.Data),
		Time:           stream.Time.Data,
		Latlng:         latlng,
		Heartrate:      heartrate,
		Moving:         moving,
		Altitude:       altitude,
		Watts:          watts,
		VelocitySmooth: velocitySmooth,
	}
}

func finiteFloat64(value float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0
	}
	return value
}

func finiteFloat32(value float64) float32 {
	if math.IsNaN(value) || math.IsInf(value, 0) || value > math.MaxFloat32 || value < -math.MaxFloat32 {
		return 0
	}
	return float32(value)
}

func finiteInt(value float64) int {
	maxInt := int(^uint(0) >> 1)
	minInt := -maxInt - 1
	if math.IsNaN(value) || math.IsInf(value, 0) || value > float64(maxInt) || value < float64(minInt) {
		return 0
	}
	return int(value)
}

func finiteFloat64Ptr(value *float64) *float64 {
	if value == nil {
		return nil
	}
	if math.IsNaN(*value) || math.IsInf(*value, 0) {
		return nil
	}
	sanitized := *value
	return &sanitized
}

func finiteEffortPower(effort *business.ActivityEffort) int {
	return finiteInt(finiteEffortPowerValue(effort))
}

func finiteEffortPowerValue(effort *business.ActivityEffort) float64 {
	if effort == nil || effort.AveragePower == nil {
		return 0
	}
	return finiteFloat64(*effort.AveragePower)
}

func finiteFloat64Slice(values []float64) []float64 {
	if values == nil {
		return nil
	}
	sanitized := make([]float64, len(values))
	for index, value := range values {
		sanitized[index] = finiteFloat64(value)
	}
	return sanitized
}

func finiteFloat64Grid(values [][]float64) [][]float64 {
	if values == nil {
		return nil
	}
	sanitized := make([][]float64, len(values))
	for index, row := range values {
		sanitized[index] = finiteFloat64Slice(row)
	}
	return sanitized
}

func parseTime(value string) time.Time {
	parsedTime, err := time.Parse(time.RFC3339, value)
	if err != nil {
		log.Printf("parseTime: failed to parse %q as RFC3339: %v", value, err)
	}
	return parsedTime
}

func ToAthleteDto(athlete strava.Athlete) AthleteDto {
	return AthleteDto{
		BadgeTypeId:           getIntValue(athlete.BadgeTypeId),
		City:                  getStringValue(athlete.City),
		Country:               getStringValue(athlete.Country),
		CreatedAt:             parseTimePtr(athlete.CreatedAt),
		Firstname:             getStringValue(athlete.Firstname),
		Id:                    athlete.Id,
		Lastname:              getStringValue(athlete.Lastname),
		Premium:               getBoolValue(athlete.Premium),
		Profile:               getStringValue(athlete.Profile),
		ProfileMedium:         getStringValue(athlete.ProfileMedium),
		ResourceState:         getIntValue(athlete.ResourceState),
		Sex:                   getStringValue(athlete.Sex),
		State:                 getStringValue(athlete.State),
		Summit:                getBoolValue(athlete.Summit),
		UpdatedAt:             parseTimePtr(athlete.UpdatedAt),
		Username:              getStringValue(athlete.Username),
		AthleteType:           getIntValue(athlete.AthleteType),
		DatePreference:        getStringValue(athlete.DatePreference),
		FollowerCount:         getIntValue(athlete.FollowerCount),
		FriendCount:           getIntValue(athlete.FriendCount),
		MeasurementPreference: getStringValue(athlete.MeasurementPreference),
		MutualFriendCount:     getIntValue(athlete.MutualFriendCount),
		Weight:                getIntValueFromFloat(athlete.Weight),
	}
}

func ToStatisticDto(statistic statistics.Statistic) StatisticDto {
	if statistic.Activity() != nil {
		return StatisticDto{
			Label: statistic.Label(),
			Value: statistic.Value(),
			Activity: &ActivityShortDto{
				ID:   statistic.Activity().Id,
				Name: statistic.Activity().Name,
				Type: statistic.Activity().Type.String(),
			},
		}
	}

	return StatisticDto{
		Label: statistic.Label(),
		Value: statistic.Value(),
	}
}

func ToPersonalRecordTimelineDto(entry business.PersonalRecordTimelineEntry) PersonalRecordTimelineDto {
	return PersonalRecordTimelineDto{
		MetricKey:     entry.MetricKey,
		MetricLabel:   entry.MetricLabel,
		ActivityDate:  entry.ActivityDate,
		Value:         entry.Value,
		PreviousValue: entry.PreviousValue,
		Improvement:   entry.Improvement,
		Activity: ActivityShortDto{
			ID:   entry.Activity.Id,
			Name: entry.Activity.Name,
			Type: entry.Activity.Type.String(),
		},
	}
}

func ToGearAnalysisDto(analysis business.GearAnalysis) GearAnalysisDto {
	items := make([]GearAnalysisItemDto, len(analysis.Items))
	for i, item := range analysis.Items {
		monthly := make([]GearAnalysisPeriodPointDto, len(item.MonthlyDistance))
		for j, point := range item.MonthlyDistance {
			monthly[j] = GearAnalysisPeriodPointDto{
				PeriodKey:     point.PeriodKey,
				Value:         point.Value,
				ActivityCount: point.ActivityCount,
			}
		}
		maintenanceTasks := make([]GearMaintenanceTaskDto, len(item.MaintenanceTasks))
		for j, task := range item.MaintenanceTasks {
			maintenanceTasks[j] = ToGearMaintenanceTaskDto(task)
		}
		maintenanceHistory := make([]GearMaintenanceRecordDto, len(item.MaintenanceHistory))
		for j, record := range item.MaintenanceHistory {
			maintenanceHistory[j] = ToGearMaintenanceRecordDto(record)
		}
		items[i] = GearAnalysisItemDto{
			ID:                       item.ID,
			Name:                     item.Name,
			Kind:                     string(item.Kind),
			Retired:                  item.Retired,
			Primary:                  item.Primary,
			MaintenanceStatus:        item.MaintenanceStatus,
			MaintenanceLabel:         item.MaintenanceLabel,
			MaintenanceTasks:         maintenanceTasks,
			MaintenanceHistory:       maintenanceHistory,
			Distance:                 item.Distance,
			TotalDistance:            item.TotalDistance,
			MovingTime:               item.MovingTime,
			ElevationGain:            item.ElevationGain,
			Activities:               item.Activities,
			AverageSpeed:             item.AverageSpeed,
			FirstUsed:                item.FirstUsed,
			LastUsed:                 item.LastUsed,
			LongestActivity:          toGearActivityShortDto(item.LongestActivity),
			BiggestElevationActivity: toGearActivityShortDto(item.BiggestElevationActivity),
			FastestActivity:          toGearActivityShortDto(item.FastestActivity),
			MonthlyDistance:          monthly,
		}
	}

	return GearAnalysisDto{
		Items: items,
		Unassigned: GearAnalysisSummaryDto{
			Distance:      analysis.Unassigned.Distance,
			MovingTime:    analysis.Unassigned.MovingTime,
			ElevationGain: analysis.Unassigned.ElevationGain,
			Activities:    analysis.Unassigned.Activities,
			AverageSpeed:  analysis.Unassigned.AverageSpeed,
		},
		Coverage: GearAnalysisCoverageDto{
			TotalActivities:      analysis.Coverage.TotalActivities,
			AssignedActivities:   analysis.Coverage.AssignedActivities,
			UnassignedActivities: analysis.Coverage.UnassignedActivities,
		},
	}
}

func ToGearMaintenanceRecordDto(record business.GearMaintenanceRecord) GearMaintenanceRecordDto {
	return GearMaintenanceRecordDto{
		ID:             record.ID,
		GearID:         record.GearID,
		GearName:       record.GearName,
		Component:      record.Component,
		ComponentLabel: record.ComponentLabel,
		Operation:      record.Operation,
		Date:           record.Date,
		Distance:       record.Distance,
		Note:           record.Note,
		CreatedAt:      record.CreatedAt,
		UpdatedAt:      record.UpdatedAt,
	}
}

func ToGearMaintenanceTaskDto(task business.GearMaintenanceTask) GearMaintenanceTaskDto {
	var lastMaintenance *GearMaintenanceRecordDto
	if task.LastMaintenance != nil {
		last := ToGearMaintenanceRecordDto(*task.LastMaintenance)
		lastMaintenance = &last
	}
	return GearMaintenanceTaskDto{
		Component:         task.Component,
		ComponentLabel:    task.ComponentLabel,
		IntervalDistance:  task.IntervalDistance,
		IntervalMonths:    task.IntervalMonths,
		Status:            task.Status,
		StatusLabel:       task.StatusLabel,
		DistanceSince:     task.DistanceSince,
		DistanceRemaining: task.DistanceRemaining,
		NextDueDistance:   task.NextDueDistance,
		MonthsSince:       task.MonthsSince,
		MonthsRemaining:   task.MonthsRemaining,
		LastMaintenance:   lastMaintenance,
	}
}

func ToGearMaintenanceRecordRequest(request GearMaintenanceRecordRequestDto) business.GearMaintenanceRecordRequest {
	return business.GearMaintenanceRecordRequest{
		GearID:    request.GearID,
		Component: request.Component,
		Operation: request.Operation,
		Date:      request.Date,
		Distance:  request.Distance,
		Note:      request.Note,
	}
}

func toGearActivityShortDto(activity *business.ActivityShort) *ActivityShortDto {
	if activity == nil {
		return nil
	}
	return &ActivityShortDto{
		ID:   activity.Id,
		Name: activity.Name,
		Type: activity.Type.String(),
	}
}

func ToSegmentClimbProgressionDto(progression business.SegmentClimbProgression) SegmentClimbProgressionDto {
	targets := make([]SegmentClimbTargetSummaryDto, len(progression.Targets))
	for i, target := range progression.Targets {
		targets[i] = ToSegmentClimbTargetSummaryDto(target)
	}

	attempts := make([]SegmentClimbAttemptDto, len(progression.Attempts))
	for i, attempt := range progression.Attempts {
		attempts[i] = ToSegmentClimbAttemptDto(attempt)
	}

	return SegmentClimbProgressionDto{
		Metric:                  progression.Metric,
		TargetTypeFilter:        progression.TargetTypeFilter,
		WeatherContextAvailable: progression.WeatherContextAvailable,
		Targets:                 targets,
		SelectedTargetId:        progression.SelectedTargetId,
		SelectedTargetType:      progression.SelectedTargetType,
		Attempts:                attempts,
	}
}

func ToSegmentClimbTargetSummaryDto(target business.SegmentClimbTargetSummary) SegmentClimbTargetSummaryDto {
	return SegmentClimbTargetSummaryDto{
		TargetId:       target.TargetId,
		TargetName:     target.TargetName,
		TargetType:     target.TargetType,
		ClimbCategory:  target.ClimbCategory,
		Distance:       target.Distance,
		AverageGrade:   target.AverageGrade,
		AttemptsCount:  target.AttemptsCount,
		BestValue:      target.BestValue,
		LatestValue:    target.LatestValue,
		Consistency:    target.Consistency,
		AveragePacing:  target.AveragePacing,
		CloseToPrCount: target.CloseToPrCount,
		RecentTrend:    target.RecentTrend,
	}
}

func ToSegmentClimbAttemptDto(attempt business.SegmentClimbAttempt) SegmentClimbAttemptDto {
	return SegmentClimbAttemptDto{
		TargetId:           attempt.TargetId,
		TargetName:         attempt.TargetName,
		TargetType:         attempt.TargetType,
		ActivityDate:       attempt.ActivityDate,
		ElapsedTimeSeconds: attempt.ElapsedTimeSeconds,
		MovingTimeSeconds:  attempt.MovingTimeSeconds,
		SpeedKph:           attempt.SpeedKph,
		Distance:           attempt.Distance,
		AverageGrade:       attempt.AverageGrade,
		ElevationGain:      attempt.ElevationGain,
		AveragePowerWatts:  attempt.AveragePowerWatts,
		AverageHeartRate:   attempt.AverageHeartRate,
		PrRank:             attempt.PrRank,
		PersonalRank:       attempt.PersonalRank,
		SetsNewPr:          attempt.SetsNewPr,
		CloseToPr:          attempt.CloseToPr,
		DeltaToPr:          attempt.DeltaToPr,
		WeatherSummary:     attempt.WeatherSummary,
		Activity: ActivityShortDto{
			ID:   attempt.Activity.Id,
			Name: attempt.Activity.Name,
			Type: attempt.Activity.Type.String(),
		},
	}
}

func ToHeartRateZoneSettingsDto(settings business.HeartRateZoneSettings) HeartRateZoneSettingsDto {
	return HeartRateZoneSettingsDto{
		MaxHr:       settings.MaxHr,
		ThresholdHr: settings.ThresholdHr,
		ReserveHr:   settings.ReserveHr,
	}
}

func ToHeartRateZoneSettings(dto HeartRateZoneSettingsDto) business.HeartRateZoneSettings {
	return business.HeartRateZoneSettings{
		MaxHr:       dto.MaxHr,
		ThresholdHr: dto.ThresholdHr,
		ReserveHr:   dto.ReserveHr,
	}
}

func ToHeartRateZoneAnalysisDto(analysis business.HeartRateZoneAnalysis) HeartRateZoneAnalysisDto {
	distributions := make([]HeartRateZoneDistributionDto, len(analysis.Zones))
	for i, zone := range analysis.Zones {
		distributions[i] = HeartRateZoneDistributionDto{
			Zone:       zone.Zone,
			Label:      zone.Label,
			Seconds:    zone.Seconds,
			Percentage: zone.Percentage,
		}
	}

	activities := make([]HeartRateZoneActivitySummaryDto, len(analysis.Activities))
	for i, activity := range analysis.Activities {
		zones := make([]HeartRateZoneDistributionDto, len(activity.Zones))
		for j, zone := range activity.Zones {
			zones[j] = HeartRateZoneDistributionDto{
				Zone:       zone.Zone,
				Label:      zone.Label,
				Seconds:    zone.Seconds,
				Percentage: zone.Percentage,
			}
		}
		activities[i] = HeartRateZoneActivitySummaryDto{
			Activity: ActivityShortDto{
				ID:   activity.Activity.Id,
				Name: activity.Activity.Name,
				Type: activity.Activity.Type.String(),
			},
			ActivityDate:        activity.ActivityDate,
			TotalTrackedSeconds: activity.TotalTrackedSeconds,
			EasySeconds:         activity.EasySeconds,
			HardSeconds:         activity.HardSeconds,
			EasyHardRatio:       activity.EasyHardRatio,
			Zones:               zones,
		}
	}

	byMonth := make([]HeartRateZonePeriodSummaryDto, len(analysis.ByMonth))
	for i, period := range analysis.ByMonth {
		zones := make([]HeartRateZoneDistributionDto, len(period.Zones))
		for j, zone := range period.Zones {
			zones[j] = HeartRateZoneDistributionDto{
				Zone:       zone.Zone,
				Label:      zone.Label,
				Seconds:    zone.Seconds,
				Percentage: zone.Percentage,
			}
		}
		byMonth[i] = HeartRateZonePeriodSummaryDto{
			Period:              period.Period,
			TotalTrackedSeconds: period.TotalTrackedSeconds,
			EasySeconds:         period.EasySeconds,
			HardSeconds:         period.HardSeconds,
			EasyHardRatio:       period.EasyHardRatio,
			Zones:               zones,
		}
	}

	byYear := make([]HeartRateZonePeriodSummaryDto, len(analysis.ByYear))
	for i, period := range analysis.ByYear {
		zones := make([]HeartRateZoneDistributionDto, len(period.Zones))
		for j, zone := range period.Zones {
			zones[j] = HeartRateZoneDistributionDto{
				Zone:       zone.Zone,
				Label:      zone.Label,
				Seconds:    zone.Seconds,
				Percentage: zone.Percentage,
			}
		}
		byYear[i] = HeartRateZonePeriodSummaryDto{
			Period:              period.Period,
			TotalTrackedSeconds: period.TotalTrackedSeconds,
			EasySeconds:         period.EasySeconds,
			HardSeconds:         period.HardSeconds,
			EasyHardRatio:       period.EasyHardRatio,
			Zones:               zones,
		}
	}

	var resolved *ResolvedHeartRateZoneSettingsDto
	if analysis.ResolvedSettings != nil {
		resolved = &ResolvedHeartRateZoneSettingsDto{
			MaxHr:       analysis.ResolvedSettings.MaxHr,
			ThresholdHr: analysis.ResolvedSettings.ThresholdHr,
			ReserveHr:   analysis.ResolvedSettings.ReserveHr,
			Method:      string(analysis.ResolvedSettings.Method),
			Source:      string(analysis.ResolvedSettings.Source),
		}
	}

	return HeartRateZoneAnalysisDto{
		Settings:            ToHeartRateZoneSettingsDto(analysis.Settings),
		ResolvedSettings:    resolved,
		HasHeartRateData:    analysis.HasHeartRateData,
		TotalTrackedSeconds: analysis.TotalTrackedSeconds,
		EasyHardRatio:       analysis.EasyHardRatio,
		Zones:               distributions,
		Activities:          activities,
		ByMonth:             byMonth,
		ByYear:              byYear,
	}
}

func ToDashboardDataDto(data business.DashboardData) DashboardDataDto {
	return DashboardDataDto{
		NbActivitiesByYear:        data.NbActivities,
		ActiveDaysByYear:          data.ActiveDaysByYear,
		ConsistencyByYear:         data.ConsistencyByYear,
		MovingTimeByYear:          data.MovingTimeByYear,
		TotalDistanceByYear:       data.TotalDistanceByYear,
		AverageDistanceByYear:     data.AverageDistanceByYear,
		MaxDistanceByYear:         data.MaxDistanceByYear,
		TotalElevationByYear:      data.TotalElevationByYear,
		AverageElevationByYear:    data.AverageElevationByYear,
		MaxElevationByYear:        data.MaxElevationByYear,
		ElevationEfficiencyByYear: data.ElevationEfficiencyByYear,
		AverageSpeedByYear:        data.AverageSpeedByYear,
		MaxSpeedByYear:            data.MaxSpeedByYear,
		AverageHeartRateByYear:    data.AverageHeartRateByYear,
		MaxHeartRateByYear:        data.MaxHeartRateByYear,
		AverageWattsByYear:        data.AverageWattsByYear,
		MaxWattsByYear:            data.MaxWattsByYear,
		AverageCadenceByYear:      data.AverageCadence,
	}
}

func getIntValue(value *int) int {
	if value != nil {
		return *value
	}
	return 0
}

func getStringValue(value *string) string {
	if value != nil {
		return *value
	}
	return ""
}

func getBoolValue(value *bool) bool {
	if value != nil {
		return *value
	}
	return false
}

func getIntValueFromFloat(value *float64) int {
	if value != nil {
		return int(*value)
	}
	return 0
}

func BuildActivityEfforts(activity *strava.DetailedActivity) []business.ActivityEffort {
	if activity == nil || activity.Stream == nil {
		return []business.ActivityEffort{}
	}

	var activityEfforts []business.ActivityEffort

	slopes := activity.Stream.ListSlopesDefault()
	// Filter slopes to keep only ascent segments
	ascentSlopes := make([]strava.Slope, 0, len(slopes))
	for _, slope := range slopes {
		if slope.Type == strava.ASCENT {
			ascentSlopes = append(ascentSlopes, slope)
		}
	}

	for index, s := range ascentSlopes {

		e := business.ActivityEffort{
			Distance:      s.Distance,
			Seconds:       s.Duration,
			DeltaAltitude: s.EndAltitude - s.StartAltitude,
			IdxStart:      s.StartIndex,
			IdxEnd:        s.EndIndex,
			AveragePower:  nil,
			Label:         fmt.Sprintf("Climb %d - %.1f km - D+ %.0f m - max %.1f%%", index+1, s.Distance/1000, math.Max(0, s.EndAltitude-s.StartAltitude), s.MaxGrade),
			ActivityShort: business.ActivityShort{
				Id:   activity.Id,
				Name: activity.Name,
				Type: business.ActivityTypes[activity.Type],
			},
		}
		activityEfforts = append(activityEfforts, e)
	}

	// Add additional efforts based on specific criteria
	bestTimeFor1000m := calculateBestTimeForDistance(activity, 1000.0)
	if bestTimeFor1000m != nil {
		activityEfforts = append(activityEfforts, *bestTimeFor1000m)
	}
	bestTimeFor5000m := calculateBestTimeForDistance(activity, 5000.0)
	if bestTimeFor5000m != nil {
		activityEfforts = append(activityEfforts, *bestTimeFor5000m)
	}
	bestTimeFor10000m := calculateBestTimeForDistance(activity, 10000.0)
	if bestTimeFor10000m != nil {
		activityEfforts = append(activityEfforts, *bestTimeFor10000m)
	}
	bestDistanceFor1Hour := calculateBestDistanceForTime(activity, 3600)
	if bestDistanceFor1Hour != nil {
		activityEfforts = append(activityEfforts, *bestDistanceFor1Hour)
	}
	bestElevationFor500m := calculateBestElevationForDistance(activity, 500.0)
	if bestElevationFor500m != nil {
		activityEfforts = append(activityEfforts, *bestElevationFor500m)
	}
	bestElevationFor1000m := calculateBestElevationForDistance(activity, 1000.0)
	if bestElevationFor1000m != nil {
		activityEfforts = append(activityEfforts, *bestElevationFor1000m)
	}
	bestElevationFor10000m := calculateBestElevationForDistance(activity, 10000.0)
	if bestElevationFor10000m != nil {
		activityEfforts = append(activityEfforts, *bestElevationFor10000m)
	}

	for _, segmentEffort := range activity.SegmentEfforts {
		if segmentEffort.Segment.ClimbCategory > 2 || segmentEffort.Segment.Starred {
			activityEfforts = append(activityEfforts, toActivityEffort(&segmentEffort, activity))
		}
	}

	return activityEfforts
}

type segmentEffortDirection int

const (
	segmentEffortDirectionUnknown segmentEffortDirection = 0
	segmentEffortDirectionAscent  segmentEffortDirection = 1
	segmentEffortDirectionDescent segmentEffortDirection = -1
)

const (
	segmentEffortDirectionMinAltitudeDeltaM = 3.0
	segmentEffortDirectionMinGradePercent   = 0.5
)

var segmentEffortDirectionLabelNormalizer = strings.NewReplacer(
	"é", "e",
	"è", "e",
	"ê", "e",
	"ë", "e",
	"à", "a",
	"â", "a",
	"ä", "a",
	"î", "i",
	"ï", "i",
	"ô", "o",
	"ö", "o",
	"ù", "u",
	"û", "u",
	"ü", "u",
	"ç", "c",
	"’", "'",
	"-", " ",
)

var ascentSegmentDirectionKeywords = []string{
	"montee",
	"ascent",
	"climb",
	"uphill",
}

var descentSegmentDirectionKeywords = []string{
	"descente",
	"descent",
	"downhill",
}

func toActivityEffort(effort *strava.SegmentEffort, activity *strava.DetailedActivity) business.ActivityEffort {
	direction := resolveSegmentEffortDirection(effort, activity)
	label := directionAwareSegmentEffortLabel(effort.Segment.Name, direction)
	deltaAltitude := resolveSegmentEffortDeltaAltitude(effort, activity, direction)

	return business.ActivityEffort{
		Distance:      effort.Distance,
		Seconds:       effort.ElapsedTime,
		DeltaAltitude: deltaAltitude,
		IdxStart:      effort.StartIndex,
		IdxEnd:        effort.EndIndex,
		AveragePower:  &effort.AverageWatts,
		Label:         label,
		ActivityShort: business.ActivityShort{
			Id:   effort.Id,
			Name: label,
			Type: business.ActivityTypes[effort.Segment.ActivityType],
		},
	}
}

func resolveSegmentEffortDirection(effort *strava.SegmentEffort, activity *strava.DetailedActivity) segmentEffortDirection {
	if direction := resolveSegmentEffortDirectionFromAltitudeStream(activity, effort); direction != segmentEffortDirectionUnknown {
		return direction
	}
	if direction := resolveSegmentEffortDirectionFromLabels(effort.Name, effort.Segment.Name); direction != segmentEffortDirectionUnknown {
		return direction
	}
	return resolveSegmentEffortDirectionFromAverageGrade(effort.Segment.AverageGrade)
}

func resolveSegmentEffortDirectionFromAltitudeStream(activity *strava.DetailedActivity, effort *strava.SegmentEffort) segmentEffortDirection {
	if activity == nil || activity.Stream == nil || activity.Stream.Altitude == nil {
		return segmentEffortDirectionUnknown
	}
	altitudeData := activity.Stream.Altitude.Data
	if len(altitudeData) == 0 {
		return segmentEffortDirectionUnknown
	}
	startIndex := effort.StartIndex
	endIndex := effort.EndIndex
	if startIndex < 0 || endIndex < 0 || startIndex >= len(altitudeData) || endIndex >= len(altitudeData) || startIndex == endIndex {
		return segmentEffortDirectionUnknown
	}
	altitudeDelta := altitudeData[endIndex] - altitudeData[startIndex]
	if !isFiniteNumber(altitudeDelta) {
		return segmentEffortDirectionUnknown
	}
	if math.Abs(altitudeDelta) < segmentEffortDirectionMinAltitudeDeltaM {
		return segmentEffortDirectionUnknown
	}
	if altitudeDelta > 0 {
		return segmentEffortDirectionAscent
	}
	return segmentEffortDirectionDescent
}

func resolveSegmentEffortDirectionFromLabels(labels ...string) segmentEffortDirection {
	for _, label := range labels {
		normalized := normalizeSegmentEffortDirectionLabel(label)
		if normalized == "" {
			continue
		}
		if hasAnySegmentEffortDirectionKeyword(normalized, descentSegmentDirectionKeywords) {
			return segmentEffortDirectionDescent
		}
		if hasAnySegmentEffortDirectionKeyword(normalized, ascentSegmentDirectionKeywords) {
			return segmentEffortDirectionAscent
		}
	}
	return segmentEffortDirectionUnknown
}

func resolveSegmentEffortDirectionFromAverageGrade(averageGrade float64) segmentEffortDirection {
	if !isFiniteNumber(averageGrade) {
		return segmentEffortDirectionUnknown
	}
	if math.Abs(averageGrade) < segmentEffortDirectionMinGradePercent {
		return segmentEffortDirectionUnknown
	}
	if averageGrade > 0 {
		return segmentEffortDirectionAscent
	}
	return segmentEffortDirectionDescent
}

func resolveSegmentEffortDeltaAltitude(
	effort *strava.SegmentEffort,
	activity *strava.DetailedActivity,
	direction segmentEffortDirection,
) float64 {
	if activity != nil && activity.Stream != nil && activity.Stream.Altitude != nil {
		altitudeData := activity.Stream.Altitude.Data
		startIndex := effort.StartIndex
		endIndex := effort.EndIndex
		if startIndex >= 0 && endIndex >= 0 && startIndex < len(altitudeData) && endIndex < len(altitudeData) {
			altitudeDelta := altitudeData[endIndex] - altitudeData[startIndex]
			if isFiniteNumber(altitudeDelta) {
				return altitudeDelta
			}
		}
	}

	baseDelta := effort.Segment.ElevationHigh - effort.Segment.ElevationLow
	if !isFiniteNumber(baseDelta) {
		baseDelta = 0
	}
	switch direction {
	case segmentEffortDirectionAscent:
		return math.Abs(baseDelta)
	case segmentEffortDirectionDescent:
		return -math.Abs(baseDelta)
	default:
		return baseDelta
	}
}

func directionAwareSegmentEffortLabel(baseLabel string, direction segmentEffortDirection) string {
	normalized := strings.ToLower(baseLabel)
	switch direction {
	case segmentEffortDirectionAscent:
		if strings.Contains(normalized, "(ascent)") {
			return baseLabel
		}
		return fmt.Sprintf("%s (ascent)", baseLabel)
	case segmentEffortDirectionDescent:
		if strings.Contains(normalized, "(descent)") {
			return baseLabel
		}
		return fmt.Sprintf("%s (descent)", baseLabel)
	default:
		return baseLabel
	}
}

func normalizeSegmentEffortDirectionLabel(label string) string {
	normalized := strings.ToLower(strings.TrimSpace(label))
	if normalized == "" {
		return ""
	}
	normalized = segmentEffortDirectionLabelNormalizer.Replace(normalized)
	return strings.Join(strings.Fields(normalized), " ")
}

func hasAnySegmentEffortDirectionKeyword(label string, keywords []string) bool {
	for _, keyword := range keywords {
		if strings.Contains(label, keyword) {
			return true
		}
	}
	return false
}

func isFiniteNumber(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

func calculateBestElevationForDistance(activity *strava.DetailedActivity, f float64) *business.ActivityEffort {
	if activity.Stream == nil {
		return nil
	}
	return statistics.BestElevationForDistance(activity.Id, activity.Name, activity.Type, activity.Stream, f)
}

func calculateBestDistanceForTime(activity *strava.DetailedActivity, i int) *business.ActivityEffort {
	if activity.Stream == nil {
		return nil
	}
	return statistics.BestDistanceForTime(activity.Id, activity.Name, activity.Type, activity.Stream, i)
}

func calculateBestTimeForDistance(activity *strava.DetailedActivity, f float64) *business.ActivityEffort {

	if activity.Stream == nil {
		return nil
	}
	return statistics.BestTimeForDistance(activity.Id, activity.Name, activity.Type, activity.Stream, f)
}

func ToBadgeCheckResultDto(result business.BadgeCheckResult, activityTypes ...business.ActivityType) BadgeCheckResultDto {
	nbCheckedActivities := len(result.Activities)
	var activities []ActivityDto
	if nbCheckedActivities > 0 {
		displayActivity, badgeEffortSeconds := selectBadgeDisplayActivity(result.Badge, result.Activities)
		if displayActivity != nil {
			activityDto := ToActivityDto(*displayActivity)
			if badgeEffortSeconds > 0 {
				activityDto.BadgeEffortSeconds = badgeEffortSeconds
			}
			activities = append(activities, activityDto)
		}
	}

	return BadgeCheckResultDto{
		Badge:               ToBadgeDto(result.Badge, activityTypes...),
		Activities:          activities,
		NbCheckedActivities: nbCheckedActivities,
	}
}

func selectBadgeDisplayActivity(badge business.Badge, activities []*strava.Activity) (*strava.Activity, int) {
	if len(activities) == 0 {
		return nil, 0
	}

	switch b := badge.(type) {
	case badges.FamousClimbBadge:
		return selectBestFamousClimbActivity(activities, b)
	default:
		// Keep current behavior for non-climb badges.
		return activities[len(activities)-1], 0
	}
}

func selectFastestActivity(activities []*strava.Activity) *strava.Activity {
	var best *strava.Activity
	for _, activity := range activities {
		if activity == nil {
			continue
		}
		if best == nil {
			best = activity
			continue
		}
		if activity.MovingTime > 0 && (best.MovingTime <= 0 || activity.MovingTime < best.MovingTime) {
			best = activity
		}
	}

	if best != nil {
		return best
	}
	return activities[len(activities)-1]
}

func selectBestFamousClimbActivity(activities []*strava.Activity, badge badges.FamousClimbBadge) (*strava.Activity, int) {
	var bestActivity *strava.Activity
	bestEffortSeconds := 0

	for _, activity := range activities {
		effortSeconds, ok := computeFamousClimbEffortSeconds(activity, badge)
		if !ok {
			continue
		}
		if bestActivity == nil || effortSeconds < bestEffortSeconds {
			bestActivity = activity
			bestEffortSeconds = effortSeconds
		}
	}

	if bestActivity != nil {
		return bestActivity, bestEffortSeconds
	}

	// Fallback when streams are unavailable: keep old behavior.
	return selectFastestActivity(activities), 0
}

func computeFamousClimbEffortSeconds(activity *strava.Activity, badge badges.FamousClimbBadge) (int, bool) {
	if activity == nil || activity.Stream == nil || activity.Stream.LatLng == nil {
		return 0, false
	}

	latlngData := activity.Stream.LatLng.Data
	timeData := activity.Stream.Time.Data
	dataSize := len(latlngData)
	if len(timeData) < dataSize {
		dataSize = len(timeData)
	}
	if dataSize == 0 {
		return 0, false
	}

	const waypointToleranceInM = 500
	startTime := 0
	seenStart := false

	for i := 0; i < dataSize; i++ {
		coords := latlngData[i]
		if len(coords) < 2 {
			continue
		}

		if !seenStart {
			if badge.Start.HaversineInM(coords[0], coords[1]) < waypointToleranceInM {
				seenStart = true
				startTime = timeData[i]
			}
			continue
		}

		if badge.End.HaversineInM(coords[0], coords[1]) < waypointToleranceInM {
			duration := timeData[i] - startTime
			if duration > 0 {
				return duration, true
			}
		}
	}

	return 0, false
}

func ToBadgeDto(badge business.Badge, activityTypes ...business.ActivityType) BadgeDto {
	if len(activityTypes) == 0 {
		return BadgeDto{}
	}

	activityType, ok := business.RepresentativeBadgeActivityType(activityTypes...)
	if !ok {
		return BadgeDto{}
	}

	switch b := badge.(type) {
	case badges.DistanceBadge:
		return BadgeDto{
			Label:       b.Label,
			Description: strconv.FormatFloat(b.Distance, 'f', 0, 64),
			Type:        activityType.String() + "DistanceBadge",
		}
	case badges.ElevationBadge:
		return BadgeDto{
			Label:       b.Label,
			Description: strconv.FormatFloat(b.TotalElevationGain, 'f', 0, 64),
			Type:        activityType.String() + "ElevationBadge",
		}
	case badges.MovingTimeBadge:
		return BadgeDto{
			Label:       b.Label,
			Description: strconv.Itoa(b.MovingTime),
			Type:        activityType.String() + "MovingTimeBadge",
		}
	case badges.FamousClimbBadge:
		return BadgeDto{
			Label:       b.Label,
			Description: b.Name,
			Type:        activityType.String() + "FamousClimbBadge",
			Category:    b.Category,
		}
	default:
		return BadgeDto{}
	}
}
