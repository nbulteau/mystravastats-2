package application

import (
	"mystravastats/domain/business"
	segmentsDomain "mystravastats/internal/segments/domain"
	"testing"
)

type segmentsReaderStub struct {
	progression       business.SegmentClimbProgression
	summaries         []business.SegmentClimbTargetSummary
	attempts          []business.SegmentClimbAttempt
	summary           *segmentsDomain.SegmentSummary
	receivedYear      *int
	receivedMetric    *string
	receivedQuery     *string
	receivedFrom      *string
	receivedTo        *string
	receivedTargetID  *int64
	receivedSegmentID int64
	receivedTypes     []business.ActivityType
}

func (stub *segmentsReaderStub) FindSegmentClimbProgressionByYearMetricTargetAndTypes(
	year *int,
	metric *string,
	targetType *string,
	targetID *int64,
	activityTypes ...business.ActivityType,
) business.SegmentClimbProgression {
	stub.receivedYear = year
	stub.receivedMetric = metric
	stub.receivedQuery = targetType
	stub.receivedTargetID = targetID
	stub.receivedTypes = append([]business.ActivityType(nil), activityTypes...)
	return stub.progression
}

func (stub *segmentsReaderStub) FindSegmentsByYearMetricQueryRangeAndTypes(
	year *int,
	metric *string,
	query *string,
	from *string,
	to *string,
	activityTypes ...business.ActivityType,
) []business.SegmentClimbTargetSummary {
	stub.receivedYear = year
	stub.receivedMetric = metric
	stub.receivedQuery = query
	stub.receivedFrom = from
	stub.receivedTo = to
	stub.receivedTypes = append([]business.ActivityType(nil), activityTypes...)
	return stub.summaries
}

func (stub *segmentsReaderStub) FindSegmentEffortsByYearMetricRangeAndTypes(
	year *int,
	metric *string,
	segmentID int64,
	from *string,
	to *string,
	activityTypes ...business.ActivityType,
) []business.SegmentClimbAttempt {
	stub.receivedYear = year
	stub.receivedMetric = metric
	stub.receivedSegmentID = segmentID
	stub.receivedFrom = from
	stub.receivedTo = to
	stub.receivedTypes = append([]business.ActivityType(nil), activityTypes...)
	return stub.attempts
}

func (stub *segmentsReaderStub) FindSegmentSummaryByYearMetricRangeAndTypes(
	year *int,
	metric *string,
	segmentID int64,
	from *string,
	to *string,
	activityTypes ...business.ActivityType,
) *segmentsDomain.SegmentSummary {
	stub.receivedYear = year
	stub.receivedMetric = metric
	stub.receivedSegmentID = segmentID
	stub.receivedFrom = from
	stub.receivedTo = to
	stub.receivedTypes = append([]business.ActivityType(nil), activityTypes...)
	return stub.summary
}

func TestGetSegmentClimbProgressionUseCase_Execute_ForwardsInputs(t *testing.T) {
	// GIVEN
	year := 2026
	metric := "TIME"
	targetType := "ALL"
	targetID := int64(123)
	reader := &segmentsReaderStub{
		progression: business.SegmentClimbProgression{
			Metric: "TIME",
		},
	}
	useCase := NewGetSegmentClimbProgressionUseCase(reader)
	inputTypes := []business.ActivityType{business.Ride}

	// WHEN
	result := useCase.Execute(&year, &metric, &targetType, &targetID, inputTypes)

	// THEN
	if result.Metric != "TIME" {
		t.Fatalf("expected metric TIME, got %s", result.Metric)
	}
	if reader.receivedTargetID == nil || *reader.receivedTargetID != targetID {
		t.Fatalf("expected targetID %d to be forwarded, got %v", targetID, reader.receivedTargetID)
	}
}

func TestListSegmentsUseCase_Execute_ReturnsEmptySliceOnNilReaderResult(t *testing.T) {
	// GIVEN
	reader := &segmentsReaderStub{
		summaries: nil,
	}
	useCase := NewListSegmentsUseCase(reader)

	// WHEN
	result := useCase.Execute(nil, nil, nil, nil, nil, []business.ActivityType{business.Ride})

	// THEN
	if result == nil {
		t.Fatal("expected non-nil empty slice when reader returns nil")
	}
	if len(result) != 0 {
		t.Fatalf("expected empty slice, got %d item(s)", len(result))
	}
}

func TestListSegmentEffortsUseCase_Execute_ReturnsEmptySliceOnNilReaderResult(t *testing.T) {
	// GIVEN
	reader := &segmentsReaderStub{
		attempts: nil,
	}
	useCase := NewListSegmentEffortsUseCase(reader)

	// WHEN
	result := useCase.Execute(nil, nil, 42, nil, nil, []business.ActivityType{business.Ride})

	// THEN
	if result == nil {
		t.Fatal("expected non-nil empty slice when reader returns nil")
	}
	if len(result) != 0 {
		t.Fatalf("expected empty slice, got %d item(s)", len(result))
	}
}

func TestGetSegmentSummaryUseCase_Execute_ReturnsSummary(t *testing.T) {
	// GIVEN
	expected := &segmentsDomain.SegmentSummary{
		Metric: "SPEED",
	}
	reader := &segmentsReaderStub{
		summary: expected,
	}
	useCase := NewGetSegmentSummaryUseCase(reader)

	// WHEN
	result := useCase.Execute(nil, nil, 42, nil, nil, []business.ActivityType{business.Ride})

	// THEN
	if result == nil {
		t.Fatal("expected summary, got nil")
	}
	if result.Metric != expected.Metric {
		t.Fatalf("expected metric %s, got %s", expected.Metric, result.Metric)
	}
}
