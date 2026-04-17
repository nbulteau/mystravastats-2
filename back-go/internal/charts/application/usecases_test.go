package application

import (
	"mystravastats/internal/shared/domain/business"
	"testing"
)

type chartsReaderStub struct {
	result []ChartPeriodPoint
}

func (stub *chartsReaderStub) FindDistanceByPeriod(_ *int, _ business.Period, _ ...business.ActivityType) []ChartPeriodPoint {
	return stub.result
}

func (stub *chartsReaderStub) FindElevationByPeriod(_ *int, _ business.Period, _ ...business.ActivityType) []ChartPeriodPoint {
	return stub.result
}

func (stub *chartsReaderStub) FindAverageSpeedByPeriod(_ *int, _ business.Period, _ ...business.ActivityType) []ChartPeriodPoint {
	return stub.result
}

func (stub *chartsReaderStub) FindAverageCadenceByPeriod(_ *int, _ business.Period, _ ...business.ActivityType) []ChartPeriodPoint {
	return stub.result
}

func TestChartsUseCases_ReturnEmptySliceOnNilReaderResult(t *testing.T) {
	// GIVEN
	reader := &chartsReaderStub{result: nil}
	year := 2025
	period := business.PeriodMonths
	activityTypes := []business.ActivityType{business.Ride}

	// WHEN
	distance := NewGetDistanceByPeriodUseCase(reader).Execute(&year, period, activityTypes)
	elevation := NewGetElevationByPeriodUseCase(reader).Execute(&year, period, activityTypes)
	speed := NewGetAverageSpeedByPeriodUseCase(reader).Execute(&year, period, activityTypes)
	cadence := NewGetAverageCadenceByPeriodUseCase(reader).Execute(&year, period, activityTypes)

	// THEN
	for _, result := range [][]ChartPeriodPoint{distance, elevation, speed, cadence} {
		if result == nil {
			t.Fatal("expected non-nil empty slice")
		}
		if len(result) != 0 {
			t.Fatalf("expected empty slice, got %d item(s)", len(result))
		}
	}
}
