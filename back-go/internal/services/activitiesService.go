package services

import (
	"fmt"
	"log"
	"mystravastats/domain/business"
	"mystravastats/domain/strava"
)

func RetrieveActivitiesByYearAndActivityTypes(year *int, activityTypes ...business.ActivityType) []*strava.Activity {
	return activityProvider.GetActivitiesByYearAndActivityTypes(year, activityTypes...)
}

func FetchAthlete() strava.Athlete {
	return activityProvider.GetAthlete()
}

func RetrieveGPXByYearAndActivityTypes(year *int, activityTypes ...business.ActivityType) [][][]float64 {

	activities := activityProvider.GetActivitiesByYearAndActivityTypes(year, activityTypes...)

	step := 100
	if year != nil {
		step = 10
	}

	var result [][][]float64
	for _, activity := range activities {
		if activity.Stream != nil && activity.Stream.LatLng != nil {
			var coordinates [][]float64
			for i, pair := range activity.Stream.LatLng.Data {
				if i%step == 0 {
					coordinates = append(coordinates, []float64{pair[0], pair[1]})
				}
			}
			result = append(result, coordinates)

		}
	}

	return result
}

// RetrieveDetailedActivity fetches detailed information about a specific activity by its ID.
func RetrieveDetailedActivity(activityId int64) (*strava.DetailedActivity, error) {
	log.Printf("Get detailed activity %d", activityId)

	detailedActivity := activityProvider.GetDetailedActivity(activityId)
	if detailedActivity == nil {
		log.Printf("Activity %d not found", activityId)
		return nil, fmt.Errorf("activity %d not found", activityId)
	}

	return detailedActivity, nil
}
