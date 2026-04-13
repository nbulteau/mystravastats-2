package services

import (
	"log"
	"math"
	"mystravastats/domain/business"
	"mystravastats/domain/statistics"
	"mystravastats/domain/strava"
	"strconv"
	"time"
)

// DashboardService provides methods to fetch various dashboard data related to Strava activities.

// FetchEddingtonNumber calculates the Eddington number for a given activity type.
func FetchEddingtonNumber(activityTypes ...business.ActivityType) business.EddingtonNumber {
	log.Printf("Get Eddington number for activity type %s", activityTypes)

	activitiesByActiveDays := activityProvider.GetActivitiesByActivityTypeGroupByActiveDays(activityTypes...)

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
		counts := make([]int, maxValue)
		for _, value := range activitiesByActiveDays {
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

// GetCumulativeDistancePerYear calculates the cumulative distance per year for a given activity type.
func GetCumulativeDistancePerYear(activityTypes ...business.ActivityType) map[string]map[string]float64 {
	log.Printf("Get cumulative distance per year for activity type %s", activityTypes)

	activitiesByYear := activityProvider.GetActivitiesByActivityTypeGroupByYear(activityTypes...)

	currentYear := time.Now().Year()
	result := make(map[string]map[string]float64)

	for year := 2010; year <= currentYear; year++ {
		yearStr := strconv.Itoa(year)
		if activities, exists := activitiesByYear[yearStr]; exists {
			activitiesByDay := groupActivitiesByDay(activities, year)
			cumulativeDistance := calculateCumulativeDistance(activitiesByDay)
			result[yearStr] = cumulativeDistance
		}
	}

	return result
}

func calculateCumulativeDistance(activitiesByDay map[string][]*strava.Activity) map[string]float64 {
	result := make(map[string]float64)

	var sum float64
	for _, day := range sortedKeys(activitiesByDay) {
		for _, activity := range activitiesByDay[day] {
			sum += activity.Distance / 1000
		}
		result[day] = sum
	}

	return result
}

func GetCumulativeElevationPerYear(activityTypes ...business.ActivityType) map[string]map[string]float64 {
	log.Printf("Get cumulative elevation per year for activity type %s", activityTypes)

	activitiesByYear := activityProvider.GetActivitiesByActivityTypeGroupByYear(activityTypes...)

	result := make(map[string]map[string]float64)
	currentYear := time.Now().Year()

	for year := 2010; year <= currentYear; year++ {
		yearStr := strconv.Itoa(year)
		if activities, ok := activitiesByYear[yearStr]; ok {
			activitiesByDay := groupActivitiesByDay(activities, year)
			result[yearStr] = cumulativeElevation(activitiesByDay)
		}
	}

	return result
}

func cumulativeElevation(activitiesByDay map[string][]*strava.Activity) map[string]float64 {
	sum := 0.0
	result := make(map[string]float64)
	for _, day := range sortedKeys(activitiesByDay) {
		for _, activity := range activitiesByDay[day] {
			sum += activity.TotalElevationGain
		}
		result[day] = sum
	}
	return result
}

func FetchDashboardData(activityTypes ...business.ActivityType) business.DashboardData {
	log.Printf("Get dashboard data for activity type %s", activityTypes)

	activitiesByYear := activityProvider.GetActivitiesByYearAndActivityTypes(nil, activityTypes...)

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

	activitiesGroupedByYear := groupActivitiesByYear(activitiesByYear)

	for year, activities := range activitiesGroupedByYear {
		nbActivitiesByYear[year] = len(activities)
		totalDistanceByYear[year] = sumDistance(activities)
		averageDistanceByYear[year] = averageDistance(activities)
		maxDistanceByYear[year] = maxDistance(activities)
		totalElevationByYear[year] = sumElevation(activities)
		averageElevationByYear[year] = averageElevation(activities)
		maxElevationByYear[year] = maxElevation(activities)
		averageSpeedByYear[year] = averageSpeed(activities)
		maxSpeedByYear[year] = maxSpeed(activities)
		averageHeartRateByYear[year] = averageHeartRate(activities)
		maxHeartRateByYear[year] = maxHeartRate(activities)
		averageWattsByYear[year] = averageWatts(activities)
		maxWattsByYear[year] = maxWatts(activities)
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
	var maxDistance float64
	for _, activity := range activities {
		distance := activity.Distance / 1000
		if distance > maxDistance {
			maxDistance = distance
		}
	}
	return maxDistance
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
	var maxElevation int
	for _, activity := range activities {
		elevation := int(activity.TotalElevationGain)
		if elevation > maxElevation {
			maxElevation = elevation
		}
	}
	return maxElevation
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

// maxHeartRate calculates the maximum heart rate for a list of Strava activities.
func maxHeartRate(activities []*strava.Activity) float64 {
	var maxHeartRate float64
	for _, activity := range activities {
		if activity.MaxHeartrate > maxHeartRate {
			maxHeartRate = activity.MaxHeartrate
		}
	}
	return maxHeartRate
}

// averageWatts calculates the average watts for a list of Strava activities.
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

// maxWatts calculates the maximum average watts for a list of Strava activities.
func maxWatts(activities []*strava.Activity) float64 {
	var maxWatts float64
	for _, activity := range activities {
		if activity.AverageWatts > maxWatts {
			maxWatts = activity.AverageWatts
		}
	}
	return maxWatts
}

type ActivityHeatmapActivity struct {
	ID             int64   `json:"id"`
	Name           string  `json:"name"`
	Type           string  `json:"type"`
	DistanceKm     float64 `json:"distanceKm"`
	ElevationGainM float64 `json:"elevationGainM"`
	DurationSec    int     `json:"durationSec"`
}

type ActivityHeatmapDay struct {
	DistanceKm     float64                   `json:"distanceKm"`
	ElevationGainM float64                   `json:"elevationGainM"`
	DurationSec    int                       `json:"durationSec"`
	ActivityCount  int                       `json:"activityCount"`
	Activities     []ActivityHeatmapActivity `json:"activities"`
}

// FetchActivityHeatmap returns daily training aggregates by year.
// Shape: year → ("MM-DD" → {distanceKm, elevationGainM, durationSec, activityCount, activities}).
func FetchActivityHeatmap(activityTypes ...business.ActivityType) map[string]map[string]ActivityHeatmapDay {
	log.Printf("Get activity heatmap for activity type %s", activityTypes)

	activitiesByYear := activityProvider.GetActivitiesByActivityTypeGroupByYear(activityTypes...)
	result := make(map[string]map[string]ActivityHeatmapDay)
	currentYear := time.Now().Year()

	for year := 2010; year <= currentYear; year++ {
		yearStr := strconv.Itoa(year)
		activities, ok := activitiesByYear[yearStr]
		if !ok {
			continue
		}
		activitiesByDay := groupActivitiesByDay(activities, year)
		dayMap := make(map[string]ActivityHeatmapDay, len(activitiesByDay))
		for day, acts := range activitiesByDay {
			var distanceKm float64
			var elevationGainM float64
			durationSec := 0
			details := make([]ActivityHeatmapActivity, 0, len(acts))

			for _, a := range acts {
				dayDistanceKm := a.Distance / 1000.0
				dayElevationGainM := a.TotalElevationGain
				dayDurationSec := a.MovingTime
				if dayDurationSec <= 0 {
					dayDurationSec = a.ElapsedTime
				}

				distanceKm += dayDistanceKm
				elevationGainM += dayElevationGainM
				durationSec += dayDurationSec

				details = append(details, ActivityHeatmapActivity{
					ID:             a.Id,
					Name:           a.Name,
					Type:           a.SportType,
					DistanceKm:     roundToOneDecimal(dayDistanceKm),
					ElevationGainM: roundToOneDecimal(dayElevationGainM),
					DurationSec:    dayDurationSec,
				})
			}

			dayMap[day] = ActivityHeatmapDay{
				DistanceKm:     roundToOneDecimal(distanceKm),
				ElevationGainM: roundToOneDecimal(elevationGainM),
				DurationSec:    durationSec,
				ActivityCount:  len(acts),
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
