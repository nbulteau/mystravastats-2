package api

import (
	"encoding/json"
	"fmt"
	"mystravastats/api/dto"
	"mystravastats/domain"
	"mystravastats/domain/business"
	"mystravastats/domain/statistics"
	"mystravastats/domain/strava"
	"net/http"
	"strconv"
	"time"
)

func getAthlete(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)

	athlete := domain.FetchAthlete()

	athleteDto := toAthleteDto(athlete)

	if err := json.NewEncoder(w).Encode(athleteDto); err != nil {
		panic(err)
	}
}

func getActivitiesByActivityType(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)

	year := getYearParam(r)
	activityType := getActivityTypeParam(r)

	activitiesByActivityTypeAndYear := domain.RetrieveActivitiesByActivityTypeAndYear(activityType, year)
	activitiesDto := make([]dto.ActivityDto, len(activitiesByActivityTypeAndYear))
	for i, activity := range activitiesByActivityTypeAndYear {
		activitiesDto[i] = toActivityDto(*activity)
	}

	if err := json.NewEncoder(w).Encode(activitiesDto); err != nil {
		panic(err)
	}
}

func getStatisticsByActivityType(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)

	year := getYearParam(r)
	activityType := getActivityTypeParam(r)

	statisticsByActivityTypeAndYear := domain.FetchStatisticsByActivityTypeAndYear(activityType, year)
	statisticsDto := make([]dto.StatisticDto, len(statisticsByActivityTypeAndYear))
	for i, statistic := range statisticsByActivityTypeAndYear {
		statisticsDto[i] = toStatisticDto(statistic)
	}

	if err := json.NewEncoder(w).Encode(statisticsDto); err != nil {
		panic(err)
	}
}

func getMapsGPX(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)

	year := getYearParam(r)
	activityType := getActivityTypeParam(r)

	gpx := domain.RetrieveGPXByActivityTypeAndYear(activityType, year)

	if err := json.NewEncoder(w).Encode(gpx); err != nil {
		panic(err)
	}
}

func getChartsDistanceByPeriod(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)

	year := getYearParam(r)
	activityType := getActivityTypeParam(r)
	period := getPeriodParam(r)

	distanceByPeriod := domain.FetchChartsDistanceByPeriod(activityType, year, period)

	if err := json.NewEncoder(w).Encode(distanceByPeriod); err != nil {
		panic(err)
	}
}

func getChartsElevationByPeriod(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)

	year := getYearParam(r)
	activityType := getActivityTypeParam(r)
	period := getPeriodParam(r)

	elevationByPeriod := domain.FetchChartsElevationByPeriod(activityType, year, period)

	if err := json.NewEncoder(w).Encode(elevationByPeriod); err != nil {
		panic(err)
	}
}

func getChartsAverageSpeedByPeriod(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)

	year := getYearParam(r)
	activityType := getActivityTypeParam(r)
	period := getPeriodParam(r)

	averageSpeedByPeriod := domain.FetchChartsAverageSpeedByPeriod(activityType, year, period)

	if err := json.NewEncoder(w).Encode(averageSpeedByPeriod); err != nil {
		panic(err)
	}
}

func getDashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)

	activityType := getActivityTypeParam(r)

	dashboardData := domain.FetchDashboardData(activityType)
	dashboardDataDto := toDashboardDataDto(dashboardData)

	if err := json.NewEncoder(w).Encode(dashboardDataDto); err != nil {
		panic(err)
	}

}

func getDashboardCumulativeDataByYear(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)

	activityType := getActivityTypeParam(r)

	cumulativeDistancePerYear := domain.GetCumulativeDistancePerYear(activityType)
	cumulativeElevationPerYear := domain.GetCumulativeElevationPerYear(activityType)

	cumulativeDataDto := dto.CumulativeDataPerYearDto{
		Distance:  cumulativeDistancePerYear,
		Elevation: cumulativeElevationPerYear,
	}

	if err := json.NewEncoder(w).Encode(cumulativeDataDto); err != nil {
		panic(err)
	}
}

