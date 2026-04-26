package infrastructure

import (
	"math"
	"mystravastats/internal/shared/domain/business"
	"mystravastats/internal/shared/domain/strava"
	"testing"
	"time"
)

func TestBuildAnnualGoals_ProjectsCurrentYearAndClassifiesStatus(t *testing.T) {
	// GIVEN
	targetDistance := 50.0
	targetActivities := 100
	now := time.Date(2026, time.April, 10, 12, 0, 0, 0, time.UTC)
	activities := []*strava.Activity{
		annualGoalActivity(1, "2026-01-01T08:00:00Z", 12000, 100, 3600),
		annualGoalActivity(2, "2026-01-03T08:00:00Z", 8000, 50, 1800),
	}

	// WHEN
	result := buildAnnualGoals(2026, "Ride", business.AnnualGoalTargets{
		DistanceKm: &targetDistance,
		Activities: &targetActivities,
	}, activities, now)

	// THEN
	distance := annualGoalProgressByMetric(result, business.AnnualGoalMetricDistanceKm)
	if distance.Current != 20 {
		t.Fatalf("expected current distance 20km, got %.1f", distance.Current)
	}
	if distance.ProjectedEndOfYear != 73 {
		t.Fatalf("expected projected distance 73km, got %.1f", distance.ProjectedEndOfYear)
	}
	if distance.RequiredPace != 0.1 {
		t.Fatalf("expected required pace 0.1km/day, got %.1f", distance.RequiredPace)
	}
	if distance.Status != business.AnnualGoalStatusAhead {
		t.Fatalf("expected distance status AHEAD, got %s", distance.Status)
	}

	activitiesProgress := annualGoalProgressByMetric(result, business.AnnualGoalMetricActivities)
	if activitiesProgress.Status != business.AnnualGoalStatusBehind {
		t.Fatalf("expected activities status BEHIND, got %s", activitiesProgress.Status)
	}
}

func TestBuildAnnualGoals_ReturnsNotSetRowsForMissingTargets(t *testing.T) {
	// GIVEN
	now := time.Date(2026, time.June, 1, 12, 0, 0, 0, time.UTC)

	// WHEN
	result := buildAnnualGoals(2026, "Ride", business.AnnualGoalTargets{}, nil, now)

	// THEN
	if len(result.Progress) != 6 {
		t.Fatalf("expected 6 annual goal rows, got %d", len(result.Progress))
	}
	for _, progress := range result.Progress {
		if progress.Status != business.AnnualGoalStatusNotSet {
			t.Fatalf("expected NOT_SET status for %s, got %s", progress.Metric, progress.Status)
		}
	}
}

func TestBuildAnnualGoals_ComputesAnnualEddingtonForSelectedYear(t *testing.T) {
	// GIVEN
	targetEddington := 2
	now := time.Date(2026, time.December, 31, 12, 0, 0, 0, time.UTC)
	activities := []*strava.Activity{
		annualGoalActivity(1, "2026-01-01T08:00:00Z", 3000, 0, 900),
		annualGoalActivity(2, "2026-01-02T08:00:00Z", 2000, 0, 900),
		annualGoalActivity(3, "2026-01-03T08:00:00Z", 1000, 0, 900),
	}

	// WHEN
	result := buildAnnualGoals(2026, "Ride", business.AnnualGoalTargets{
		Eddington: &targetEddington,
	}, activities, now)

	// THEN
	eddington := annualGoalProgressByMetric(result, business.AnnualGoalMetricEddington)
	if eddington.Current != 2 {
		t.Fatalf("expected Eddington 2, got %.1f", eddington.Current)
	}
	if eddington.Status != business.AnnualGoalStatusOnTrack {
		t.Fatalf("expected Eddington ON_TRACK, got %s", eddington.Status)
	}
}

func TestBuildAnnualGoals_AddsMonthlyTrendAndAdjustmentSuggestion(t *testing.T) {
	// GIVEN
	targetDistance := 500.0
	now := time.Date(2026, time.April, 10, 12, 0, 0, 0, time.UTC)
	activities := []*strava.Activity{
		annualGoalActivity(1, "2026-01-01T08:00:00Z", 10000, 100, 3600),
		annualGoalActivity(2, "2026-03-20T08:00:00Z", 20000, 200, 7200),
		annualGoalActivity(3, "2026-04-05T08:00:00Z", 5000, 50, 1800),
	}

	// WHEN
	result := buildAnnualGoals(2026, "Ride", business.AnnualGoalTargets{
		DistanceKm: &targetDistance,
	}, activities, now)

	// THEN
	distance := annualGoalProgressByMetric(result, business.AnnualGoalMetricDistanceKm)
	if distance.Last30Days != 25 {
		t.Fatalf("expected 25km over last 30 days, got %.1f", distance.Last30Days)
	}
	if distance.Last30DaysWeeklyPace != 5.8 {
		t.Fatalf("expected recent weekly pace 5.8km/week, got %.1f", distance.Last30DaysWeeklyPace)
	}
	if distance.RequiredWeeklyPace != 12.3 {
		t.Fatalf("expected required weekly pace 12.3km/week, got %.1f", distance.RequiredWeeklyPace)
	}
	if distance.WeeklyPaceGap != 6.4 {
		t.Fatalf("expected weekly pace gap 6.4km/week, got %.1f", distance.WeeklyPaceGap)
	}
	if distance.SuggestedTarget == nil {
		t.Fatalf("expected suggested target 127.7km, got nil")
	}
	if math.Abs(*distance.SuggestedTarget-127.7) > 0.001 {
		t.Fatalf("expected suggested target 127.7km, got %.3f", *distance.SuggestedTarget)
	}
	if len(distance.Monthly) != 12 {
		t.Fatalf("expected 12 monthly entries, got %d", len(distance.Monthly))
	}
	if distance.Monthly[0].Value != 10 || distance.Monthly[2].Value != 20 || distance.Monthly[3].Cumulative != 35 {
		t.Fatalf("unexpected monthly distance breakdown: %#v", distance.Monthly[:4])
	}
}

func annualGoalProgressByMetric(result business.AnnualGoals, metric business.AnnualGoalMetric) business.AnnualGoalProgress {
	for _, progress := range result.Progress {
		if progress.Metric == metric {
			return progress
		}
	}
	return business.AnnualGoalProgress{}
}

func annualGoalActivity(id int64, startDateLocal string, distanceMeters float64, elevationMeters float64, movingTimeSeconds int) *strava.Activity {
	return &strava.Activity{
		Id:                 id,
		StartDateLocal:     startDateLocal,
		Distance:           distanceMeters,
		TotalElevationGain: elevationMeters,
		MovingTime:         movingTimeSeconds,
	}
}
