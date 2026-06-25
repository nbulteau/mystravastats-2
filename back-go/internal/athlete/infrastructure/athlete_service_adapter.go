package infrastructure

import (
	dataqualityInfra "mystravastats/internal/dataquality/infrastructure"
	"mystravastats/internal/platform/activityprovider"
	"mystravastats/internal/shared/domain/business"
	"mystravastats/internal/shared/domain/strava"
)

// AthleteServiceAdapter bridges the current internal/services layer
// to the hexagonal outbound ports used by athlete use cases.
type AthleteServiceAdapter struct{}

func NewAthleteServiceAdapter() *AthleteServiceAdapter {
	return &AthleteServiceAdapter{}
}

func (adapter *AthleteServiceAdapter) FindAthlete() strava.Athlete {
	return activityprovider.Get().GetAthlete()
}

func (adapter *AthleteServiceAdapter) FindActivitiesByYearAndTypes(year *int, activityTypes ...business.ActivityType) []*strava.Activity {
	return dataqualityInfra.FilterExcludedFromStats(activityprovider.Get().GetActivitiesByYearAndActivityTypes(year, activityTypes...))
}

func (adapter *AthleteServiceAdapter) FindPerformanceSettings() business.AthletePerformanceSettings {
	return activityprovider.Get().GetPerformanceSettings()
}

func (adapter *AthleteServiceAdapter) SavePerformanceSettings(settings business.AthletePerformanceSettings) business.AthletePerformanceSettings {
	return activityprovider.Get().SavePerformanceSettings(settings)
}
