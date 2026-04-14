package services

import (
	"mystravastats/domain/business"
	"mystravastats/domain/strava"
	"testing"
)

func TestBuildPersonalRecordsTimeline_BestDistance1HIsChronological(t *testing.T) {
	// GIVEN
	activities := []*strava.Activity{
		rideActivityWithOneHourStream(202503, "2025-03-10T06:00:00Z", "2025-03-10T08:00:00+01:00", 30000),
		rideActivityWithOneHourStream(202501, "2025-01-10T07:00:00Z", "2025-01-10T08:00:00+0100", 20000),
		rideActivityWithOneHourStream(202502, "2025-02-10T07:00:00Z", "2025-02-10T08:00:00Z", 25000),
	}
	metric := "best-distance-1h"

	// WHEN
	timeline := buildPersonalRecordsTimeline(activities, &metric, []business.ActivityType{business.Ride})

	// THEN
	if len(timeline) != 3 {
		t.Fatalf("expected 3 PR events, got %d", len(timeline))
	}

	if timeline[0].ActivityDate != "2025-01-10T08:00:00+0100" {
		t.Fatalf("expected first PR event to be oldest, got %s", timeline[0].ActivityDate)
	}
	if timeline[0].PreviousValue != nil || timeline[0].Improvement != nil {
		t.Fatalf("expected first PR event to be initial PR")
	}

	if timeline[2].ActivityDate != "2025-03-10T08:00:00+01:00" {
		t.Fatalf("expected last PR event to be most recent, got %s", timeline[2].ActivityDate)
	}
	if timeline[2].PreviousValue == nil || timeline[2].Improvement == nil {
		t.Fatalf("expected latest PR event to include previous value and improvement")
	}
}

func rideActivityWithOneHourStream(id int64, startDate string, startDateLocal string, bestDistanceFor1hMeters float64) *strava.Activity {
	return &strava.Activity{
		Id:             id,
		Name:           "Ride timeline",
		Type:           "Ride",
		StartDate:      startDate,
		StartDateLocal: startDateLocal,
		Distance:       bestDistanceFor1hMeters,
		MovingTime:     3600,
		ElapsedTime:    3600,
		Stream: &strava.Stream{
			Distance: strava.DistanceStream{
				Data:         []float64{0, bestDistanceFor1hMeters},
				OriginalSize: 2,
				Resolution:   "high",
				SeriesType:   "distance",
			},
			Time: strava.TimeStream{
				Data:         []int{0, 3600},
				OriginalSize: 2,
				Resolution:   "high",
				SeriesType:   "time",
			},
			Altitude: &strava.AltitudeStream{
				Data:         []float64{100, 120},
				OriginalSize: 2,
				Resolution:   "high",
				SeriesType:   "distance",
			},
		},
	}
}
