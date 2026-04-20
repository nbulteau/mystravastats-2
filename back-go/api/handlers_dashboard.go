package api

import (
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
	activityTypes, err := getActivityTypeParam(request)
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
	activityTypes, err := getActivityTypeParam(request)
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
	activityTypes, err := getActivityTypeParam(request)
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
	activityTypes, err := getActivityTypeParam(request)
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
