package infrastructure

import (
	"mystravastats/internal/shared/domain/strava"
	"testing"
	"time"
)

func TestBuildRoutingHistoryProfileFromActivities_FiltersByRouteType(t *testing.T) {
	// GIVEN
	now := time.Date(2026, time.April, 20, 10, 0, 0, 0, time.UTC)
	activities := []*strava.Activity{
		buildHistoryProfileActivity(
			1,
			"2026-04-15T08:00:00Z",
			"GravelRide",
			"Ride",
			[][]float64{{45.00, 6.00}, {45.01, 6.02}, {45.02, 6.03}},
		),
		buildHistoryProfileActivity(
			2,
			"2026-04-16T08:00:00Z",
			"Ride",
			"Ride",
			[][]float64{{45.10, 6.10}, {45.11, 6.11}, {45.12, 6.12}},
		),
	}

	// WHEN
	profile := buildRoutingHistoryProfileFromActivities(activities, "GRAVEL", now, 75)

	// THEN
	if profile == nil {
		t.Fatal("expected non-nil history profile for GRAVEL")
	}
	if profile.RouteType != "GRAVEL" {
		t.Fatalf("expected routeType GRAVEL, got %q", profile.RouteType)
	}
	if profile.ActivityCount != 1 {
		t.Fatalf("expected one contributing activity, got %d", profile.ActivityCount)
	}
	if len(profile.AxisScores) == 0 {
		t.Fatal("expected non-empty axis scores")
	}
	if len(profile.ZoneScores) == 0 {
		t.Fatal("expected non-empty zone scores")
	}
}

func TestBuildRoutingHistoryProfileFromActivities_AppliesRecencyDecay(t *testing.T) {
	// GIVEN
	now := time.Date(2026, time.April, 20, 10, 0, 0, 0, time.UTC)
	recentTrack := [][]float64{{45.00, 6.00}, {45.02, 6.00}}
	oldTrack := [][]float64{{46.00, 7.00}, {46.02, 7.00}}
	activities := []*strava.Activity{
		buildHistoryProfileActivity(11, "2026-04-10T08:00:00Z", "Ride", "Ride", recentTrack),
		buildHistoryProfileActivity(12, "2025-04-10T08:00:00Z", "Ride", "Ride", oldTrack),
	}
	recentAxis := historyAxisKey(recentTrack[0][0], recentTrack[0][1], recentTrack[1][0], recentTrack[1][1])
	oldAxis := historyAxisKey(oldTrack[0][0], oldTrack[0][1], oldTrack[1][0], oldTrack[1][1])

	// WHEN
	profile := buildRoutingHistoryProfileFromActivities(activities, "RIDE", now, 75)

	// THEN
	if profile == nil {
		t.Fatal("expected non-nil history profile for RIDE")
	}
	recentScore, recentOK := profile.AxisScores[recentAxis]
	oldScore, oldOK := profile.AxisScores[oldAxis]
	if !recentOK || !oldOK {
		t.Fatalf("expected both recent and old axis scores, got keys=%v", profile.AxisScores)
	}
	if recentScore <= oldScore {
		t.Fatalf("expected recent axis score > old axis score, recent=%.2f old=%.2f", recentScore, oldScore)
	}
}

func TestBuildRoutingHistoryProfileFromActivities_ReturnsNilWhenNoMatchingTrack(t *testing.T) {
	// GIVEN
	now := time.Date(2026, time.April, 20, 10, 0, 0, 0, time.UTC)
	activities := []*strava.Activity{
		buildHistoryProfileActivity(21, "2026-03-01T08:00:00Z", "Ride", "Ride", [][]float64{{45.00, 6.00}, {45.01, 6.01}}),
		{Id: 22, SportType: "Hike", Type: "Hike"},
	}

	// WHEN
	profile := buildRoutingHistoryProfileFromActivities(activities, "HIKE", now, 75)

	// THEN
	if profile != nil {
		t.Fatalf("expected nil history profile when no matching tracked activity exists, got %+v", profile)
	}
}

func buildHistoryProfileActivity(
	id int64,
	startDateLocal string,
	sportType string,
	legacyType string,
	track [][]float64,
) *strava.Activity {
	return &strava.Activity{
		Id:             id,
		SportType:      sportType,
		Type:           legacyType,
		StartDate:      startDateLocal,
		StartDateLocal: startDateLocal,
		Distance:       10000.0,
		MovingTime:     3600,
		ElapsedTime:    3600,
		Stream: &strava.Stream{
			LatLng: &strava.LatLngStream{Data: track},
		},
	}
}
