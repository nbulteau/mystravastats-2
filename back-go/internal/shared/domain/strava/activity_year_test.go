package strava

import "testing"

func TestParseYearRejectsInvalidValues(t *testing.T) {
	for _, value := range []string{"", "202", "nope-01-01", "0000-01-01", "9999-01-01"} {
		if year, ok := ParseYear(value); ok {
			t.Fatalf("ParseYear(%q) = %d, true; want invalid", value, year)
		}
	}
}

func TestActivityYearFallsBackToUTCDate(t *testing.T) {
	activity := &Activity{StartDateLocal: "", StartDate: "2025-04-03T08:00:00Z"}
	year, ok := activity.Year()
	if !ok || year != 2025 {
		t.Fatalf("activity.Year() = %d, %v; want 2025, true", year, ok)
	}
}
