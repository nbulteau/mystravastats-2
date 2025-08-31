package services

import (
	"fmt"
	"log"
	"mystravastats/domain/business"
	"mystravastats/domain/strava"
	"sort"
	"time"
)

// FetchChartsDistanceByPeriod retrieves the total distance of activities by period for a given activity type and year.
func FetchChartsDistanceByPeriod(year *int, period business.Period, activityTypes ...business.ActivityType) []map[string]float64 {
	log.Printf("Get distance by %s by activity (%s) type by year (%d)", period, activityTypes, *year)

	activities := activityProvider.GetActivitiesByYearAndActivityTypes(year, activityTypes...)
	activitiesByPeriod := activitiesByPeriod(activities, *year, period)

	result := make([]map[string]float64, 0)
	for period, activities := range activitiesByPeriod {
		totalDistance := 0.0
		for _, activity := range activities {
			totalDistance += activity.Distance / 1000
		}
		result = append(result, map[string]float64{period: totalDistance})
	}

	return sortResultByKey(result)
}

// FetchChartsElevationByPeriod retrieves the total elevation gain of activities by period for a given activity type and year.
func FetchChartsElevationByPeriod(year *int, period business.Period, activityTypes ...business.ActivityType) []map[string]float64 {
	log.Printf("Get elevation by %s by activity (%s) type by year (%d)", period, activityTypes, *year)

	activities := activityProvider.GetActivitiesByYearAndActivityTypes(year, activityTypes...)
	activitiesByPeriod := activitiesByPeriod(activities, *year, period)

	size := 12
	switch period {
	case business.PeriodWeeks:
		size = 52
	case business.PeriodDays:
		size = 365
	}

	result := make([]map[string]float64, size)
	for period, activities := range activitiesByPeriod {
		totalElevation := 0.0
		for _, activity := range activities {
			totalElevation += activity.TotalElevationGain
		}
		result = append(result, map[string]float64{period: totalElevation})
	}

	return sortResultByKey(result)
}

func FetchChartsAverageSpeedByPeriod(year *int, period business.Period, activityTypes ...business.ActivityType) []map[string]float64 {
	log.Printf("Get average speed by %s by activity (%s) type by year (%d)", period, activityTypes, *year)

	activities := activityProvider.GetActivitiesByYearAndActivityTypes(year, activityTypes...)
	activitiesByPeriod := activitiesByPeriod(activities, *year, period)

	size := 12
	switch period {
	case business.PeriodWeeks:
		size = 52
	case business.PeriodDays:
		size = 365
	}

	result := make([]map[string]float64, size)
	for period, activities := range activitiesByPeriod {
		if len(activities) == 0 {
			result = append(result, map[string]float64{period: 0.0})
			continue
		}
		totalSpeed := 0.0
		for _, activity := range activities {
			totalSpeed += activity.AverageSpeed
		}
		averageSpeed := totalSpeed / float64(len(activities))
		result = append(result, map[string]float64{period: averageSpeed})
	}

	return sortResultByKey(result)
}
func FetchChartsAverageCadenceByPeriod(year *int, period business.Period, activityTypes ...business.ActivityType) []map[string]float64 {
	log.Printf("Get average cadence by %s by activity (%s) type by year (%d)", period, activityTypes, *year)

	activities := activityProvider.GetActivitiesByYearAndActivityTypes(year, activityTypes...)
	activitiesByPeriod := activitiesByPeriod(activities, *year, period)

	size := 12
	switch period {
	case business.PeriodWeeks:
		size = 52
	case business.PeriodDays:
		size = 365
	}

	result := make([]map[string]float64, size)
	for period, activities := range activitiesByPeriod {
		if len(activities) == 0 {
			result = append(result, map[string]float64{period: 0.0})
			continue
		}
		totalCadence := 0.0
		for _, activity := range activities {
			totalCadence += activity.AverageCadence
		}
		averageCadence := totalCadence / float64(len(activities))
		if activityTypes[0] == business.Run || activityTypes[0] == business.TrailRun {
			averageCadence = averageCadence * 2 // Strava reports a half-cadence for running activities
		}
		result = append(result, map[string]float64{period: averageCadence})
	}

	return sortResultByKey(result)
}

func activitiesByPeriod(activities []*strava.Activity, year int, period business.Period) map[string][]*strava.Activity {
	switch period {
	case business.PeriodMonths:
		return groupActivitiesByMonth(activities)
	case business.PeriodWeeks:
		return groupActivitiesByWeek(activities)
	case business.PeriodDays:
		return groupActivitiesByDay(activities, year)
	default:
		return nil
	}
}

func groupActivitiesByMonth(activities []*strava.Activity) map[string][]*strava.Activity {
	activitiesByMonth := make(map[string][]*strava.Activity)
	for _, activity := range activities {
		month := activity.StartDateLocal[5:7]
		activitiesByMonth[month] = append(activitiesByMonth[month], activity)
	}

	// Add months without activities
	for month := 1; month <= 12; month++ {
		monthStr := fmt.Sprintf("%02d", month)
		if _, exists := activitiesByMonth[monthStr]; !exists {
			activitiesByMonth[monthStr] = []*strava.Activity{}
		}
	}

	return activitiesByMonth
}

func groupActivitiesByWeek(activities []*strava.Activity) map[string][]*strava.Activity {
	activitiesByWeek := make(map[string][]*strava.Activity)
	for _, activity := range activities {
		date, _ := time.Parse("2006-01-02T15:04:05Z", activity.StartDateLocal)
		_, week := date.ISOWeek()
		weekStr := fmt.Sprintf("%02d", week)
		activitiesByWeek[weekStr] = append(activitiesByWeek[weekStr], activity)
	}

	// Add weeks without activities
	for week := 1; week <= 52; week++ {
		weekStr := fmt.Sprintf("%02d", week)
		if _, exists := activitiesByWeek[weekStr]; !exists {
			activitiesByWeek[weekStr] = []*strava.Activity{}
		}
	}

	return activitiesByWeek
}

func sortResultByKey(result []map[string]float64) []map[string]float64 {
	// Extract keys and sort them
	keys := make([]string, 0)
	for _, m := range result {
		for k := range m {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	// Create a new sorted slice of maps
	sortedResult := make([]map[string]float64, 0, len(result))
	for _, k := range keys {
		for _, m := range result {
			if val, ok := m[k]; ok {
				sortedResult = append(sortedResult, map[string]float64{k: val})
			}
		}
	}
	return sortedResult
}
