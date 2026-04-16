package infrastructure

import (
	"log"
	"math"
	"mystravastats/domain/statistics"
	dashboardDomain "mystravastats/internal/dashboard/domain"
	"mystravastats/internal/platform/activityprovider"
	"mystravastats/internal/shared/domain/business"
	"mystravastats/internal/shared/domain/strava"
	"sort"
	"strconv"
	"time"
)

func computeEddingtonNumber(activityTypes ...business.ActivityType) business.EddingtonNumber {
	log.Printf("Get Eddington number for activity type %s", activityTypes)

	activitiesByActiveDays := activityprovider.Get().GetActivitiesByActivityTypeGroupByActiveDays(activityTypes...)
	return computeEddingtonFromDailyTotals(activitiesByActiveDays)
}

func computeEddingtonFromDailyTotals(activitiesByActiveDays map[string]int) business.EddingtonNumber {
	var eddingtonList []int
	if len(activitiesByActiveDays) == 0 {
		eddingtonList = []int{}
	} else {
		maxValue := 0
		for _, value := range activitiesByActiveDays {
			if value > maxValue {
				maxValue = value
			}
		}
		if maxValue <= 0 {
			return business.EddingtonNumber{Number: 0, List: []int{}}
		}
		counts := make([]int, maxValue)
		for _, value := range activitiesByActiveDays {
			if value <= 0 {
				continue
			}
			for day := value; day > 0; day-- {
				counts[day-1]++
			}
		}
		eddingtonList = counts
	}

	eddingtonNumber := 0
	for day := len(eddingtonList); day > 0; day-- {
		if eddingtonList[day-1] >= day {
			eddingtonNumber = day
			break
		}
	}

	return business.EddingtonNumber{Number: eddingtonNumber, List: eddingtonList}
}

func computeCumulativeDistancePerYear(activityTypes ...business.ActivityType) map[string]map[string]float64 {
	log.Printf("Get cumulative distance per year for activity type %s", activityTypes)

	activitiesByYear := activityprovider.Get().GetActivitiesByActivityTypeGroupByYear(activityTypes...)
	currentYear := time.Now().Year()
	result := make(map[string]map[string]float64)

	for year := 2010; year <= currentYear; year++ {
		yearStr := strconv.Itoa(year)
		if activities, exists := activitiesByYear[yearStr]; exists {
			activitiesByDay := groupActivitiesByDay(activities, year)
			result[yearStr] = calculateCumulativeDistance(activitiesByDay)
		}
	}

	return result
}

func calculateCumulativeDistance(activitiesByDay map[string][]*strava.Activity) map[string]float64 {
	result := make(map[string]float64)
	var sum float64
	for _, day := range sortedDayKeys(activitiesByDay) {
		for _, activity := range activitiesByDay[day] {
			sum += activity.Distance / 1000
		}
		result[day] = sum
	}
	return result
}

func computeCumulativeElevationPerYear(activityTypes ...business.ActivityType) map[string]map[string]float64 {
	log.Printf("Get cumulative elevation per year for activity type %s", activityTypes)

	activitiesByYear := activityprovider.Get().GetActivitiesByActivityTypeGroupByYear(activityTypes...)
	result := make(map[string]map[string]float64)
	currentYear := time.Now().Year()

	for year := 2010; year <= currentYear; year++ {
		yearStr := strconv.Itoa(year)
		if activities, ok := activitiesByYear[yearStr]; ok {
			activitiesByDay := groupActivitiesByDay(activities, year)
			result[yearStr] = calculateCumulativeElevation(activitiesByDay)
		}
	}

	return result
}

func calculateCumulativeElevation(activitiesByDay map[string][]*strava.Activity) map[string]float64 {
	sum := 0.0
	result := make(map[string]float64)
	for _, day := range sortedDayKeys(activitiesByDay) {
		for _, activity := range activitiesByDay[day] {
			sum += activity.TotalElevationGain
		}
		result[day] = sum
	}
	return result
}

