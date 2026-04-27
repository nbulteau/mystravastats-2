package application

import "mystravastats/internal/shared/domain/business"

type GetMapPassagesUseCase struct {
	reader ActivitiesPassagesReader
}

func NewGetMapPassagesUseCase(reader ActivitiesPassagesReader) *GetMapPassagesUseCase {
	return &GetMapPassagesUseCase{
		reader: reader,
	}
}

func (uc *GetMapPassagesUseCase) Execute(year *int, activityTypes []business.ActivityType) MapPassagesResponse {
	if uc.reader == nil {
		return emptyMapPassagesResponse()
	}

	response := uc.reader.FindPassagesByYearAndTypes(year, activityTypes...)
	if response.Segments == nil {
		response.Segments = []MapPassageSegment{}
	}
	return response
}

func emptyMapPassagesResponse() MapPassagesResponse {
	return MapPassagesResponse{
		Segments:         []MapPassageSegment{},
		ResolutionMeters: 120,
		MinPassageCount:  1,
	}
}