func getDashboardEddingtonNumber(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)

	activityType := getActivityTypeParam(r)

	edNum := domain.FetchEddingtonNumber(activityType)
	edNumDto := dto.EddingtonNumberDto{
		EddingtonNumber: edNum.Number,
		EddingtonList:   edNum.List,
	}

	if err := json.NewEncoder(w).Encode(edNumDto); err != nil {
		panic(err)
	}
}

func getBadges(w http.ResponseWriter, r *http.Request) {
	year := getYearParam(r)
	activityType := getActivityTypeParam(r)
	badgeSetParam := r.URL.Query().Get("badgeSet")

	var badgeSet business.BadgeSetEnum
	if badgeSetParam != "" {
		badgeSet = business.BadgeSetEnum(badgeSetParam)
	}

	var badges []business.BadgeCheckResult
	switch badgeSet {
	case business.GENERAL:
		badges = domain.GetGeneralBadges(activityType, year)
	case business.FAMOUS:
		badges = domain.GetFamousBadges(activityType, year)
	default:
		generalBadges := domain.GetGeneralBadges(activityType, year)
		famousBadges := domain.GetFamousBadges(activityType, year)
		badges = append(generalBadges, famousBadges...)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(badges); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}

}

func getActivityTypeParam(r *http.Request) business.ActivityType {
	var activityType business.ActivityType
	activityTypeStr := r.URL.Query().Get("activityType")
	if activityTypeStr != "" {
		activityType = business.ActivityTypes[activityTypeStr]
	}
	return activityType
}

func getYearParam(r *http.Request) *int {
	var year *int
	yearStr := r.URL.Query().Get("year")
	if yearStr != "" {
		y, _ := strconv.Atoi(yearStr)
		year = &y
	}
	return year
}

func getPeriodParam(r *http.Request) business.Period {
	periodParam := r.URL.Query().Get("period")
	var period business.Period
	if periodParam != "" {
		period = business.Period(periodParam)
	}
	return period
}

func toAthleteDto(athlete strava.Athlete) dto.AthleteDto {
	return dto.AthleteDto{
		BadgeTypeId:           getIntValue(athlete.BadgeTypeId),
		City:                  getStringValue(athlete.City),
		Country:               getStringValue(athlete.Country),
		CreatedAt:             parseTime(athlete.CreatedAt),
		Firstname:             getStringValue(athlete.Firstname),
		Id:                    athlete.Id,
		Lastname:              getStringValue(athlete.Lastname),
		Premium:               getBoolValue(athlete.Premium),
		Profile:               getStringValue(athlete.Profile),
		ProfileMedium:         getStringValue(athlete.ProfileMedium),
		ResourceState:         getIntValue(athlete.ResourceState),
		Sex:                   getStringValue(athlete.Sex),
		State:                 getStringValue(athlete.State),
		Summit:                getBoolValue(athlete.Summit),
		UpdatedAt:             parseTime(athlete.UpdatedAt),
		Username:              getStringValue(athlete.Username),
		AthleteType:           getIntValue(athlete.AthleteType),
		DatePreference:        getStringValue(athlete.DatePreference),
		FollowerCount:         getIntValue(athlete.FollowerCount),
		FriendCount:           getIntValue(athlete.FriendCount),
		MeasurementPreference: getStringValue(athlete.MeasurementPreference),
		MutualFriendCount:     getIntValue(athlete.MutualFriendCount),
		Weight:                getIntValueFromFloat(athlete.Weight),
	}
}
func toStatisticDto(statistic statistics.Statistic) dto.StatisticDto {
	if statistic.Activity() != nil {
		return dto.StatisticDto{
			Label: statistic.Label(),
			Value: statistic.Value(),
			Activity: &dto.ActivityShortDto{
				ID:   statistic.Activity().Id,
				Name: statistic.Activity().Name,
				Type: statistic.Activity().Type.String(),
			},
		}
	} else {
		return dto.StatisticDto{
			Label: statistic.Label(),
			Value: statistic.Value(),
		}
	}
}

