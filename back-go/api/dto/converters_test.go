package dto

import (
	"math"
	"strings"
	"testing"

	"mystravastats/domain/badges"
	"mystravastats/internal/shared/domain/business"
	"mystravastats/internal/shared/domain/strava"
)

func TestToAthleteDto_NilDates(t *testing.T) {
	// GIVEN
	athlete := strava.Athlete{Id: 123}

	// WHEN
	dto := ToAthleteDto(athlete)

	// THEN
	if !dto.CreatedAt.IsZero() {
		t.Fatalf("expected CreatedAt to be zero value, got %v", dto.CreatedAt)
	}
	if !dto.UpdatedAt.IsZero() {
		t.Fatalf("expected UpdatedAt to be zero value, got %v", dto.UpdatedAt)
	}
}

func TestBuildActivityEfforts_NilStream(t *testing.T) {
	// GIVEN
	detailedActivity := &strava.DetailedActivity{Id: 42, Stream: nil}

	// WHEN
	efforts := BuildActivityEfforts(detailedActivity)

	// THEN
	if len(efforts) != 0 {
		t.Fatalf("expected no efforts when stream is nil, got %d", len(efforts))
	}
}

func TestBuildActivityEfforts_DirectionAwareSegmentLabels(t *testing.T) {
	// GIVEN
	segmentName := "MURAILLE DE CHINE <Alpe d'Huez>"
	detailedActivity := &strava.DetailedActivity{
		Id:   42,
		Name: "Direction check",
		Type: "Ride",
		Stream: &strava.Stream{
			Distance: strava.DistanceStream{Data: []float64{0, 100, 200, 300, 400, 500, 600}},
			Time:     strava.TimeStream{Data: []int{0, 10, 20, 30, 40, 50, 60}},
			Altitude: &strava.AltitudeStream{Data: []float64{100, 102, 105, 108, 106, 104, 102}},
		},
		SegmentEfforts: []strava.SegmentEffort{
			{
				Id:          1001,
				Name:        "Muraille montée",
				Distance:    300,
				ElapsedTime: 30,
				StartIndex:  0,
				EndIndex:    3,
				Segment: strava.Segment{
					Id:            9001,
					Name:          segmentName,
					ActivityType:  "Ride",
					AverageGrade:  8.0,
					ClimbCategory: 4,
					ElevationHigh: 108,
					ElevationLow:  100,
				},
			},
			{
				Id:          1002,
				Name:        "Muraille descente",
				Distance:    300,
				ElapsedTime: 20,
				StartIndex:  3,
				EndIndex:    6,
				Segment: strava.Segment{
					Id:            9002,
					Name:          segmentName,
					ActivityType:  "Ride",
					AverageGrade:  -7.5,
					ClimbCategory: 4,
					ElevationHigh: 108,
					ElevationLow:  100,
				},
			},
		},
	}

	// WHEN
	efforts := BuildActivityEfforts(detailedActivity)

	// THEN
	foundAscent := false
	foundDescent := false
	for _, effort := range efforts {
		if !strings.Contains(effort.Label, segmentName) {
			continue
		}
		if strings.Contains(effort.Label, "(ascent)") {
			foundAscent = true
			if effort.DeltaAltitude <= 0 {
				t.Fatalf("expected ascent delta altitude to be positive, got %.2f", effort.DeltaAltitude)
			}
		}
		if strings.Contains(effort.Label, "(descent)") {
			foundDescent = true
			if effort.DeltaAltitude >= 0 {
				t.Fatalf("expected descent delta altitude to be negative, got %.2f", effort.DeltaAltitude)
			}
		}
	}

	if !foundAscent {
		t.Fatalf("expected ascent segment effort label for %q", segmentName)
	}
	if !foundDescent {
		t.Fatalf("expected descent segment effort label for %q", segmentName)
	}
}

func TestToBadgeDto_UsesRepresentativeBadgeActivityTypeForRideVariants(t *testing.T) {
	dto := ToBadgeDto(badges.DistanceRideLevel1, business.GravelRide, business.MountainBikeRide, business.Ride)

	if dto.Type != "RideDistanceBadge" {
		t.Fatalf("expected RideDistanceBadge, got %q", dto.Type)
	}
}

func TestToBadgeDto_UsesRepresentativeBadgeActivityTypeForTrailRun(t *testing.T) {
	dto := ToBadgeDto(badges.DistanceRunLevel1, business.TrailRun)

	if dto.Type != "RunDistanceBadge" {
		t.Fatalf("expected RunDistanceBadge, got %q", dto.Type)
	}
}

