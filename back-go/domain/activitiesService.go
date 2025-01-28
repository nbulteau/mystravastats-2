package domain

import (
	"mystravastats/domain/business"
	"mystravastats/domain/strava"
)

func FetchActivitiesByActivityTypeAndYear(activityType business.ActivityType, year *int) []strava.Activity {
	return activityProvider.GetActivitiesByActivityTypeAndYear(activityType, year)
}

func FetchAthlete() strava.Athlete {
	return activityProvider.GetAthlete()
}
