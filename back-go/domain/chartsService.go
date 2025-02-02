package domain

import (
	"log"
	"mystravastats/domain/business"
	"mystravastats/domain/strava"
	"strconv"
	"time"
)

func FetchChartsDistanceByPeriod(activityType business.ActivityType, year *int, period business.Period) []map[string]float64 {
	log.Printf("Get distance by %s by activity (%s) type by year (%d)", period, activityType, year)

	activities := activityProvider.GetActivitiesByActivityTypeAndYear(activityType, year)
	activitiesByPeriod := activitiesByPeriod(activities, *year, period)

	result := make([]map[string]float64, 0)
	for period, activities := range activitiesByPeriod {
		totalDistance := 0.0
		for _, activity := range activities {
			totalDistance += activity.Distance / 1000
		}
		result = append(result, map[string]float64{period: totalDistance})
	}

	return result
}

func FetchChartsElevationByPeriod(activityType business.ActivityType, year *int, period business.Period) []map[string]float64 {
	log.Printf("Get elevation by %s by activity (%s) type by year (%d)", period, activityType, year)

	activities := activityProvider.GetActivitiesByActivityTypeAndYear(activityType, year)
	activitiesByPeriod := activitiesByPeriod(activities, *year, period)

	result := make([]map[string]float64, 0)
	for period, activities := range activitiesByPeriod {
		totalElevation := 0.0
		for _, activity := range activities {
			totalElevation += activity.TotalElevationGain
		}
		result = append(result, map[string]float64{period: totalElevation})
	}

	return result
}

func FetchChartsAverageSpeedByPeriod(activityType business.ActivityType, year *int, period business.Period) []map[string]float64 {
	log.Printf("Get average speed by %s by activity (%s) type by year (%d)", period, activityType, year)

	activities := activityProvider.GetActivitiesByActivityTypeAndYear(activityType, year)
	activitiesByPeriod := activitiesByPeriod(activities, *year, period)

	result := make([]map[string]float64, 0)
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

	return result
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
	return activitiesByMonth
}

func groupActivitiesByWeek(activities []*strava.Activity) map[string][]*strava.Activity {
	activitiesByWeek := make(map[string][]*strava.Activity)
	for _, activity := range activities {
		year := time.Now().Year()
		month, _ := strconv.Atoi(activity.StartDateLocal[5:7])
		day, _ := strconv.Atoi(activity.StartDateLocal[8:10])
		week, _ := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC).ISOWeek()
		weekStr := strconv.Itoa(week)
		activitiesByWeek[weekStr] = append(activitiesByWeek[weekStr], activity)
	}
	return activitiesByWeek
}
