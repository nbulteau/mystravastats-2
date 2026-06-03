package infrastructure

import (
	"fmt"
	"math"
	"mystravastats/internal/shared/domain/business"
	"mystravastats/internal/shared/domain/strava"
	"strconv"
	"testing"
	"time"
)

func TestComputeEddingtonFromDailyTotals_ReturnsZeroForEmptyInput(t *testing.T) {
	// GIVEN
	dailyTotals := map[string]int{}

	// WHEN
	result := computeEddingtonFromDailyTotals(business.EddingtonScopeLifetime, dailyTotals)

	// THEN
	if result.Number != 0 {
		t.Fatalf("expected eddington number 0, got %d", result.Number)
	}
	if len(result.List) != 0 {
		t.Fatalf("expected empty eddington list, got %d entries", len(result.List))
	}
	if result.NextTarget != 1 || result.QualifyingDays != 0 || result.MissingDays != 1 {
		t.Fatalf("expected next target progress 0/1, got target=%d qualifying=%d missing=%d", result.NextTarget, result.QualifyingDays, result.MissingDays)
	}
}

func TestComputeEddingtonFromDailyTotals_DoesNotRoundUpOnExactBoundary(t *testing.T) {
	// GIVEN
	dailyTotals := make(map[string]int, 49)
	for day := 1; day <= 49; day++ {
		dailyTotals[fmt.Sprintf("2024-01-%02d", day)] = 51
	}

	// WHEN
	result := computeEddingtonFromDailyTotals(business.EddingtonScopeLifetime, dailyTotals)

	// THEN
	if result.Number != 49 {
		t.Fatalf("expected eddington number 49, got %d", result.Number)
	}
	if len(result.List) != 51 {
		t.Fatalf("expected eddington list length 51, got %d", len(result.List))
	}
	if result.List[48] != 49 {
		t.Fatalf("expected 49 days at >=49km, got %d", result.List[48])
	}
	if result.List[49] != 49 {
		t.Fatalf("expected 49 days at >=50km, got %d", result.List[49])
	}
	if result.NextTarget != 50 || result.QualifyingDays != 49 || result.MissingDays != 1 {
		t.Fatalf("expected next target 50 with one missing day, got target=%d qualifying=%d missing=%d", result.NextTarget, result.QualifyingDays, result.MissingDays)
	}
}

func TestComputeEddingtonFromDailyTotals_IgnoresNonPositiveDailyTotals(t *testing.T) {
	// GIVEN
	dailyTotals := map[string]int{
		"2025-01-01": 4,
		"2025-01-02": 4,
		"2025-01-03": 4,
		"2025-01-04": 4,
		"2025-01-05": 0,
		"2025-01-06": -2,
	}

	// WHEN
	result := computeEddingtonFromDailyTotals(business.EddingtonScopeLifetime, dailyTotals)

	// THEN
	if result.Number != 4 {
		t.Fatalf("expected eddington number 4, got %d", result.Number)
	}
	if len(result.List) != 4 {
		t.Fatalf("expected eddington list length 4, got %d", len(result.List))
	}
	for day, count := range result.List {
		if count != 4 {
			t.Fatalf("expected 4 days for threshold index %d, got %d", day, count)
		}
	}
}

func TestComputeEddingtonFromValues_SupportsElevationActivities(t *testing.T) {
	// GIVEN
	values := []int{4, 4, 4, 2}

	// WHEN
	result := computeEddingtonFromValues(
		business.EddingtonScopeLifetime,
		business.EddingtonMetricElevation,
		business.EddingtonBasisActivities,
		values,
	)

	// THEN
	if result.Number != 3 {
		t.Fatalf("expected eddington number 3, got %d", result.Number)
	}
	if result.Metric != business.EddingtonMetricElevation || result.Basis != business.EddingtonBasisActivities || result.Unit != "m" || result.ThresholdScale != 100 {
		t.Fatalf("expected elevation/activity metadata, got metric=%s basis=%s unit=%s scale=%d", result.Metric, result.Basis, result.Unit, result.ThresholdScale)
	}
	if result.NextTarget != 4 || result.QualifyingCount != 3 || result.MissingCount != 1 {
		t.Fatalf("expected next target 4 with one missing activity, got target=%d qualifying=%d missing=%d", result.NextTarget, result.QualifyingCount, result.MissingCount)
	}
}

func TestEddingtonActivityValue_UsesHundredMeterElevationBuckets(t *testing.T) {
	// GIVEN
	activity := &strava.Activity{TotalElevationGain: 349}

	// WHEN
	value := eddingtonActivityValue(activity, business.EddingtonMetricElevation)

	// THEN
	if value != 3 {
		t.Fatalf("expected 349m to count as 3 hundred-meter buckets, got %d", value)
	}
}

func TestCountActiveDays_CountsUniqueCalendarDatesOnly(t *testing.T) {
	// GIVEN
	activities := []*strava.Activity{
		{StartDateLocal: "2025-01-01T08:00:00Z"},
		{StartDateLocal: "2025-01-01T18:00:00Z"},
		{StartDateLocal: "2025-01-03T07:30:00Z"},
	}

	// WHEN
	activeDays := countActiveDays(activities)

	// THEN
	if activeDays != 2 {
		t.Fatalf("expected 2 active days, got %d", activeDays)
	}
}

func TestSumMovingTime_FallsBackToElapsedTimeWhenMovingTimeIsZero(t *testing.T) {
	// GIVEN
	activities := []*strava.Activity{
		{MovingTime: 3600, ElapsedTime: 3700},
		{MovingTime: 0, ElapsedTime: 1800},
		{MovingTime: 1200, ElapsedTime: 1500},
	}

	// WHEN
	totalMoving := sumMovingTime(activities)

	// THEN
	expected := 3600 + 1800 + 1200
	if totalMoving != expected {
		t.Fatalf("expected moving time %d, got %d", expected, totalMoving)
	}
}

func TestComputeConsistencyByYear_UsesFullYearForPastYears(t *testing.T) {
	// GIVEN
	year := "2024"
	activeDays := 183

	// WHEN
	consistency := computeConsistencyByYear(year, activeDays)

	// THEN
	expected := 50.0 // 183 / 366 * 100 rounded to 1 decimal
	if math.Abs(consistency-expected) > 0.0001 {
		t.Fatalf("expected consistency %.1f, got %.1f", expected, consistency)
	}
}

func TestComputeConsistencyByYear_UsesYearToDateForCurrentYear(t *testing.T) {
	// GIVEN
	currentYear := strconv.Itoa(time.Now().Year())
	activeDays := 1

	// WHEN
	consistency := computeConsistencyByYear(currentYear, activeDays)

	// THEN
	daysInScope := time.Now().YearDay()
	expected := math.Round((float64(activeDays)/float64(daysInScope))*1000) / 10
	if math.Abs(consistency-expected) > 0.0001 {
		t.Fatalf("expected consistency %.1f, got %.1f", expected, consistency)
	}
}
