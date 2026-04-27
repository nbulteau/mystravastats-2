package api

import (
	"errors"
	"fmt"
	"io"
	"log"
	"mystravastats/api/dto"
	activitiesDomain "mystravastats/internal/activities/domain"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

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
func getActivitiesByActivityType(writer http.ResponseWriter, request *http.Request) {
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

	activities := getContainer().listActivitiesUseCase.Execute(year, activityTypes)
	activitiesDto := make([]dto.ActivityDto, len(activities))
	for i, activity := range activities {
		activitiesDto[i] = dto.ToActivityDto(*activity)
	}

	if err := writeJSON(writer, http.StatusOK, activitiesDto); err != nil {
		log.Printf("failed to write activities response: %v", err)
		writeInternalServerError(writer, "Failed to encode activities response")
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
	activityId, err := strconv.ParseInt(mux.Vars(request)["activityId"], 10, 64)
	if err != nil {
		writeBadRequest(writer, "Invalid request parameters", "invalid activityId")
		return
	}
	activity, err := getContainer().getDetailedActivityUseCase.Execute(activityId)
	if err != nil {
		if errors.Is(err, activitiesDomain.ErrInvalidActivityID) {
			writeBadRequest(writer, "Invalid request parameters", "activityId must be > 0")
			return
		}
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
	csvData := getContainer().exportActivitiesCSVUseCase.Execute(year, activityTypes)

	writer.Header().Set("Content-Type", "text/csv")
	writer.Header().Set("Content-Disposition", "attachment; filename=\"activities.csv\"")
	writer.WriteHeader(http.StatusOK)
	if _, err := io.WriteString(writer, csvData); err != nil {
		log.Printf("failed to write CSV response: %v", err)
		return
	}
	log.Println("CSV export successful")
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
func getMapsGPX(writer http.ResponseWriter, request *http.Request) {
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

	gpx := getContainer().getMapsGPXUseCase.Execute(year, activityTypes)
	if err := writeJSON(writer, http.StatusOK, gpx); err != nil {
		log.Printf("failed to write gpx response: %v", err)
		writeInternalServerError(writer, "Failed to encode gpx response")
	}
}

// getMapPassages godoc
// @Summary Get map passage density data
// @Description Returns aggregated passage corridors from activity GPS streams for map display
// @Tags maps
// @Produce json
// @Param year query int false "Year"
// @Param activityType query string true "Activity type"
// @Success 200 {object} object "Map passage data"
// @Failure 400 {string} string "Invalid parameters"
// @Failure 500 {string} string "Internal server error"
// @Router /api/maps/passages [get]
func getMapPassages(writer http.ResponseWriter, request *http.Request) {
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

	passages := getContainer().getMapPassagesUseCase.Execute(year, activityTypes)
	if err := writeJSON(writer, http.StatusOK, passages); err != nil {
		log.Printf("failed to write map passages response: %v", err)
		writeInternalServerError(writer, "Failed to encode map passages response")
	}
}
