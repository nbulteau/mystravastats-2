package domain

import (
	"mystravastats/adapters/stravaapi"
	"mystravastats/domain/business"
	"mystravastats/domain/strava"
)

var ActivityProvider = stravaapi.NewStravaActivityProvider("strava-cache")

func FetchActivitiesByActivityTypeAndYear(activityType business.ActivityType, year *int) []strava.Activity {
	return ActivityProvider.GetActivitiesByActivityTypeAndYear(activityType, year)
}
