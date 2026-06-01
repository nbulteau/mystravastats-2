package application

import (
	"mystravastats/internal/shared/domain/business"
	"mystravastats/internal/shared/domain/strava"
)

// AthleteReader is an outbound port used by athlete use cases.
// Infrastructure adapters implement this interface.
type AthleteReader interface {
	FindAthlete() strava.Athlete
	FindPerformanceSettings() business.AthletePerformanceSettings
	SavePerformanceSettings(settings business.AthletePerformanceSettings) business.AthletePerformanceSettings
}
