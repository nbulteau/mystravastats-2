package infrastructure

import (
	"log"
	"math"
	"mystravastats/internal/platform/activityprovider"
	"mystravastats/internal/shared/domain/business"
	"mystravastats/internal/shared/domain/strava"
	"mystravastats/internal/shared/infrastructure/localrepository"
	"sort"
	"strconv"
	"strings"
	"time"
)

type annualGoalMetricDefinition struct {
	metric           business.AnnualGoalMetric
	label            string
	unit             string
	requiredPaceUnit string
	current          float64
	target           *float64
	monthlyValues    []float64
	last30DaysValue  float64
}

func computeAnnualGoals(year int, targets business.AnnualGoalTargets, activityTypes ...business.ActivityType) business.AnnualGoals {
	log.Printf("Get annual goals for year %d and activity type %s", year, activityTypes)

	normalizedTargets := normalizeAnnualGoalTargets(targets)
	activities := activityprovider.Get().GetActivitiesByYearAndActivityTypes(&year, activityTypes...)
	return buildAnnualGoals(year, activityTypeKey(activityTypes...), normalizedTargets, activities, time.Now())
}

func saveAnnualGoals(year int, targets business.AnnualGoalTargets, activityTypes ...business.ActivityType) business.AnnualGoals {
	normalizedTargets := normalizeAnnualGoalTargets(targets)
	key := annualGoalTargetsKey(year, activityTypes...)
	provider := activityprovider.Get()
	repository := localrepository.NewStravaRepository(provider.CacheRootPath())
	repository.SaveAnnualGoalTargets(provider.ClientID(), key, normalizedTargets)

	activities := provider.GetActivitiesByYearAndActivityTypes(&year, activityTypes...)
	return buildAnnualGoals(year, activityTypeKey(activityTypes...), normalizedTargets, activities, time.Now())
}

func loadAnnualGoals(year int, activityTypes ...business.ActivityType) business.AnnualGoals {
	key := annualGoalTargetsKey(year, activityTypes...)
	provider := activityprovider.Get()
	repository := localrepository.NewStravaRepository(provider.CacheRootPath())
	targets := repository.LoadAnnualGoalTargets(provider.ClientID(), key)

	activities := provider.GetActivitiesByYearAndActivityTypes(&year, activityTypes...)
	return buildAnnualGoals(year, activityTypeKey(activityTypes...), normalizeAnnualGoalTargets(targets), activities, time.Now())
}

func buildAnnualGoals(
	year int,
	activityTypeKey string,
	targets business.AnnualGoalTargets,
	activities []*strava.Activity,
	now time.Time,
) business.AnnualGoals {
	current := annualGoalCurrentValues(activities)
	eddington := computeEddingtonFromDailyTotals(annualGoalDailyDistanceTotals(activities))
	current.eddington = float64(eddington.Number)
	monthlyValues := annualGoalMonthlyValues(activities)
	last30DaysValues, last30DaysWindowDays := annualGoalLast30DaysValues(year, activities, now)

	definitions := []annualGoalMetricDefinition{
		{
			metric:           business.AnnualGoalMetricDistanceKm,
			label:            "Distance",
			unit:             "km",
			requiredPaceUnit: "km/day",
			current:          current.distanceKm,
			target:           floatTarget(targets.DistanceKm),
			monthlyValues:    monthlyValues[business.AnnualGoalMetricDistanceKm],
			last30DaysValue:  last30DaysValues.distanceKm,
		},
		{
			metric:           business.AnnualGoalMetricElevationMeters,
			label:            "Elevation",
			unit:             "m",
			requiredPaceUnit: "m/day",
			current:          current.elevationMeters,
			target:           intTarget(targets.ElevationMeters),
			monthlyValues:    monthlyValues[business.AnnualGoalMetricElevationMeters],
			last30DaysValue:  last30DaysValues.elevationMeters,
		},
		{
			metric:           business.AnnualGoalMetricMovingTimeSeconds,
			label:            "Moving time",
			unit:             "s",
			requiredPaceUnit: "s/day",
			current:          current.movingTimeSeconds,
			target:           intTarget(targets.MovingTimeSeconds),
			monthlyValues:    monthlyValues[business.AnnualGoalMetricMovingTimeSeconds],
			last30DaysValue:  last30DaysValues.movingTimeSeconds,
		},
		{
			metric:           business.AnnualGoalMetricActivities,
			label:            "Activities",
			unit:             "activities",
			requiredPaceUnit: "activities/day",
			current:          current.activities,
			target:           intTarget(targets.Activities),
			monthlyValues:    monthlyValues[business.AnnualGoalMetricActivities],
			last30DaysValue:  last30DaysValues.activities,
		},
		{
			metric:           business.AnnualGoalMetricActiveDays,
			label:            "Active days",
			unit:             "days",
			requiredPaceUnit: "days/day",
			current:          current.activeDays,
			target:           intTarget(targets.ActiveDays),
			monthlyValues:    monthlyValues[business.AnnualGoalMetricActiveDays],
			last30DaysValue:  last30DaysValues.activeDays,
		},
		{
			metric:           business.AnnualGoalMetricEddington,
			label:            "Eddington",
			unit:             "level",
			requiredPaceUnit: "level/day",
			current:          current.eddington,
			target:           intTarget(targets.Eddington),
			monthlyValues:    monthlyValues[business.AnnualGoalMetricEddington],
			last30DaysValue:  last30DaysValues.eddington,
		},
	}

	progress := make([]business.AnnualGoalProgress, 0, len(definitions))
	for _, definition := range definitions {
		progress = append(progress, buildAnnualGoalProgress(year, now, definition, last30DaysWindowDays))
	}

	return business.AnnualGoals{
		Year:            year,
		ActivityTypeKey: activityTypeKey,
		Targets:         targets,
		Progress:        progress,
	}
}

