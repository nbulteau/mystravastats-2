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
	maintenanceRecords := loadCurrentProviderGearMaintenanceRecords()
	return buildGearAnalysis(activities, athlete, maintenanceRecords)
}

func (adapter *GearAnalysisServiceAdapter) SaveGearMaintenanceRecord(request business.GearMaintenanceRecordRequest) (business.GearMaintenanceRecord, error) {
	return saveCurrentProviderGearMaintenanceRecord(request)
}

func (adapter *GearAnalysisServiceAdapter) DeleteGearMaintenanceRecord(recordID string) error {
	return deleteCurrentProviderGearMaintenanceRecord(recordID)
}
