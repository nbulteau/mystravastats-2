package application

import (
	dashboardDomain "mystravastats/internal/dashboard/domain"
	"mystravastats/internal/shared/domain/business"
	"testing"
)

type dashboardReaderStub struct {
	dashboardData business.DashboardData
	distance      map[string]map[string]float64
	elevation     map[string]map[string]float64
	heatmap       map[string]map[string]dashboardDomain.ActivityHeatmapDay
	eddington     business.EddingtonNumber
}

func (stub *dashboardReaderStub) FindDashboardData(_ ...business.ActivityType) business.DashboardData {
	return stub.dashboardData
}

func (stub *dashboardReaderStub) FindCumulativeDistancePerYear(_ ...business.ActivityType) map[string]map[string]float64 {
	return stub.distance
}

func (stub *dashboardReaderStub) FindCumulativeElevationPerYear(_ ...business.ActivityType) map[string]map[string]float64 {
	return stub.elevation
}

func (stub *dashboardReaderStub) FindActivityHeatmap(_ ...business.ActivityType) map[string]map[string]dashboardDomain.ActivityHeatmapDay {
	return stub.heatmap
}

func (stub *dashboardReaderStub) FindEddingtonNumber(_ ...business.ActivityType) business.EddingtonNumber {
	return stub.eddington
}

func TestGetCumulativeDataPerYearUseCase_Execute_ReturnsEmptyMapsOnNilReaderResult(t *testing.T) {
	// GIVEN
	reader := &dashboardReaderStub{distance: nil, elevation: nil}
	useCase := NewGetCumulativeDataPerYearUseCase(reader)

	// WHEN
	result := useCase.Execute([]business.ActivityType{business.Ride})

	// THEN
	if result.Distance == nil || result.Elevation == nil {
		t.Fatal("expected non-nil maps")
	}
}

func TestGetActivityHeatmapUseCase_Execute_ReturnsEmptyMapOnNilReaderResult(t *testing.T) {
	// GIVEN
	reader := &dashboardReaderStub{heatmap: nil}
	useCase := NewGetActivityHeatmapUseCase(reader)

	// WHEN
	result := useCase.Execute([]business.ActivityType{business.Ride})

	// THEN
	if result == nil {
		t.Fatal("expected non-nil map")
	}
}

func TestGetEddingtonNumberUseCase_Execute_ReturnsResult(t *testing.T) {
	// GIVEN
	reader := &dashboardReaderStub{eddington: business.EddingtonNumber{Number: 42}}
	useCase := NewGetEddingtonNumberUseCase(reader)

	// WHEN
	result := useCase.Execute([]business.ActivityType{business.Ride})

	// THEN
	if result.Number != 42 {
		t.Fatalf("expected eddington number 42, got %d", result.Number)
	}
}
