package domain

import (
	"mystravastats/adapters/stravaapi"
	"mystravastats/domain/strava"
	"sort"
	"time"
)

var activityProvider = stravaapi.NewStravaActivityProvider("strava-cache")

func groupActivitiesByDay(activities []*strava.Activity, year int) map[string][]*strava.Activity {
	activitiesByDay := make(map[string][]*strava.Activity)

	for _, activity := range activities {
		day := activity.StartDateLocal[5:10]
		activitiesByDay[day] = append(activitiesByDay[day], activity)
	}

	currentDate := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 365+1; i++ {
		dayString := currentDate.Format("01-02")
		if _, exists := activitiesByDay[dayString]; !exists {
			activitiesByDay[dayString] = []*strava.Activity{}
		}
		currentDate = currentDate.AddDate(0, 0, 1)
	}

	// Sort the map keys
	sortedKeys := sortedKeys(activitiesByDay)

	sortedActivitiesByDay := make(map[string][]*strava.Activity)
	for _, k := range sortedKeys {
		sortedActivitiesByDay[k] = activitiesByDay[k]
	}

	return sortedActivitiesByDay
}

func sortedKeys(m map[string][]*strava.Activity) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	return keys
}
