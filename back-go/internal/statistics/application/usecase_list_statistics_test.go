package application

import (
	"mystravastats/domain/business"
	domainStatistics "mystravastats/domain/statistics"
	"testing"
)

type statisticStub struct{}

func (s *statisticStub) Label() string {
	return "stub"
}

func (s *statisticStub) Value() string {
	return "stub"
}

func (s *statisticStub) Activity() *business.ActivityShort {
	return nil
}

type statisticsReaderStub struct {
	statistics    []domainStatistics.Statistic
	receivedYear  *int
	receivedTypes []business.ActivityType
	calls         int
}

func (stub *statisticsReaderStub) FindStatisticsByYearAndTypes(year *int, activityTypes ...business.ActivityType) []domainStatistics.Statistic {
	stub.calls++
	stub.receivedYear = year
	stub.receivedTypes = append([]business.ActivityType(nil), activityTypes...)
	return stub.statistics
}

func TestListStatisticsUseCase_Execute_ForwardsInputsAndReturnsStatistics(t *testing.T) {
	// GIVEN
	year := 2025
	expectedStatistics := []domainStatistics.Statistic{
		&statisticStub{},
		&statisticStub{},
	}
	reader := &statisticsReaderStub{
		statistics: expectedStatistics,
	}
	useCase := NewListStatisticsUseCase(reader)
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
	if len(result) != len(expectedStatistics) {
		t.Fatalf("expected %d statistics, got %d", len(expectedStatistics), len(result))
	}
}

func TestListStatisticsUseCase_Execute_ReturnsEmptySliceOnNilReaderResult(t *testing.T) {
	// GIVEN
	reader := &statisticsReaderStub{
		statistics: nil,
	}
	useCase := NewListStatisticsUseCase(reader)

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
