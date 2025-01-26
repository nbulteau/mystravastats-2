package services

import "mystravastats/domain/services/strava"

var ActivityProvider = NewStravaActivityProvider("strava-cache")

func FetchActivitiesByActivityTypeAndYear(activityType ActivityType, year *int) []strava.Activity {
	// Implement the logic to filter activities by activityType and year

	return ActivityProvider.GetActivitiesByActivityTypeAndYear(activityType, year)
}
