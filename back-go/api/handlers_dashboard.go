package api

import (
	"encoding/json"
	"log"
	"mystravastats/api/dto"
	"net/http"
)

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
func getDashboard(writer http.ResponseWriter, request *http.Request) {
	_, activityTypes, err := parseActivityRequestParams(request)
	if err != nil {
		writeBadRequest(writer, "Invalid request parameters", err.Error())
		return
	}

	dashboardData := getContainer().getDashboardDataUseCase.Execute(activityTypes)
	dashboardDataDto := dto.ToDashboardDataDto(dashboardData)

	if err := writeJSON(writer, http.StatusOK, dashboardDataDto); err != nil {
		log.Printf("failed to write dashboard response: %v", err)
		writeInternalServerError(writer, "Failed to encode dashboard response")
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
func getDashboardCumulativeDataByYear(writer http.ResponseWriter, request *http.Request) {
	_, activityTypes, err := parseActivityRequestParams(request)
	if err != nil {
		writeBadRequest(writer, "Invalid request parameters", err.Error())
		return
	}

	cumulativeData := getContainer().getCumulativeDataPerYearUseCase.Execute(activityTypes)
	cumulativeDataDto := dto.CumulativeDataPerYearDto{
		Distance:  cumulativeData.Distance,
		Elevation: cumulativeData.Elevation,
	}

	if err := writeJSON(writer, http.StatusOK, cumulativeDataDto); err != nil {
		log.Printf("failed to write cumulative dashboard response: %v", err)
		writeInternalServerError(writer, "Failed to encode cumulative dashboard response")
	}
}

// getDashboardActivityHeatmap godoc
// @Summary Get activity heatmap data
// @Description Returns daily distance/elevation/duration and activity details per day per year for heatmap display
// @Tags dashboard
// @Produce json
// @Param activityType query string true "Activity type"
// @Success 200 {object} map[string]map[string]interface{}
// @Failure 400 {string} string "Invalid parameters"
// @Failure 500 {string} string "Internal server error"
// @Router /api/dashboard/activity-heatmap [get]
func getDashboardActivityHeatmap(writer http.ResponseWriter, request *http.Request) {
	_, activityTypes, err := parseActivityRequestParams(request)
	if err != nil {
		writeBadRequest(writer, "Invalid request parameters", err.Error())
		return
	}

	heatmap := getContainer().getActivityHeatmapUseCase.Execute(activityTypes)
	if err := writeJSON(writer, http.StatusOK, heatmap); err != nil {
		log.Printf("failed to write activity heatmap response: %v", err)
		writeInternalServerError(writer, "Failed to encode activity heatmap response")
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
func getDashboardEddingtonNumber(writer http.ResponseWriter, request *http.Request) {
	_, activityTypes, err := parseActivityRequestParams(request)
	if err != nil {
		writeBadRequest(writer, "Invalid request parameters", err.Error())
		return
	}

	edNum := getContainer().getEddingtonNumberUseCase.Execute(activityTypes)
	edNumDto := dto.EddingtonNumberDto{
		EddingtonNumber: edNum.Number,
		EddingtonList:   edNum.List,
	}

	if err := writeJSON(writer, http.StatusOK, edNumDto); err != nil {
		log.Printf("failed to write eddington response: %v", err)
		writeInternalServerError(writer, "Failed to encode eddington response")
	}
}

// getDashboardAnnualGoals godoc
// @Summary Get annual goals and projections
// @Description Returns persisted annual goals and computed projections for a year/activity filter
// @Tags dashboard
// @Produce json
// @Param year query int true "Year"
// @Param activityType query string true "Activity type"
// @Success 200 {object} dto.AnnualGoalsDto
// @Failure 400 {string} string "Invalid parameters"
// @Failure 500 {string} string "Internal server error"
// @Router /api/dashboard/annual-goals [get]
func getDashboardAnnualGoals(writer http.ResponseWriter, request *http.Request) {
	year, activityTypes, err := parseActivityRequestParams(request)
	if err != nil {
		writeBadRequest(writer, "Invalid request parameters", err.Error())
		return
	}
	if year == nil {
		writeBadRequest(writer, "Invalid request parameters", "year is required")
		return
	}

	goals := getContainer().getAnnualGoalsUseCase.Execute(*year, activityTypes)
	if err := writeJSON(writer, http.StatusOK, dto.ToAnnualGoalsDto(goals)); err != nil {
		log.Printf("failed to write annual goals response: %v", err)
		writeInternalServerError(writer, "Failed to encode annual goals response")
	}
}

// putDashboardAnnualGoals godoc
// @Summary Save annual goals
// @Description Persists annual goals locally in the athlete cache and returns updated projections
// @Tags dashboard
// @Accept json
// @Produce json
// @Param year query int true "Year"
// @Param activityType query string true "Activity type"
// @Param targets body dto.AnnualGoalTargetsDto true "Annual goal targets"
// @Success 200 {object} dto.AnnualGoalsDto
// @Failure 400 {string} string "Invalid parameters"
// @Failure 500 {string} string "Internal server error"
// @Router /api/dashboard/annual-goals [put]
func putDashboardAnnualGoals(writer http.ResponseWriter, request *http.Request) {
	year, activityTypes, err := parseActivityRequestParams(request)
	if err != nil {
		writeBadRequest(writer, "Invalid request parameters", err.Error())
		return
	}
	if year == nil {
		writeBadRequest(writer, "Invalid request parameters", "year is required")
		return
	}

	var requestDto dto.AnnualGoalTargetsDto
	if err := json.NewDecoder(request.Body).Decode(&requestDto); err != nil {
		writeBadRequest(writer, "Invalid request body", err.Error())
		return
	}

	goals := getContainer().updateAnnualGoalsUseCase.Execute(*year, dto.ToAnnualGoalTargets(requestDto), activityTypes)
	if err := writeJSON(writer, http.StatusOK, dto.ToAnnualGoalsDto(goals)); err != nil {
		log.Printf("failed to write annual goals response: %v", err)
		writeInternalServerError(writer, "Failed to encode annual goals response")
	}
}
