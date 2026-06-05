package infrastructure

import (
	"log"
	"math"
	"mystravastats/domain/statistics"
	dashboardDomain "mystravastats/internal/dashboard/domain"
	dataqualityInfra "mystravastats/internal/dataquality/infrastructure"
	"mystravastats/internal/platform/activityprovider"
	"mystravastats/internal/shared/domain/business"
	"mystravastats/internal/shared/domain/strava"
	"sort"
	"strconv"
	"time"
)

func computeEddingtonNumber(scope business.EddingtonScope, metric business.EddingtonMetric, basis business.EddingtonBasis, year *int, activityTypes ...business.ActivityType) business.EddingtonNumber {
	if scope == "" {
		scope = business.EddingtonScopeLifetime
	}
	if metric == "" {
		metric = business.EddingtonMetricDistance
	}
	if basis == "" {
		basis = business.EddingtonBasisDays
	}
	log.Printf("Get Eddington number for scope %s, metric %s, basis %s and activity type %s", scope, metric, basis, activityTypes)

	activities := eddingtonActivitiesForScope(scope, year, activityTypes...)
	return computeEddingtonFromValues(scope, metric, basis, eddingtonValues(activities, metric, basis))
}

func eddingtonActivitiesForScope(scope business.EddingtonScope, year *int, activityTypes ...business.ActivityType) []*strava.Activity {
	switch scope {
	case business.EddingtonScopeYear:
		return dataqualityInfra.FilterExcludedFromStats(activityprovider.Get().GetActivitiesByYearAndActivityTypes(year, activityTypes...))
	case business.EddingtonScopeRolling12Months:
		activities := dataqualityInfra.FilterExcludedFromStats(activityprovider.Get().GetActivitiesByYearAndActivityTypes(nil, activityTypes...))
		now := time.Now()
		return filterActivitiesByDateRange(activities, now.AddDate(-1, 0, 0), now)
	default:
		return dataqualityInfra.FilterExcludedFromStats(activityprovider.Get().GetActivitiesByYearAndActivityTypes(nil, activityTypes...))
	}
}

func computeEddingtonFromDailyTotals(scope business.EddingtonScope, activitiesByActiveDays map[string]int) business.EddingtonNumber {
	return computeEddingtonFromValues(scope, business.EddingtonMetricDistance, business.EddingtonBasisDays, mapValues(activitiesByActiveDays))
}

func computeEddingtonFromValues(scope business.EddingtonScope, metric business.EddingtonMetric, basis business.EddingtonBasis, values []int) business.EddingtonNumber {
	var eddingtonList []int
	if len(values) == 0 {
		eddingtonList = []int{}
	} else {
		maxValue := 0
		for _, value := range values {
			if value > maxValue {
				maxValue = value
			}
		}
		if maxValue <= 0 {
			return withEddingtonProgress(business.EddingtonNumber{Number: 0, List: []int{}, Scope: scope, Metric: metric, Basis: basis, Unit: eddingtonUnit(metric), ThresholdScale: eddingtonThresholdScale(metric)})
		}
		counts := make([]int, maxValue)
		for _, value := range values {
			if value <= 0 {
				continue
			}
			for threshold := value; threshold > 0; threshold-- {
				counts[threshold-1]++
			}
		}
		eddingtonList = counts
	}

	eddingtonNumber := 0
	for threshold := len(eddingtonList); threshold > 0; threshold-- {
		if eddingtonList[threshold-1] >= threshold {
			eddingtonNumber = threshold
			break
		}
	}

	return withEddingtonProgress(business.EddingtonNumber{Number: eddingtonNumber, List: eddingtonList, Scope: scope, Metric: metric, Basis: basis, Unit: eddingtonUnit(metric), ThresholdScale: eddingtonThresholdScale(metric)})
}

func eddingtonValues(activities []*strava.Activity, metric business.EddingtonMetric, basis business.EddingtonBasis) []int {
	if basis == business.EddingtonBasisActivities {
		values := make([]int, 0, len(activities))
		for _, activity := range activities {
			if activity == nil {
				continue
			}
			values = append(values, eddingtonActivityValue(activity, metric))
		}
		return values
	}

	return mapValues(dailyMetricTotals(activities, metric))
}

func dailyMetricTotals(activities []*strava.Activity, metric business.EddingtonMetric) map[string]int {
	result := make(map[string]int)
	for _, activity := range activities {
		if activity == nil || len(activity.StartDateLocal) < 10 {
			continue
		}
		day := activity.StartDateLocal[:10]
		result[day] += eddingtonActivityValue(activity, metric)
	}
	return result
}

