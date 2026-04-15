package application

import (
	"mystravastats/domain/business"
	"testing"
)

type activitiesExportAndMapsStub struct {
	csv           string
	gpx           [][][]float64
	receivedYear  *int
	receivedTypes []business.ActivityType
}

func (stub *activitiesExportAndMapsStub) ExportCSVByYearAndTypes(year *int, activityTypes ...business.ActivityType) string {
	stub.receivedYear = year
	stub.receivedTypes = append([]business.ActivityType(nil), activityTypes...)
	return stub.csv
}

func (stub *activitiesExportAndMapsStub) FindGPXByYearAndTypes(year *int, activityTypes ...business.ActivityType) [][][]float64 {
	stub.receivedYear = year
	stub.receivedTypes = append([]business.ActivityType(nil), activityTypes...)
	return stub.gpx
}

func TestExportActivitiesCSVUseCase_Execute_ForwardsInputs(t *testing.T) {
	// GIVEN
	year := 2025
	types := []business.ActivityType{business.Ride}
	expected := "csv-content"
	stub := &activitiesExportAndMapsStub{csv: expected}
	useCase := NewExportActivitiesCSVUseCase(stub)

	// WHEN
	result := useCase.Execute(&year, types)

	// THEN
	if result != expected {
		t.Fatalf("expected %q, got %q", expected, result)
	}
	if stub.receivedYear == nil || *stub.receivedYear != year {
		t.Fatalf("expected year %d, got %+v", year, stub.receivedYear)
	}
}

func TestGetMapsGPXUseCase_Execute_ReturnsEmptySliceOnNilReaderResult(t *testing.T) {
	// GIVEN
	stub := &activitiesExportAndMapsStub{gpx: nil}
	useCase := NewGetMapsGPXUseCase(stub)

	// WHEN
	result := useCase.Execute(nil, []business.ActivityType{business.Ride})

	// THEN
	if result == nil {
		t.Fatal("expected non-nil empty slice")
	}
	if len(result) != 0 {
		t.Fatalf("expected empty slice, got %d", len(result))
	}
}
