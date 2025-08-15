package services

import (
	"fmt"
	"log"
	"mystravastats/domain/business"
	"mystravastats/domain/strava"
)

func RetrieveActivitiesByActivityTypeAndYear(year *int, activityTypes ...business.ActivityType) []*strava.Activity {
	return activityProvider.GetActivitiesByActivityTypeAndYear(year, activityTypes...)
}

func FetchAthlete() strava.Athlete {
	return activityProvider.GetAthlete()
}

// RetrieveGPXByActivityTypeAndYear retrieves GPX coordinates for activities of a specific type and year.
func RetrieveGPXByActivityTypeAndYear(year *int, activityTypes ...business.ActivityType) [][][]float64 {

	activities := activityProvider.GetActivitiesByActivityTypeAndYear(year, activityTypes...)

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