func dailyDistanceTotals(activities []*strava.Activity) map[string]int {
	return dailyMetricTotals(activities, business.EddingtonMetricDistance)
}

func eddingtonActivityValue(activity *strava.Activity, metric business.EddingtonMetric) int {
	switch metric {
	case business.EddingtonMetricElevation:
		return int(activity.TotalElevationGain / float64(eddingtonThresholdScale(metric)))
	default:
		return int(activity.Distance / 1000)
	}
}

func eddingtonUnit(metric business.EddingtonMetric) string {
	if metric == business.EddingtonMetricElevation {
		return "m"
	}
	return "km"
}

func eddingtonThresholdScale(metric business.EddingtonMetric) int {
	if metric == business.EddingtonMetricElevation {
		return 100
	}
	return 1
}

func mapValues(valuesByKey map[string]int) []int {
	values := make([]int, 0, len(valuesByKey))
	for _, value := range valuesByKey {
		values = append(values, value)
	}
	return values
}

func withEddingtonProgress(eddington business.EddingtonNumber) business.EddingtonNumber {
	nextTarget := eddington.Number + 1
	qualifyingCount := 0
	if nextTarget > 0 && nextTarget <= len(eddington.List) {
		qualifyingCount = eddington.List[nextTarget-1]
	}
	missingCount := nextTarget - qualifyingCount
	if missingCount < 0 {
		missingCount = 0
	}
	eddington.NextTarget = nextTarget
	eddington.QualifyingCount = qualifyingCount
	eddington.MissingCount = missingCount
	eddington.QualifyingDays = qualifyingCount
	eddington.MissingDays = missingCount
	return eddington
}

func filterActivitiesByDateRange(activities []*strava.Activity, start time.Time, end time.Time) []*strava.Activity {
	result := make([]*strava.Activity, 0, len(activities))
	startDate := truncateDate(start)
	endDate := truncateDate(end)
	for _, activity := range activities {
		if activity == nil || len(activity.StartDateLocal) < 10 {
			continue
		}
		activityDate, err := time.Parse("2006-01-02", activity.StartDateLocal[:10])
		if err != nil {
			continue
		}
		if !activityDate.Before(startDate) && !activityDate.After(endDate) {
			result = append(result, activity)
		}
	}
	return result
}

func truncateDate(value time.Time) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
}

