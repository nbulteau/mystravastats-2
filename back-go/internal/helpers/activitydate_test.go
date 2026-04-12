package helpers

import (
	"sort"
	"testing"
	"time"
)

func TestParseActivityDateSupportsExtendedFormats(t *testing.T) {
	t.Helper()

	tests := []string{
		"2024-08-16T08:00:00+0200", // timezone offset without colon
		"2024-08-16T08:00:00",      // local date-time without timezone
		"2024-08-16",               // day precision
	}

	for _, value := range tests {
		parsed, ok := ParseActivityDate(value)
		if !ok {
			t.Fatalf("ParseActivityDate should parse %q", value)
		}
		if parsed.Year() != 2024 {
			t.Fatalf("expected year 2024 for %q, got %d", value, parsed.Year())
		}
	}
}

func TestActivitySortingKeepsChronologicalOrderWithMixedDateFormats(t *testing.T) {
	t.Helper()

	type dateEntry struct {
		id             int64
		startDate      string
		startDateLocal string
	}

	entries := []dateEntry{
		{
			id:             1,
			startDateLocal: "2026-04-05T08:00:00+02:00",
			startDate:      "2026-04-05T06:00:00Z",
		},
		{
			id:             2,
			startDateLocal: "2025-08-04T08:00:00+02:00",
			startDate:      "2025-08-04T06:00:00Z",
		},
		{
			id:             3,
			startDateLocal: "2024-08-16T08:00:00+0200", // legacy/non ISO offset format
			startDate:      "2024-08-16T06:00:00Z",
		},
	}

	sort.Slice(entries, func(i, j int) bool {
		left := entries[i]
		right := entries[j]

		leftDay := FirstNonEmpty(ExtractSortableDay(left.startDateLocal), ExtractSortableDay(left.startDate))
		rightDay := FirstNonEmpty(ExtractSortableDay(right.startDateLocal), ExtractSortableDay(right.startDate))
		if leftDay != rightDay {
			return leftDay < rightDay
		}

		leftDateValue := FirstNonEmpty(left.startDateLocal, left.startDate)
		rightDateValue := FirstNonEmpty(right.startDateLocal, right.startDate)
		if leftDateValue != rightDateValue {
			return IsBeforeActivityDate(leftDateValue, rightDateValue)
		}

		return left.id < right.id
	})

	if entries[0].id != 3 || entries[1].id != 2 || entries[2].id != 1 {
		t.Fatalf("unexpected order: got IDs [%d, %d, %d], want [3, 2, 1]", entries[0].id, entries[1].id, entries[2].id)
	}
}

func TestExtractSortableDay(t *testing.T) {
	t.Helper()

	if day := ExtractSortableDay("2024-08-16T08:00:00+0200"); day != "2024-08-16" {
		t.Fatalf("unexpected day extraction, got %q", day)
	}
	if day := ExtractSortableDay("invalid"); day != "" {
		t.Fatalf("invalid value should return empty day, got %q", day)
	}
}

func TestIsBeforeActivityDateWithMixedFormats(t *testing.T) {
	t.Helper()

	left := "2024-08-16T08:00:00+0200"
	right := "2025-08-04T08:00:00+02:00"
	if !IsBeforeActivityDate(left, right) {
		t.Fatalf("expected %q to be before %q", left, right)
	}

	// Ensure deterministic behavior for same instant but different representations.
	sameInstantLeft := "2026-04-05T06:00:00Z"
	sameInstantRight := "2026-04-05T08:00:00+02:00"
	leftParsed, _ := time.Parse(time.RFC3339, sameInstantLeft)
	rightParsed, _ := time.Parse(time.RFC3339, sameInstantRight)
	if !leftParsed.Equal(rightParsed) {
		t.Fatalf("expected test setup to represent same instant")
	}
	if IsBeforeActivityDate(sameInstantLeft, sameInstantRight) {
		t.Fatalf("same instants should not be considered strictly before each other")
	}
}
