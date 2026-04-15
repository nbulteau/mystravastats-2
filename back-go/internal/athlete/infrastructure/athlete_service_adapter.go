package infrastructure

import (
	"mystravastats/domain/strava"
	"mystravastats/internal/platform/activityprovider"
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
