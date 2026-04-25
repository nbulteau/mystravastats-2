package infrastructure

import (
	"testing"

	"mystravastats/domain/badges"
	"mystravastats/internal/shared/domain/business"
	"mystravastats/internal/shared/domain/strava"
)

func TestCheckGeneralBadges_UsesCyclingFamilyForRideVariants(t *testing.T) {
	results := checkGeneralBadges(
		[]*strava.Activity{
			activity("GravelRide", 60000, 1200, 7200),
		},
		business.GravelRide,
		business.MountainBikeRide,
		business.Ride,
	)

	assertCompletedBadge(t, results, isDistanceBadge)
	assertCompletedBadge(t, results, isElevationBadge)
	assertCompletedBadge(t, results, isMovingTimeBadge)
}

func TestCheckGeneralBadges_UsesRunningFamilyForTrailRun(t *testing.T) {
	results := checkGeneralBadges(
		[]*strava.Activity{
			activity("TrailRun", 12000, 300, 4200),
		},
		business.TrailRun,
	)

	assertCompletedBadge(t, results, isDistanceBadge)
	assertCompletedBadge(t, results, isElevationBadge)
	assertCompletedBadge(t, results, isMovingTimeBadge)
}

func TestCheckGeneralBadges_UsesHikingFamilyForWalk(t *testing.T) {
	results := checkGeneralBadges(
		[]*strava.Activity{
			activity("Walk", 11000, 1100, 4000),
		},
		business.Walk,
	)

	assertCompletedBadge(t, results, isDistanceBadge)
	assertCompletedBadge(t, results, isElevationBadge)
	assertCompletedBadge(t, results, isMovingTimeBadge)
}

func TestCheckGeneralBadges_ReturnsEmptyForUnsupportedFamily(t *testing.T) {
	results := checkGeneralBadges(
		[]*strava.Activity{
			activity("AlpineSki", 20000, 1000, 3600),
		},
		business.AlpineSki,
	)

	if len(results) != 0 {
		t.Fatalf("expected no badges for unsupported family, got %d", len(results))
	}
}

func assertCompletedBadge(t *testing.T, results []business.BadgeCheckResult, matches func(business.Badge) bool) {
	t.Helper()
	for _, result := range results {
		if matches(result.Badge) && result.IsCompleted {
			return
		}
	}
	t.Fatalf("expected a completed badge of requested type")
}

func isDistanceBadge(badge business.Badge) bool {
	_, ok := badge.(badges.DistanceBadge)
	return ok
}

func isElevationBadge(badge business.Badge) bool {
	_, ok := badge.(badges.ElevationBadge)
	return ok
}

func isMovingTimeBadge(badge business.Badge) bool {
	_, ok := badge.(badges.MovingTimeBadge)
	return ok
}

func activity(activityType string, distance float64, elevation float64, movingTime int) *strava.Activity {
	return &strava.Activity{
		Type:               activityType,
		SportType:          activityType,
		Distance:           distance,
		TotalElevationGain: elevation,
		MovingTime:         movingTime,
	}
}