type annualGoalValues struct {
	distanceKm        float64
	elevationMeters   float64
	movingTimeSeconds float64
	activities        float64
	activeDays        float64
	eddington         float64
}

func annualGoalCurrentValues(activities []*strava.Activity) annualGoalValues {
	return annualGoalValues{
		distanceKm:        sumDistance(activities),
		elevationMeters:   float64(sumElevation(activities)),
		movingTimeSeconds: float64(sumMovingTime(activities)),
		activities:        float64(len(activities)),
		activeDays:        float64(countActiveDays(activities)),
	}
}

func annualGoalDailyDistanceTotals(activities []*strava.Activity) map[string]int {
	result := make(map[string]int)
	for _, activity := range activities {
		if activity == nil || len(activity.StartDateLocal) < 10 {
			continue
		}
		day := strings.Split(activity.StartDateLocal, "T")[0]
		result[day] += int(activity.Distance / 1000)
	}
	return result
}

func annualGoalMonthlyValues(activities []*strava.Activity) map[business.AnnualGoalMetric][]float64 {
	values := map[business.AnnualGoalMetric][]float64{
		business.AnnualGoalMetricDistanceKm:        make([]float64, 12),
		business.AnnualGoalMetricElevationMeters:   make([]float64, 12),
		business.AnnualGoalMetricMovingTimeSeconds: make([]float64, 12),
		business.AnnualGoalMetricActivities:        make([]float64, 12),
		business.AnnualGoalMetricActiveDays:        make([]float64, 12),
		business.AnnualGoalMetricEddington:         make([]float64, 12),
	}
	activeDaysByMonth := make([]map[string]struct{}, 12)
	dailyDistanceByMonth := make([]map[string]int, 12)
	for month := 0; month < 12; month++ {
		activeDaysByMonth[month] = map[string]struct{}{}
		dailyDistanceByMonth[month] = map[string]int{}
	}

	for _, activity := range activities {
		activityDate, ok := annualGoalActivityDate(activity)
		if !ok {
			continue
		}
		monthIndex := int(activityDate.Month()) - 1
		day := activityDate.Format("2006-01-02")
		values[business.AnnualGoalMetricDistanceKm][monthIndex] += activity.Distance / 1000
		values[business.AnnualGoalMetricElevationMeters][monthIndex] += activity.TotalElevationGain
		values[business.AnnualGoalMetricMovingTimeSeconds][monthIndex] += float64(activityMovingTimeSeconds(activity))
		values[business.AnnualGoalMetricActivities][monthIndex]++
		activeDaysByMonth[monthIndex][day] = struct{}{}
		dailyDistanceByMonth[monthIndex][day] += int(activity.Distance / 1000)
	}

	for month := 0; month < 12; month++ {
		values[business.AnnualGoalMetricActiveDays][month] = float64(len(activeDaysByMonth[month]))
		values[business.AnnualGoalMetricEddington][month] = float64(computeEddingtonFromDailyTotals(dailyDistanceByMonth[month]).Number)
	}
	return values
}

func annualGoalLast30DaysValues(year int, activities []*strava.Activity, now time.Time) (annualGoalValues, int) {
	windowStart, windowEnd, windowDays := annualGoalLast30DaysWindow(year, now)
	if windowDays <= 0 {
		return annualGoalValues{}, 0
	}

	filtered := make([]*strava.Activity, 0)
	for _, activity := range activities {
		activityDate, ok := annualGoalActivityDate(activity)
		if !ok || activityDate.Before(windowStart) || activityDate.After(windowEnd) {
			continue
		}
		filtered = append(filtered, activity)
	}
	values := annualGoalCurrentValues(filtered)
	eddington := computeEddingtonFromDailyTotals(annualGoalDailyDistanceTotals(filtered))
	values.eddington = float64(eddington.Number)
	return values, windowDays
}

