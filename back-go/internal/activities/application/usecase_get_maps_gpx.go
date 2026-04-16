package application

import "mystravastats/internal/shared/domain/business"

type GetMapsGPXUseCase struct {
	reader ActivitiesGPXReader
}

func NewGetMapsGPXUseCase(reader ActivitiesGPXReader) *GetMapsGPXUseCase {
	return &GetMapsGPXUseCase{
		reader: reader,
	}
}

func (uc *GetMapsGPXUseCase) Execute(year *int, activityTypes []business.ActivityType) []MapTrack {
	gpx := uc.reader.FindGPXByYearAndTypes(year, activityTypes...)
	if gpx == nil {
		return []MapTrack{}
	}

	return gpx
}
