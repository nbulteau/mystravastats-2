package infrastructure

import (
	"mystravastats/internal/shared/domain/business"
	"mystravastats/internal/shared/domain/strava"
	"testing"
)

func TestResolveMapTrackActivityType_ReturnsCommuteWhenCommuteIsTrue(t *testing.T) {
	// GIVEN
	activity := &strava.Activity{
		Commute:   true,
		SportType: business.Ride.String(),
		Type:      business.Ride.String(),
	}

	// WHEN
	got := resolveMapTrackActivityType(activity)

	// THEN
	if got != business.Commute.String() {
		t.Fatalf("expected %q, got %q", business.Commute.String(), got)
	}
}

func TestResolveMapTrackActivityType_ReturnsSportTypeWhenNotCommute(t *testing.T) {
	// GIVEN
	activity := &strava.Activity{
		Commute:   false,
		SportType: business.MountainBikeRide.String(),
		Type:      business.Ride.String(),
	}

	// WHEN
	got := resolveMapTrackActivityType(activity)

	// THEN
	if got != business.MountainBikeRide.String() {
		t.Fatalf("expected %q, got %q", business.MountainBikeRide.String(), got)
	}
}

func TestResolveMapTrackActivityType_FallsBackToTypeThenRide(t *testing.T) {
	// GIVEN
	activityWithType := &strava.Activity{
		Type: business.VirtualRide.String(),
	}
	activityWithoutType := &strava.Activity{}

	// WHEN
	gotWithType := resolveMapTrackActivityType(activityWithType)
	gotWithoutType := resolveMapTrackActivityType(activityWithoutType)

	// THEN
	if gotWithType != business.VirtualRide.String() {
		t.Fatalf("expected %q, got %q", business.VirtualRide.String(), gotWithType)
	}
	if gotWithoutType != business.Ride.String() {
		t.Fatalf("expected %q, got %q", business.Ride.String(), gotWithoutType)
	}
}
