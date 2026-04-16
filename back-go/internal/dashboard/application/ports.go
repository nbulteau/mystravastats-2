package application

import (
	dashboardDomain "mystravastats/internal/dashboard/domain"
	"mystravastats/internal/shared/domain/business"
)

// DashboardReader is an outbound port used by dashboard use cases.
// Infrastructure adapters implement this interface.
type DashboardReader interface {
	FindDashboardData(activityTypes ...business.ActivityType) business.DashboardData
	FindCumulativeDistancePerYear(activityTypes ...business.ActivityType) map[string]map[string]float64
	FindCumulativeElevationPerYear(activityTypes ...business.ActivityType) map[string]map[string]float64
	FindActivityHeatmap(activityTypes ...business.ActivityType) map[string]map[string]dashboardDomain.ActivityHeatmapDay
	FindEddingtonNumber(activityTypes ...business.ActivityType) business.EddingtonNumber
}
