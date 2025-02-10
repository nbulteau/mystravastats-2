package statistics

import (
	"mystravastats/domain/strava"
	"testing"
)

func bestDayFunction(activities []*strava.Activity) *Pair {
	if len(activities) == 0 {
		return nil
	}
	return &Pair{Date: "2025-02-10", Value: 100.0}
}

func TestNewBestDayStatistic(t *testing.T) {
	activities := []*strava.Activity{
		{StartDate: "2025-02-10"},
	}

	stat := NewBestDayStatistic("Best Day", activities, "%.2f on %s", bestDayFunction)

	if stat == nil {
		t.Errorf("Expected BestDayStatistic to be created, but got nil")
	}
}

func TestBestDayStatistic_Value(t *testing.T) {
	activities := []*strava.Activity{
		{StartDate: "2025-02-10"},
	}

	stat := NewBestDayStatistic("Best Day", activities, "%.2f on %s", bestDayFunction)
	value := stat.Value()

	expected := "100.00 on Mon 10 February 2025"
	if value != expected {
		t.Errorf("Expected value to be %s, but got %s", expected, value)
	}

	// Test with no activities
	stat = NewBestDayStatistic("Best Day", []*strava.Activity{}, "%.2f on %s", bestDayFunction)
	value = stat.Value()

	expected = "Not available"
	if value != expected {
		t.Errorf("Expected value to be %s, but got %s", expected, value)
	}
}
