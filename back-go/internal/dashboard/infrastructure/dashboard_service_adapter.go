package infrastructure

import (
	"mystravastats/domain/business"
	dashboardDomain "mystravastats/internal/dashboard/domain"
)

// DashboardServiceAdapter computes dashboard read models directly from provider data.
type DashboardServiceAdapter struct{}

func NewDashboardServiceAdapter() *DashboardServiceAdapter {
	return &DashboardServiceAdapter{}
}

func (adapter *DashboardServiceAdapter) FindDashboardData(activityTypes ...business.ActivityType) business.DashboardData {
	return computeDashboardData(activityTypes...)
}

func (adapter *DashboardServiceAdapter) FindCumulativeDistancePerYear(activityTypes ...business.ActivityType) map[string]map[string]float64 {
	return computeCumulativeDistancePerYear(activityTypes...)
}

func (adapter *DashboardServiceAdapter) FindCumulativeElevationPerYear(activityTypes ...business.ActivityType) map[string]map[string]float64 {
	return computeCumulativeElevationPerYear(activityTypes...)
}

func (adapter *DashboardServiceAdapter) FindActivityHeatmap(activityTypes ...business.ActivityType) map[string]map[string]dashboardDomain.ActivityHeatmapDay {
	return computeActivityHeatmap(activityTypes...)
}

func (adapter *DashboardServiceAdapter) FindEddingtonNumber(activityTypes ...business.ActivityType) business.EddingtonNumber {
	return computeEddingtonNumber(activityTypes...)
}