func annualGoalLast30DaysWindow(year int, now time.Time) (time.Time, time.Time, int) {
	if year > now.Year() {
		return time.Time{}, time.Time{}, 0
	}

	yearStart := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC)
	windowEnd := time.Date(year, time.December, 31, 0, 0, 0, 0, time.UTC)
	if year == now.Year() {
		windowEnd = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	}
	windowStart := windowEnd.AddDate(0, 0, -29)
	if windowStart.Before(yearStart) {
		windowStart = yearStart
	}

	windowDays := int(windowEnd.Sub(windowStart).Hours()/24) + 1
	return windowStart, windowEnd, windowDays
}

func annualGoalActivityDate(activity *strava.Activity) (time.Time, bool) {
	if activity == nil {
		return time.Time{}, false
	}
	date := activity.StartDateLocal
	if len(date) < 10 {
		date = activity.StartDate
	}
	if len(date) < 10 {
		return time.Time{}, false
	}
	parsed, err := time.Parse("2006-01-02", date[:10])
	if err != nil {
		return time.Time{}, false
	}
	return parsed, true
}

func activityMovingTimeSeconds(activity *strava.Activity) int {
	if activity.MovingTime > 0 {
		return activity.MovingTime
	}
	return activity.ElapsedTime
}

func buildAnnualGoalProgress(year int, now time.Time, definition annualGoalMetricDefinition, last30DaysWindowDays int) business.AnnualGoalProgress {
	elapsedDays := elapsedDaysForAnnualGoal(year, now)
	remainingDays := remainingDaysForAnnualGoal(year, now)
	expectedProgress := annualExpectedProgressPercent(year, now)
	projected := definition.current
	if year == now.Year() && elapsedDays > 0 {
		projected = definition.current / float64(elapsedDays) * float64(daysInYear(year))
	}
	last30DaysWeeklyPace := 0.0
	if last30DaysWindowDays > 0 {
		last30DaysWeeklyPace = definition.last30DaysValue / float64(last30DaysWindowDays) * 7
	}

	target := 0.0
	if definition.target != nil {
		target = *definition.target
	}
	if target <= 0 {
		return business.AnnualGoalProgress{
			Metric:                  definition.metric,
			Label:                   definition.label,
			Unit:                    definition.unit,
			Current:                 roundAnnualGoalValue(definition.current),
			Target:                  0,
			ProgressPercent:         0,
			ExpectedProgressPercent: roundAnnualGoalValue(expectedProgress),
			ProjectedEndOfYear:      roundAnnualGoalValue(projected),
			RequiredPace:            0,
			RequiredPaceUnit:        definition.requiredPaceUnit,
			RequiredWeeklyPace:      0,
			Last30Days:              roundAnnualGoalValue(definition.last30DaysValue),
			Last30DaysWeeklyPace:    roundAnnualGoalValue(last30DaysWeeklyPace),
			WeeklyPaceGap:           0,
			SuggestedTarget:         nil,
			Monthly:                 buildAnnualGoalMonthlyProgress(year, definition.monthlyValues, 0),
			Status:                  business.AnnualGoalStatusNotSet,
		}
	}

	progressPercent := definition.current / target * 100.0
	requiredPace := 0.0
	if remainingDays > 0 {
		requiredPace = math.Max(target-definition.current, 0) / float64(remainingDays)
	}
	requiredWeeklyPace := requiredPace * 7
	weeklyPaceGap := math.Max(requiredWeeklyPace-last30DaysWeeklyPace, 0)
	suggestedTarget := suggestedAnnualGoalTarget(year, now, target, projected, progressPercent, expectedProgress)

	return business.AnnualGoalProgress{
		Metric:                  definition.metric,
		Label:                   definition.label,
		Unit:                    definition.unit,
		Current:                 roundAnnualGoalValue(definition.current),
		Target:                  roundAnnualGoalValue(target),
		ProgressPercent:         roundAnnualGoalValue(progressPercent),
		ExpectedProgressPercent: roundAnnualGoalValue(expectedProgress),
		ProjectedEndOfYear:      roundAnnualGoalValue(projected),
		RequiredPace:            roundAnnualGoalValue(requiredPace),
		RequiredPaceUnit:        definition.requiredPaceUnit,
		RequiredWeeklyPace:      roundAnnualGoalValue(requiredWeeklyPace),
		Last30Days:              roundAnnualGoalValue(definition.last30DaysValue),
		Last30DaysWeeklyPace:    roundAnnualGoalValue(last30DaysWeeklyPace),
		WeeklyPaceGap:           roundAnnualGoalValue(weeklyPaceGap),
		SuggestedTarget:         suggestedTarget,
		Monthly:                 buildAnnualGoalMonthlyProgress(year, definition.monthlyValues, target),
		Status:                  annualGoalStatus(progressPercent, expectedProgress),
	}
}