func computeDashboardData(activityTypes ...business.ActivityType) business.DashboardData {
	log.Printf("Get dashboard data for activity type %s", activityTypes)

	activities := activityprovider.Get().GetActivitiesByYearAndActivityTypes(nil, activityTypes...)
	activitiesGroupedByYear := groupActivitiesByYear(activities)

	nbActivitiesByYear := make(map[string]int)
	totalDistanceByYear := make(map[string]float64)
	averageDistanceByYear := make(map[string]float64)
	maxDistanceByYear := make(map[string]float64)
	totalElevationByYear := make(map[string]int)
	averageElevationByYear := make(map[string]int)
	maxElevationByYear := make(map[string]int)
	averageSpeedByYear := make(map[string]float64)
	maxSpeedByYear := make(map[string]float64)
	averageHeartRateByYear := make(map[string]int)
	maxHeartRateByYear := make(map[string]float64)
	averageWattsByYear := make(map[string]float64)
	maxWattsByYear := make(map[string]float64)

	for year, yearActivities := range activitiesGroupedByYear {
		nbActivitiesByYear[year] = len(yearActivities)
		totalDistanceByYear[year] = sumDistance(yearActivities)
		averageDistanceByYear[year] = averageDistance(yearActivities)
		maxDistanceByYear[year] = maxDistance(yearActivities)
		totalElevationByYear[year] = sumElevation(yearActivities)
		averageElevationByYear[year] = averageElevation(yearActivities)
		maxElevationByYear[year] = maxElevation(yearActivities)
		averageSpeedByYear[year] = averageSpeed(yearActivities)
		maxSpeedByYear[year] = maxSpeed(yearActivities)
		averageHeartRateByYear[year] = averageHeartRate(yearActivities)
		maxHeartRateByYear[year] = maxHeartRate(yearActivities)
		averageWattsByYear[year] = averageWatts(yearActivities)
		maxWattsByYear[year] = maxWatts(yearActivities)
	}

	return business.DashboardData{
		NbActivities:           nbActivitiesByYear,
		TotalDistanceByYear:    totalDistanceByYear,
		AverageDistanceByYear:  averageDistanceByYear,
		MaxDistanceByYear:      maxDistanceByYear,
		TotalElevationByYear:   totalElevationByYear,
		AverageElevationByYear: averageElevationByYear,
		MaxElevationByYear:     maxElevationByYear,
		AverageSpeedByYear:     averageSpeedByYear,
		MaxSpeedByYear:         maxSpeedByYear,
		AverageHeartRateByYear: averageHeartRateByYear,
		MaxHeartRateByYear:     maxHeartRateByYear,
		AverageWattsByYear:     averageWattsByYear,
		MaxWattsByYear:         maxWattsByYear,
	}
}

func groupActivitiesByYear(activities []*strava.Activity) map[string][]*strava.Activity {
	activitiesByYear := make(map[string][]*strava.Activity)
	for _, activity := range activities {
		year := activity.StartDateLocal[:4]
		activitiesByYear[year] = append(activitiesByYear[year], activity)
	}
	return activitiesByYear
}

func sumDistance(activities []*strava.Activity) float64 {
	var sum float64
	for _, activity := range activities {
		sum += activity.Distance / 1000
	}
	return sum
}

func averageDistance(activities []*strava.Activity) float64 {
	if len(activities) == 0 {
		return 0
	}
	return sumDistance(activities) / float64(len(activities))
}

func maxDistance(activities []*strava.Activity) float64 {
	var maxDistanceValue float64
	for _, activity := range activities {
		distance := activity.Distance / 1000
		if distance > maxDistanceValue {
			maxDistanceValue = distance
		}
	}
	return maxDistanceValue
}

func sumElevation(activities []*strava.Activity) int {
	var sum int
	for _, activity := range activities {
		sum += int(activity.TotalElevationGain)
	}
	return sum
}

func averageElevation(activities []*strava.Activity) int {
	if len(activities) == 0 {
		return 0
	}
	return sumElevation(activities) / len(activities)
}

func maxElevation(activities []*strava.Activity) int {
	var maxElevationValue int
	for _, activity := range activities {
		elevation := int(activity.TotalElevationGain)
		if elevation > maxElevationValue {
			maxElevationValue = elevation
		}
	}
	return maxElevationValue
}

