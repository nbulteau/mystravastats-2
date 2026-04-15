package application

import (
	"mystravastats/domain/business"
	"mystravastats/domain/strava"
	"testing"
)

type activitiesReaderStub struct {
	activities    []*strava.Activity
	receivedYear  *int
	receivedTypes []business.ActivityType
	calls         int
}

func (stub *activitiesReaderStub) FindActivitiesByYearAndTypes(year *int, activityTypes ...business.ActivityType) []*strava.Activity {
	stub.calls++
	stub.receivedYear = year
	stub.receivedTypes = append([]business.ActivityType(nil), activityTypes...)
	return stub.activities
}

func TestListActivitiesUseCase_Execute_ForwardsInputsAndReturnsActivities(t *testing.T) {
	// GIVEN
	year := 2025
	expectedActivities := []*strava.Activity{
		{Id: 1, Name: "Ride A"},
		{Id: 2, Name: "Ride B"},
	}
	reader := &activitiesReaderStub{
		activities: expectedActivities,
	}
	useCase := NewListActivitiesUseCase(reader)
	inputTypes := []business.ActivityType{business.Ride, business.Commute}

	// WHEN
	result := useCase.Execute(&year, inputTypes)

	// THEN
	if reader.calls != 1 {
		t.Fatalf("expected reader to be called once, got %d", reader.calls)
	}
	if reader.receivedYear == nil || *reader.receivedYear != year {
		t.Fatalf("expected year %d to be forwarded, got %v", year, reader.receivedYear)
	}
	if len(reader.receivedTypes) != len(inputTypes) {
		t.Fatalf("expected %d activity types, got %d", len(inputTypes), len(reader.receivedTypes))
	}
	if len(result) != len(expectedActivities) {
		t.Fatalf("expected %d activities, got %d", len(expectedActivities), len(result))
	}
}

func TestListActivitiesUseCase_Execute_ReturnsEmptySliceOnNilReaderResult(t *testing.T) {
	// GIVEN
	reader := &activitiesReaderStub{
		activities: nil,
	}
	useCase := NewListActivitiesUseCase(reader)

	// WHEN
	result := useCase.Execute(nil, []business.ActivityType{business.Ride})

	// THEN
	if result == nil {
		t.Fatal("expected non-nil empty slice when reader returns nil")
	}
	if len(result) != 0 {
		t.Fatalf("expected empty slice, got %d item(s)", len(result))
	}
}
