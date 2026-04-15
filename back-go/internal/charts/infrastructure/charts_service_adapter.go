package infrastructure

import (
	"mystravastats/domain/business"
	"mystravastats/internal/services"
)

// ChartsServiceAdapter bridges the current internal/services layer
// to the hexagonal outbound ports used by chart use cases.
type ChartsServiceAdapter struct{}

func NewChartsServiceAdapter() *ChartsServiceAdapter {
	return &ChartsServiceAdapter{}
}

func (adapter *ChartsServiceAdapter) FindDistanceByPeriod(year *int, period business.Period, activityTypes ...business.ActivityType) []map[string]float64 {
	return services.FetchChartsDistanceByPeriod(year, period, activityTypes...)
}

func (adapter *ChartsServiceAdapter) FindElevationByPeriod(year *int, period business.Period, activityTypes ...business.ActivityType) []map[string]float64 {
	return services.FetchChartsElevationByPeriod(year, period, activityTypes...)
}

func (adapter *ChartsServiceAdapter) FindAverageSpeedByPeriod(year *int, period business.Period, activityTypes ...business.ActivityType) []map[string]float64 {
	return services.FetchChartsAverageSpeedByPeriod(year, period, activityTypes...)
}

func (adapter *ChartsServiceAdapter) FindAverageCadenceByPeriod(year *int, period business.Period, activityTypes ...business.ActivityType) []map[string]float64 {
	return services.FetchChartsAverageCadenceByPeriod(year, period, activityTypes...)
}
