package infrastructure

import (
	"mystravastats/internal/routes/application"
	routesDomain "mystravastats/internal/routes/domain"
	"testing"
)

func TestOSMRoutingAdapter_WhenDisabled_ReturnsDisabledHealthAndNoGeneratedLoops(t *testing.T) {
	// GIVEN
	t.Setenv("OSM_ROUTING_ENABLED", "false")
	adapter := NewOSMRoutingAdapter()
	request := application.RoutingEngineRequest{
		StartPoint:       routesDomain.Coordinates{Lat: 48.13, Lng: -1.63},
		DistanceTargetKm: 40.0,
		StartDirection:   "N",
		RouteType:        "RIDE",
		Limit:            4,
	}

	// WHEN
	health := adapter.HealthDetails()
	routes, err := adapter.GenerateTargetLoops(request)

	// THEN
	if err != nil {
		t.Fatalf("expected no error when adapter is disabled, got %v", err)
	}
	if len(routes) != 0 {
		t.Fatalf("expected no generated routes when adapter is disabled, got %d", len(routes))
	}
	if got := health["status"]; got != "disabled" {
		t.Fatalf("expected health status=disabled, got %v", got)
	}
	if got := health["enabled"]; got != false {
		t.Fatalf("expected enabled=false, got %v", got)
	}
}

func TestRespectsHalfPlaneDirection_NorthRejectsPointsSouthOfStart(t *testing.T) {
	// GIVEN
	start := routesDomain.Coordinates{Lat: 48.13000, Lng: -1.63000}
	points := [][]float64{
		{48.13010, -1.63000},
		{48.13050, -1.62000},
		{48.12840, -1.61000}, // south of start by more than tolerance
	}

	// WHEN
	ok := respectsHalfPlaneDirection(points, start, "N", 120.0)

	// THEN
	if ok {
		t.Fatal("expected north direction filter to reject points south of start")
	}
}

func TestHasOppositeEdgeTraversal_DetectsBacktracking(t *testing.T) {
	// GIVEN
	points := [][]float64{
		{48.13000, -1.63000},
		{48.13100, -1.62900},
		{48.13200, -1.62800},
		{48.13100, -1.62900}, // traverses previous edge in reverse
	}

	// WHEN
	hasBacktracking := hasOppositeEdgeTraversal(points)

	// THEN
	if !hasBacktracking {
		t.Fatal("expected opposite-direction edge traversal to be detected")
	}
}
