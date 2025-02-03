package api

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"mystravastats/api/dto"
	"mystravastats/domain"
	"mystravastats/domain/business"
	"mystravastats/domain/strava"
	"net/http"
	"strconv"
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

func getDetailedActivity(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Content-Type", "application/json; charset=UTF-8")
	writer.WriteHeader(http.StatusOK)

	vars := mux.Vars(request)
	activityId, err := strconv.ParseInt(vars["activityId"], 10, 64)
	if err != nil {
		http.Error(writer, "Invalid activityId", http.StatusBadRequest)
		return
	}
	activity, err := domain.RetrieveDetailedActivity(activityId)
	if err != nil {
		http.Error(writer, fmt.Sprintf("Activity %d not found", activityId), http.StatusNotFound)
		return
	}

	activityDto := toDetailedActivityDto(*activity)

	if err := json.NewEncoder(writer).Encode(activityDto); err != nil {
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

	var badgeSet strava.BadgeSetEnum
	if badgeSetParam != "" {
		badgeSet = strava.BadgeSetEnum(badgeSetParam)
	}

	var badges []strava.BadgeCheckResult
	switch badgeSet {
	case strava.GENERAL:
		badges = domain.GetGeneralBadges(activityType, year)
	case strava.FAMOUS:
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
