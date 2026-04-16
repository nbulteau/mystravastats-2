package infrastructure

import (
	"fmt"
	"log"
	"mystravastats/internal/platform/activityprovider"
	"mystravastats/internal/shared/domain/business"
	"mystravastats/internal/shared/domain/strava"
	"sort"
	"time"
)

// ChartsServiceAdapter provides chart projections directly from provider data.
type ChartsServiceAdapter struct{}

func NewChartsServiceAdapter() *ChartsServiceAdapter {
	return &ChartsServiceAdapter{}
}

func (adapter *ChartsServiceAdapter) FindDistanceByPeriod(year *int, period business.Period, activityTypes ...business.ActivityType) []map[string]float64 {
	resolvedYear, ok := resolveChartYear(year, period, activityTypes, "distance")
	if !ok {
		return []map[string]float64{}
	}

	log.Printf("Get distance by %s by activity (%v) type by year (%d)", period, activityTypes, resolvedYear)

	activities := activityprovider.Get().GetActivitiesByYearAndActivityTypes(year, activityTypes...)
	activitiesByPeriod := activitiesByPeriod(activities, resolvedYear, period)

	result := make([]map[string]float64, 0, len(activitiesByPeriod))
	for _, periodKey := range sortedPeriodKeys(activitiesByPeriod) {
		periodActivities := activitiesByPeriod[periodKey]
		totalDistance := 0.0
		for _, activity := range periodActivities {
			totalDistance += activity.Distance / 1000
		}
		result = append(result, map[string]float64{periodKey: totalDistance})
	}

	return result
}

func (adapter *ChartsServiceAdapter) FindElevationByPeriod(year *int, period business.Period, activityTypes ...business.ActivityType) []map[string]float64 {
	resolvedYear, ok := resolveChartYear(year, period, activityTypes, "elevation")
	if !ok {
		return []map[string]float64{}
	}

	log.Printf("Get elevation by %s by activity (%v) type by year (%d)", period, activityTypes, resolvedYear)

	activities := activityprovider.Get().GetActivitiesByYearAndActivityTypes(year, activityTypes...)
	activitiesByPeriod := activitiesByPeriod(activities, resolvedYear, period)

	size := 12
	switch period {
	case business.PeriodWeeks:
		size = 52
	case business.PeriodDays:
		size = 365
	}

	result := make([]map[string]float64, 0, size)
	for _, periodKey := range sortedPeriodKeys(activitiesByPeriod) {
		periodActivities := activitiesByPeriod[periodKey]
		totalElevation := 0.0
		for _, activity := range periodActivities {
			totalElevation += activity.TotalElevationGain
		}
		result = append(result, map[string]float64{periodKey: totalElevation})
	}

	return result
}

func (adapter *ChartsServiceAdapter) FindAverageSpeedByPeriod(year *int, period business.Period, activityTypes ...business.ActivityType) []map[string]float64 {
	resolvedYear, ok := resolveChartYear(year, period, activityTypes, "average speed")
	if !ok {
		return []map[string]float64{}
	}

	log.Printf("Get average speed by %s by activity (%v) type by year (%d)", period, activityTypes, resolvedYear)

	activities := activityprovider.Get().GetActivitiesByYearAndActivityTypes(year, activityTypes...)
	activitiesByPeriod := activitiesByPeriod(activities, resolvedYear, period)

	size := 12
	switch period {
	case business.PeriodWeeks:
		size = 52
	case business.PeriodDays:
		size = 365
	}

	result := make([]map[string]float64, 0, size)
	for _, periodKey := range sortedPeriodKeys(activitiesByPeriod) {
		periodActivities := activitiesByPeriod[periodKey]
		if len(periodActivities) == 0 {
			result = append(result, map[string]float64{periodKey: 0.0})
			continue
		}
		totalSpeed := 0.0
		for _, activity := range periodActivities {
			totalSpeed += activity.AverageSpeed
		}
		averageSpeed := totalSpeed / float64(len(periodActivities))
		result = append(result, map[string]float64{periodKey: averageSpeed})
	}

	return result
}

