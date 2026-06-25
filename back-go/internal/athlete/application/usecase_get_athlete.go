package application

import (
	"fmt"
	"math"
	"mystravastats/domain/statistics"
	"sort"
	"time"

	"mystravastats/internal/shared/domain/business"
	"mystravastats/internal/shared/domain/strava"
)

const defaultFtpEstimateWindowDays = 180

type GetAthleteUseCase struct {
	reader AthleteReader
}

func NewGetAthleteUseCase(reader AthleteReader) *GetAthleteUseCase {
	return &GetAthleteUseCase{
		reader: reader,
	}
}

func (uc *GetAthleteUseCase) Execute() strava.Athlete {
	return uc.reader.FindAthlete()
}

type GetPerformanceSettingsUseCase struct {
	reader AthleteReader
}

func NewGetPerformanceSettingsUseCase(reader AthleteReader) *GetPerformanceSettingsUseCase {
	return &GetPerformanceSettingsUseCase{reader: reader}
}

func (uc *GetPerformanceSettingsUseCase) Execute() business.AthletePerformanceSettings {
	return normalizePerformanceSettings(uc.reader.FindPerformanceSettings())
}

type UpdatePerformanceSettingsUseCase struct {
	reader AthleteReader
}

func NewUpdatePerformanceSettingsUseCase(reader AthleteReader) *UpdatePerformanceSettingsUseCase {
	return &UpdatePerformanceSettingsUseCase{reader: reader}
}

func (uc *UpdatePerformanceSettingsUseCase) Execute(settings business.AthletePerformanceSettings) business.AthletePerformanceSettings {
	return uc.reader.SavePerformanceSettings(normalizePerformanceSettings(settings))
}

type GetFtpEstimateUseCase struct {
	reader AthleteReader
	now    func() time.Time
}

func NewGetFtpEstimateUseCase(reader AthleteReader) *GetFtpEstimateUseCase {
	return &GetFtpEstimateUseCase{
		reader: reader,
		now:    time.Now,
	}
}

func DefaultFtpEstimateActivityTypes() []business.ActivityType {
	return []business.ActivityType{
		business.Commute,
		business.GravelRide,
		business.MountainBikeRide,
		business.Ride,
		business.VirtualRide,
	}
}

func (uc *GetFtpEstimateUseCase) Execute(activityTypes []business.ActivityType, windowDays int) business.FtpEstimate {
	normalizedWindowDays := normalizeFtpEstimateWindowDays(windowDays)
	if len(activityTypes) == 0 {
		activityTypes = DefaultFtpEstimateActivityTypes()
	}

	activities := uc.reader.FindActivitiesByYearAndTypes(nil, activityTypes...)
	if len(activities) == 0 {
		return unavailableFtpEstimate(normalizedWindowDays, 0, "No activities available")
	}

	referenceDate := latestActivityDate(activities, uc.now())
	recentCutoff := referenceDate.AddDate(0, 0, -normalizedWindowDays)

	candidateGroups := []ftpEstimateCandidateGroup{
		{
			activities:  filterFtpEstimateActivities(activities, recentCutoff, true),
			source:      fmt.Sprintf("Power meter, last %d days", normalizedWindowDays),
			sourceKind:  "power-meter",
			recent:      true,
			deviceWatts: true,
		},
		{
			activities:  filterFtpEstimateActivities(activities, time.Time{}, true),
			source:      "Power meter, all time",
			sourceKind:  "power-meter",
			deviceWatts: true,
		},
		{
			activities: filterFtpEstimateActivities(activities, recentCutoff, false),
			source:     fmt.Sprintf("All power data, last %d days", normalizedWindowDays),
			sourceKind: "all-power",
			recent:     true,
		},
		{
			activities: filterFtpEstimateActivities(activities, time.Time{}, false),
			source:     "All power data, all time",
			sourceKind: "all-power",
		},
	}

	for _, group := range candidateGroups {
		if estimate, ok := estimateFtpFromGroup(group, normalizedWindowDays); ok {
			return estimate
		}
	}

	return unavailableFtpEstimate(normalizedWindowDays, len(activities), "No usable power stream available")
}

func normalizePerformanceSettings(settings business.AthletePerformanceSettings) business.AthletePerformanceSettings {
	normalized := business.AthletePerformanceSettings{}
	if settings.WeightKg != nil && *settings.WeightKg > 0 {
		weight := *settings.WeightKg
		normalized.WeightKg = &weight
	}

	byDate := make(map[string]business.AthleteFtpSetting)
	for _, entry := range settings.FtpHistory {
		if entry.Ftp <= 0 {
			continue
		}
		if _, err := time.Parse("2006-01-02", entry.EffectiveFrom); err != nil {
			continue
		}
		byDate[entry.EffectiveFrom] = business.AthleteFtpSetting{
			EffectiveFrom: entry.EffectiveFrom,
			Ftp:           entry.Ftp,
		}
	}

	for _, entry := range byDate {
		normalized.FtpHistory = append(normalized.FtpHistory, entry)
	}
	sort.Slice(normalized.FtpHistory, func(i, j int) bool {
		return normalized.FtpHistory[i].EffectiveFrom < normalized.FtpHistory[j].EffectiveFrom
	})

	return normalized
}

type ftpEstimateCandidateGroup struct {
	activities  []*strava.Activity
	source      string
	sourceKind  string
	recent      bool
	deviceWatts bool
}