func TestToBadgeDto_UsesRepresentativeBadgeActivityTypeForWalk(t *testing.T) {
	dto := ToBadgeDto(badges.DistanceHikeLevel1, business.Walk)

	if dto.Type != "HikeDistanceBadge" {
		t.Fatalf("expected HikeDistanceBadge, got %q", dto.Type)
	}
}

func TestBuildActivityEfforts_NaNAltitudeFallsBackToSegmentDelta(t *testing.T) {
	// GIVEN
	detailedActivity := &strava.DetailedActivity{
		Id:   52,
		Name: "NaN direction check",
		Type: "Ride",
		Stream: &strava.Stream{
			Distance: strava.DistanceStream{Data: []float64{0, 100, 200}},
			Time:     strava.TimeStream{Data: []int{0, 10, 20}},
			Altitude: &strava.AltitudeStream{Data: []float64{100, math.NaN(), 110}},
		},
		SegmentEfforts: []strava.SegmentEffort{
			{
				Id:          2001,
				Name:        "NaN climb",
				Distance:    200,
				ElapsedTime: 20,
				StartIndex:  0,
				EndIndex:    2,
				Segment: strava.Segment{
					Id:            9901,
					Name:          "NaN climb segment",
					ActivityType:  "Ride",
					AverageGrade:  5.0,
					ClimbCategory: 4,
					ElevationHigh: 120,
					ElevationLow:  100,
				},
			},
		},
	}

	// WHEN
	efforts := BuildActivityEfforts(detailedActivity)

	// THEN
	for _, effort := range efforts {
		if !strings.Contains(effort.Label, "NaN climb segment") {
			continue
		}
		if math.IsNaN(effort.DeltaAltitude) || math.IsInf(effort.DeltaAltitude, 0) {
			t.Fatalf("expected finite delta altitude, got %.2f", effort.DeltaAltitude)
		}
		if effort.DeltaAltitude <= 0 {
			t.Fatalf("expected fallback ascent delta altitude to remain positive, got %.2f", effort.DeltaAltitude)
		}
		return
	}

	t.Fatalf("expected segment effort to be present")
}

func TestToStreamDto_MapsValues(t *testing.T) {
	// GIVEN
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

	// WHEN
	dto := toStreamDto(stream)

	// THEN
	if dto == nil {
		t.Fatal("expected non-nil dto")
	}

	if dto.Moving[0] != true || dto.Moving[1] != false {
		t.Fatalf("unexpected moving values: %v %v", dto.Moving[0], dto.Moving[1])
	}

	if dto.Altitude[0] != 100.5 || dto.Altitude[1] != 101.5 {
		t.Fatalf("unexpected altitude values: %.2f %.2f", dto.Altitude[0], dto.Altitude[1])
	}

	if dto.Watts[0] != 210.0 || dto.Watts[1] != 220.0 {
		t.Fatalf("unexpected watts values: %.1f %.1f", dto.Watts[0], dto.Watts[1])
	}

	if dto.VelocitySmooth[0] != 8.5 || dto.VelocitySmooth[1] != 8.8 {
		t.Fatalf("unexpected velocity values: %.1f %.1f", dto.VelocitySmooth[0], dto.VelocitySmooth[1])
	}

	if dto.Latlng[0][0] != 10.1 || dto.Latlng[0][1] != 20.2 {
		t.Fatalf("unexpected first latlng values: %.1f %.1f", dto.Latlng[0][0], dto.Latlng[0][1])
	}
	if dto.Latlng[1][0] != 30.3 || dto.Latlng[1][1] != 40.4 {
		t.Fatalf("unexpected second latlng values: %.1f %.1f", dto.Latlng[1][0], dto.Latlng[1][1])
	}
}

func TestComputeFamousClimbEffortSeconds_UsesSegmentDurationNotActivityDuration(t *testing.T) {
	// GIVEN
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

	// WHEN
	effortSeconds, ok := computeFamousClimbEffortSeconds(activity, badge)

	// THEN
	if !ok {
		t.Fatalf("expected effort duration to be detected")
	}
	if effortSeconds != 1100 {
		t.Fatalf("expected effortSeconds=1100, got %d", effortSeconds)
	}
}

func TestComputeFamousClimbEffortSeconds_RejectsDescentOrder(t *testing.T) {
	// GIVEN
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

	// WHEN
	_, ok := computeFamousClimbEffortSeconds(activity, badge)

	// THEN
	if ok {
		t.Fatalf("expected descent-only order to be rejected")
	}
}
