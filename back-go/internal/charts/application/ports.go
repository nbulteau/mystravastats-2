package application

import "mystravastats/internal/shared/domain/business"

type ChartPeriodPoint struct {
	PeriodKey     string  `json:"periodKey"`
	Value         float64 `json:"value"`
	ActivityCount int     `json:"activityCount"`
}

// ChartsReader is an outbound port used by chart use cases.
// Infrastructure adapters implement this interface.
type ChartsReader interface {
	FindDistanceByPeriod(year *int, period business.Period, activityTypes ...business.ActivityType) []ChartPeriodPoint
	FindElevationByPeriod(year *int, period business.Period, activityTypes ...business.ActivityType) []ChartPeriodPoint
	FindAverageSpeedByPeriod(year *int, period business.Period, activityTypes ...business.ActivityType) []ChartPeriodPoint
	FindAverageCadenceByPeriod(year *int, period business.Period, activityTypes ...business.ActivityType) []ChartPeriodPoint
}
