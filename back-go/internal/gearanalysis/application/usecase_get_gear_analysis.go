package application

import "mystravastats/internal/shared/domain/business"

type GetGearAnalysisUseCase struct {
	reader GearAnalysisReader
}

func NewGetGearAnalysisUseCase(reader GearAnalysisReader) *GetGearAnalysisUseCase {
	return &GetGearAnalysisUseCase{reader: reader}
}

func (uc *GetGearAnalysisUseCase) Execute(year *int, activityTypes []business.ActivityType) business.GearAnalysis {
	if uc == nil || uc.reader == nil {
		return business.GearAnalysis{}
	}
	return uc.reader.FindGearAnalysis(year, activityTypes...)
}
