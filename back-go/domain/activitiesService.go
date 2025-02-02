package domain

import (
	"mystravastats/domain/business"
	"mystravastats/domain/strava"
)

func RetrieveActivitiesByActivityTypeAndYear(activityType business.ActivityType, year *int) []*strava.Activity {
	return activityProvider.GetActivitiesByActivityTypeAndYear(activityType, year)
}

func FetchAthlete() strava.Athlete {
	return activityProvider.GetAthlete()
}

func RetrieveGPXByActivityTypeAndYear(activityType business.ActivityType, year *int) [][][]float64 {

	activities := activityProvider.GetActivitiesByActivityTypeAndYear(activityType, year)

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
