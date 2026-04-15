package infrastructure

import (
	"mystravastats/domain/business"
	dashboardDomain "mystravastats/internal/dashboard/domain"
	"mystravastats/internal/services"
)

// DashboardServiceAdapter bridges the current internal/services layer
// to the hexagonal outbound ports used by dashboard use cases.
type DashboardServiceAdapter struct{}

func NewDashboardServiceAdapter() *DashboardServiceAdapter {
	return &DashboardServiceAdapter{}
}

func (adapter *DashboardServiceAdapter) FindDashboardData(activityTypes ...business.ActivityType) business.DashboardData {
	return services.FetchDashboardData(activityTypes...)
}

func (adapter *DashboardServiceAdapter) FindCumulativeDistancePerYear(activityTypes ...business.ActivityType) map[string]map[string]float64 {
	return services.GetCumulativeDistancePerYear(activityTypes...)
}

func (adapter *DashboardServiceAdapter) FindCumulativeElevationPerYear(activityTypes ...business.ActivityType) map[string]map[string]float64 {
	return services.GetCumulativeElevationPerYear(activityTypes...)
}

func (adapter *DashboardServiceAdapter) FindActivityHeatmap(activityTypes ...business.ActivityType) map[string]map[string]dashboardDomain.ActivityHeatmapDay {
	raw := services.FetchActivityHeatmap(activityTypes...)
	if raw == nil {
		return nil
	}

	result := make(map[string]map[string]dashboardDomain.ActivityHeatmapDay, len(raw))
	for year, days := range raw {
		dayMap := make(map[string]dashboardDomain.ActivityHeatmapDay, len(days))
		for day, value := range days {
			activities := make([]dashboardDomain.ActivityHeatmapActivity, 0, len(value.Activities))
			for _, activity := range value.Activities {
				activities = append(activities, dashboardDomain.ActivityHeatmapActivity{
					ID:             activity.ID,
					Name:           activity.Name,
					Type:           activity.Type,
					DistanceKm:     activity.DistanceKm,
					ElevationGainM: activity.ElevationGainM,
					DurationSec:    activity.DurationSec,
				})
			}
			dayMap[day] = dashboardDomain.ActivityHeatmapDay{
				DistanceKm:     value.DistanceKm,
				ElevationGainM: value.ElevationGainM,
				DurationSec:    value.DurationSec,
				ActivityCount:  value.ActivityCount,
				Activities:     activities,
			}
		}
		result[year] = dayMap
	}

	return result
}

func (adapter *DashboardServiceAdapter) FindEddingtonNumber(activityTypes ...business.ActivityType) business.EddingtonNumber {
	return services.FetchEddingtonNumber(activityTypes...)
}