func estimateFtpFromGroup(group ftpEstimateCandidateGroup, windowDays int) (business.FtpEstimate, bool) {
	if len(group.activities) == 0 {
		return business.FtpEstimate{}, false
	}

	if effort := bestPowerEffortByAveragePower(group.activities, 60*60); effort != nil {
		bestPower := int(math.Round(*effort.AveragePower))
		return ftpEstimateFromEffort(group, *effort, bestPower, bestPower, 1, "best-60min", "Best 60 min power", windowDays), true
	}

	if effort := bestPowerEffortByAveragePower(group.activities, 20*60); effort != nil {
		bestPower := int(math.Round(*effort.AveragePower))
		ftp := int(math.Round(*effort.AveragePower * 0.95))
		return ftpEstimateFromEffort(group, *effort, ftp, bestPower, 0.95, "95-percent-20min", "95% of best 20 min power", windowDays), true
	}

	return business.FtpEstimate{}, false
}

func bestPowerEffortByAveragePower(activities []*strava.Activity, seconds int) *business.ActivityEffort {
	var bestEffort *business.ActivityEffort
	for _, activity := range activities {
		if activity == nil {
			continue
		}
		effort := statistics.BestPowerForTime(*activity, seconds)
		if effort == nil || effort.AveragePower == nil {
			continue
		}
		if bestEffort == nil || *effort.AveragePower > *bestEffort.AveragePower {
			bestEffort = effort
		}
	}
	return bestEffort
}

func ftpEstimateFromEffort(
	group ftpEstimateCandidateGroup,
	effort business.ActivityEffort,
	ftp int,
	bestPower int,
	multiplier float64,
	method string,
	methodLabel string,
	windowDays int,
) business.FtpEstimate {
	return business.FtpEstimate{
		Available:      ftp > 0,
		Ftp:            ftp,
		Method:         method,
		MethodLabel:    methodLabel,
		BestPower:      bestPower,
		Multiplier:     multiplier,
		BasedOnSeconds: effort.Seconds,
		Confidence:     ftpEstimateConfidence(group, method),
		Source:         group.source,
		SourceKind:     group.sourceKind,
		ActivityID:     effort.ActivityShort.Id,
		ActivityName:   effort.ActivityShort.Name,
		ActivityType:   effort.ActivityShort.Type.String(),
		ActivityDate:   ftpEstimateActivityDate(group.activities, effort.ActivityShort.Id),
		WindowDays:     windowDays,
		ActivityCount:  len(group.activities),
	}
}

func ftpEstimateConfidence(group ftpEstimateCandidateGroup, method string) string {
	if group.recent && group.deviceWatts && method == "best-60min" {
		return "high"
	}
	if group.deviceWatts && (group.recent || method == "best-60min") {
		return "medium"
	}
	return "low"
}

func filterFtpEstimateActivities(activities []*strava.Activity, cutoff time.Time, deviceOnly bool) []*strava.Activity {
	filtered := make([]*strava.Activity, 0, len(activities))
	for _, activity := range activities {
		if activity == nil {
			continue
		}
		if deviceOnly && !activity.DeviceWatts {
			continue
		}
		if !cutoff.IsZero() {
			activityDate, ok := parseFtpEstimateActivityDate(activity)
			if !ok || activityDate.Before(cutoff) {
				continue
			}
		}
		filtered = append(filtered, activity)
	}
	return filtered
}

func latestActivityDate(activities []*strava.Activity, fallback time.Time) time.Time {
	var latest time.Time
	for _, activity := range activities {
		activityDate, ok := parseFtpEstimateActivityDate(activity)
		if !ok {
			continue
		}
		if latest.IsZero() || activityDate.After(latest) {
			latest = activityDate
		}
	}
	if latest.IsZero() {
		return fallback
	}
	return latest
}

func parseFtpEstimateActivityDate(activity *strava.Activity) (time.Time, bool) {
	if activity == nil {
		return time.Time{}, false
	}
	for _, value := range []string{activity.StartDateLocal, activity.StartDate} {
		if value == "" {
			continue
		}
		if parsed, err := time.Parse(time.RFC3339, value); err == nil {
			return parsed, true
		}
		if len(value) >= 10 {
			if parsed, err := time.Parse("2006-01-02", value[:10]); err == nil {
				return parsed, true
			}
		}
	}
	return time.Time{}, false
}

func ftpEstimateActivityDate(activities []*strava.Activity, activityID int64) string {
	for _, activity := range activities {
		if activity == nil || activity.Id != activityID {
			continue
		}
		if date, ok := extractFtpEstimateDateOnly(activity.StartDateLocal); ok {
			return date
		}
		if date, ok := extractFtpEstimateDateOnly(activity.StartDate); ok {
			return date
		}
	}
	return ""
}

func extractFtpEstimateDateOnly(value string) (string, bool) {
	if len(value) < 10 {
		return "", false
	}
	date := value[:10]
	if _, err := time.Parse("2006-01-02", date); err != nil {
		return "", false
	}
	return date, true
}

func normalizeFtpEstimateWindowDays(windowDays int) int {
	if windowDays <= 0 {
		return defaultFtpEstimateWindowDays
	}
	if windowDays < 30 {
		return 30
	}
	if windowDays > 730 {
		return 730
	}
	return windowDays
}

func unavailableFtpEstimate(windowDays int, activityCount int, source string) business.FtpEstimate {
	return business.FtpEstimate{
		Available:     false,
		Confidence:    "unavailable",
		Source:        source,
		SourceKind:    "none",
		WindowDays:    windowDays,
		ActivityCount: activityCount,
	}
}
