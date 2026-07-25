package infrastructure

import (
	"testing"

	"mystravastats/internal/shared/domain/business"
)

func TestResolveStatisticsActivityType_UsesRideFamilyForCyclingSelection(t *testing.T) {
	activityType := resolveStatisticsActivityType([]business.ActivityType{
		business.Commute,
		business.GravelRide,
		business.MountainBikeRide,
		business.Ride,
		business.VirtualRide,
	})

	if activityType != business.Ride {
		t.Fatalf("expected Ride for cycling family selection, got %s", activityType.String())
	}
}

func TestResolveStatisticsActivityType_UsesHikeFamilyForHikeSelection(t *testing.T) {
	activityType := resolveStatisticsActivityType([]business.ActivityType{
		business.Walk,
		business.Hike,
	})

	if activityType != business.Hike {
		t.Fatalf("expected Hike for hiking family selection, got %s", activityType.String())
	}
}

func TestResolveStatisticsActivityType_PreservesSingleSpecializedCyclingSelection(t *testing.T) {
	if activityType := resolveStatisticsActivityType([]business.ActivityType{business.Commute}); activityType != business.Commute {
		t.Fatalf("expected Commute for commute-only selection, got %s", activityType.String())
	}

	if activityType := resolveStatisticsActivityType([]business.ActivityType{business.VirtualRide}); activityType != business.VirtualRide {
		t.Fatalf("expected VirtualRide for virtual-only selection, got %s", activityType.String())
	}
}
