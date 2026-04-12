package services

import (
	"math"
	"mystravastats/domain/business"
	"mystravastats/domain/strava"
	"sort"
)

var heartRateZoneCodes = []string{"Z1", "Z2", "Z3", "Z4", "Z5"}
var heartRateZoneLabels = []string{"Recovery", "Endurance", "Tempo", "Threshold", "VO2 Max"}

func FetchHeartRateZoneAnalysisByActivityTypeAndYear(year *int, activityTypes ...business.ActivityType) business.HeartRateZoneAnalysis {
	settings := normalizeHeartRateZoneSettings(activityProvider.GetHeartRateZoneSettings())
	activities := activityProvider.GetActivitiesByYearAndActivityTypes(year, activityTypes...)
	sort.Slice(activities, func(i, j int) bool {
		return activities[i].StartDateLocal < activities[j].StartDateLocal
	})

	resolvedSettings := resolveHeartRateZoneSettings(settings, activities)
	if resolvedSettings == nil {
		return emptyHeartRateZoneAnalysis(settings, nil)
	}

	summaries := make([]business.HeartRateZoneActivitySummary, 0, len(activities))
	for _, activity := range activities {
		summary := buildHeartRateZoneActivitySummary(activity, resolvedSettings)
		if summary != nil {
			summaries = append(summaries, *summary)
		}
	}

	if len(summaries) == 0 {
		return emptyHeartRateZoneAnalysis(settings, resolvedSettings)
	}

	globalTotals := make([]int, len(heartRateZoneCodes))
	totalTracked := 0
	easySeconds := 0
	hardSeconds := 0
	for _, summary := range summaries {
		totalTracked += summary.TotalTrackedSeconds
		easySeconds += summary.EasySeconds
		hardSeconds += summary.HardSeconds
		for idx, zone := range summary.Zones {
			globalTotals[idx] += zone.Seconds
		}
	}

	return business.HeartRateZoneAnalysis{
		Settings:            settings,
		ResolvedSettings:    resolvedSettings,
		HasHeartRateData:    true,
		TotalTrackedSeconds: totalTracked,
		EasyHardRatio:       calculateEasyHardRatio(easySeconds, hardSeconds),
		Zones:               buildHeartRateDistributions(globalTotals, totalTracked),
		Activities:          summaries,
		ByMonth: summarizeHeartRateByPeriod(summaries, func(summary business.HeartRateZoneActivitySummary) string {
			return safeDateSlice(summary.ActivityDate, 7)
		}),
		ByYear: summarizeHeartRateByPeriod(summaries, func(summary business.HeartRateZoneActivitySummary) string {
			return safeDateSlice(summary.ActivityDate, 4)
		}),
	}
}

func buildHeartRateZoneActivitySummary(activity *strava.Activity, settings *business.ResolvedHeartRateZoneSettings) *business.HeartRateZoneActivitySummary {
	if activity == nil || activity.Stream == nil || activity.Stream.HeartRate == nil {
		return nil
	}

	heartRateData := activity.Stream.HeartRate.Data
	timeData := activity.Stream.Time.Data
	sampleSize := minInt(len(heartRateData), len(timeData))
	if sampleSize < 2 {
		return nil
	}

	zoneTotals := make([]int, len(heartRateZoneCodes))
	totalTracked := 0
	for i := 0; i < sampleSize-1; i++ {
		hr := heartRateData[i]
		delta := timeData[i+1] - timeData[i]
		if hr <= 0 || delta <= 0 {
			continue
		}
		zoneIdx := resolveHeartRateZoneIndex(hr, settings)
		zoneTotals[zoneIdx] += delta
		totalTracked += delta
	}

	if totalTracked <= 0 {
		return nil
	}

	easy := zoneTotals[0] + zoneTotals[1]
	hard := zoneTotals[3] + zoneTotals[4]
	return &business.HeartRateZoneActivitySummary{
		Activity: business.ActivityShort{
			Id:   activity.Id,
			Name: activity.Name,
			Type: resolveActivityTypeForSummary(activity),
		},
		ActivityDate:        activity.StartDateLocal,
		TotalTrackedSeconds: totalTracked,
		EasySeconds:         easy,
		HardSeconds:         hard,
		EasyHardRatio:       calculateEasyHardRatio(easy, hard),
		Zones:               buildHeartRateDistributions(zoneTotals, totalTracked),
	}
}

