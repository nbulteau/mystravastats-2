package infrastructure

import (
	"fmt"
	"testing"
)

func TestComputeEddingtonFromDailyTotals_ReturnsZeroForEmptyInput(t *testing.T) {
	// GIVEN
	dailyTotals := map[string]int{}

	// WHEN
	result := computeEddingtonFromDailyTotals(dailyTotals)

	// THEN
	if result.Number != 0 {
		t.Fatalf("expected eddington number 0, got %d", result.Number)
	}
	if len(result.List) != 0 {
		t.Fatalf("expected empty eddington list, got %d entries", len(result.List))
	}
}

func TestComputeEddingtonFromDailyTotals_DoesNotRoundUpOnExactBoundary(t *testing.T) {
	// GIVEN
	dailyTotals := make(map[string]int, 49)
	for day := 1; day <= 49; day++ {
		dailyTotals[fmt.Sprintf("2024-01-%02d", day)] = 51
	}

	// WHEN
	result := computeEddingtonFromDailyTotals(dailyTotals)

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
	result := computeEddingtonFromDailyTotals(dailyTotals)

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
