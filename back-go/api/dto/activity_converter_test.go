package dto

import (
	"testing"

	"mystravastats/internal/shared/domain/strava"
)

func TestToActivityDto_MapsCommuteAndHeartRate(t *testing.T) {
	// GIVEN
	activity := strava.Activity{
		Id:                 42,
		Name:               "Lunch ride",
		Type:               "Ride",
		Commute:            true,
		AverageHeartrate:   156.7,
		AverageWatts:       212.4,
		Distance:           1000,
		ElapsedTime:        180,
		MovingTime:         170,
		TotalElevationGain: 42,
		AverageSpeed:       6.1,
		StartDateLocal:     "2026-04-17T12:00:00Z",
		Stream: &strava.Stream{
			Distance: strava.DistanceStream{Data: []float64{0, 1000}},
			Time:     strava.TimeStream{Data: []int{0, 180}},
			Altitude: &strava.AltitudeStream{Data: []float64{100, 142}},
		},
	}

	// WHEN
	dto := ToActivityDto(activity)

	// THEN
	if !dto.Commute {
		t.Fatalf("expected commute to be true")
	}
	if dto.AverageHeartrate != 156 {
		t.Fatalf("expected averageHeartrate=156, got %d", dto.AverageHeartrate)
	}
	if dto.AverageWatts != 212 {
		t.Fatalf("expected averageWatts=212, got %d", dto.AverageWatts)
	}
}