func suggestedAnnualGoalTarget(year int, now time.Time, target float64, projected float64, progressPercent float64, expectedProgressPercent float64) *float64 {
	if year != now.Year() || target <= 0 || projected <= 0 || progressPercent >= expectedProgressPercent-5 {
		return nil
	}
	if projected >= target*0.9 {
		return nil
	}
	suggested := roundAnnualGoalValue(math.Max(projected, 0))
	return &suggested
}

func buildAnnualGoalMonthlyProgress(year int, monthlyValues []float64, target float64) []business.AnnualGoalMonth {
	months := make([]business.AnnualGoalMonth, 0, 12)
	cumulative := 0.0
	for month := 1; month <= 12; month++ {
		value := 0.0
		if len(monthlyValues) >= month {
			value = monthlyValues[month-1]
		}
		cumulative += value
		expectedCumulative := 0.0
		if target > 0 {
			expectedCumulative = target * float64(dayOfYearAtMonthEnd(year, time.Month(month))) / float64(daysInYear(year))
		}
		months = append(months, business.AnnualGoalMonth{
			Month:              month,
			Value:              roundAnnualGoalValue(value),
			Cumulative:         roundAnnualGoalValue(cumulative),
			ExpectedCumulative: roundAnnualGoalValue(expectedCumulative),
		})
	}
	return months
}

func elapsedDaysForAnnualGoal(year int, now time.Time) int {
	switch {
	case year < now.Year():
		return daysInYear(year)
	case year > now.Year():
		return 0
	default:
		return now.YearDay()
	}
}

func remainingDaysForAnnualGoal(year int, now time.Time) int {
	switch {
	case year < now.Year():
		return 0
	case year > now.Year():
		return daysInYear(year)
	default:
		return daysInYear(year) - now.YearDay()
	}
}

func annualExpectedProgressPercent(year int, now time.Time) float64 {
	elapsedDays := elapsedDaysForAnnualGoal(year, now)
	if elapsedDays <= 0 {
		return 0
	}
	return float64(elapsedDays) / float64(daysInYear(year)) * 100.0
}

func dayOfYearAtMonthEnd(year int, month time.Month) int {
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).YearDay()
}

func annualGoalStatus(progressPercent float64, expectedProgressPercent float64) business.AnnualGoalStatus {
	switch {
	case progressPercent >= expectedProgressPercent+5:
		return business.AnnualGoalStatusAhead
	case progressPercent >= expectedProgressPercent-5:
		return business.AnnualGoalStatusOnTrack
	default:
		return business.AnnualGoalStatusBehind
	}
}

func annualGoalTargetsKey(year int, activityTypes ...business.ActivityType) string {
	return strconv.Itoa(year) + ":" + activityTypeKey(activityTypes...)
}

func activityTypeKey(activityTypes ...business.ActivityType) string {
	names := make([]string, 0, len(activityTypes))
	for _, activityType := range activityTypes {
		names = append(names, activityType.String())
	}
	sort.Strings(names)
	return strings.Join(names, "_")
}

func normalizeAnnualGoalTargets(targets business.AnnualGoalTargets) business.AnnualGoalTargets {
	return business.AnnualGoalTargets{
		DistanceKm:        positiveFloatPointer(targets.DistanceKm),
		ElevationMeters:   positiveIntPointer(targets.ElevationMeters),
		MovingTimeSeconds: positiveIntPointer(targets.MovingTimeSeconds),
		Activities:        positiveIntPointer(targets.Activities),
		ActiveDays:        positiveIntPointer(targets.ActiveDays),
		Eddington:         positiveIntPointer(targets.Eddington),
	}
}

func positiveFloatPointer(value *float64) *float64 {
	if value == nil || *value <= 0 {
		return nil
	}
	normalized := *value
	return &normalized
}

func positiveIntPointer(value *int) *int {
	if value == nil || *value <= 0 {
		return nil
	}
	normalized := *value
	return &normalized
}

func floatTarget(value *float64) *float64 {
	if value == nil {
		return nil
	}
	target := *value
	return &target
}

func intTarget(value *int) *float64 {
	if value == nil {
		return nil
	}
	target := float64(*value)
	return &target
}

func roundAnnualGoalValue(value float64) float64 {
	return math.Round(value*10) / 10
}