func averageSpeed(activities []*strava.Activity) float64 {
	var sum float64
	var count int
	for _, activity := range activities {
		if activity.AverageSpeed > 0 {
			sum += activity.AverageSpeed
			count++
		}
	}
	if count == 0 {
		return 0
	}
	return sum / float64(count)
}

func maxSpeed(activities []*strava.Activity) float64 {
	activityEffort := statistics.FindBestActivityEffort(activities, 200.0)
	if activityEffort == nil {
		return 0.0
	}
	return activityEffort.GetMSSpeed()
}

func averageHeartRate(activities []*strava.Activity) int {
	var sum int
	var count int
	for _, activity := range activities {
		if activity.AverageHeartrate > 0 {
			sum += int(activity.AverageHeartrate)
			count++
		}
	}
	if count == 0 {
		return 0
	}
	return sum / count
}

func maxHeartRate(activities []*strava.Activity) float64 {
	var maxHeartRateValue float64
	for _, activity := range activities {
		if activity.MaxHeartrate > maxHeartRateValue {
			maxHeartRateValue = activity.MaxHeartrate
		}
	}
	return maxHeartRateValue
}

func averageWatts(activities []*strava.Activity) float64 {
	var sum float64
	var count float64
	for _, activity := range activities {
		if activity.AverageWatts > 0 {
			sum += activity.AverageWatts
			count++
		}
	}
	if count == 0 {
		return 0
	}
	return sum / count
}

func maxWatts(activities []*strava.Activity) float64 {
	var maxWattsValue float64
	for _, activity := range activities {
		if activity.AverageWatts > maxWattsValue {
			maxWattsValue = activity.AverageWatts
		}
	}
	return maxWattsValue
}

func computeActivityHeatmap(activityTypes ...business.ActivityType) map[string]map[string]dashboardDomain.ActivityHeatmapDay {
	log.Printf("Get activity heatmap for activity type %s", activityTypes)

	activitiesByYear := activityprovider.Get().GetActivitiesByActivityTypeGroupByYear(activityTypes...)
	result := make(map[string]map[string]dashboardDomain.ActivityHeatmapDay)
	currentYear := time.Now().Year()

	for year := 2010; year <= currentYear; year++ {
		yearStr := strconv.Itoa(year)
		activities, ok := activitiesByYear[yearStr]
		if !ok {
			continue
		}
		activitiesByDay := groupActivitiesByDay(activities, year)
		dayMap := make(map[string]dashboardDomain.ActivityHeatmapDay, len(activitiesByDay))
		for day, dayActivities := range activitiesByDay {
			var distanceKm float64
			var elevationGainM float64
			durationSec := 0
			details := make([]dashboardDomain.ActivityHeatmapActivity, 0, len(dayActivities))

			for _, activity := range dayActivities {
				dayDistanceKm := activity.Distance / 1000.0
				dayElevationGainM := activity.TotalElevationGain
				dayDurationSec := activity.MovingTime
				if dayDurationSec <= 0 {
					dayDurationSec = activity.ElapsedTime
				}

				distanceKm += dayDistanceKm
				elevationGainM += dayElevationGainM
				durationSec += dayDurationSec

				details = append(details, dashboardDomain.ActivityHeatmapActivity{
					ID:             activity.Id,
					Name:           activity.Name,
					Type:           activity.SportType,
					DistanceKm:     roundToOneDecimal(dayDistanceKm),
					ElevationGainM: roundToOneDecimal(dayElevationGainM),
					DurationSec:    dayDurationSec,
				})
			}

			dayMap[day] = dashboardDomain.ActivityHeatmapDay{
				DistanceKm:     roundToOneDecimal(distanceKm),
				ElevationGainM: roundToOneDecimal(elevationGainM),
				DurationSec:    durationSec,
				ActivityCount:  len(dayActivities),
				Activities:     details,
			}
		}
		result[yearStr] = dayMap
	}

	return result
}

func roundToOneDecimal(value float64) float64 {
	return math.Round(value*10) / 10
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

func sortedDayKeys(m map[string][]*strava.Activity) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
