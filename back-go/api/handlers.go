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
		writeInternalServerError(w, "Failed to encode athlete response")
	}
}

func getAthleteHeartRateZones(w http.ResponseWriter, _ *http.Request) {
	settings := services2.FetchHeartRateZoneSettings()
	settingsDto := dto.ToHeartRateZoneSettingsDto(settings)

	if err := writeJSON(w, http.StatusOK, settingsDto); err != nil {
		log.Printf("failed to write heart rate settings response: %v", err)
		writeInternalServerError(w, "Failed to encode heart rate settings response")
	}
}

func putAthleteHeartRateZones(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var settingsDto dto.HeartRateZoneSettingsDto
	if err := json.NewDecoder(r.Body).Decode(&settingsDto); err != nil {
		writeBadRequest(w, "Invalid request body", "heart rate zone settings payload is invalid")
		return
	}

	settings := dto.ToHeartRateZoneSettings(settingsDto)
	updatedSettings := services2.UpdateHeartRateZoneSettings(settings)
	updatedSettingsDto := dto.ToHeartRateZoneSettingsDto(updatedSettings)

	if err := writeJSON(w, http.StatusOK, updatedSettingsDto); err != nil {
		log.Printf("failed to write updated heart rate settings response: %v", err)
		writeInternalServerError(w, "Failed to encode updated heart rate settings response")
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
		writeBadRequest(w, "Invalid request parameters", err.Error())
		return
	}
	activityTypes, err := getActivityTypeParam(r)
	if err != nil {
		writeBadRequest(w, "Invalid request parameters", err.Error())
		return
	}

	activitiesByActivityTypeAndYear := services2.RetrieveActivitiesByYearAndActivityTypes(year, activityTypes...)
	activitiesDto := make([]dto.ActivityDto, len(activitiesByActivityTypeAndYear))
	for i, activity := range activitiesByActivityTypeAndYear {
		activitiesDto[i] = dto.ToActivityDto(*activity)
	}

	if err := writeJSON(w, http.StatusOK, activitiesDto); err != nil {
		log.Printf("failed to write activities response: %v", err)
		writeInternalServerError(w, "Failed to encode activities response")
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
		writeBadRequest(writer, "Invalid request parameters", "invalid activityId")
		return
	}
	activity, err := services2.RetrieveDetailedActivity(activityId)
	if err != nil {
		writeNotFound(writer, "Resource not found", fmt.Sprintf("Activity %d not found", activityId))
		return
	}

	detailedActivityDto := dto.ToDetailedActivityDto(activity)

	if err := writeJSON(writer, http.StatusOK, detailedActivityDto); err != nil {
		log.Printf("failed to write detailed activity response: %v", err)
		writeInternalServerError(writer, "Failed to encode detailed activity response")
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
		writeBadRequest(writer, "Invalid request parameters", err.Error())
		return
	}
	activityTypes, err := getActivityTypeParam(request)
	if err != nil {
		writeBadRequest(writer, "Invalid request parameters", err.Error())
		return
	}
	csvData := services2.ExportCSV(year, activityTypes...)

	writer.Header().Set("Content-Type", "text/csv")
	writer.Header().Set("Content-Disposition", "attachment; filename=\"activities.csv\"")
	writer.WriteHeader(http.StatusOK)
	if _, err := writer.Write([]byte(csvData)); err != nil {
		log.Printf("failed to write CSV response: %v", err)
		return
	}
	log.Println("CSV export successful")
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
		writeBadRequest(w, "Invalid request parameters", err.Error())
		return
	}
	activityTypes, err := getActivityTypeParam(r)
	if err != nil {
		writeBadRequest(w, "Invalid request parameters", err.Error())
		return
	}

	statisticsByActivityTypeAndYear := services2.FetchStatisticsByActivityTypeAndYear(year, activityTypes...)
	statisticsDto := make([]dto.StatisticDto, len(statisticsByActivityTypeAndYear))
	for i, statistic := range statisticsByActivityTypeAndYear {
		statisticsDto[i] = dto.ToStatisticDto(statistic)
	}

	if err := writeJSON(w, http.StatusOK, statisticsDto); err != nil {
		log.Printf("failed to write statistics response: %v", err)
		writeInternalServerError(w, "Failed to encode statistics response")
	}
}

