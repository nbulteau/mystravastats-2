package application

import "mystravastats/internal/shared/domain/business"

type SaveGearMaintenanceRecordUseCase struct {
	writer GearMaintenanceWriter
}

func NewSaveGearMaintenanceRecordUseCase(writer GearMaintenanceWriter) *SaveGearMaintenanceRecordUseCase {
	return &SaveGearMaintenanceRecordUseCase{writer: writer}
}

func (uc *SaveGearMaintenanceRecordUseCase) Execute(request business.GearMaintenanceRecordRequest) (business.GearMaintenanceRecord, error) {
	return uc.writer.SaveGearMaintenanceRecord(request)
}

type DeleteGearMaintenanceRecordUseCase struct {
	writer GearMaintenanceWriter
}

func NewDeleteGearMaintenanceRecordUseCase(writer GearMaintenanceWriter) *DeleteGearMaintenanceRecordUseCase {
	return &DeleteGearMaintenanceRecordUseCase{writer: writer}
}

func (uc *DeleteGearMaintenanceRecordUseCase) Execute(recordID string) error {
	return uc.writer.DeleteGearMaintenanceRecord(recordID)
}
