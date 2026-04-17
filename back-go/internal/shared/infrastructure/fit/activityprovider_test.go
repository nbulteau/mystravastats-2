package fit

import (
	"testing"

	"mystravastats/internal/shared/domain/business"
	"mystravastats/internal/shared/domain/strava"
)

func TestNewFITActivityProvider_EmptyDirectory(t *testing.T) {
	// GIVEN
	fitDirectory := t.TempDir()

	// WHEN
	provider := NewFITActivityProvider(fitDirectory)

	// THEN
	if provider == nil {
		t.Fatal("expected provider to be initialized")
	}
	if provider.GetAthlete().Firstname == nil || *provider.GetAthlete().Firstname == "" {
		t.Fatal("expected FIT profile first name to be initialized")
	}
	activities := provider.GetActivitiesByYearAndActivityTypes(nil, business.Ride)
	if len(activities) != 0 {
		t.Fatalf("expected no activities for empty FIT directory, got %d", len(activities))
	}
	diagnostics := provider.CacheDiagnostics()
	if diagnostics["provider"] != "fit" {
		t.Fatalf("expected diagnostics provider=fit, got %#v", diagnostics["provider"])
	}
}

func TestFITActivityProvider_GetActivitiesByYearAndType_UsesDefensiveCopy(t *testing.T) {
	// GIVEN
	provider := NewFITActivityProvider(t.TempDir())
	provider.replaceActivities([]*strava.Activity{
		{Id: 3, Type: "Ride", SportType: "Ride", StartDateLocal: "2025-09-01T10:00:00+02:00", Distance: 50_000},
		{Id: 2, Type: "Ride", SportType: "Ride", StartDateLocal: "2025-06-01T10:00:00+02:00", Distance: 40_000},
		{Id: 1, Type: "Run", SportType: "Run", StartDateLocal: "2024-05-01T10:00:00+02:00", Distance: 10_000},
	})
	year := 2025

	// WHEN
	firstCall := provider.GetActivitiesByYearAndActivityTypes(&year, business.Ride)
	if len(firstCall) != 2 {
		t.Fatalf("expected 2 ride activities in 2025, got %d", len(firstCall))
	}
	firstCall[0], firstCall[1] = firstCall[1], firstCall[0]
	secondCall := provider.GetActivitiesByYearAndActivityTypes(&year, business.Ride)

	// THEN
	if len(secondCall) != 2 {
		t.Fatalf("expected 2 ride activities in 2025 on second call, got %d", len(secondCall))
	}
	if secondCall[0].Id != 3 {
		t.Fatalf("expected cached ordering to be preserved, got first id=%d", secondCall[0].Id)
	}
}

func TestNormalizeCoordinates_FillsMissingValues(t *testing.T) {
	// GIVEN
	rawCoordinates := [][]float64{
		{0, 0},
		{0, 0},
		{48.1000, -1.7000},
		{0, 0},
		{48.2000, -1.6000},
		{0, 0},
	}

	// WHEN
	normalized, ok := normalizeCoordinates(rawCoordinates)

	// THEN
	if !ok {
		t.Fatal("expected coordinates to be considered valid")
	}
	if len(normalized) != len(rawCoordinates) {
		t.Fatalf("expected same coordinate count, got %d", len(normalized))
	}
	if normalized[0][0] == 0 && normalized[0][1] == 0 {
		t.Fatal("expected leading invalid coordinates to be fixed")
	}
	if normalized[3][0] == 0 && normalized[3][1] == 0 {
		t.Fatal("expected middle invalid coordinates to be fixed")
	}
	if normalized[5][0] == 0 && normalized[5][1] == 0 {
		t.Fatal("expected trailing invalid coordinates to be fixed")
	}
}
