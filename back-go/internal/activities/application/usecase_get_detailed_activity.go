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
	return uc.execute(activityID, false)
}

func (uc *GetDetailedActivityUseCase) ExecuteRaw(activityID int64) (*strava.DetailedActivity, error) {
	return uc.execute(activityID, true)
}

func (uc *GetDetailedActivityUseCase) execute(activityID int64, raw bool) (*strava.DetailedActivity, error) {
	if activityID <= 0 {
		return nil, activitiesDomain.ErrInvalidActivityID
	}

	var detailedActivity *strava.DetailedActivity
	var err error
	if raw {
		if rawReader, ok := uc.reader.(RawDetailedActivityReader); ok {
			detailedActivity, err = rawReader.FindRawDetailedActivityByID(activityID)
		} else {
			detailedActivity, err = uc.reader.FindDetailedActivityByID(activityID)
		}
	} else {
		detailedActivity, err = uc.reader.FindDetailedActivityByID(activityID)
	}
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
