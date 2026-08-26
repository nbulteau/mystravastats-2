package fit

import (
	"math"
	"path/filepath"
	"testing"

	"mystravastats/internal/shared/domain/business"
	"mystravastats/internal/shared/domain/strava"

	fitparser "github.com/tormoder/fit"
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

func TestDecodeFITActivity_DoesNotMarkLocalFileAsStravaUpload(t *testing.T) {
	// GIVEN
	fitFile := filepath.Join("..", "..", "..", "..", "..", "test-fixtures", "source-modes", "fit", "2026", "smoke-ride.fit")

	// WHEN
	activity, err := DecodeFITActivity(fitFile, 42)

	// THEN
	if err != nil {
		t.Fatalf("expected FIT activity to decode, got error: %v", err)
	}
	if activity.UploadId != 0 {
		t.Fatalf("expected local FIT activity to have no Strava upload id, got %d", activity.UploadId)
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

func TestClassifyFITSport_UsesSportAndSubSport(t *testing.T) {
	tests := []struct {
		name     string
		sport    fitparser.Sport
		subSport fitparser.SubSport
		expected fitActivityClassification
	}{
		{
			name:     "generic cycling",
			sport:    fitparser.SportCycling,
			subSport: fitparser.SubSportGeneric,
			expected: fitActivityTypeClassification(business.Ride.String()),
		},
		{
			name:     "mountain cycling",
			sport:    fitparser.SportCycling,
			subSport: fitparser.SubSportMountain,
			expected: fitActivityTypeClassification(business.MountainBikeRide.String()),
		},
		{
			name:     "e-bike mountain cycling",
			sport:    fitparser.SportCycling,
			subSport: fitparser.SubSportEBikeMountain,
			expected: fitActivityTypeClassification(business.MountainBikeRide.String()),
		},
		{
			name:     "gravel cycling",
			sport:    fitparser.SportCycling,
			subSport: fitparser.SubSportGravelCycling,
			expected: fitActivityTypeClassification(business.GravelRide.String()),
		},
		{
			name:     "mixed surface cycling",
			sport:    fitparser.SportCycling,
			subSport: fitparser.SubSportMixedSurface,
			expected: fitActivityTypeClassification(business.GravelRide.String()),
		},
		{
			name:     "virtual cycling",
			sport:    fitparser.SportCycling,
			subSport: fitparser.SubSportVirtualActivity,
			expected: fitActivityTypeClassification(business.VirtualRide.String()),
		},
		{
			name:     "indoor cycling",
			sport:    fitparser.SportCycling,
			subSport: fitparser.SubSportIndoorCycling,
			expected: fitActivityTypeClassification(business.VirtualRide.String()),
		},
		{
			name:     "fitness equipment indoor cycling",
			sport:    fitparser.SportFitnessEquipment,
			subSport: fitparser.SubSportIndoorCycling,
			expected: fitActivityTypeClassification(business.VirtualRide.String()),
		},
		{
			name:     "cycling commute",
			sport:    fitparser.SportCycling,
			subSport: fitparser.SubSportCommuting,
			expected: fitCommuteClassification(business.Ride.String()),
		},
		{
			name:     "generic running",
			sport:    fitparser.SportRunning,
			subSport: fitparser.SubSportGeneric,
			expected: fitActivityTypeClassification(business.Run.String()),
		},
		{
			name:     "trail running",
			sport:    fitparser.SportRunning,
			subSport: fitparser.SubSportTrail,
			expected: fitActivityTypeClassification(business.TrailRun.String()),
		},
		{
			name:     "fitness equipment treadmill",
			sport:    fitparser.SportFitnessEquipment,
			subSport: fitparser.SubSportTreadmill,
			expected: fitActivityTypeClassification(business.Run.String()),
		},
		{
			name:     "walking",
			sport:    fitparser.SportWalking,
			subSport: fitparser.SubSportGeneric,
			expected: fitActivityTypeClassification(business.Walk.String()),
		},
		{
			name:     "fitness equipment indoor walking",
			sport:    fitparser.SportFitnessEquipment,
			subSport: fitparser.SubSportIndoorWalking,
			expected: fitActivityTypeClassification(business.Walk.String()),
		},
		{
			name:     "hiking",
			sport:    fitparser.SportHiking,
			subSport: fitparser.SubSportGeneric,
			expected: fitActivityTypeClassification(business.Hike.String()),
		},
		{
			name:     "mountaineering",
			sport:    fitparser.SportMountaineering,
			subSport: fitparser.SubSportGeneric,
			expected: fitActivityTypeClassification(business.Hike.String()),
		},
		{
			name:     "alpine skiing",
			sport:    fitparser.SportAlpineSkiing,
			subSport: fitparser.SubSportGeneric,
			expected: fitActivityTypeClassification(business.AlpineSki.String()),
		},
		{
			name:     "inline skating",
			sport:    fitparser.SportInlineSkating,
			subSport: fitparser.SubSportGeneric,
			expected: fitActivityTypeClassification(business.InlineSkate.String()),
		},
		{
			name:     "generic e-biking",
			sport:    fitparser.SportEBiking,
			subSport: fitparser.SubSportGeneric,
			expected: fitActivityTypeClassification(business.Ride.String()),
		},
		{
			name:     "virtual e-biking",
			sport:    fitparser.SportEBiking,
			subSport: fitparser.SubSportVirtualActivity,
			expected: fitActivityTypeClassification(business.VirtualRide.String()),
		},
		{
			name:     "unknown fallback",
			sport:    fitparser.SportInvalid,
			subSport: fitparser.SubSportInvalid,
			expected: fitActivityTypeClassification(business.Ride.String()),
		},
	}

	coveredTypes := make(map[string]bool)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := classifyFITSport(tt.sport, tt.subSport)
			if actual != tt.expected {
				t.Fatalf("expected %+v, got %+v", tt.expected, actual)
			}
		})
		coveredTypes[tt.expected.Type] = true
	}

	for _, activityType := range []business.ActivityType{
		business.AlpineSki,
		business.Commute,
		business.GravelRide,
		business.Hike,
		business.InlineSkate,
		business.MountainBikeRide,
		business.Ride,
		business.Run,
		business.TrailRun,
		business.VirtualRide,
		business.Walk,
	} {
		if !coveredTypes[activityType.String()] {
			t.Fatalf("expected FIT mapping coverage for %s", activityType.String())
		}
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

func TestResolveFITMovingTime_UsesTotalMovingTimeWhenPresent(t *testing.T) {
	stream := movingTimeTestStream([]int{0, 10, 20}, []bool{false, true, true})

	movingTime := resolveFITMovingTime(12, 20, 20, stream)

	if movingTime != 12 {
		t.Fatalf("expected total moving time to win, got %d", movingTime)
	}
}

func TestResolveFITMovingTime_UsesStreamWhenItRemovesStopTimeFromTimer(t *testing.T) {
	stream := movingTimeTestStream([]int{0, 100, 200, 300, 400}, []bool{false, true, false, true, true})

	movingTime := resolveFITMovingTime(0, 400, 405, stream)

	if movingTime != 300 {
		t.Fatalf("expected stream moving fallback, got %d", movingTime)
	}
}

func TestResolveFITMovingTime_UsesStreamForGarminTimerWithLongStops(t *testing.T) {
	stream := movingTimeTestStream([]int{0, 12701, 16328}, []bool{false, true, false})

	movingTime := resolveFITMovingTime(0, 16328, 18581, stream)

	if movingTime != 12701 {
		t.Fatalf("expected stream moving time from FIT records, got %d", movingTime)
	}
}

func TestResolveFITMovingTime_KeepsTimerWhenTimerAlreadyExcludesStops(t *testing.T) {
	stream := movingTimeTestStream([]int{0, 100, 200, 300, 400}, []bool{false, true, true, true, true})

	movingTime := resolveFITMovingTime(0, 220, 900, stream)

	if movingTime != 220 {
		t.Fatalf("expected total timer time to win, got %d", movingTime)
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

func movingTimeTestStream(times []int, moving []bool) *strava.Stream {
	return &strava.Stream{
		Time: strava.TimeStream{Data: times},
		Moving: &strava.MovingStream{
			Data: moving,
		},
	}
}

func assertFloatEquals(t *testing.T, expected float64, actual float64) {
	t.Helper()
	if math.Abs(expected-actual) > 0.0001 {
		t.Fatalf("expected %.4f, got %.4f", expected, actual)
	}
}
