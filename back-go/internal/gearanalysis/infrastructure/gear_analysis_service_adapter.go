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
	lifetimeActivities := dataqualityInfra.FilterExcludedFromStats(provider.GetActivitiesByYearAndActivityTypes(nil, allGearActivityTypes()...))
	athlete := provider.GetAthlete()
	maintenanceRecords := loadCurrentProviderGearMaintenanceRecords()
	return buildGearAnalysis(activities, lifetimeActivities, athlete, maintenanceRecords)
}

func (adapter *GearAnalysisServiceAdapter) SaveGearMaintenanceRecord(request business.GearMaintenanceRecordRequest) (business.GearMaintenanceRecord, error) {
	return saveCurrentProviderGearMaintenanceRecord(request)
}

func (adapter *GearAnalysisServiceAdapter) DeleteGearMaintenanceRecord(recordID string) error {
	return deleteCurrentProviderGearMaintenanceRecord(recordID)
}

func allGearActivityTypes() []business.ActivityType {
	return []business.ActivityType{
		business.Run,
		business.TrailRun,
		business.Ride,
		business.GravelRide,
		business.MountainBikeRide,
		business.InlineSkate,
		business.Hike,
		business.Walk,
		business.Commute,
		business.AlpineSki,
		business.VirtualRide,
	}
}
