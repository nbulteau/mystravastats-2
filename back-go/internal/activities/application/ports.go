package application

import (
	"mystravastats/internal/shared/domain/business"
	"mystravastats/internal/shared/domain/strava"
)

type MapTrack struct {
	ActivityID     int64       `json:"activityId"`
	ActivityName   string      `json:"activityName"`
	ActivityDate   string      `json:"activityDate"`
	ActivityType   string      `json:"activityType"`
	DistanceKm     float64     `json:"distanceKm"`
	ElevationGainM float64     `json:"elevationGainM"`
	Coordinates    [][]float64 `json:"coordinates"`
}

// DetailedActivityReader is an outbound port used by the use case.
// Infrastructure adapters implement this interface.
type DetailedActivityReader interface {
	FindDetailedActivityByID(activityID int64) (*strava.DetailedActivity, error)
}

// ActivitiesReader is an outbound port used by list activities use cases.
// Infrastructure adapters implement this interface.
type ActivitiesReader interface {
	FindActivitiesByYearAndTypes(year *int, activityTypes ...business.ActivityType) []*strava.Activity
}

// ActivitiesCSVExporter is an outbound port used by CSV export use cases.
// Infrastructure adapters implement this interface.
type ActivitiesCSVExporter interface {
	ExportCSVByYearAndTypes(year *int, activityTypes ...business.ActivityType) string
}

// ActivitiesGPXReader is an outbound port used by map/GPX use cases.
// Infrastructure adapters implement this interface.
type ActivitiesGPXReader interface {
	FindGPXByYearAndTypes(year *int, activityTypes ...business.ActivityType) []MapTrack
}
