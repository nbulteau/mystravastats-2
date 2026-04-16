package application

import "mystravastats/internal/shared/domain/strava"

type GetAthleteUseCase struct {
	reader AthleteReader
}

func NewGetAthleteUseCase(reader AthleteReader) *GetAthleteUseCase {
	return &GetAthleteUseCase{
		reader: reader,
	}
}

func (uc *GetAthleteUseCase) Execute() strava.Athlete {
	return uc.reader.FindAthlete()
}
