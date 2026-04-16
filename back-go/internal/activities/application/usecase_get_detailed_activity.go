package application

import (
	"errors"
	activitiesDomain "mystravastats/internal/activities/domain"
	"mystravastats/internal/shared/domain/strava"
)

type GetDetailedActivityUseCase struct {
	reader DetailedActivityReader
}

func NewGetDetailedActivityUseCase(reader DetailedActivityReader) *GetDetailedActivityUseCase {
	return &GetDetailedActivityUseCase{
		reader: reader,
	}
}

func (uc *GetDetailedActivityUseCase) Execute(activityID int64) (*strava.DetailedActivity, error) {
	if activityID <= 0 {
		return nil, activitiesDomain.ErrInvalidActivityID
	}

	detailedActivity, err := uc.reader.FindDetailedActivityByID(activityID)
	if err != nil {
		return nil, err
	}
	if detailedActivity == nil {
		return nil, activitiesDomain.ErrDetailedActivityNotFound
	}

	return detailedActivity, nil
}

func IsNotFoundError(err error) bool {
	return errors.Is(err, activitiesDomain.ErrDetailedActivityNotFound)
}