func (adapter *ChartsServiceAdapter) FindAverageCadenceByPeriod(year *int, period business.Period, activityTypes ...business.ActivityType) []map[string]float64 {
	resolvedYear, ok := resolveChartYear(year, period, activityTypes, "average cadence")
	if !ok {
		return []map[string]float64{}
	}

	log.Printf("Get average cadence by %s by activity (%v) type by year (%d)", period, activityTypes, resolvedYear)

	activities := activityprovider.Get().GetActivitiesByYearAndActivityTypes(year, activityTypes...)
	activitiesByPeriod := activitiesByPeriod(activities, resolvedYear, period)

	size := 12
	switch period {
	case business.PeriodWeeks:
		size = 52
	case business.PeriodDays:
		size = 365
	}

	result := make([]map[string]float64, 0, size)
	for _, periodKey := range sortedPeriodKeys(activitiesByPeriod) {
		periodActivities := activitiesByPeriod[periodKey]
		if len(periodActivities) == 0 {
			result = append(result, map[string]float64{periodKey: 0.0})
			continue
		}
		totalCadence := 0.0
		nbActivitiesWithCadence := 0
		for _, activity := range periodActivities {
			if activity.AverageCadence == 0 {
				continue
			}
			nbActivitiesWithCadence++
			totalCadence += activity.AverageCadence
		}
		if nbActivitiesWithCadence == 0 {
			result = append(result, map[string]float64{periodKey: 0.0})
			continue
		}
		averageCadence := totalCadence / float64(nbActivitiesWithCadence)
		// Strava reports running cadence as half-cadence in the activity payload.
		if len(activityTypes) > 0 && (activityTypes[0] == business.Run || activityTypes[0] == business.TrailRun) {
			averageCadence = averageCadence * 2
		}
		result = append(result, map[string]float64{periodKey: averageCadence})
	}

	return result
}

func resolveChartYear(year *int, period business.Period, activityTypes []business.ActivityType, metric string) (int, bool) {
	if year == nil {
		log.Printf("Skip %s by %s by activity (%v): missing year", metric, period, activityTypes)
		return 0, false
	}
	return *year, true
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
		return map[string][]*strava.Activity{}
	}
}

func groupActivitiesByMonth(activities []*strava.Activity) map[string][]*strava.Activity {
	activitiesByMonth := make(map[string][]*strava.Activity)
	for _, activity := range activities {
		month := activity.StartDateLocal[5:7]
		activitiesByMonth[month] = append(activitiesByMonth[month], activity)
	}

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
		date, err := time.Parse("2006-01-02T15:04:05Z", activity.StartDateLocal)
		if err != nil {
			continue
		}
		_, week := date.ISOWeek()
		weekStr := fmt.Sprintf("%02d", week)
		activitiesByWeek[weekStr] = append(activitiesByWeek[weekStr], activity)
	}

	for week := 1; week <= 52; week++ {
		weekStr := fmt.Sprintf("%02d", week)
		if _, exists := activitiesByWeek[weekStr]; !exists {
			activitiesByWeek[weekStr] = []*strava.Activity{}
		}
	}
	return activitiesByWeek
}

func groupActivitiesByDay(activities []*strava.Activity, year int) map[string][]*strava.Activity {
	activitiesByDay := make(map[string][]*strava.Activity)

	for _, activity := range activities {
		startDate, err := time.Parse("2006-01-02T15:04:05Z", activity.StartDateLocal)
		if err != nil {
			continue
		}
		day := startDate.Format("01-02")
		activitiesByDay[day] = append(activitiesByDay[day], activity)
	}

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

	return activitiesByDay
}

func isLeapYear(year int) bool {
	return year%4 == 0 && (year%100 != 0 || year%400 == 0)
}

func sortedPeriodKeys(periods map[string][]*strava.Activity) []string {
	keys := make([]string, 0, len(periods))
	for key := range periods {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
