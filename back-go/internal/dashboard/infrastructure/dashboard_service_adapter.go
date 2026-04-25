package infrastructure

import (
	dashboardDomain "mystravastats/internal/dashboard/domain"
	"mystravastats/internal/shared/domain/business"
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

func (adapter *DashboardServiceAdapter) FindAnnualGoals(year int, activityTypes ...business.ActivityType) business.AnnualGoals {
	return loadAnnualGoals(year, activityTypes...)
}

func (adapter *DashboardServiceAdapter) SaveAnnualGoals(year int, targets business.AnnualGoalTargets, activityTypes ...business.ActivityType) business.AnnualGoals {
	return saveAnnualGoals(year, targets, activityTypes...)
}
