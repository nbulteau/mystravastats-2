package application

import "mystravastats/domain/strava"

// AthleteReader is an outbound port used by athlete use cases.
// Infrastructure adapters implement this interface.
type AthleteReader interface {
	FindAthlete() strava.Athlete
}