func resolveActivityTypeForSummary(activity *strava.Activity) business.ActivityType {
	if activity == nil {
		return business.Ride
	}

	if value, ok := business.ActivityTypes[activity.SportType]; ok {
		return value
	}
	if value, ok := business.ActivityTypes[activity.Type]; ok {
		return value
	}
	return business.Ride
}

func summarizeHeartRateByPeriod(
	summaries []business.HeartRateZoneActivitySummary,
	keySelector func(summary business.HeartRateZoneActivitySummary) string,
) []business.HeartRateZonePeriodSummary {
	grouped := make(map[string][]business.HeartRateZoneActivitySummary)
	for _, summary := range summaries {
		key := keySelector(summary)
		grouped[key] = append(grouped[key], summary)
	}

	keys := make([]string, 0, len(grouped))
	for key := range grouped {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	result := make([]business.HeartRateZonePeriodSummary, 0, len(keys))
	for _, key := range keys {
		values := grouped[key]
		totals := make([]int, len(heartRateZoneCodes))
		totalTracked := 0
		easy := 0
		hard := 0
		for _, summary := range values {
			totalTracked += summary.TotalTrackedSeconds
			easy += summary.EasySeconds
			hard += summary.HardSeconds
			for idx, zone := range summary.Zones {
				totals[idx] += zone.Seconds
			}
		}

		result = append(result, business.HeartRateZonePeriodSummary{
			Period:              key,
			TotalTrackedSeconds: totalTracked,
			EasySeconds:         easy,
			HardSeconds:         hard,
			EasyHardRatio:       calculateEasyHardRatio(easy, hard),
			Zones:               buildHeartRateDistributions(totals, totalTracked),
		})
	}

	return result
}

func resolveHeartRateZoneSettings(
	settings business.HeartRateZoneSettings,
	activities []*strava.Activity,
) *business.ResolvedHeartRateZoneSettings {
	maxHr := settings.MaxHr
	thresholdHr := settings.ThresholdHr
	reserveHr := settings.ReserveHr

	if thresholdHr != nil {
		resolvedMax := thresholdHr
		source := business.HeartRateZoneSourceAthleteSettings
		if maxHr != nil {
			resolvedMax = maxHr
		} else if derived := deriveMaxHeartRateFromActivities(activities); derived != nil {
			resolvedMax = derived
			source = business.HeartRateZoneSourceDerivedFromData
		}

		return &business.ResolvedHeartRateZoneSettings{
			MaxHr:       *resolvedMax,
			ThresholdHr: thresholdHr,
			ReserveHr:   reserveHr,
			Method:      business.HeartRateZoneMethodThreshold,
			Source:      source,
		}
	}

	if maxHr != nil && reserveHr != nil && *reserveHr > 0 && *reserveHr < *maxHr {
		return &business.ResolvedHeartRateZoneSettings{
			MaxHr:       *maxHr,
			ThresholdHr: nil,
			ReserveHr:   reserveHr,
			Method:      business.HeartRateZoneMethodReserve,
			Source:      business.HeartRateZoneSourceAthleteSettings,
		}
	}

	if maxHr != nil {
		return &business.ResolvedHeartRateZoneSettings{
			MaxHr:       *maxHr,
			ThresholdHr: nil,
			ReserveHr:   reserveHr,
			Method:      business.HeartRateZoneMethodMax,
			Source:      business.HeartRateZoneSourceAthleteSettings,
		}
	}

	derivedMax := deriveMaxHeartRateFromActivities(activities)
	if derivedMax == nil {
		return nil
	}

	return &business.ResolvedHeartRateZoneSettings{
		MaxHr:       *derivedMax,
		ThresholdHr: nil,
		ReserveHr:   nil,
		Method:      business.HeartRateZoneMethodMax,
		Source:      business.HeartRateZoneSourceDerivedFromData,
	}
}

func deriveMaxHeartRateFromActivities(activities []*strava.Activity) *int {
	maxValue := 0
	for _, activity := range activities {
		if activity == nil {
			continue
		}
		current := int(activity.MaxHeartrate)
		if current > maxValue {
			maxValue = current
		}
	}
	if maxValue <= 0 {
		return nil
	}
	return &maxValue
}

func resolveHeartRateZoneIndex(hr int, settings *business.ResolvedHeartRateZoneSettings) int {
	if settings == nil {
		return 0
	}

	var z1Upper float64
	var z2Upper float64
	var z3Upper float64
	var z4Upper float64

	switch settings.Method {
	case business.HeartRateZoneMethodThreshold:
		threshold := float64(settings.MaxHr)
		if settings.ThresholdHr != nil {
			threshold = float64(*settings.ThresholdHr)
		}
		z1Upper = threshold * 0.81
		z2Upper = threshold * 0.89
		z3Upper = threshold * 0.93
		z4Upper = threshold * 0.99
	case business.HeartRateZoneMethodReserve:
		reserve := float64(0)
		if settings.ReserveHr != nil {
			reserve = float64(*settings.ReserveHr)
		}
		resting := math.Max(float64(settings.MaxHr)-reserve, 35)
		z1Upper = resting + reserve*0.60
		z2Upper = resting + reserve*0.70
		z3Upper = resting + reserve*0.80
		z4Upper = resting + reserve*0.90
	default:
		maxHr := float64(settings.MaxHr)
		z1Upper = maxHr * 0.60
		z2Upper = maxHr * 0.70
		z3Upper = maxHr * 0.80
		z4Upper = maxHr * 0.90
	}

	switch {
	case float64(hr) <= z1Upper:
		return 0
	case float64(hr) <= z2Upper:
		return 1
	case float64(hr) <= z3Upper:
		return 2
	case float64(hr) <= z4Upper:
		return 3
	default:
		return 4
	}
}

func calculateEasyHardRatio(easySeconds int, hardSeconds int) *float64 {
	if easySeconds <= 0 || hardSeconds <= 0 {
		return nil
	}
	ratio := math.Round((float64(easySeconds)/float64(hardSeconds))*100) / 100
	return &ratio
}

func buildHeartRateDistributions(zoneTotals []int, totalTracked int) []business.HeartRateZoneDistribution {
	distributions := make([]business.HeartRateZoneDistribution, 0, len(heartRateZoneCodes))
	for idx, zone := range heartRateZoneCodes {
		seconds := 0
		if idx < len(zoneTotals) {
			seconds = zoneTotals[idx]
		}
		percentage := 0.0
		if totalTracked > 0 {
			percentage = math.Round((float64(seconds)/float64(totalTracked))*10000) / 100
		}

		distributions = append(distributions, business.HeartRateZoneDistribution{
			Zone:       zone,
			Label:      heartRateZoneLabels[idx],
			Seconds:    seconds,
			Percentage: percentage,
		})
	}
	return distributions
}

func emptyHeartRateZoneAnalysis(
	settings business.HeartRateZoneSettings,
	resolvedSettings *business.ResolvedHeartRateZoneSettings,
) business.HeartRateZoneAnalysis {
	zeros := make([]int, len(heartRateZoneCodes))
	return business.HeartRateZoneAnalysis{
		Settings:            settings,
		ResolvedSettings:    resolvedSettings,
		HasHeartRateData:    false,
		TotalTrackedSeconds: 0,
		EasyHardRatio:       nil,
		Zones:               buildHeartRateDistributions(zeros, 0),
		Activities:          []business.HeartRateZoneActivitySummary{},
		ByMonth:             []business.HeartRateZonePeriodSummary{},
		ByYear:              []business.HeartRateZonePeriodSummary{},
	}
}

func normalizeHeartRateZoneSettings(settings business.HeartRateZoneSettings) business.HeartRateZoneSettings {
	return business.HeartRateZoneSettings{
		MaxHr:       normalizeIntPointer(settings.MaxHr),
		ThresholdHr: normalizeIntPointer(settings.ThresholdHr),
		ReserveHr:   normalizeIntPointer(settings.ReserveHr),
	}
}

func normalizeIntPointer(value *int) *int {
	if value == nil || *value <= 0 {
		return nil
	}
	return value
}

func safeDateSlice(value string, length int) string {
	if len(value) < length {
		return value
	}
	return value[:length]
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
