package infrastructure

import (
	"mystravastats/internal/platform/activityprovider"
	"mystravastats/internal/shared/domain/strava"
)

// AthleteServiceAdapter bridges the current internal/services layer
// to the hexagonal outbound ports used by athlete use cases.
type AthleteServiceAdapter struct{}

func NewAthleteServiceAdapter() *AthleteServiceAdapter {
	return &AthleteServiceAdapter{}
}

func (adapter *AthleteServiceAdapter) FindAthlete() strava.Athlete {
	return activityprovider.Get().GetAthlete()
}
