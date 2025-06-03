package statistics

import (
	"mystravastats/domain/strava"
	"testing"
)

func TestNewHighestPointStatistic(t *testing.T) {
	// Test with normal activities
	activities := []*strava.Activity{
		{Id: 1, Name: "Activity 1", Type: "Run", ElevHigh: 100.0},
		{Id: 2, Name: "Activity 2", Type: "Ride", ElevHigh: 500.0},
		{Id: 3, Name: "Activity 3", Type: "Run", ElevHigh: 300.0},
	}

	stat := NewHighestPointStatistic(activities)

	if stat.highestElevation == nil {
		t.Errorf("Expected highestElevation to be calculated, got nil")
	}

	expectedElevation := 500.0
	if *stat.highestElevation != expectedElevation {
		t.Errorf("Expected highestElevation to be %.2f, got %.2f", expectedElevation, *stat.highestElevation)
	}

	if stat.activity == nil {
		t.Errorf("Expected activity to be set, got nil")
	} else {
		expectedId := int64(2)
		if stat.activity.Id != expectedId {
			t.Errorf("Expected activity Id to be %d, got %d", expectedId, stat.activity.Id)
		}

		expectedName := "Activity 2"
		if stat.activity.Name != expectedName {
			t.Errorf("Expected activity Name to be %s, got %s", expectedName, stat.activity.Name)
		}
	}

	// Test with empty activities slice
	var emptyActivities []*strava.Activity
	emptyStat := NewHighestPointStatistic(emptyActivities)

	if emptyStat.highestElevation != nil {
		t.Errorf("Expected highestElevation to be nil for empty activities, got %.2f", *emptyStat.highestElevation)
	}

	if emptyStat.activity != nil {
		t.Errorf("Expected activity to be nil for empty activities, got %v", emptyStat.activity)
	}

	// Test Value method
	expectedValue := "500.00 m"
	if stat.Value() != expectedValue {
		t.Errorf("Expected Value to be %s, got %s", expectedValue, stat.Value())
	}

	expectedEmptyValue := "Not available"
	if emptyStat.Value() != expectedEmptyValue {
		t.Errorf("Expected Value for empty activities to be %s, got %s", expectedEmptyValue, emptyStat.Value())
	}
}