func computeCumulativeDistancePerYear(activityTypes ...business.ActivityType) map[string]map[string]float64 {
	log.Printf("Get cumulative distance per year for activity type %s", activityTypes)

	activitiesByYear := groupActivitiesByYear(dataqualityInfra.FilterExcludedFromStats(activityprovider.Get().GetActivitiesByYearAndActivityTypes(nil, activityTypes...)))
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

	activitiesByYear := groupActivitiesByYear(dataqualityInfra.FilterExcludedFromStats(activityprovider.Get().GetActivitiesByYearAndActivityTypes(nil, activityTypes...)))
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

	activities := dataqualityInfra.FilterExcludedFromStats(activityprovider.Get().GetActivitiesByYearAndActivityTypes(nil, activityTypes...))
	activitiesGroupedByYear := groupActivitiesByYear(activities)

	nbActivitiesByYear := make(map[string]int)
	activeDaysByYear := make(map[string]int)
	consistencyByYear := make(map[string]float64)
	movingTimeByYear := make(map[string]int)
	totalDistanceByYear := make(map[string]float64)
	averageDistanceByYear := make(map[string]float64)
	maxDistanceByYear := make(map[string]float64)
	totalElevationByYear := make(map[string]int)
	averageElevationByYear := make(map[string]int)
	maxElevationByYear := make(map[string]int)
	elevationEfficiencyByYear := make(map[string]float64)
	averageSpeedByYear := make(map[string]float64)
	maxSpeedByYear := make(map[string]float64)
	averageHeartRateByYear := make(map[string]int)
	maxHeartRateByYear := make(map[string]float64)
	averageWattsByYear := make(map[string]float64)
	maxWattsByYear := make(map[string]float64)
	deviceAverageWattsByYear := make(map[string]float64)
	deviceMaxWattsByYear := make(map[string]float64)

	for year, yearActivities := range activitiesGroupedByYear {
		nbActivitiesByYear[year] = len(yearActivities)
		activeDaysByYear[year] = countActiveDays(yearActivities)
		consistencyByYear[year] = computeConsistencyByYear(year, activeDaysByYear[year])
		movingTimeByYear[year] = sumMovingTime(yearActivities)
		totalDistanceByYear[year] = sumDistance(yearActivities)
		averageDistanceByYear[year] = averageDistance(yearActivities)
		maxDistanceByYear[year] = maxDistance(yearActivities)
		totalElevationByYear[year] = sumElevation(yearActivities)
		averageElevationByYear[year] = averageElevation(yearActivities)
		maxElevationByYear[year] = maxElevation(yearActivities)
		if totalDistanceByYear[year] > 0 {
			elevationEfficiencyByYear[year] = (float64(totalElevationByYear[year]) / totalDistanceByYear[year]) * 10.0
		}
		averageSpeedByYear[year] = averageSpeed(yearActivities)
		maxSpeedByYear[year] = maxSpeed(yearActivities)
		averageHeartRateByYear[year] = averageHeartRate(yearActivities)
		maxHeartRateByYear[year] = maxHeartRate(yearActivities)
		averageWattsByYear[year] = averageWatts(yearActivities)
		maxWattsByYear[year] = maxWatts(yearActivities)
		if deviceAverageWatts := averageDeviceWatts(yearActivities); deviceAverageWatts > 0 {
			deviceAverageWattsByYear[year] = deviceAverageWatts
		}
		if deviceMaxWatts := maxDeviceWatts(yearActivities); deviceMaxWatts > 0 {
			deviceMaxWattsByYear[year] = deviceMaxWatts
		}
	}

	return business.DashboardData{
		NbActivities:              nbActivitiesByYear,
		ActiveDaysByYear:          activeDaysByYear,
		ConsistencyByYear:         consistencyByYear,
		MovingTimeByYear:          movingTimeByYear,
		TotalDistanceByYear:       totalDistanceByYear,
		AverageDistanceByYear:     averageDistanceByYear,
		MaxDistanceByYear:         maxDistanceByYear,
		TotalElevationByYear:      totalElevationByYear,
		AverageElevationByYear:    averageElevationByYear,
		MaxElevationByYear:        maxElevationByYear,
		ElevationEfficiencyByYear: elevationEfficiencyByYear,
		AverageSpeedByYear:        averageSpeedByYear,
		MaxSpeedByYear:            maxSpeedByYear,
		AverageHeartRateByYear:    averageHeartRateByYear,
		MaxHeartRateByYear:        maxHeartRateByYear,
		AverageWattsByYear:        averageWattsByYear,
		MaxWattsByYear:            maxWattsByYear,
		DeviceAverageWattsByYear:  deviceAverageWattsByYear,
		DeviceMaxWattsByYear:      deviceMaxWattsByYear,
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

func countActiveDays(activities []*strava.Activity) int {
	uniqueDays := make(map[string]struct{})
	for _, activity := range activities {
		if len(activity.StartDateLocal) < 10 {
			continue
		}
		dayKey := activity.StartDateLocal[:10]
		uniqueDays[dayKey] = struct{}{}
	}
	return len(uniqueDays)
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

func averageDeviceWatts(activities []*strava.Activity) float64 {
	var sum float64
	var count float64
	for _, activity := range activities {
		if activity.DeviceWatts && activity.AverageWatts > 0 {
			sum += activity.AverageWatts
			count++
		}
	}
	if count == 0 {
		return 0
	}
	return sum / count
}

func maxDeviceWatts(activities []*strava.Activity) float64 {
	var maxWattsValue float64
	for _, activity := range activities {
		if activity.DeviceWatts && activity.AverageWatts > maxWattsValue {
			maxWattsValue = activity.AverageWatts
		}
	}
	return maxWattsValue
}

func sumMovingTime(activities []*strava.Activity) int {
	var sum int
	for _, activity := range activities {
		movingTime := activity.MovingTime
		if movingTime <= 0 {
			movingTime = activity.ElapsedTime
		}
		sum += movingTime
	}
	return sum
}

func computeConsistencyByYear(year string, activeDays int) float64 {
	yearNumber, err := strconv.Atoi(year)
	if err != nil || activeDays <= 0 {
		return 0
	}
	now := time.Now()
	daysScope := daysInScopeForYear(yearNumber, now)
	if daysScope <= 0 {
		return 0
	}
	return math.Round((float64(activeDays)/float64(daysScope))*1000) / 10
}

func daysInScopeForYear(year int, now time.Time) int {
	if year == now.Year() {
		return now.YearDay()
	}
	return daysInYear(year)
}

func daysInYear(year int) int {
	if isLeapYear(year) {
		return 366
	}
	return 365
}

func computeActivityHeatmap(activityTypes ...business.ActivityType) map[string]map[string]dashboardDomain.ActivityHeatmapDay {
	log.Printf("Get activity heatmap for activity type %s", activityTypes)

	activitiesByYear := groupActivitiesByYear(dataqualityInfra.FilterExcludedFromStats(activityprovider.Get().GetActivitiesByYearAndActivityTypes(nil, activityTypes...)))
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