func toDashboardDataDto(data business.DashboardData) dto.DashboardDataDto {
	return dto.DashboardDataDto{
		NbActivities:           data.NbActivities,
		TotalDistanceByYear:    data.TotalDistanceByYear,
		AverageDistanceByYear:  data.AverageDistanceByYear,
		MaxDistanceByYear:      data.MaxDistanceByYear,
		TotalElevationByYear:   data.TotalElevationByYear,
		AverageElevationByYear: data.AverageElevationByYear,
		MaxElevationByYear:     data.MaxElevationByYear,
		AverageSpeedByYear:     data.AverageSpeedByYear,
		MaxSpeedByYear:         data.MaxSpeedByYear,
		AverageHeartRateByYear: data.AverageHeartRateByYear,
		MaxHeartRateByYear:     data.MaxHeartRateByYear,
		AverageWattsByYear:     data.AverageWattsByYear,
		MaxWattsByYear:         data.MaxWattsByYear,
		AverageCadence:         data.AverageCadence,
	}
}

func getIntValue(value *int) int {
	if value != nil {
		return *value
	}
	return 0
}

func getStringValue(value *string) string {
	if value != nil {
		return *value
	}
	return ""
}

func getBoolValue(value *bool) bool {
	if value != nil {
		return *value
	}
	return false
}

func parseTime(value *string) time.Time {
	if value != nil {
		parsedTime, _ := time.Parse(time.RFC3339, *value)
		return parsedTime
	}
	return time.Time{}
}

func getIntValueFromFloat(value *float64) int {
	if value != nil {
		return int(*value)
	}
	return 0
}

func toActivityDto(activity strava.Activity) dto.ActivityDto {
	bestPowerFor20Minutes := statistics.BestPowerForTime(activity, 20*60)
	bestPowerFor60Minutes := statistics.BestPowerForTime(activity, 60*60)

	var ftp string
	if bestPowerFor60Minutes != nil {
		ftp = fmt.Sprintf("%d", bestPowerFor60Minutes.AveragePower)
	} else if bestPowerFor20Minutes != nil {
		ftp = fmt.Sprintf("%d", int(float64(*bestPowerFor20Minutes.AveragePower)*0.95))
	} else {
		ftp = ""
	}

	link := ""
	if activity.UploadId != 0 {
		link = fmt.Sprintf("https://www.strava.com/activities/%d", activity.Id)
	}

	bestPowerFor20MinutesStr := ""
	if bestPowerFor20Minutes != nil {
		bestPowerFor20MinutesStr = bestPowerFor20Minutes.GetFormattedPower()
	}

	bestPowerFor60MinutesStr := ""
	if bestPowerFor60Minutes != nil {
		bestPowerFor60MinutesStr = bestPowerFor60Minutes.GetFormattedPower()
	}

	bestTimeForDistanceFor1000m := 0.0
	if bestTimeForDistance := statistics.BestActivityEffort(activity, 1000.0); bestTimeForDistance != nil {
		bestTimeForDistanceFor1000m = bestTimeForDistance.GetMSSpeed()
	}

	bestElevationForDistanceFor500m := 0.0
	if bestElevationForDistance := statistics.BestElevationEffort(activity, 500.0); bestElevationForDistance != nil {
		bestElevationForDistanceFor500m = bestElevationForDistance.GetGradient()
	}

	bestElevationForDistanceFor1000m := 0.0
	if bestElevationForDistance := statistics.BestElevationEffort(activity, 1000.0); bestElevationForDistance != nil {
		bestElevationForDistanceFor1000m = bestElevationForDistance.GetGradient()
	}

	return dto.ActivityDto{
		Id:                               activity.Id,
		Name:                             activity.Name,
		Type:                             activity.Type,
		Link:                             link,
		Distance:                         int(activity.Distance),
		ElapsedTime:                      activity.ElapsedTime,
		TotalElevationGain:               int(activity.TotalElevationGain),
		AverageSpeed:                     activity.AverageSpeed,
		BestTimeForDistanceFor1000m:      bestTimeForDistanceFor1000m,
		BestElevationForDistanceFor500m:  bestElevationForDistanceFor500m,
		BestElevationForDistanceFor1000m: bestElevationForDistanceFor1000m,
		Date:                             activity.StartDateLocal,
		AverageWatts:                     int(activity.AverageWatts),
		WeightedAverageWatts:             strconv.Itoa(activity.WeightedAverageWatts),
		BestPowerFor20Minutes:            bestPowerFor20MinutesStr,
		BestPowerFor60Minutes:            bestPowerFor60MinutesStr,
		FTP:                              ftp,
	}
}
