package badges

import (
	"testing"

	"mystravastats/internal/shared/domain/strava"
)

func TestHikingAdventureBadgeSet_ChecksOutdoorSpecificBadges(t *testing.T) {
	activities := []*strava.Activity{
		hikingBadgeActivity(1, "Summit loop", "2026-06-20T09:00:00Z", 21000, 650, 2350, []float64{45.1001, 6.1001}),
		hikingBadgeActivity(2, "Sunday recovery", "2026-06-21T09:00:00Z", 8000, 120, 1200, []float64{45.2001, 6.2001}),
		hikingBadgeActivity(3, "Highest trail", "2026-07-04T09:00:00Z", 9000, 450, 2650, []float64{45.3001, 6.3001}),
	}

	results := HikingAdventureBadgeSet.Check(activities)
	completed := map[string]int{}
	for _, result := range results {
		if result.IsCompleted {
			completed[result.Badge.String()] = len(result.Activities)
		}
	}

	expectedLabels := []string{
		"Summit Day",
		"Back-to-back Hiking Weekend",
		"High Point PR",
		"New Trail",
	}
	for _, label := range expectedLabels {
		if completed[label] == 0 {
			t.Fatalf("expected %q badge to be completed, got completed=%v", label, completed)
		}
	}
	if completed["High Point PR"] != 1 {
		t.Fatalf("expected High Point PR to expose one representative activity, got %d", completed["High Point PR"])
	}
}

func TestHikingDistanceAndElevationLabels_AreOutdoorSpecific(t *testing.T) {
	if DistanceHikeLevel2.Label != "Long Hike 15 km" {
		t.Fatalf("unexpected hike distance label %q", DistanceHikeLevel2.Label)
	}
	if ElevationHikeLevel1.Label != "Vertical Kilometer" {
		t.Fatalf("unexpected hike elevation label %q", ElevationHikeLevel1.Label)
	}
	if ElevationHikeBadgeSet.Name != "Hiking elevation" {
		t.Fatalf("expected Hiking elevation badge set, got %q", ElevationHikeBadgeSet.Name)
	}
}

func hikingBadgeActivity(
	id int64,
	name string,
	startDateLocal string,
	distance float64,
	elevationGain float64,
	elevHigh float64,
	startLatLng []float64,
) *strava.Activity {
	return &strava.Activity{
		Id:                 id,
		Name:               name,
		Type:               "Hike",
		SportType:          "Hike",
		StartDateLocal:     startDateLocal,
		StartLatlng:        startLatLng,
		Distance:           distance,
		TotalElevationGain: elevationGain,
		ElevHigh:           elevHigh,
	}
}