// getPersonalRecordsTimelineByActivityType godoc
// @Summary Get personal records timeline by activity type
// @Description Returns chronological personal record events for a given year and activity types
// @Tags statistics
// @Produce json
// @Param year query int false "Year"
// @Param activityType query string true "Activity type"
// @Param metric query string false "Metric key"
// @Success 200 {array} dto.PersonalRecordTimelineDto
// @Failure 400 {string} string "Invalid parameters"
// @Failure 500 {string} string "Internal server error"
// @Router /api/statistics/personal-records-timeline [get]
func getPersonalRecordsTimelineByActivityType(w http.ResponseWriter, r *http.Request) {
	year, err := getYearParam(r)
	if err != nil {
		writeBadRequest(w, "Invalid request parameters", err.Error())
		return
	}
	activityTypes, err := getActivityTypeParam(r)
	if err != nil {
		writeBadRequest(w, "Invalid request parameters", err.Error())
		return
	}
	metric := getMetricParam(r)

	timeline := services2.FetchPersonalRecordsTimelineByActivityTypeAndYear(year, metric, activityTypes...)
	timelineDto := make([]dto.PersonalRecordTimelineDto, len(timeline))
	for i, entry := range timeline {
		timelineDto[i] = dto.ToPersonalRecordTimelineDto(entry)
	}

	if err := writeJSON(w, http.StatusOK, timelineDto); err != nil {
		log.Printf("failed to write personal records timeline response: %v", err)
		writeInternalServerError(w, "Failed to encode personal records timeline response")
	}
}

func getHeartRateZoneAnalysisByActivityType(w http.ResponseWriter, r *http.Request) {
	year, err := getYearParam(r)
	if err != nil {
		writeBadRequest(w, "Invalid request parameters", err.Error())
		return
	}
	activityTypes, err := getActivityTypeParam(r)
	if err != nil {
		writeBadRequest(w, "Invalid request parameters", err.Error())
		return
	}

	analysis := services2.FetchHeartRateZoneAnalysisByActivityTypeAndYear(year, activityTypes...)
	analysisDto := dto.ToHeartRateZoneAnalysisDto(analysis)

	if err := writeJSON(w, http.StatusOK, analysisDto); err != nil {
		log.Printf("failed to write heart rate zone analysis response: %v", err)
		writeInternalServerError(w, "Failed to encode heart rate zone analysis response")
	}
}

