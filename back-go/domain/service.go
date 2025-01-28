package domain

import "mystravastats/adapters/stravaapi"

var activityProvider = stravaapi.NewStravaActivityProvider("strava-cache")
