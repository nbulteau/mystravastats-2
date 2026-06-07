package statistics

import (
	"math"
	"mystravastats/internal/shared/domain/strava"
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
	if effort.ElevationGain == nil || math.Abs(*effort.ElevationGain-15) > 1e-6 {
		t.Fatalf("unexpected elevation gain: got %v, want 15", effort.ElevationGain)
	}
	if effort.ElevationLoss == nil || math.Abs(*effort.ElevationLoss) > 1e-6 {
		t.Fatalf("unexpected elevation loss: got %v, want 0", effort.ElevationLoss)
	}
}

func TestBestDistanceForTime_ComputesCumulativeElevationWhenNetDeltaIsZero(t *testing.T) {
	// GIVEN
	stream := syntheticStream(
		[]float64{0, 100, 200, 300, 400},
		[]int{0, 10, 20, 30, 40},
		[]float64{100, 120, 100, 125, 100},
	)

	// WHEN
	effort := BestDistanceForTime(1, "Rolling ride", "Ride", stream, 40)

	// THEN
	if effort == nil {
		t.Fatalf("expected effort, got nil")
	}
	if math.Abs(effort.DeltaAltitude) > 1e-6 {
		t.Fatalf("unexpected net altitude delta: got %.2f, want 0", effort.DeltaAltitude)
	}
	if effort.ElevationGain == nil || math.Abs(*effort.ElevationGain-45) > 1e-6 {
		t.Fatalf("unexpected elevation gain: got %v, want 45", effort.ElevationGain)
	}
	if effort.ElevationLoss == nil || math.Abs(*effort.ElevationLoss-45) > 1e-6 {
		t.Fatalf("unexpected elevation loss: got %v, want 45", effort.ElevationLoss)
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

func TestBestPowerForTime_ReturnsNilWhenPowerStreamIsTruncated(t *testing.T) {
	// GIVEN
	stream := syntheticStream(
		[]float64{0, 100, 200, 300},
		[]int{0, 10, 20, 30},
		[]float64{100, 102, 104, 106},
	)
	stream.Watts = &strava.PowerStream{Data: []float64{180}}
	activity := strava.Activity{
		Id:     43,
		Name:   "Truncated virtual ride",
		Type:   "VirtualRide",
		Stream: stream,
	}

	// WHEN
	effort := BestPowerForTime(activity, 20)

	// THEN
	if effort != nil {
		t.Fatalf("expected nil effort when power stream is truncated, got %+v", effort)
	}
}

func TestBestPowerForDistance_WithSyntheticStream(t *testing.T) {
	// GIVEN
	stream := syntheticStream(
		[]float64{0, 500, 1000, 1500},
		[]int{0, 30, 60, 90},
		[]float64{100, 105, 110, 120},
	)
	stream.Watts = &strava.PowerStream{Data: []float64{100, 150, 200, 400}}
	activity := strava.Activity{
		Id:     44,
		Name:   "Power test",
		Type:   "Ride",
		Stream: stream,
	}

	// WHEN
	effort := BestPowerForDistance(activity, 1000)

	// THEN
	if effort == nil {
		t.Fatalf("expected effort, got nil")
	}
	if effort.AveragePower == nil || math.Abs(*effort.AveragePower-250) > 1e-6 {
		t.Fatalf("unexpected average power: got %v, want 250", effort.AveragePower)
	}
	if effort.Label != "Best Power for 1000 m" {
		t.Fatalf("unexpected label: got %q", effort.Label)
	}
	if effort.IdxStart != 1 || effort.IdxEnd != 3 {
		t.Fatalf("unexpected indexes: got %d-%d, want 1-3", effort.IdxStart, effort.IdxEnd)
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
