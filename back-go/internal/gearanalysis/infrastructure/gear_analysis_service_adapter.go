package infrastructure

import (
	dataqualityInfra "mystravastats/internal/dataquality/infrastructure"
	"mystravastats/internal/platform/activityprovider"
	"mystravastats/internal/shared/domain/business"
)

type GearAnalysisServiceAdapter struct{}

func NewGearAnalysisServiceAdapter() *GearAnalysisServiceAdapter {
	return &GearAnalysisServiceAdapter{}
}

func (adapter *GearAnalysisServiceAdapter) FindGearAnalysis(year *int, activityTypes ...business.ActivityType) business.GearAnalysis {
	provider := activityprovider.Get()
	activities := dataqualityInfra.FilterExcludedFromStats(provider.GetActivitiesByYearAndActivityTypes(year, activityTypes...))
	athlete := provider.GetAthlete()
	return buildGearAnalysis(activities, athlete)
}
