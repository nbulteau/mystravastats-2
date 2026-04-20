package api

import (
	"log"
	"net/http"
)

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
func getChartsDistanceByPeriod(writer http.ResponseWriter, request *http.Request) {
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

	distanceByPeriod := getContainer().getDistanceByPeriodUseCase.Execute(year, period, activityTypes)
	if err := writeJSON(writer, http.StatusOK, distanceByPeriod); err != nil {
		log.Printf("failed to write distance chart response: %v", err)
		writeInternalServerError(writer, "Failed to encode distance chart response")
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

	elevationByPeriod := getContainer().getElevationByPeriodUseCase.Execute(year, period, activityTypes)
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

	averageSpeedByPeriod := getContainer().getAverageSpeedByPeriodUseCase.Execute(year, period, activityTypes)
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

	averageCadenceByPeriod := getContainer().getAverageCadenceByPeriodUseCase.Execute(year, period, activityTypes)
	if err := writeJSON(writer, http.StatusOK, averageCadenceByPeriod); err != nil {
		log.Printf("failed to write average cadence chart response: %v", err)
		writeInternalServerError(writer, "Failed to encode average cadence chart response")
	}
}
