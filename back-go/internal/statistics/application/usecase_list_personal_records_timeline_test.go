package application

import (
	"mystravastats/internal/shared/domain/business"
	"testing"
)

type personalRecordsTimelineReaderStub struct {
	timeline       []business.PersonalRecordTimelineEntry
	receivedYear   *int
	receivedMetric *string
	receivedTypes  []business.ActivityType
	calls          int
}

func (stub *personalRecordsTimelineReaderStub) FindPersonalRecordsTimelineByYearMetricAndTypes(year *int, metric *string, activityTypes ...business.ActivityType) []business.PersonalRecordTimelineEntry {
	stub.calls++
	stub.receivedYear = year
	stub.receivedMetric = metric
	stub.receivedTypes = append([]business.ActivityType(nil), activityTypes...)
	return stub.timeline
}

func TestListPersonalRecordsTimelineUseCase_Execute_ForwardsInputsAndReturnsTimeline(t *testing.T) {
	// GIVEN
	year := 2025
	metric := "best-distance-1h"
	expectedTimeline := []business.PersonalRecordTimelineEntry{
		{MetricKey: "best-distance-1h"},
		{MetricKey: "max-distance-activity"},
	}
	reader := &personalRecordsTimelineReaderStub{
		timeline: expectedTimeline,
	}
	useCase := NewListPersonalRecordsTimelineUseCase(reader)
	inputTypes := []business.ActivityType{business.Ride, business.Commute}

	// WHEN
	result := useCase.Execute(&year, &metric, inputTypes)

	// THEN
	if reader.calls != 1 {
		t.Fatalf("expected reader to be called once, got %d", reader.calls)
	}
	if reader.receivedYear == nil || *reader.receivedYear != year {
		t.Fatalf("expected year %d to be forwarded, got %v", year, reader.receivedYear)
	}
	if reader.receivedMetric == nil || *reader.receivedMetric != metric {
		t.Fatalf("expected metric %s to be forwarded, got %v", metric, reader.receivedMetric)
	}
	if len(reader.receivedTypes) != len(inputTypes) {
		t.Fatalf("expected %d activity types, got %d", len(inputTypes), len(reader.receivedTypes))
	}
	if len(result) != len(expectedTimeline) {
		t.Fatalf("expected %d timeline entries, got %d", len(expectedTimeline), len(result))
	}
}

func TestListPersonalRecordsTimelineUseCase_Execute_ReturnsEmptySliceOnNilReaderResult(t *testing.T) {
	// GIVEN
	reader := &personalRecordsTimelineReaderStub{
		timeline: nil,
	}
	useCase := NewListPersonalRecordsTimelineUseCase(reader)

	// WHEN
	result := useCase.Execute(nil, nil, []business.ActivityType{business.Ride})

	// THEN
	if result == nil {
		t.Fatal("expected non-nil empty slice when reader returns nil")
	}
	if len(result) != 0 {
		t.Fatalf("expected empty slice, got %d item(s)", len(result))
	}
}
