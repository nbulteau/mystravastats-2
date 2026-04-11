package dto

import (
	"testing"

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
