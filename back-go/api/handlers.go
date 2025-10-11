package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"mystravastats/api/dto"
	"mystravastats/domain/business"
	services2 "mystravastats/internal/services"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
)

// getAthlete godoc
// @Summary Get athlete information
// @Description Returns the current athlete information
// @Tags athlete
// @Produce json
// @Success 200 {object} dto.AthleteDto
// @Failure 500 {string} string "Internal server error"
// @Router /api/athletes/me [get]
func getAthlete(w http.ResponseWriter, _ *http.Request) {
	athlete := services2.FetchAthlete()
	athleteDto := dto.ToAthleteDto(athlete)

	if err := writeJSON(w, http.StatusOK, athleteDto); err != nil {
		log.Printf("failed to write athlete response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// getActivitiesByActivityType godoc
// @Summary List activities by type
// @Description Returns activities filtered by year and type
// @Tags activities
// @Produce json
// @Param year query int false "Year"
// @Param activityType query string true "Activity type"
// @Success 200 {array} dto.ActivityDto
// @Failure 400 {string} string "Invalid parameters"
// @Failure 500 {string} string "Internal server error"
// @Router /api/activities [get]
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

	activitiesByActivityTypeAndYear := services2.RetrieveActivitiesByYearAndActivityTypes(year, activityTypes...)
	activitiesDto := make([]dto.ActivityDto, len(activitiesByActivityTypeAndYear))
	for i, activity := range activitiesByActivityTypeAndYear {
		activitiesDto[i] = dto.ToActivityDto(*activity)
	}

	if err := writeJSON(w, http.StatusOK, activitiesDto); err != nil {
		log.Printf("failed to write activities response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// getDetailedActivity godoc
// @Summary Get activity details
// @Description Returns detailed information about a specific activity
// @Tags activities
// @Produce json
// @Param activityId path int true "Activity ID"
// @Success 200 {object} dto.DetailedActivityDto
// @Failure 400 {string} string "Invalid activity ID"
// @Failure 404 {string} string "Activity not found"
// @Failure 500 {string} string "Internal server error"
// @Router /api/activities/{activityId} [get]
func getDetailedActivity(writer http.ResponseWriter, request *http.Request) {
	vars := mux.Vars(request)
	activityId, err := strconv.ParseInt(vars["activityId"], 10, 64)
	if err != nil {
		http.Error(writer, "Invalid activityId", http.StatusBadRequest)
		return
	}
	activity, err := services2.RetrieveDetailedActivity(activityId)
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

// getExportCSV godoc
// @Summary Export activities to CSV
// @Description Generates and returns a CSV file containing activities filtered by year and type
// @Tags activities
// @Produce text/csv
// @Param year query int false "Year"
// @Param activityType query string true "Activity type"
// @Success 200 {file} file "CSV file of activities"
// @Failure 400 {string} string "Invalid parameters"
// @Failure 500 {string} string "Internal server error"
// @Router /api/activities/csv [get]
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
	csvData := services2.ExportCSV(year, activityTypes...)

	writer.Header().Set("Content-Type", "text/csv")
	writer.Header().Set("Content-Disposition", "attachment; filename=\"activities.csv\"")
	writer.WriteHeader(http.StatusOK)
	if _, err := writer.Write([]byte(csvData)); err != nil {
		log.Printf("failed to write CSV response: %v", err)
		http.Error(writer, "Failed to write CSV response", http.StatusInternalServerError)
		return
	}
	log.Println("CSV export successful")
	if _, err = writer.Write([]byte(csvData)); err != nil {
		log.Printf("failed to write CSV response: %v", err)
	}
	if _, err := writer.Write([]byte(csvData)); err != nil {
		log.Printf("failed to write CSV response: %v", err)
		http.Error(writer, "Failed to write CSV response", http.StatusInternalServerError)
		return
	}
}

// getStatisticsByActivityType godoc
// @Summary Get statistics by activity type
// @Description Returns calculated statistics for a given year and activity types
// @Tags statistics
// @Produce json
// @Param year query int false "Year"
// @Param activityType query string true "Activity type"
// @Success 200 {array} dto.StatisticDto
// @Failure 400 {string} string "Invalid parameters"
// @Failure 500 {string} string "Internal server error"
// @Router /api/statistics [get]
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

	statisticsByActivityTypeAndYear := services2.FetchStatisticsByActivityTypeAndYear(year, activityTypes...)
	statisticsDto := make([]dto.StatisticDto, len(statisticsByActivityTypeAndYear))
	for i, statistic := range statisticsByActivityTypeAndYear {
		statisticsDto[i] = dto.ToStatisticDto(statistic)
	}

	if err := writeJSON(w, http.StatusOK, statisticsDto); err != nil {
		log.Printf("failed to write statistics response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// getMapsGPX godoc
// @Summary Get GPX data for maps
// @Description Returns GPX data from activities for map display
// @Tags maps
// @Produce json
// @Param year query int false "Year"
// @Param activityType query string true "Activity type"
// @Success 200 {object} object "GPX data"
// @Failure 400 {string} string "Invalid parameters"
// @Failure 500 {string} string "Internal server error"
// @Router /api/maps/gpx [get]
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

	gpx := services2.RetrieveGPXByYearAndActivityTypes(year, activityTypes...)

	if err := writeJSON(w, http.StatusOK, gpx); err != nil {
		log.Printf("failed to write gpx response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// getChartsDistanceByPeriod godoc
// @Summary Get distance data by period
// @Description Returns distance data aggregated by period for charts
// @Tags charts
// @Produce json
// @Param year query int false "Year"
// @Param activityType query string true "Activity type"
// @Param period query string false "Aggregation period"
// @Success 200 {object} object "Distance data by period"
// @Failure 400 {string} string "Invalid parameters"
// @Failure 500 {string} string "Internal server error"
// @Router /api/charts/distance-by-period [get]
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

	distanceByPeriod := services2.FetchChartsDistanceByPeriod(year, period, activityTypes...)

	if err := writeJSON(w, http.StatusOK, distanceByPeriod); err != nil {
		log.Printf("failed to write distance chart response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// getChartsElevationByPeriod godoc
// @Summary Get elevation data by period
// @Description Returns elevation data aggregated by period for charts
// @Tags charts
// @Produce json
// @Param year query int false "Year"
// @Param activityType query string true "Activity type"
// @Param period query string false "Aggregation period"
// @Success 200 {object} object "Elevation data by period"
// @Failure 400 {string} string "Invalid parameters"
// @Failure 500 {string} string "Internal server error"
// @Router /api/charts/elevation-by-period [get]
func getChartsElevationByPeriod(writer http.ResponseWriter, request *http.Request) {
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
	period := getPeriodParam(request)

	elevationByPeriod := services2.FetchChartsElevationByPeriod(year, period, activityTypes...)

	if err := writeJSON(writer, http.StatusOK, elevationByPeriod); err != nil {
		log.Printf("failed to write elevation chart response: %v", err)
		http.Error(writer, "Failed to encode response", http.StatusInternalServerError)
	}
}

// getChartsAverageSpeedByPeriod godoc
// @Summary Get average speed data by period
// @Description Returns average speed data aggregated by period for charts
// @Tags charts
// @Produce json
// @Param year query int false "Year"
// @Param activityType query string true "Activity type"
// @Param period query string false "Aggregation period"
// @Success 200 {object} object "Average speed data by period"
// @Failure 400 {string} string "Invalid parameters"
// @Failure 500 {string} string "Internal server error"
// @Router /api/charts/average-speed-by-period [get]
func getChartsAverageSpeedByPeriod(writer http.ResponseWriter, request *http.Request) {
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
	period := getPeriodParam(request)

	averageSpeedByPeriod := services2.FetchChartsAverageSpeedByPeriod(year, period, activityTypes...)

	if err := writeJSON(writer, http.StatusOK, averageSpeedByPeriod); err != nil {
		log.Printf("failed to write average speed chart response: %v", err)
		http.Error(writer, "Failed to encode response", http.StatusInternalServerError)
	}
}

// getChartsAverageCadenceByPeriod godoc
// @Summary Get average cadence data by period
// @Description Returns average cadence data aggregated by period for charts
// @Tags charts
// @Produce json
// @Param year query int false "Year"
// @Param activityType query string true "Activity type"
// @Param period query string false "Aggregation period"
// @Success 200 {object} object "Average cadence data by period"
// @Failure 400 {string} string "Invalid parameters"
// @Failure 500 {string} string "Internal server error"
// @Router /api/charts/average-cadence-by-period [get]
func getChartsAverageCadenceByPeriod(writer http.ResponseWriter, request *http.Request) {
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
	period := getPeriodParam(request)

	averageCadenceByPeriod := services2.FetchChartsAverageCadenceByPeriod(year, period, activityTypes...)

	if err := writeJSON(writer, http.StatusOK, averageCadenceByPeriod); err != nil {
		log.Printf("failed to write average cadence chart response: %v", err)
		http.Error(writer, "Failed to encode response", http.StatusInternalServerError)
	}
}

// getDashboard godoc
// @Summary Get dashboard data
// @Description Returns main data for dashboard display
// @Tags dashboard
// @Produce json
// @Param activityType query string true "Activity type"
// @Success 200 {object} dto.DashboardDataDto
// @Failure 400 {string} string "Invalid parameters"
// @Failure 500 {string} string "Internal server error"
// @Router /api/dashboard [get]
func getDashboard(w http.ResponseWriter, r *http.Request) {
	activityTypes, err := getActivityTypeParam(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	dashboardData := services2.FetchDashboardData(activityTypes...)
	dashboardDataDto := dto.ToDashboardDataDto(dashboardData)

	if err := writeJSON(w, http.StatusOK, dashboardDataDto); err != nil {
		log.Printf("failed to write dashboard response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// getDashboardCumulativeDataByYear godoc
// @Summary Get cumulative data by year
// @Description Returns cumulative distance and elevation data by year
// @Tags dashboard
// @Produce json
// @Param activityType query string true "Activity type"
// @Success 200 {object} dto.CumulativeDataPerYearDto
// @Failure 400 {string} string "Invalid parameters"
// @Failure 500 {string} string "Internal server error"
// @Router /api/dashboard/cumulative-data-per-year [get]
func getDashboardCumulativeDataByYear(w http.ResponseWriter, r *http.Request) {
	activityTypes, err := getActivityTypeParam(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	cumulativeDistancePerYear := services2.GetCumulativeDistancePerYear(activityTypes...)
	cumulativeElevationPerYear := services2.GetCumulativeElevationPerYear(activityTypes...)

	cumulativeDataDto := dto.CumulativeDataPerYearDto{
		Distance:  cumulativeDistancePerYear,
		Elevation: cumulativeElevationPerYear,
	}

	if err := writeJSON(w, http.StatusOK, cumulativeDataDto); err != nil {
		log.Printf("failed to write cumulative dashboard response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// getDashboardEddingtonNumber godoc
// @Summary Get Eddington number
// @Description Returns the Eddington number and associated list
// @Tags dashboard
// @Produce json
// @Param activityType query string true "Activity type"
// @Success 200 {object} dto.EddingtonNumberDto
// @Failure 400 {string} string "Invalid parameters"
// @Failure 500 {string} string "Internal server error"
// @Router /api/dashboard/eddington-number [get]
func getDashboardEddingtonNumber(w http.ResponseWriter, r *http.Request) {
	activityTypes, err := getActivityTypeParam(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	edNum := services2.FetchEddingtonNumber(activityTypes...)
	edNumDto := dto.EddingtonNumberDto{
		EddingtonNumber: edNum.Number,
		EddingtonList:   edNum.List,
	}

	if err := writeJSON(w, http.StatusOK, edNumDto); err != nil {
		log.Printf("failed to write eddington response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// getBadges godoc
// @Summary Get badges
// @Description Returns badges earned or in progress for a given year and activity types
// @Tags badges
// @Produce json
// @Param year query int false "Year"
// @Param activityType query string true "Activity type"
// @Param badgeSet query string false "Badge set (GENERAL, FAMOUS)"
// @Success 200 {array} dto.BadgeCheckResultDto
// @Failure 400 {string} string "Invalid parameters"
// @Failure 500 {string} string "Internal server error"
// @Router /api/badges [get]
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
		badges = services2.GetGeneralBadges(year, activityTypes...)
	case business.FAMOUS:
		badges = services2.GetFamousBadges(year, activityTypes...)
	default:
		generalBadges := services2.GetGeneralBadges(year, activityTypes...)
		famousBadges := services2.GetFamousBadges(year, activityTypes...)
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
