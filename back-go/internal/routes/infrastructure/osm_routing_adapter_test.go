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

func TestHasMinimumSegmentDiversity_RejectsOverusedSegments(t *testing.T) {
	// GIVEN
	points := [][]float64{
		{48.13000, -1.63000},
		{48.13100, -1.62900},
		{48.13200, -1.62800},
		{48.13300, -1.62700},
		{48.13200, -1.62800},
		{48.13300, -1.62700},
		{48.13200, -1.62800},
	}

	// WHEN
	ok := hasMinimumSegmentDiversity(points, "RIDE")

	// THEN
	if ok {
		t.Fatal("expected diversity filter to reject a route reusing the same segment too often")
	}
}

func TestBuildOSRMScoringProfile_CalibratesWeightsByRouteType(t *testing.T) {
	// GIVEN
	rideProfile := buildOSRMScoringProfile("RIDE", true, false)
	hikeProfile := buildOSRMScoringProfile("HIKE", true, false)

	// WHEN
	hikeElevationHigher := hikeProfile.elevationWeight > rideProfile.elevationWeight
	hikeDistanceLower := hikeProfile.distanceWeight < rideProfile.distanceWeight

	// THEN
	if !hikeElevationHigher {
		t.Fatalf("expected hike profile elevation weight to be higher than ride, ride=%f hike=%f", rideProfile.elevationWeight, hikeProfile.elevationWeight)
	}
	if !hikeDistanceLower {
		t.Fatalf("expected hike profile distance weight to be lower than ride, ride=%f hike=%f", rideProfile.distanceWeight, hikeProfile.distanceWeight)
	}
}

func TestOSRMMatchScore_PenalizesOppositeDirection(t *testing.T) {
	// GIVEN
	elevationTarget := 600.0
	request := application.RoutingEngineRequest{
		StartPoint:       routesDomain.Coordinates{Lat: 48.13000, Lng: -1.63000},
		DistanceTargetKm: 40.0,
		ElevationTargetM: &elevationTarget,
		StartDirection:   "N",
		RouteType:        "RIDE",
		Limit:            4,
	}
	northPoints := [][]float64{
		{48.13000, -1.63000},
		{48.15000, -1.63000},
		{48.13000, -1.62000},
		{48.13000, -1.63000},
	}
	southPoints := [][]float64{
		{48.13000, -1.63000},
		{48.11000, -1.63000},
		{48.13000, -1.62000},
		{48.13000, -1.63000},
	}

	// WHEN
	northScore := osrmMatchScore(request, 40.0, 600.0, northPoints)
	southScore := osrmMatchScore(request, 40.0, 600.0, southPoints)

	// THEN
	if northScore <= southScore {
		t.Fatalf("expected north-aligned route to score higher than south-aligned route, north=%f south=%f", northScore, southScore)
	}
}

func TestSelectCandidatesWithRelaxation_WhenStrictFails_ThenRelaxedCandidateCanStillBeSelected(t *testing.T) {
	// GIVEN
	request := application.RoutingEngineRequest{
		StartPoint:       routesDomain.Coordinates{Lat: 48.13000, Lng: -1.63000},
		DistanceTargetKm: 40.0,
		StartDirection:   "N",
		RouteType:        "RIDE",
		Limit:            3,
	}
	candidates := []osrmRouteCandidate{
		{
			recommendation: routesDomain.RouteRecommendation{
				RouteID:       "candidate-relaxed",
				MatchScore:    88.0,
				PreviewLatLng: [][]float64{{48.13, -1.63}, {48.16, -1.62}, {48.13, -1.63}},
			},
			directionPenalty:    0.24, // strict rejects (>0.18), balanced accepts (<=0.28)
			backtrackingRatio:   0.04,
			segmentDiversity:    0.40,
			distanceDeltaRatio:  0.32,
			effectiveMatchScore: 82.0,
		},
	}
	rejectCounts := map[string]int{}

	// WHEN
	recommendations := selectCandidatesWithRelaxation(request, candidates, rejectCounts)

	// THEN
	if len(recommendations) != 1 {
		t.Fatalf("expected 1 recommendation after relaxation, got %d", len(recommendations))
	}
	if recommendations[0].RouteID != "candidate-relaxed" {
		t.Fatalf("expected candidate-relaxed to be selected, got %s", recommendations[0].RouteID)
	}
}

func TestDirectionalLobePenalty_PenalizesRoutesDominatingOppositeDirection(t *testing.T) {
	// GIVEN
	start := routesDomain.Coordinates{Lat: 48.13000, Lng: -1.63000}
	northDominant := [][]float64{
		{48.13000, -1.63000},
		{48.14600, -1.62500},
		{48.14200, -1.61800},
		{48.13100, -1.62200},
		{48.13000, -1.63000},
	}
	southDominant := [][]float64{
		{48.13000, -1.63000},
		{48.11200, -1.62500},
		{48.11000, -1.61700},
		{48.12600, -1.62000},
		{48.13000, -1.63000},
	}

	// WHEN
	northPenalty := directionalLobePenalty(northDominant, start, "N")
	southPenalty := directionalLobePenalty(southDominant, start, "N")

	// THEN
	if northPenalty >= southPenalty {
		t.Fatalf("expected north-dominant route to have lower lobe penalty, north=%.3f south=%.3f", northPenalty, southPenalty)
	}
}

