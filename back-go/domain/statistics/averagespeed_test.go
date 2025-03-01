package statistics

import (
	"mystravastats/domain/strava"
	"testing"
)

func TestNewAverageSpeedStatistic(t *testing.T) {
	activities := []*strava.Activity{
		{AverageSpeed: 5.0},
		{AverageSpeed: 10.0},
		{AverageSpeed: 15.0},
	}

	stat := NewAverageSpeedStatistic(activities)

	if stat.averageSpeed == nil {
		t.Errorf("Expected averageSpeed to be calculated, got nil")
	}

	expectedAverageSpeed := 10.0
	if *stat.averageSpeed != expectedAverageSpeed {
		t.Errorf("Expected averageSpeed to be %.2f, got %.2f", expectedAverageSpeed, *stat.averageSpeed)
	}
}

func TestAverageSpeedStatistic_for_Ride_activities_Value(t *testing.T) {
	activities := []*strava.Activity{
		{AverageSpeed: 5.0, Type: "Ride"},
		{AverageSpeed: 10.0, Type: "Ride"},
		{AverageSpeed: 15.0, Type: "Ride"},
	}

	stat := NewAverageSpeedStatistic(activities)
	expectedValue := "36.00 km/h"

	if stat.Value() != expectedValue {
		t.Errorf("Expected Value to be %s, got %s", expectedValue, stat.Value())
	}
}

func TestAverageSpeedStatistic_NoActivities(t *testing.T) {
	var activities []*strava.Activity

	stat := NewAverageSpeedStatistic(activities)

	if stat.averageSpeed != nil {
		t.Errorf("Expected averageSpeed to be nil, got %.2f", *stat.averageSpeed)
	}

	expectedValue := "Not available"
	if stat.Value() != expectedValue {
		t.Errorf("Expected Value to be %s, got %s", expectedValue, stat.Value())
	}
}

func TestAverageSpeedStatistic_ZeroSpeedActivities(t *testing.T) {
	activities := []*strava.Activity{
		{AverageSpeed: 0.0},
		{AverageSpeed: 0.0},
	}

	stat := NewAverageSpeedStatistic(activities)

	if stat.averageSpeed != nil {
		t.Errorf("Expected averageSpeed to be nil, got %.2f", *stat.averageSpeed)
	}

	expectedValue := "Not available"
	if stat.Value() != expectedValue {
		t.Errorf("Expected Value to be %s, got %s", expectedValue, stat.Value())
	}
}
