package services

import "mystravastats/domain/services/strava"

var ActivityProvider = NewStravaActivityProvider("strava-cache")

func FetchActivitiesByActivityTypeAndYear(activityType ActivityType, year *int) []strava.Activity {
	return ActivityProvider.GetActivitiesByActivityTypeAndYear(activityType, year)
}
