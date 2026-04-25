package infrastructure

import (
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