func TestSyntheticLoopWaypoints_WithNorthDirection_StayInForwardHemisphere(t *testing.T) {
	// GIVEN
	adapter := NewOSMRoutingAdapter()
	start := routesDomain.Coordinates{Lat: 48.13000, Lng: -1.63000}

	// WHEN
	waypoints := adapter.syntheticLoopWaypoints(start, 6.0, 0.0, "N", 0)

	// THEN
	if len(waypoints) < 3 {
		t.Fatalf("expected generated waypoints, got %d", len(waypoints))
	}
	for index, waypoint := range waypoints {
		// start/end are expected to be at start point; we only assert intermediate points.
		if index == 0 || index == len(waypoints)-1 {
			continue
		}
		if waypoint.Lat < start.Lat-0.0005 {
			t.Fatalf("expected directional waypoint %d to stay north-oriented, startLat=%.5f gotLat=%.5f", index, start.Lat, waypoint.Lat)
		}
	}
}

func TestBuildRouteRelaxationLevels_WhenDirectionStrict_ThenUsesStricterDirectionThresholds(t *testing.T) {
	// GIVEN
	regular := buildRouteRelaxationLevels("RIDE", true, false)
	strict := buildRouteRelaxationLevels("RIDE", true, true)

	// WHEN
	regularStrictMax := regular[0].maxDirectionPenalty
	strictStrictMax := strict[0].maxDirectionPenalty
	regularFallbackMax := regular[len(regular)-1].maxDirectionPenalty
	strictFallbackMax := strict[len(strict)-1].maxDirectionPenalty

	// THEN
	if strictStrictMax >= regularStrictMax {
		t.Fatalf("expected strict mode to tighten strict-level direction threshold, strict=%f regular=%f", strictStrictMax, regularStrictMax)
	}
	if strictFallbackMax >= regularFallbackMax {
		t.Fatalf("expected strict mode to tighten fallback-level direction threshold, strict=%f regular=%f", strictFallbackMax, regularFallbackMax)
	}
}

func TestHalfPlaneViolationRatio_WhenPointsCrossForbiddenHalfPlane_ThenPenaltyIsPositive(t *testing.T) {
	// GIVEN
	start := routesDomain.Coordinates{Lat: 48.13000, Lng: -1.63000}
	points := [][]float64{
		{48.13010, -1.63000},
		{48.12980, -1.62900},
		{48.12890, -1.62800}, // south of start for a north direction request
	}

	// WHEN
	penalty := halfPlaneViolationRatio(points, start, "N", 120.0)

	// THEN
	if penalty <= 0 {
		t.Fatalf("expected positive half-plane violation penalty, got %.3f", penalty)
	}
}

func TestSelectCandidatesWithRelaxation_PrioritizesLowerBacktracking(t *testing.T) {
	// GIVEN
	request := application.RoutingEngineRequest{
		StartPoint:       routesDomain.Coordinates{Lat: 48.13000, Lng: -1.63000},
		DistanceTargetKm: 40.0,
		StartDirection:   "N",
		RouteType:        "RIDE",
		Limit:            1,
	}
	candidates := []osrmRouteCandidate{
		{
			recommendation:      routesDomain.RouteRecommendation{RouteID: "high-backtracking", MatchScore: 95.0},
			directionPenalty:    0.10,
			backtrackingRatio:   0.22,
			segmentDiversity:    0.60,
			distanceDeltaRatio:  0.10,
			effectiveMatchScore: 94.0,
		},
		{
			recommendation:      routesDomain.RouteRecommendation{RouteID: "low-backtracking", MatchScore: 90.0},
			directionPenalty:    0.10,
			backtrackingRatio:   0.02,
			segmentDiversity:    0.60,
			distanceDeltaRatio:  0.10,
			effectiveMatchScore: 88.0,
		},
	}
	rejectCounts := map[string]int{}

	// WHEN
	recommendations := selectCandidatesWithRelaxation(request, candidates, rejectCounts)

	// THEN
	if len(recommendations) != 1 {
		t.Fatalf("expected 1 recommendation, got %d", len(recommendations))
	}
	if recommendations[0].RouteID != "low-backtracking" {
		t.Fatalf("expected low-backtracking route to be selected first, got %s", recommendations[0].RouteID)
	}
}

func TestCorridorOverlapRatio_DetectsNearParallelOutAndBackCorridor(t *testing.T) {
	// GIVEN
	points := [][]float64{
		{48.13000, -1.63000},
		{48.13000, -1.62000},
		{48.13020, -1.62000},
		{48.13020, -1.63000},
		{48.13000, -1.63000},
	}

	// WHEN
	overlapRatio := corridorOverlapRatio(points)

	// THEN
	if overlapRatio <= 0.0 {
		t.Fatalf("expected a positive corridor overlap ratio, got %.3f", overlapRatio)
	}
}
