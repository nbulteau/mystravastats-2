package application

import "mystravastats/domain/business"

// ChartsReader is an outbound port used by chart use cases.
// Infrastructure adapters implement this interface.
type ChartsReader interface {
	FindDistanceByPeriod(year *int, period business.Period, activityTypes ...business.ActivityType) []map[string]float64
	FindElevationByPeriod(year *int, period business.Period, activityTypes ...business.ActivityType) []map[string]float64
	FindAverageSpeedByPeriod(year *int, period business.Period, activityTypes ...business.ActivityType) []map[string]float64
	FindAverageCadenceByPeriod(year *int, period business.Period, activityTypes ...business.ActivityType) []map[string]float64
}
