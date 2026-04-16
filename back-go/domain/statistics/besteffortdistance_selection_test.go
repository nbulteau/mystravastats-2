package statistics

import (
	"mystravastats/internal/shared/domain/strava"
	"testing"
)

func TestFindBestActivityEffort_SelectsFastestEffortAcrossActivities(t *testing.T) {
	// GIVEN
	ClearBestEffortCache()
	slowActivity := &strava.Activity{
		Id:   7101,
		Name: "Slow effort",
		Type: "Ride",
		Stream: syntheticStream(
			[]float64{0, 100, 200, 300},
			[]int{0, 20, 40, 60},
			[]float64{100, 102, 104, 106},
		),
	}
	fastActivity := &strava.Activity{
		Id:   7102,
		Name: "Fast effort",
		Type: "Ride",
		Stream: syntheticStream(
			[]float64{0, 100, 200, 300},
			[]int{0, 10, 20, 30},
			[]float64{100, 103, 106, 109},
		),
	}

	// WHEN
	result := FindBestActivityEffort([]*strava.Activity{slowActivity, fastActivity}, 200.0)

	// THEN
	if result == nil {
		t.Fatal("expected a best effort, got nil")
	}
	if result.Seconds != 20 {
		t.Fatalf("expected best effort to be 20s, got %ds", result.Seconds)
	}
	if result.ActivityShort.Id != 7102 {
		t.Fatalf("expected best effort to come from activity 7102, got %d", result.ActivityShort.Id)
	}
}

func TestFindBestActivityEffort_SkipsInvalidStreamsAndReturnsNilWhenNoValidEffort(t *testing.T) {
	// GIVEN
	ClearBestEffortCache()
	withoutStream := &strava.Activity{
		Id:     7201,
		Name:   "No stream",
		Type:   "Ride",
		Stream: nil,
	}
	withoutAltitude := &strava.Activity{
		Id:   7202,
		Name: "No altitude",
		Type: "Ride",
		Stream: &strava.Stream{
			Distance: strava.DistanceStream{Data: []float64{0, 100, 200}},
			Time:     strava.TimeStream{Data: []int{0, 10, 20}},
			Altitude: nil,
		},
	}
	withEmptyAltitude := &strava.Activity{
		Id:   7203,
		Name: "Empty altitude",
		Type: "Ride",
		Stream: &strava.Stream{
			Distance: strava.DistanceStream{Data: []float64{0, 100, 200}},
			Time:     strava.TimeStream{Data: []int{0, 10, 20}},
			Altitude: &strava.AltitudeStream{Data: []float64{}},
		},
	}

	// WHEN
	result := FindBestActivityEffort([]*strava.Activity{
		withoutStream,
		withoutAltitude,
		withEmptyAltitude,
	}, 200.0)

	// THEN
	if result != nil {
		t.Fatalf("expected nil best effort when no valid stream is available, got %+v", result)
	}
}
