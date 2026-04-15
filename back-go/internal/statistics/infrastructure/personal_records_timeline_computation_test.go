package infrastructure

import (
	"mystravastats/domain/business"
	"mystravastats/domain/strava"
	"testing"
)

func TestBuildPersonalRecordsTimeline_OrdersChronologicallyWithInitialPREarliest(t *testing.T) {
	// GIVEN
	metric := "  max-distance-activity  "
	activities := []*strava.Activity{
		{Id: 901, Name: "June ride", Type: "Ride", StartDateLocal: "2025-06-15T08:00:00Z", Distance: 14000, MovingTime: 2500},
		{Id: 902, Name: "January ride", Type: "Ride", StartDateLocal: "2025-01-10T08:00:00Z", Distance: 9000, MovingTime: 1800},
		{Id: 903, Name: "February ride", Type: "Ride", StartDateLocal: "2025-02-01T08:00:00Z", Distance: 10000, MovingTime: 1900},
		{Id: 904, Name: "March ride", Type: "Ride", StartDateLocal: "2025-03-01T08:00:00Z", Distance: 15000, MovingTime: 2600},
	}

	// WHEN
	timeline := buildPersonalRecordsTimeline(activities, &metric, []business.ActivityType{business.Ride})

	// THEN
	if len(timeline) != 3 {
		t.Fatalf("expected 3 PR events, got %d", len(timeline))
	}
	if timeline[0].ActivityDate != "2025-01-10T08:00:00Z" {
		t.Fatalf("expected first PR on 2025-01-10, got %s", timeline[0].ActivityDate)
	}
	if timeline[0].PreviousValue != nil {
		t.Fatalf("expected first PR to have no previous value, got %v", *timeline[0].PreviousValue)
	}
	if timeline[0].Improvement != nil {
		t.Fatalf("expected first PR to have no improvement label, got %v", *timeline[0].Improvement)
	}
	if timeline[1].ActivityDate != "2025-02-01T08:00:00Z" {
		t.Fatalf("expected second PR on 2025-02-01, got %s", timeline[1].ActivityDate)
	}
	if timeline[1].PreviousValue == nil || *timeline[1].PreviousValue != "9.00 km" {
		t.Fatalf("expected second PR previous value to be 9.00 km, got %v", timeline[1].PreviousValue)
	}
	if timeline[1].Improvement == nil || *timeline[1].Improvement != "1.00 km farther" {
		t.Fatalf("expected second PR improvement to be 1.00 km farther, got %v", timeline[1].Improvement)
	}
	if timeline[2].ActivityDate != "2025-03-01T08:00:00Z" {
		t.Fatalf("expected third PR on 2025-03-01, got %s", timeline[2].ActivityDate)
	}
	if timeline[2].PreviousValue == nil || *timeline[2].PreviousValue != "10.00 km" {
		t.Fatalf("expected third PR previous value to be 10.00 km, got %v", timeline[2].PreviousValue)
	}
}

func TestBuildPersonalRecordsTimeline_ReturnsEmptyWhenMetricDoesNotExist(t *testing.T) {
	// GIVEN
	metric := "non-existent-metric"
	activities := []*strava.Activity{
		{Id: 910, Name: "Ride", Type: "Ride", StartDateLocal: "2025-01-01T08:00:00Z", Distance: 12000, MovingTime: 2000},
	}

	// WHEN
	timeline := buildPersonalRecordsTimeline(activities, &metric, []business.ActivityType{business.Ride})

	// THEN
	if len(timeline) != 0 {
		t.Fatalf("expected no timeline entries for unknown metric, got %d", len(timeline))
	}
}
