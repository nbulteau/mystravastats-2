package infrastructure

import (
	"fmt"
	"mystravastats/domain/business"
	"mystravastats/domain/strava"
	"mystravastats/internal/platform/activityprovider"
	"mystravastats/internal/services"
)

// DetailedActivityServiceAdapter bridges the current internal/services layer
// to the hexagonal outbound port used by activities use cases.
type DetailedActivityServiceAdapter struct{}

func NewDetailedActivityServiceAdapter() *DetailedActivityServiceAdapter {
	return &DetailedActivityServiceAdapter{}
}

func (adapter *DetailedActivityServiceAdapter) FindDetailedActivityByID(activityID int64) (*strava.DetailedActivity, error) {
	detailedActivity := activityprovider.Get().GetDetailedActivity(activityID)
	if detailedActivity == nil {
		return nil, fmt.Errorf("activity %d not found", activityID)
	}
	return detailedActivity, nil
}

func (adapter *DetailedActivityServiceAdapter) FindActivitiesByYearAndTypes(year *int, activityTypes ...business.ActivityType) []*strava.Activity {
	return activityprovider.Get().GetActivitiesByYearAndActivityTypes(year, activityTypes...)
}

func (adapter *DetailedActivityServiceAdapter) ExportCSVByYearAndTypes(year *int, activityTypes ...business.ActivityType) string {
	return services.ExportCSV(year, activityTypes...)
}

func (adapter *DetailedActivityServiceAdapter) FindGPXByYearAndTypes(year *int, activityTypes ...business.ActivityType) [][][]float64 {
	activities := activityprovider.Get().GetActivitiesByYearAndActivityTypes(year, activityTypes...)

	step := 100
	if year != nil {
		step = 10
	}

	var result [][][]float64
	for _, activity := range activities {
		if activity.Stream == nil || activity.Stream.LatLng == nil {
			continue
		}
		var coordinates [][]float64
		for i, pair := range activity.Stream.LatLng.Data {
			if i%step == 0 {
				coordinates = append(coordinates, []float64{pair[0], pair[1]})
			}
		}
		result = append(result, coordinates)
	}

	return result
}
