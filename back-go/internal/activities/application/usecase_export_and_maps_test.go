package application

import (
	"mystravastats/internal/shared/domain/business"
	"testing"
)

type activitiesExportAndMapsStub struct {
	csv             string
	gpx             []MapTrack
	passages        MapPassagesResponse
	receivedYear    *int
	receivedTypes   []business.ActivityType
	receivedPassage bool
}

func (stub *activitiesExportAndMapsStub) ExportCSVByYearAndTypes(year *int, activityTypes ...business.ActivityType) string {
	stub.receivedYear = year
	stub.receivedTypes = append([]business.ActivityType(nil), activityTypes...)
	return stub.csv
}

func (stub *activitiesExportAndMapsStub) FindGPXByYearAndTypes(year *int, activityTypes ...business.ActivityType) []MapTrack {
	stub.receivedYear = year
	stub.receivedTypes = append([]business.ActivityType(nil), activityTypes...)
	return stub.gpx
}

func (stub *activitiesExportAndMapsStub) FindPassagesByYearAndTypes(year *int, activityTypes ...business.ActivityType) MapPassagesResponse {
	stub.receivedYear = year
	stub.receivedTypes = append([]business.ActivityType(nil), activityTypes...)
	stub.receivedPassage = true
	return stub.passages
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

func TestGetMapsGPXUseCase_Execute_ReturnsMapTrackMetadata(t *testing.T) {
	// GIVEN
	year := 2026
	expected := []MapTrack{
		{
			ActivityID:     123,
			ActivityName:   "Morning Ride",
			ActivityDate:   "2026-04-16T08:00:00Z",
			ActivityType:   "Ride",
			DistanceKm:     42.3,
			ElevationGainM: 720,
			Coordinates:    [][]float64{{48.1, 2.3}, {48.2, 2.4}},
		},
	}
	stub := &activitiesExportAndMapsStub{gpx: expected}
	useCase := NewGetMapsGPXUseCase(stub)

	// WHEN
	result := useCase.Execute(&year, []business.ActivityType{business.Ride})

	// THEN
	if len(result) != 1 {
		t.Fatalf("expected 1 track, got %d", len(result))
	}
	if result[0].ActivityID != expected[0].ActivityID {
		t.Fatalf("expected activity id %d, got %d", expected[0].ActivityID, result[0].ActivityID)
	}
	if result[0].ActivityName != expected[0].ActivityName {
		t.Fatalf("expected activity name %q, got %q", expected[0].ActivityName, result[0].ActivityName)
	}
	if len(result[0].Coordinates) != 2 {
		t.Fatalf("expected 2 coordinates, got %d", len(result[0].Coordinates))
	}
}

func TestGetMapPassagesUseCase_Execute_ReturnsEmptySegmentsOnNilReaderResult(t *testing.T) {
	// GIVEN
	stub := &activitiesExportAndMapsStub{}
	useCase := NewGetMapPassagesUseCase(stub)

	// WHEN
	result := useCase.Execute(nil, []business.ActivityType{business.Ride})

	// THEN
	if result.Segments == nil {
		t.Fatal("expected non-nil empty segments")
	}
	if len(result.Segments) != 0 {
		t.Fatalf("expected empty segments, got %d", len(result.Segments))
	}
	if !stub.receivedPassage {
		t.Fatal("expected passages reader to be called")
	}
}

func TestGetMapPassagesUseCase_Execute_ForwardsPassageResponse(t *testing.T) {
	// GIVEN
	year := 2026
	expected := MapPassagesResponse{
		Segments: []MapPassageSegment{
			{
				Coordinates:        [][]float64{{48.1, 2.3}, {48.2, 2.4}},
				PassageCount:       3,
				ActivityCount:      3,
				DistanceKm:         1.2,
				ActivityTypeCounts: map[string]int{"Ride": 3},
			},
		},
		IncludedActivities: 3,
		ResolutionMeters:   120,
	}
	stub := &activitiesExportAndMapsStub{passages: expected}
	useCase := NewGetMapPassagesUseCase(stub)

	// WHEN
	result := useCase.Execute(&year, []business.ActivityType{business.Ride})

	// THEN
	if len(result.Segments) != 1 {
		t.Fatalf("expected 1 segment, got %d", len(result.Segments))
	}
	if result.Segments[0].PassageCount != 3 {
		t.Fatalf("expected passage count 3, got %d", result.Segments[0].PassageCount)
	}
	if result.IncludedActivities != 3 {
		t.Fatalf("expected 3 included activities, got %d", result.IncludedActivities)
	}
	if stub.receivedYear == nil || *stub.receivedYear != year {
		t.Fatalf("expected year %d, got %+v", year, stub.receivedYear)
	}
}
