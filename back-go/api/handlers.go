package api

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"mystravastats/api/dto"
	"mystravastats/domain/business"
	"mystravastats/domain/services"
	"net/http"
	"strconv"
)

func getAthlete(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)

	athlete := services.FetchAthlete()

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

	activitiesByActivityTypeAndYear := services.RetrieveActivitiesByActivityTypeAndYear(activityType, year)
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
	activity, err := services.RetrieveDetailedActivity(activityId)
	if err != nil {
		http.Error(writer, fmt.Sprintf("Activity %d not found", activityId), http.StatusNotFound)
		return
	}

	detailedActivityDto := toDetailedActivityDto(activity)

	if err := json.NewEncoder(writer).Encode(detailedActivityDto); err != nil {
		panic(err)
	}
}

func getStatisticsByActivityType(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)

	year := getYearParam(r)
	activityType := getActivityTypeParam(r)

	statisticsByActivityTypeAndYear := services.FetchStatisticsByActivityTypeAndYear(activityType, year)
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

	gpx := services.RetrieveGPXByActivityTypeAndYear(activityType, year)

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

	distanceByPeriod := services.FetchChartsDistanceByPeriod(activityType, year, period)

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

	elevationByPeriod := services.FetchChartsElevationByPeriod(activityType, year, period)

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

	averageSpeedByPeriod := services.FetchChartsAverageSpeedByPeriod(activityType, year, period)

	if err := json.NewEncoder(w).Encode(averageSpeedByPeriod); err != nil {
		panic(err)
	}
}

func getDashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)

	activityType := getActivityTypeParam(r)

	dashboardData := services.FetchDashboardData(activityType)
	dashboardDataDto := toDashboardDataDto(dashboardData)

	if err := json.NewEncoder(w).Encode(dashboardDataDto); err != nil {
		panic(err)
	}

}

func getDashboardCumulativeDataByYear(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)

	activityType := getActivityTypeParam(r)

	cumulativeDistancePerYear := services.GetCumulativeDistancePerYear(activityType)
	cumulativeElevationPerYear := services.GetCumulativeElevationPerYear(activityType)

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

	edNum := services.FetchEddingtonNumber(activityType)
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
		badges = services.GetGeneralBadges(activityType, year)
	case business.FAMOUS:
		badges = services.GetFamousBadges(activityType, year)
	default:
		generalBadges := services.GetGeneralBadges(activityType, year)
		famousBadges := services.GetFamousBadges(activityType, year)
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