// getSegmentClimbProgressionByActivityType godoc
// @Summary Get segment and climb progression
// @Description Returns progression for favorite segments and climbs (attempts, PR progression, consistency, pacing and trends)
// @Tags statistics
// @Produce json
// @Param year query int false "Year"
// @Param activityType query string true "Activity type"
// @Param metric query string false "Metric (TIME or SPEED)"
// @Param targetType query string false "Target type filter (ALL, SEGMENT, CLIMB)"
// @Param targetId query int false "Target id"
// @Success 200 {object} dto.SegmentClimbProgressionDto
// @Failure 400 {string} string "Invalid parameters"
// @Failure 500 {string} string "Internal server error"
// @Router /api/statistics/segment-climb-progression [get]
func getSegmentClimbProgressionByActivityType(w http.ResponseWriter, r *http.Request) {
	year, err := getYearParam(r)
	if err != nil {
		writeBadRequest(w, "Invalid request parameters", err.Error())
		return
	}
	activityTypes, err := getActivityTypeParam(r)
	if err != nil {
		writeBadRequest(w, "Invalid request parameters", err.Error())
		return
	}
	metric := getMetricParam(r)
	targetType := getTargetTypeParam(r)
	targetId, err := getTargetIDParam(r)
	if err != nil {
		writeBadRequest(w, "Invalid request parameters", err.Error())
		return
	}

	log.Printf(
		"Segment progression request: year=%v metric=%v targetType=%v targetId=%v activityTypes=%v",
		year,
		metric,
		targetType,
		targetId,
		activityTypes,
	)

	progression := services2.FetchSegmentClimbProgressionByActivityTypeAndYear(year, metric, targetType, targetId, activityTypes...)
	progressionDto := dto.ToSegmentClimbProgressionDto(progression)

	if err := writeJSON(w, http.StatusOK, progressionDto); err != nil {
		log.Printf("failed to write segment/climb progression response: %v", err)
		writeInternalServerError(w, "Failed to encode segment/climb progression response")
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
		writeBadRequest(w, "Invalid request parameters", err.Error())
		return
	}
	activityTypes, err := getActivityTypeParam(r)
	if err != nil {
		writeBadRequest(w, "Invalid request parameters", err.Error())
		return
	}

	gpx := services2.RetrieveGPXByYearAndActivityTypes(year, activityTypes...)

	if err := writeJSON(w, http.StatusOK, gpx); err != nil {
		log.Printf("failed to write gpx response: %v", err)
		writeInternalServerError(w, "Failed to encode gpx response")
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
		writeBadRequest(w, "Invalid request parameters", err.Error())
		return
	}
	if year == nil {
		writeBadRequest(w, "Invalid request parameters", "year is required")
		return
	}
	activityTypes, err := getActivityTypeParam(r)
	if err != nil {
		writeBadRequest(w, "Invalid request parameters", err.Error())
		return
	}
	period, err := getPeriodParam(r)
	if err != nil {
		writeBadRequest(w, "Invalid request parameters", err.Error())
		return
	}

	distanceByPeriod := services2.FetchChartsDistanceByPeriod(year, period, activityTypes...)

	if err := writeJSON(w, http.StatusOK, distanceByPeriod); err != nil {
		log.Printf("failed to write distance chart response: %v", err)
		writeInternalServerError(w, "Failed to encode distance chart response")
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
		writeBadRequest(writer, "Invalid request parameters", err.Error())
		return
	}
	if year == nil {
		writeBadRequest(writer, "Invalid request parameters", "year is required")
		return
	}
	activityTypes, err := getActivityTypeParam(request)
	if err != nil {
		writeBadRequest(writer, "Invalid request parameters", err.Error())
		return
	}
	period, err := getPeriodParam(request)
	if err != nil {
		writeBadRequest(writer, "Invalid request parameters", err.Error())
		return
	}

	elevationByPeriod := services2.FetchChartsElevationByPeriod(year, period, activityTypes...)

	if err := writeJSON(writer, http.StatusOK, elevationByPeriod); err != nil {
		log.Printf("failed to write elevation chart response: %v", err)
		writeInternalServerError(writer, "Failed to encode elevation chart response")
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
		writeBadRequest(writer, "Invalid request parameters", err.Error())
		return
	}
	if year == nil {
		writeBadRequest(writer, "Invalid request parameters", "year is required")
		return
	}
	activityTypes, err := getActivityTypeParam(request)
	if err != nil {
		writeBadRequest(writer, "Invalid request parameters", err.Error())
		return
	}
	period, err := getPeriodParam(request)
	if err != nil {
		writeBadRequest(writer, "Invalid request parameters", err.Error())
		return
	}

	averageSpeedByPeriod := services2.FetchChartsAverageSpeedByPeriod(year, period, activityTypes...)

	if err := writeJSON(writer, http.StatusOK, averageSpeedByPeriod); err != nil {
		log.Printf("failed to write average speed chart response: %v", err)
		writeInternalServerError(writer, "Failed to encode average speed chart response")
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
		writeBadRequest(writer, "Invalid request parameters", err.Error())
		return
	}
	if year == nil {
		writeBadRequest(writer, "Invalid request parameters", "year is required")
		return
	}
	activityTypes, err := getActivityTypeParam(request)
	if err != nil {
		writeBadRequest(writer, "Invalid request parameters", err.Error())
		return
	}
	period, err := getPeriodParam(request)
	if err != nil {
		writeBadRequest(writer, "Invalid request parameters", err.Error())
		return
	}

	averageCadenceByPeriod := services2.FetchChartsAverageCadenceByPeriod(year, period, activityTypes...)

	if err := writeJSON(writer, http.StatusOK, averageCadenceByPeriod); err != nil {
		log.Printf("failed to write average cadence chart response: %v", err)
		writeInternalServerError(writer, "Failed to encode average cadence chart response")
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
		writeBadRequest(w, "Invalid request parameters", err.Error())
		return
	}

	dashboardData := services2.FetchDashboardData(activityTypes...)
	dashboardDataDto := dto.ToDashboardDataDto(dashboardData)

	if err := writeJSON(w, http.StatusOK, dashboardDataDto); err != nil {
		log.Printf("failed to write dashboard response: %v", err)
		writeInternalServerError(w, "Failed to encode dashboard response")
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
		writeBadRequest(w, "Invalid request parameters", err.Error())
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
		writeInternalServerError(w, "Failed to encode cumulative dashboard response")
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
		writeBadRequest(w, "Invalid request parameters", err.Error())
		return
	}

	edNum := services2.FetchEddingtonNumber(activityTypes...)
	edNumDto := dto.EddingtonNumberDto{
		EddingtonNumber: edNum.Number,
		EddingtonList:   edNum.List,
	}

	if err := writeJSON(w, http.StatusOK, edNumDto); err != nil {
		log.Printf("failed to write eddington response: %v", err)
		writeInternalServerError(w, "Failed to encode eddington response")
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
		writeBadRequest(w, "Invalid request parameters", err.Error())
		return
	}
	activityTypes, err := getActivityTypeParam(r)
	if err != nil {
		writeBadRequest(w, "Invalid request parameters", err.Error())
		return
	}
	badgeSet, err := getBadgeSetParam(r)
	if err != nil {
		writeBadRequest(w, "Invalid request parameters", err.Error())
		return
	}

	var badges []business.BadgeCheckResult
	switch {
	case badgeSet != nil && *badgeSet == business.GENERAL:
		badges = services2.GetGeneralBadges(year, activityTypes...)
	case badgeSet != nil && *badgeSet == business.FAMOUS:
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
		writeInternalServerError(w, "Failed to encode badges response")
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

func getPeriodParam(r *http.Request) (business.Period, error) {
	periodParam := r.URL.Query().Get("period")
	if periodParam == "" {
		return "", fmt.Errorf("period is required")
	}

	period := business.Period(periodParam)
	switch period {
	case business.PeriodDays, business.PeriodWeeks, business.PeriodMonths:
		return period, nil
	default:
		return "", fmt.Errorf("invalid period: %q", periodParam)
	}
}

func getBadgeSetParam(r *http.Request) (*business.BadgeSetEnum, error) {
	value := strings.TrimSpace(r.URL.Query().Get("badgeSet"))
	if value == "" {
		return nil, nil
	}

	badgeSet := business.BadgeSetEnum(value)
	switch badgeSet {
	case business.GENERAL, business.FAMOUS:
		return &badgeSet, nil
	default:
		return nil, fmt.Errorf("invalid badgeSet: %q", value)
	}
}

func getMetricParam(r *http.Request) *string {
	metric := strings.TrimSpace(r.URL.Query().Get("metric"))
	if metric == "" {
		return nil
	}
	return &metric
}

func getTargetTypeParam(r *http.Request) *string {
	targetType := strings.TrimSpace(r.URL.Query().Get("targetType"))
	if targetType == "" {
		return nil
	}
	return &targetType
}

func getTargetIDParam(r *http.Request) (*int64, error) {
	targetID := strings.TrimSpace(r.URL.Query().Get("targetId"))
	if targetID == "" {
		return nil, nil
	}

	id, err := strconv.ParseInt(targetID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid targetId: %q", targetID)
	}

	return &id, nil
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

func writeBadRequest(w http.ResponseWriter, message string, description string) {
	writeAPIError(w, http.StatusBadRequest, message, description)
}

func writeNotFound(w http.ResponseWriter, message string, description string) {
	writeAPIError(w, http.StatusNotFound, message, description)
}

func writeInternalServerError(w http.ResponseWriter, description string) {
	writeAPIError(w, http.StatusInternalServerError, "Internal server error", description)
}

func writeAPIError(w http.ResponseWriter, status int, message string, description string) {
	if err := writeJSON(w, status, dto.ErrorResponseMessageDto{
		Message:     message,
		Description: description,
		Code:        1,
	}); err != nil {
		log.Printf("failed to write API error response: %v", err)
	}
}
