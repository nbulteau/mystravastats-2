package application

import "mystravastats/internal/shared/domain/business"

type GearAnalysisReader interface {
	FindGearAnalysis(year *int, activityTypes ...business.ActivityType) business.GearAnalysis
}

type GearMaintenanceWriter interface {
	SaveGearMaintenanceRecord(request business.GearMaintenanceRecordRequest) (business.GearMaintenanceRecord, error)
	DeleteGearMaintenanceRecord(recordID string) error
}
