package application

import (
	"mystravastats/domain/business"
	routesDomain "mystravastats/internal/routes/domain"
	"testing"
)

type routesReaderStub struct {
	result        routesDomain.RouteExplorerResult
	receivedYear  *int
	receivedReq   routesDomain.RouteExplorerRequest
	receivedTypes []business.ActivityType
}

func (stub *routesReaderStub) FindRouteExplorerByYearAndTypes(
	year *int,
	request routesDomain.RouteExplorerRequest,
	activityTypes ...business.ActivityType,
) routesDomain.RouteExplorerResult {
	stub.receivedYear = year
	stub.receivedReq = request
	stub.receivedTypes = append([]business.ActivityType(nil), activityTypes...)
	return stub.result
}

func TestGetRouteExplorerUseCase_Execute_ForwardsInputs(t *testing.T) {
	// GIVEN
	year := 2025
	targetDistance := 80.0
	reader := &routesReaderStub{
		result: routesDomain.RouteExplorerResult{
			ClosestLoops: []routesDomain.RouteRecommendation{{ActivityDate: "2025-07-10"}},
		},
	}
	useCase := NewGetRouteExplorerUseCase(reader)

	// WHEN
	result := useCase.Execute(
		&year,
		routesDomain.RouteExplorerRequest{
			DistanceTargetKm: &targetDistance,
		},
		[]business.ActivityType{business.Ride},
	)

	// THEN
	if len(result.ClosestLoops) != 1 {
		t.Fatalf("expected one closest loop, got %d", len(result.ClosestLoops))
	}
	if reader.receivedYear == nil || *reader.receivedYear != year {
		t.Fatalf("expected year %d, got %v", year, reader.receivedYear)
	}
	if reader.receivedReq.DistanceTargetKm == nil || *reader.receivedReq.DistanceTargetKm != targetDistance {
		t.Fatalf("expected forwarded target distance %.1f, got %v", targetDistance, reader.receivedReq.DistanceTargetKm)
	}
	if len(reader.receivedTypes) != 1 || reader.receivedTypes[0] != business.Ride {
		t.Fatalf("expected forwarded activity type Ride, got %v", reader.receivedTypes)
	}
}

func TestGetRouteExplorerUseCase_Execute_NormalizesNilSlices(t *testing.T) {
	// GIVEN
	reader := &routesReaderStub{
		result: routesDomain.RouteExplorerResult{},
	}
	useCase := NewGetRouteExplorerUseCase(reader)

	// WHEN
	result := useCase.Execute(nil, routesDomain.RouteExplorerRequest{}, []business.ActivityType{business.Ride})

	// THEN
	if result.ClosestLoops == nil {
		t.Fatal("expected non-nil closest loops")
	}
	if result.Variants == nil {
		t.Fatal("expected non-nil variants")
	}
	if result.Seasonal == nil {
		t.Fatal("expected non-nil seasonal")
	}
	if result.ShapeMatches == nil {
		t.Fatal("expected non-nil shape matches")
	}
	if result.ShapeRemixes == nil {
		t.Fatal("expected non-nil shape remixes")
	}
}
