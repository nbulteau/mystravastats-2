package infrastructure

import (
	"mystravastats/internal/shared/domain/strava"
	"strings"
	"testing"
)

func TestGenerateRideHeaderIncludesEnrichedColumns(t *testing.T) {
	header := generateRideHeader()

	for _, column := range []string{"Activity ID", "Gear ID", "Has GPS stream", "Data quality flags"} {
		if !strings.Contains(header, column) {
			t.Fatalf("expected enriched column %q in header %q", column, header)
		}
	}
}

func TestGenerateRideActivityIncludesEnrichedDataAndEscapedFields(t *testing.T) {
	gearID := "b123"
	activity := &strava.Activity{
		Id:                 42,
		Name:               "Morning, ride",
		Type:               "Ride",
		SportType:          "GravelRide",
		Commute:            true,
		GearId:             &gearID,
		Distance:           10000,
		ElapsedTime:        1800,
		MovingTime:         1700,
		StartDateLocal:     "2026-04-26T08:30:00Z",
		TotalElevationGain: 250,
		UploadId:           99,
	}

	line := generateRideActivity(activity)

	for _, expected := range []string{
		"\"Morning, ride\"",
		"42",
		"GravelRide",
		"b123",
		"https://www.strava.com/activities/42",
		"yes",
		"missing_stream",
	} {
		if !strings.Contains(line, expected) {
			t.Fatalf("expected %q in CSV line %q", expected, line)
		}
	}
}
