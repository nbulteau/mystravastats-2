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

	definitions := []annualGoalMetricDefinition{
		{
			metric:           business.AnnualGoalMetricDistanceKm,
			label:            "Distance",
			unit:             "km",
			requiredPaceUnit: "km/day",
			current:          current.distanceKm,
			target:           floatTarget(targets.DistanceKm),
		},
		{
			metric:           business.AnnualGoalMetricElevationMeters,
			label:            "Elevation",
			unit:             "m",
			requiredPaceUnit: "m/day",
			current:          current.elevationMeters,
			target:           intTarget(targets.ElevationMeters),
		},
		{
			metric:           business.AnnualGoalMetricMovingTimeSeconds,
			label:            "Moving time",
			unit:             "s",
			requiredPaceUnit: "s/day",
			current:          current.movingTimeSeconds,
			target:           intTarget(targets.MovingTimeSeconds),
		},
		{
			metric:           business.AnnualGoalMetricActivities,
			label:            "Activities",
			unit:             "activities",
			requiredPaceUnit: "activities/day",
			current:          current.activities,
			target:           intTarget(targets.Activities),
		},
		{
			metric:           business.AnnualGoalMetricActiveDays,
			label:            "Active days",
			unit:             "days",
			requiredPaceUnit: "days/day",
			current:          current.activeDays,
			target:           intTarget(targets.ActiveDays),
		},
		{
			metric:           business.AnnualGoalMetricEddington,
			label:            "Eddington",
			unit:             "level",
			requiredPaceUnit: "level/day",
			current:          current.eddington,
			target:           intTarget(targets.Eddington),
		},
	}

	progress := make([]business.AnnualGoalProgress, 0, len(definitions))
	for _, definition := range definitions {
		progress = append(progress, buildAnnualGoalProgress(year, now, definition))
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

func buildAnnualGoalProgress(year int, now time.Time, definition annualGoalMetricDefinition) business.AnnualGoalProgress {
	elapsedDays := elapsedDaysForAnnualGoal(year, now)
	remainingDays := remainingDaysForAnnualGoal(year, now)
	expectedProgress := annualExpectedProgressPercent(year, now)
	projected := definition.current
	if year == now.Year() && elapsedDays > 0 {
		projected = definition.current / float64(elapsedDays) * float64(daysInYear(year))
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
			Status:                  business.AnnualGoalStatusNotSet,
		}
	}

	progressPercent := definition.current / target * 100.0
	requiredPace := 0.0
	if remainingDays > 0 {
		requiredPace = math.Max(target-definition.current, 0) / float64(remainingDays)
	}

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
		Status:                  annualGoalStatus(progressPercent, expectedProgress),
	}
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
