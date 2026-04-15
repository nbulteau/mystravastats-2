package infrastructure

import (
	"mystravastats/domain/business"
	"mystravastats/domain/strava"
	"mystravastats/internal/platform/activityprovider"
	"mystravastats/internal/services"
)

// DetailedActivityServiceAdapter bridges the current internal/services layer
// to the hexagonal outbound port used by activities use cases.
type DetailedActivityServiceAdapter struct{}

func NewDetailedActivityServiceAdapter() *DetailedActivityServiceAdapter {
	return &DetailedActivityServiceAdapter{}
}

func (adapter *DetailedActivityServiceAdapter) FindDetailedActivityByID(activityID int64) (*strava.DetailedActivity, error) {
	return services.RetrieveDetailedActivity(activityID)
}

func (adapter *DetailedActivityServiceAdapter) FindActivitiesByYearAndTypes(year *int, activityTypes ...business.ActivityType) []*strava.Activity {
	return activityprovider.Get().GetActivitiesByYearAndActivityTypes(year, activityTypes...)
}

func (adapter *DetailedActivityServiceAdapter) ExportCSVByYearAndTypes(year *int, activityTypes ...business.ActivityType) string {
	return services.ExportCSV(year, activityTypes...)
}

func (adapter *DetailedActivityServiceAdapter) FindGPXByYearAndTypes(year *int, activityTypes ...business.ActivityType) [][][]float64 {
	return services.RetrieveGPXByYearAndActivityTypes(year, activityTypes...)
}
