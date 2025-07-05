package services

import (
	"mystravastats/adapters/stravaapi"
	"mystravastats/domain/helpers"
	"mystravastats/domain/strava"
	"sort"
	"time"
)

var activityProvider = stravaapi.NewStravaActivityProvider(helpers.StravaCachePath)

// groupActivitiesByDay groups activities by day and fills in missing days
func groupActivitiesByDay(activities []*strava.Activity, year int) map[string][]*strava.Activity {
	activitiesByDay := make(map[string][]*strava.Activity)

	// Sort activities by start date
	for _, activity := range activities {
		startDate, err := time.Parse("2006-01-02T15:04:05Z", activity.StartDateLocal)
		if err != nil {
			continue // Ignore activities with invalid dates
		}
		day := startDate.Format("01-02")
		activitiesByDay[day] = append(activitiesByDay[day], activity)
	}

	// Fill in missing days
	fillMissingDays(activitiesByDay, year)

	// Sort the map keys
	sortedKeys := sortedKeys(activitiesByDay)

	// Create a new map with sorted keys
	sortedActivitiesByDay := make(map[string][]*strava.Activity)
	for _, k := range sortedKeys {
		sortedActivitiesByDay[k] = activitiesByDay[k]
	}

	return sortedActivitiesByDay
}

// fillMissingDays fills in missing days for a given year
func fillMissingDays(activitiesByDay map[string][]*strava.Activity, year int) {
	currentDate := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	daysInYear := 365
	if isLeapYear(year) {
		daysInYear = 366
	}

	for i := 0; i < daysInYear; i++ {
		dayString := currentDate.Format("01-02")
		if _, exists := activitiesByDay[dayString]; !exists {
			activitiesByDay[dayString] = []*strava.Activity{}
		}
		currentDate = currentDate.AddDate(0, 0, 1)
	}
}

// isLeapYear checks if a year is a leap year
func isLeapYear(year int) bool {
	return year%4 == 0 && (year%100 != 0 || year%400 == 0)
}

// sortedKeys returns a sorted slice of keys from the given map
func sortedKeys(m map[string][]*strava.Activity) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
