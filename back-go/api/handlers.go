package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"mystravastats/api/dto"
	"mystravastats/domain/business"
	"mystravastats/domain/services"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
)

func getAthlete(w http.ResponseWriter, _ *http.Request) {
	athlete := services.FetchAthlete()
	athleteDto := dto.ToAthleteDto(athlete)

	if err := writeJSON(w, http.StatusOK, athleteDto); err != nil {
		log.Printf("failed to write athlete response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func getActivitiesByActivityType(w http.ResponseWriter, r *http.Request) {
	year, err := getYearParam(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	activityTypes, err := getActivityTypeParam(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	activitiesByActivityTypeAndYear := services.RetrieveActivitiesByYearAndActivityTypes(year, activityTypes...)
	activitiesDto := make([]dto.ActivityDto, len(activitiesByActivityTypeAndYear))
	for i, activity := range activitiesByActivityTypeAndYear {
		activitiesDto[i] = dto.ToActivityDto(*activity)
	}

	if err := writeJSON(w, http.StatusOK, activitiesDto); err != nil {
		log.Printf("failed to write activities response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func getDetailedActivity(writer http.ResponseWriter, request *http.Request) {
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

	detailedActivityDto := dto.ToDetailedActivityDto(activity)

	if err := writeJSON(writer, http.StatusOK, detailedActivityDto); err != nil {
		log.Printf("failed to write detailed activity response: %v", err)
		http.Error(writer, "Failed to encode response", http.StatusInternalServerError)
	}
}

func getExportCSV(writer http.ResponseWriter, request *http.Request) {
	year, err := getYearParam(request)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}
	activityTypes, err := getActivityTypeParam(request)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}
	csvData, err := services.ExportCSV(year, activityTypes...)
	if err != nil {
		log.Printf("failed to export CSV: %v", err)
		http.Error(writer, "Failed to export CSV", http.StatusInternalServerError)
		return
	}
	writer.Header().Set("Content-Type", "text/csv")
	writer.Header().Set("Content-Disposition", "attachment; filename=\"activities.csv\"")
	writer.WriteHeader(http.StatusOK)
	if _, err := writer.Write([]byte(csvData)); err != nil {
		log.Printf("failed to write CSV response: %v", err)
		http.Error(writer, "Failed to write CSV response", http.StatusInternalServerError)
		return
	}
	log.Println("CSV export successful")
	writer.Write([]byte(csvData))
	log.Println("CSV export successful")
	writer.WriteHeader(http.StatusOK)
	if _, err := writer.Write([]byte(csvData)); err != nil {
		log.Printf("failed to write CSV response: %v", err)
		http.Error(writer, "Failed to write CSV response", http.StatusInternalServerError)
		return
	}
}

func getStatisticsByActivityType(w http.ResponseWriter, r *http.Request) {
	year, err := getYearParam(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	activityTypes, err := getActivityTypeParam(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	statisticsByActivityTypeAndYear := services.FetchStatisticsByActivityTypeAndYear(year, activityTypes...)
	statisticsDto := make([]dto.StatisticDto, len(statisticsByActivityTypeAndYear))
	for i, statistic := range statisticsByActivityTypeAndYear {
		statisticsDto[i] = dto.ToStatisticDto(statistic)
	}

	if err := writeJSON(w, http.StatusOK, statisticsDto); err != nil {
		log.Printf("failed to write statistics response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func getMapsGPX(w http.ResponseWriter, r *http.Request) {
	year, err := getYearParam(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	activityTypes, err := getActivityTypeParam(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	gpx := services.RetrieveGPXByYearAndActivityTypes(year, activityTypes...)

	if err := writeJSON(w, http.StatusOK, gpx); err != nil {
		log.Printf("failed to write gpx response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func getChartsDistanceByPeriod(w http.ResponseWriter, r *http.Request) {
	year, err := getYearParam(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	activityTypes, err := getActivityTypeParam(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	period := getPeriodParam(r)

	distanceByPeriod := services.FetchChartsDistanceByPeriod(year, period, activityTypes...)

	if err := writeJSON(w, http.StatusOK, distanceByPeriod); err != nil {
		log.Printf("failed to write distance chart response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func getChartsElevationByPeriod(w http.ResponseWriter, r *http.Request) {
	year, err := getYearParam(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	activityTypes, err := getActivityTypeParam(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	period := getPeriodParam(r)

	elevationByPeriod := services.FetchChartsElevationByPeriod(year, period, activityTypes...)

	if err := writeJSON(w, http.StatusOK, elevationByPeriod); err != nil {
		log.Printf("failed to write elevation chart response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func getChartsAverageSpeedByPeriod(w http.ResponseWriter, r *http.Request) {
	year, err := getYearParam(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	activityTypes, err := getActivityTypeParam(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	period := getPeriodParam(r)

	averageSpeedByPeriod := services.FetchChartsAverageSpeedByPeriod(year, period, activityTypes...)

	if err := writeJSON(w, http.StatusOK, averageSpeedByPeriod); err != nil {
		log.Printf("failed to write average speed chart response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func getDashboard(w http.ResponseWriter, r *http.Request) {
	activityTypes, err := getActivityTypeParam(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	dashboardData := services.FetchDashboardData(activityTypes...)
	dashboardDataDto := dto.ToDashboardDataDto(dashboardData)

	if err := writeJSON(w, http.StatusOK, dashboardDataDto); err != nil {
		log.Printf("failed to write dashboard response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func getDashboardCumulativeDataByYear(w http.ResponseWriter, r *http.Request) {
	activityTypes, err := getActivityTypeParam(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	cumulativeDistancePerYear := services.GetCumulativeDistancePerYear(activityTypes...)
	cumulativeElevationPerYear := services.GetCumulativeElevationPerYear(activityTypes...)

	cumulativeDataDto := dto.CumulativeDataPerYearDto{
		Distance:  cumulativeDistancePerYear,
		Elevation: cumulativeElevationPerYear,
	}

	if err := writeJSON(w, http.StatusOK, cumulativeDataDto); err != nil {
		log.Printf("failed to write cumulative dashboard response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func getDashboardEddingtonNumber(w http.ResponseWriter, r *http.Request) {
	activityTypes, err := getActivityTypeParam(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	edNum := services.FetchEddingtonNumber(activityTypes...)
	edNumDto := dto.EddingtonNumberDto{
		EddingtonNumber: edNum.Number,
		EddingtonList:   edNum.List,
	}

	if err := writeJSON(w, http.StatusOK, edNumDto); err != nil {
		log.Printf("failed to write eddington response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func getBadges(w http.ResponseWriter, r *http.Request) {
	year, err := getYearParam(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	activityTypes, err := getActivityTypeParam(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	badgeSetParam := r.URL.Query().Get("badgeSet")

	var badgeSet business.BadgeSetEnum
	if badgeSetParam != "" {
		badgeSet = business.BadgeSetEnum(badgeSetParam)
	}

	var badges []business.BadgeCheckResult
	switch badgeSet {
	case business.GENERAL:
		badges = services.GetGeneralBadges(year, activityTypes...)
	case business.FAMOUS:
		badges = services.GetFamousBadges(year, activityTypes...)
	default:
		generalBadges := services.GetGeneralBadges(year, activityTypes...)
		famousBadges := services.GetFamousBadges(year, activityTypes...)
		badges = append(generalBadges, famousBadges...)
	}

	badgesDto := make([]dto.BadgeCheckResultDto, len(badges))
	for i, badge := range badges {
		badgesDto[i] = dto.ToBadgeCheckResultDto(badge, activityTypes...)
	}

	if err := writeJSON(w, http.StatusOK, badgesDto); err != nil {
		log.Printf("failed to write badges response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func getActivityTypeParam(r *http.Request) ([]business.ActivityType, error) {
	activityTypeStr := r.URL.Query().Get("activityType")

	if activityTypeStr == "" {
		return nil, fmt.Errorf("activity type must not be empty")
	}

	parts := strings.Split(activityTypeStr, "_")
	activityTypes := make(map[business.ActivityType]struct{}, len(parts))

	for _, p := range parts {
		if p == "" {
			return nil, fmt.Errorf("activity type must not be empty")
		}
		t, ok := business.ActivityTypes[p]
		if !ok {
			return nil, fmt.Errorf("unknown activity type: %s", p)
		}
		activityTypes[t] = struct{}{}
	}

	types := make([]business.ActivityType, 0, len(activityTypes))
	for t := range activityTypes {
		types = append(types, t)
	}

	sort.Slice(types, func(i, j int) bool { return types[i] < types[j] })

	return types, nil
}

func getYearParam(r *http.Request) (*int, error) {
	yearStr := r.URL.Query().Get("year")
	if yearStr == "" {
		return nil, nil
	}
	y, err := strconv.Atoi(yearStr)
	if err != nil {
		return nil, fmt.Errorf("invalid year: %q", yearStr)
	}
	return &y, nil
}

func getPeriodParam(r *http.Request) business.Period {
	periodParam := r.URL.Query().Get("period")
	var period business.Period
	if periodParam != "" {
		period = business.Period(periodParam)
	}
	return period
}

func writeJSON(w http.ResponseWriter, status int, v any) error {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(true)

	if err := enc.Encode(v); err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(buf.Bytes())
	return nil
}
