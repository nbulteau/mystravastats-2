package infrastructure

import (
	"mystravastats/domain/strava"
	routesDomain "mystravastats/internal/routes/domain"
	"testing"
)

func TestComputeRouteExplorerFromActivities_ReturnsMVPAndVariants(t *testing.T) {
	// GIVEN
	season := "SPRING"
	shape := "LOOP"
	targetDistance := 45.0
	targetElevation := 650.0
	targetDuration := 140
	activities := []*strava.Activity{
		buildRouteActivity(1, "Loop Base", "2025-04-01T08:00:00Z", 43.5, 620, 8100, []float64{45.0, 6.0}, [][]float64{
			{45.0, 6.0}, {45.01, 6.03}, {45.03, 6.02}, {45.0, 6.0},
		}),
		buildRouteActivity(2, "Short Tempo", "2025-04-10T08:00:00Z", 31.2, 420, 5600, []float64{45.1, 6.1}, [][]float64{
			{45.1, 6.1}, {45.12, 6.11}, {45.1, 6.1},
		}),
		buildRouteActivity(3, "Long Endurance", "2025-04-20T08:00:00Z", 72.4, 760, 13200, []float64{45.2, 6.2}, [][]float64{
			{45.2, 6.2}, {45.25, 6.24}, {45.29, 6.22}, {45.2, 6.2},
		}),
		buildRouteActivity(4, "Hill Repeats", "2025-04-23T08:00:00Z", 44.0, 1310, 9200, []float64{45.3, 6.3}, [][]float64{
			{45.3, 6.3}, {45.32, 6.31}, {45.35, 6.33}, {45.3, 6.3},
		}),
	}
	request := routesDomain.RouteExplorerRequest{
		DistanceTargetKm:  &targetDistance,
		ElevationTargetM:  &targetElevation,
		DurationTargetMin: &targetDuration,
		Season:            &season,
		Shape:             &shape,
		Limit:             6,
	}

	// WHEN
	result := computeRouteExplorerFromActivities(activities, request)

	// THEN
	if len(result.ClosestLoops) == 0 {
		t.Fatal("expected at least one closest loop recommendation")
	}
	if len(result.Variants) < 3 {
		t.Fatalf("expected smart variants (shorter/longer/hillier), got %d", len(result.Variants))
	}
	if len(result.Seasonal) == 0 {
		t.Fatal("expected seasonal recommendations")
	}
	if len(result.ShapeMatches) == 0 {
		t.Fatal("expected shape match recommendations for LOOP")
	}
}

func TestComputeRouteExplorerFromActivities_BuildsExperimentalShapeRemix(t *testing.T) {
	// GIVEN
	targetDistance := 35.0
	activities := []*strava.Activity{
		buildRouteActivity(11, "Outbound Segment", "2025-06-05T07:00:00Z", 18.0, 340, 3600, []float64{45.00, 6.00}, [][]float64{
			{45.00, 6.00}, {45.02, 6.02}, {45.05, 6.05},
		}),
		buildRouteActivity(12, "Return Segment", "2025-06-06T07:00:00Z", 17.5, 320, 3500, []float64{45.05, 6.05}, [][]float64{
			{45.05, 6.05}, {45.02, 6.02}, {45.00, 6.00},
		}),
	}
	request := routesDomain.RouteExplorerRequest{
		DistanceTargetKm: &targetDistance,
		IncludeRemix:     true,
		Limit:            4,
	}

	// WHEN
	result := computeRouteExplorerFromActivities(activities, request)

	// THEN
	if len(result.ShapeRemixes) == 0 {
		t.Fatal("expected at least one shape remix recommendation")
	}
	if !result.ShapeRemixes[0].Experimental {
		t.Fatal("expected shape remix recommendation to be experimental")
	}
	if len(result.ShapeRemixes[0].Components) != 2 {
		t.Fatalf("expected remix to contain 2 components, got %d", len(result.ShapeRemixes[0].Components))
	}
}

func buildRouteActivity(
	id int64,
	name string,
	startDate string,
	distanceKm float64,
	elevationM float64,
	durationSec int,
	start []float64,
	latLng [][]float64,
) *strava.Activity {
	return &strava.Activity{
		Id:                 id,
		Name:               name,
		Type:               "Ride",
		SportType:          "Ride",
		StartDate:          startDate,
		StartDateLocal:     startDate,
		Distance:           distanceKm * 1000.0,
		TotalElevationGain: elevationM,
		MovingTime:         durationSec,
		ElapsedTime:        durationSec,
		StartLatlng:        start,
		Stream: &strava.Stream{
			LatLng: &strava.LatLngStream{
				Data: latLng,
			},
		},
	}
}
