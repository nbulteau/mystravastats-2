package statistics

import (
	"math"
	"mystravastats/domain/strava"
	"testing"
)

func TestBestTimeForDistance_WithSyntheticStream(t *testing.T) {
	// GIVEN
	stream := syntheticStream(
		[]float64{0, 100, 200, 300, 400},
		[]int{0, 10, 20, 35, 50},
		[]float64{100, 105, 115, 118, 130},
	)

	// WHEN
	effort := BestTimeForDistance(1, "Test ride", "Ride", stream, 200)

	// THEN
	if effort == nil {
		t.Fatalf("expected effort, got nil")
	}
	if effort.Seconds != 20 {
		t.Fatalf("unexpected best time: got %d, want 20", effort.Seconds)
	}
	if math.Abs(effort.DeltaAltitude-15) > 1e-6 {
		t.Fatalf("unexpected delta altitude: got %.2f, want 15", effort.DeltaAltitude)
	}
}

func TestBestTimeForDistance_ReturnsNilWhenTargetTooLong(t *testing.T) {
	// GIVEN
	stream := syntheticStream(
		[]float64{0, 100, 200, 300},
		[]int{0, 10, 20, 30},
		[]float64{100, 102, 104, 106},
	)

	// WHEN
	effort := BestTimeForDistance(1, "Test ride", "Ride", stream, 1000)

	// THEN
	if effort != nil {
		t.Fatalf("expected nil effort for unreachable distance, got %+v", effort)
	}
}

func TestBestDistanceForTime_WithSyntheticStream(t *testing.T) {
	// GIVEN
	stream := syntheticStream(
		[]float64{0, 100, 200, 300, 400},
		[]int{0, 10, 20, 35, 50},
		[]float64{100, 105, 115, 118, 130},
	)

	// WHEN
	effort := BestDistanceForTime(1, "Test ride", "Ride", stream, 20)

	// THEN
	if effort == nil {
		t.Fatalf("expected effort, got nil")
	}
	if math.Abs(effort.Distance-200) > 1e-6 {
		t.Fatalf("unexpected distance: got %.2f, want 200", effort.Distance)
	}
	if effort.Seconds != 20 {
		t.Fatalf("unexpected duration: got %d, want 20", effort.Seconds)
	}
}

func TestBestDistanceEffort_ReturnsNilWhenAltitudeDataIsEmpty(t *testing.T) {
	// GIVEN
	activity := strava.Activity{
		Id:   42,
		Name: "Missing altitude",
		Type: "Ride",
		Stream: &strava.Stream{
			Distance: strava.DistanceStream{Data: []float64{0, 100}},
			Time:     strava.TimeStream{Data: []int{0, 10}},
			Altitude: &strava.AltitudeStream{Data: []float64{}},
		},
	}

	// WHEN
	effort := BestDistanceEffort(activity, 10)

	// THEN
	if effort != nil {
		t.Fatalf("expected nil effort when altitude data is empty, got %+v", effort)
	}
}

func TestBestElevationForDistance_WithSyntheticStream(t *testing.T) {
	// GIVEN
	stream := syntheticStream(
		[]float64{0, 100, 200, 300, 400},
		[]int{0, 10, 20, 35, 50},
		[]float64{100, 105, 115, 118, 130},
	)

	// WHEN
	effort := BestElevationForDistance(1, "Test ride", "Ride", stream, 200)

	// THEN
	if effort == nil {
		t.Fatalf("expected effort, got nil")
	}
	if math.Abs(effort.DeltaAltitude-15) > 1e-6 {
		t.Fatalf("unexpected delta altitude: got %.2f, want 15", effort.DeltaAltitude)
	}
	if effort.Seconds != 20 {
		t.Fatalf("unexpected duration: got %d, want 20", effort.Seconds)
	}
}

func TestBestElevationForDistance_ReturnsNilWhenTargetTooLong(t *testing.T) {
	// GIVEN
	stream := syntheticStream(
		[]float64{0, 100, 200, 300},
		[]int{0, 10, 20, 30},
		[]float64{100, 102, 104, 106},
	)

	// WHEN
	effort := BestElevationForDistance(1, "Test ride", "Ride", stream, 2000)

	// THEN
	if effort != nil {
		t.Fatalf("expected nil effort for unreachable distance, got %+v", effort)
	}
}

func syntheticStream(distances []float64, times []int, altitudes []float64) *strava.Stream {
	return &strava.Stream{
		Distance: strava.DistanceStream{
			Data:         distances,
			OriginalSize: len(distances),
			Resolution:   "high",
			SeriesType:   "distance",
		},
		Time: strava.TimeStream{
			Data:         times,
			OriginalSize: len(times),
			Resolution:   "high",
			SeriesType:   "time",
		},
		Altitude: &strava.AltitudeStream{
			Data:         altitudes,
			OriginalSize: len(altitudes),
			Resolution:   "high",
			SeriesType:   "distance",
		},
	}
}
