package dto

import (
	"testing"

	"mystravastats/domain/badges"
	"mystravastats/domain/business"
	"mystravastats/domain/strava"
)

func TestToAthleteDto_NilDates(t *testing.T) {
	athlete := strava.Athlete{Id: 123}

	dto := ToAthleteDto(athlete)

	if !dto.CreatedAt.IsZero() {
		t.Fatalf("expected CreatedAt to be zero value, got %v", dto.CreatedAt)
	}
	if !dto.UpdatedAt.IsZero() {
		t.Fatalf("expected UpdatedAt to be zero value, got %v", dto.UpdatedAt)
	}
}

func TestBuildActivityEfforts_NilStream(t *testing.T) {
	efforts := BuildActivityEfforts(&strava.DetailedActivity{Id: 42, Stream: nil})

	if len(efforts) != 0 {
		t.Fatalf("expected no efforts when stream is nil, got %d", len(efforts))
	}
}

func TestToStreamDto_DistinctPointersAndValues(t *testing.T) {
	stream := &strava.Stream{
		Distance: strava.DistanceStream{Data: []float64{1, 2}},
		Time:     strava.TimeStream{Data: []int{10, 20}},
		LatLng: &strava.LatLngStream{Data: [][]float64{
			{10.1, 20.2},
			{30.3, 40.4},
		}},
		Moving:         &strava.MovingStream{Data: []bool{true, false}},
		Altitude:       &strava.AltitudeStream{Data: []float64{100.5, 101.5}},
		Watts:          &strava.PowerStream{Data: []float64{210.0, 220.0}},
		VelocitySmooth: &strava.SmoothVelocityStream{Data: []float64{8.5, 8.8}},
	}

	dto := toStreamDto(stream)
	if dto == nil {
		t.Fatal("expected non-nil dto")
	}

	if dto.Moving[0] == dto.Moving[1] {
		t.Fatal("expected moving pointers to be distinct")
	}
	if *dto.Moving[0] != true || *dto.Moving[1] != false {
		t.Fatalf("unexpected moving values: %v %v", *dto.Moving[0], *dto.Moving[1])
	}

	if dto.Altitude[0] == dto.Altitude[1] {
		t.Fatal("expected altitude pointers to be distinct")
	}
	if *dto.Altitude[0] != 100.5 || *dto.Altitude[1] != 101.5 {
		t.Fatalf("unexpected altitude values: %.2f %.2f", *dto.Altitude[0], *dto.Altitude[1])
	}

	if dto.Watts[0] == dto.Watts[1] {
		t.Fatal("expected watts pointers to be distinct")
	}
	if *dto.Watts[0] != 210.0 || *dto.Watts[1] != 220.0 {
		t.Fatalf("unexpected watts values: %.1f %.1f", *dto.Watts[0], *dto.Watts[1])
	}

	if dto.VelocitySmooth[0] == dto.VelocitySmooth[1] {
		t.Fatal("expected velocity pointers to be distinct")
	}
	if *dto.VelocitySmooth[0] != 8.5 || *dto.VelocitySmooth[1] != 8.8 {
		t.Fatalf("unexpected velocity values: %.1f %.1f", *dto.VelocitySmooth[0], *dto.VelocitySmooth[1])
	}

	if dto.Latlng[0][0] == dto.Latlng[0][1] {
		t.Fatal("expected latitude and longitude pointers to be distinct")
	}
	if *dto.Latlng[0][0] != 10.1 || *dto.Latlng[0][1] != 20.2 {
		t.Fatalf("unexpected first latlng values: %.1f %.1f", *dto.Latlng[0][0], *dto.Latlng[0][1])
	}
	if *dto.Latlng[1][0] != 30.3 || *dto.Latlng[1][1] != 40.4 {
		t.Fatalf("unexpected second latlng values: %.1f %.1f", *dto.Latlng[1][0], *dto.Latlng[1][1])
	}
}

func TestComputeFamousClimbEffortSeconds_UsesSegmentDurationNotActivityDuration(t *testing.T) {
	badge := badges.FamousClimbBadge{
		Start: business.GeoCoordinate{Latitude: 45.2178751, Longitude: 6.4750846},
		End:   business.GeoCoordinate{Latitude: 45.2026999, Longitude: 6.4446143},
	}

	activity := &strava.Activity{
		MovingTime: 12000,
		Stream: &strava.Stream{
			Time: strava.TimeStream{Data: []int{0, 100, 700, 1200}},
			LatLng: &strava.LatLngStream{Data: [][]float64{
				{45.1000000, 6.3000000},
				{45.2178751, 6.4750846}, // start at t=100
				{45.2100000, 6.4600000},
				{45.2026999, 6.4446143}, // summit at t=1200
			}},
		},
	}

	effortSeconds, ok := computeFamousClimbEffortSeconds(activity, badge)
	if !ok {
		t.Fatalf("expected effort duration to be detected")
	}
	if effortSeconds != 1100 {
		t.Fatalf("expected effortSeconds=1100, got %d", effortSeconds)
	}
}

func TestComputeFamousClimbEffortSeconds_RejectsDescentOrder(t *testing.T) {
	badge := badges.FamousClimbBadge{
		Start: business.GeoCoordinate{Latitude: 45.2178751, Longitude: 6.4750846},
		End:   business.GeoCoordinate{Latitude: 45.2026999, Longitude: 6.4446143},
	}

	activity := &strava.Activity{
		Stream: &strava.Stream{
			Time: strava.TimeStream{Data: []int{0, 400, 900}},
			LatLng: &strava.LatLngStream{Data: [][]float64{
				{45.2026999, 6.4446143}, // summit first
				{45.2100000, 6.4600000},
				{45.2178751, 6.4750846}, // start after summit
			}},
		},
	}

	_, ok := computeFamousClimbEffortSeconds(activity, badge)
	if ok {
		t.Fatalf("expected descent-only order to be rejected")
	}
}
