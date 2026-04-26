package fit

import (
	"math"
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

func TestComputeFITPowerMetrics_UsesPowerStreamWhenSessionPowerIsMissing(t *testing.T) {
	// GIVEN
	stream := &strava.Stream{
		Watts: &strava.PowerStream{
			Data: []float64{0, 100, 200, 300},
		},
	}

	// WHEN
	metrics := computeFITPowerMetrics(0, stream, 100)

	// THEN
	assertFloatEquals(t, 150, metrics.averageWatts)
	if metrics.weightedAverageWatts != 150 {
		t.Fatalf("expected weighted watts fallback=150, got %d", metrics.weightedAverageWatts)
	}
	assertFloatEquals(t, 12.906, metrics.kilojoules)
	if !metrics.hasDeviceWatts {
		t.Fatal("expected device watts to be true when FIT records contain power")
	}
}

func TestComputeFITPowerMetrics_KeepsSessionAveragePowerWhenPresent(t *testing.T) {
	// GIVEN
	stream := &strava.Stream{
		Watts: &strava.PowerStream{
			Data: []float64{0, 100, 200},
		},
	}

	// WHEN
	metrics := computeFITPowerMetrics(250, stream, 120)

	// THEN
	assertFloatEquals(t, 250, metrics.averageWatts)
	if metrics.weightedAverageWatts != 250 {
		t.Fatalf("expected session average to be reused as weighted watts, got %d", metrics.weightedAverageWatts)
	}
	assertFloatEquals(t, 25.812, metrics.kilojoules)
	if !metrics.hasDeviceWatts {
		t.Fatal("expected device watts to stay true when session power is present")
	}
}

func TestComputeFITPowerMetrics_IgnoresEmptyPowerStream(t *testing.T) {
	// GIVEN
	stream := &strava.Stream{
		Watts: &strava.PowerStream{
			Data: []float64{0, 0, math.NaN(), -20, float64(fitInvalidUint16)},
		},
	}

	// WHEN
	metrics := computeFITPowerMetrics(0, stream, 100)

	// THEN
	assertFloatEquals(t, 0, metrics.averageWatts)
	if metrics.weightedAverageWatts != 0 {
		t.Fatalf("expected empty weighted watts, got %d", metrics.weightedAverageWatts)
	}
	assertFloatEquals(t, 0, metrics.kilojoules)
	if metrics.hasDeviceWatts {
		t.Fatal("expected device watts to be false without positive FIT power samples")
	}
}

func TestComputeFITPowerMetrics_IgnoresInvalidSessionPower(t *testing.T) {
	// GIVEN
	stream := &strava.Stream{
		Watts: &strava.PowerStream{
			Data: []float64{0, float64(fitInvalidUint16)},
		},
	}

	// WHEN
	metrics := computeFITPowerMetrics(validFITUint16Float(fitInvalidUint16), stream, 100)

	// THEN
	assertFloatEquals(t, 0, metrics.averageWatts)
	if metrics.weightedAverageWatts != 0 {
		t.Fatalf("expected invalid weighted watts to be zero, got %d", metrics.weightedAverageWatts)
	}
	assertFloatEquals(t, 0, metrics.kilojoules)
	if metrics.hasDeviceWatts {
		t.Fatal("expected device watts to be false for invalid FIT power sentinels")
	}
}

func TestFITNumericHelpers_IgnoreNonFiniteValues(t *testing.T) {
	if firstPositiveFinite(math.NaN(), math.Inf(1), -1, 12.5) != 12.5 {
		t.Fatal("expected firstPositiveFinite to skip NaN, Inf and negative values")
	}
	if nonNegativeFinite(math.NaN()) != 0 || nonNegativeFinite(math.Inf(1)) != 0 || nonNegativeFinite(-1) != 0 {
		t.Fatal("expected nonNegativeFinite to coerce invalid values to zero")
	}
	if roundedNonNegative(math.NaN()) != 0 || roundedNonNegative(math.Inf(1)) != 0 || roundedNonNegative(-1) != 0 {
		t.Fatal("expected roundedNonNegative to coerce invalid values to zero")
	}
	if maxFloat64Slice([]float64{math.NaN(), 0, 12.5, math.Inf(1)}) != 12.5 {
		t.Fatal("expected maxFloat64Slice to ignore non-finite values")
	}
	if asUint8(fitInvalidUint8) != 0 || asUint8(uint16(fitInvalidUint8)) != 0 || asUint8(float64(fitInvalidUint8)) != 0 {
		t.Fatal("expected asUint8 to coerce FIT invalid sentinels to zero")
	}
	if validFITUint8(fitInvalidUint8) != 0 || validFITUint16Float(fitInvalidUint16) != 0 {
		t.Fatal("expected FIT sentinel helpers to coerce invalid values to zero")
	}
}

func assertFloatEquals(t *testing.T, expected float64, actual float64) {
	t.Helper()
	if math.Abs(expected-actual) > 0.0001 {
		t.Fatalf("expected %.4f, got %.4f", expected, actual)
	}
}
