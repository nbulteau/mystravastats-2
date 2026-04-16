package application

import (
	"mystravastats/internal/shared/domain/business"
	"mystravastats/internal/shared/domain/strava"
)

type ListActivitiesUseCase struct {
	reader ActivitiesReader
}

func NewListActivitiesUseCase(reader ActivitiesReader) *ListActivitiesUseCase {
	return &ListActivitiesUseCase{
		reader: reader,
	}
}

func (uc *ListActivitiesUseCase) Execute(year *int, activityTypes []business.ActivityType) []*strava.Activity {
	activities := uc.reader.FindActivitiesByYearAndTypes(year, activityTypes...)
	if activities == nil {
		return []*strava.Activity{}
	}

	return activities
}
