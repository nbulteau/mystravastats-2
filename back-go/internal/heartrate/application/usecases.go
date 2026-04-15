package application

import "mystravastats/domain/business"

type GetHeartRateZoneSettingsUseCase struct {
	reader HeartRateReader
}

func NewGetHeartRateZoneSettingsUseCase(reader HeartRateReader) *GetHeartRateZoneSettingsUseCase {
	return &GetHeartRateZoneSettingsUseCase{reader: reader}
}

func (uc *GetHeartRateZoneSettingsUseCase) Execute() business.HeartRateZoneSettings {
	return uc.reader.FindHeartRateZoneSettings()
}

type UpdateHeartRateZoneSettingsUseCase struct {
	reader HeartRateReader
}

func NewUpdateHeartRateZoneSettingsUseCase(reader HeartRateReader) *UpdateHeartRateZoneSettingsUseCase {
	return &UpdateHeartRateZoneSettingsUseCase{reader: reader}
}

func (uc *UpdateHeartRateZoneSettingsUseCase) Execute(settings business.HeartRateZoneSettings) business.HeartRateZoneSettings {
	return uc.reader.SaveHeartRateZoneSettings(settings)
}

type GetHeartRateZoneAnalysisUseCase struct {
	reader HeartRateReader
}

func NewGetHeartRateZoneAnalysisUseCase(reader HeartRateReader) *GetHeartRateZoneAnalysisUseCase {
	return &GetHeartRateZoneAnalysisUseCase{reader: reader}
}

func (uc *GetHeartRateZoneAnalysisUseCase) Execute(year *int, activityTypes []business.ActivityType) business.HeartRateZoneAnalysis {
	return uc.reader.FindHeartRateZoneAnalysisByYearAndTypes(year, activityTypes...)
}
