package api

import (
	"errors"
	"fmt"
	"io"
	"log"
	"mystravastats/api/dto"
	activitiesApp "mystravastats/internal/activities/application"
	activitiesDomain "mystravastats/internal/activities/domain"
	"mystravastats/internal/shared/domain/strava"
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
	year, activityTypes, err := parseActivityRequestParams(request)
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
	rawVersion := request.URL.Query().Get("version") == "raw"
	var detailedActivity *strava.DetailedActivity
	if rawVersion {
		detailedActivity, err = getContainer().getDetailedActivityUseCase.ExecuteRaw(activityId)
	} else {
		detailedActivity, err = getContainer().getDetailedActivityUseCase.Execute(activityId)
	}
	if err != nil {
		if errors.Is(err, activitiesDomain.ErrInvalidActivityID) {
			writeBadRequest(writer, "Invalid request parameters", "activityId must be > 0")
			return
		}
		writeNotFound(writer, "Resource not found", fmt.Sprintf("Activity %d not found", activityId))
		return
	}

	detailedActivityDto := dto.ToDetailedActivityDto(detailedActivity)
	if getContainer().getActivityComparisonUseCase != nil {
		detailedActivityDto.ActivityComparison = toActivityComparisonDto(
			getContainer().getActivityComparisonUseCase.Execute(detailedActivity),
		)
	}
	if err := writeJSON(writer, http.StatusOK, detailedActivityDto); err != nil {
		log.Printf("failed to write detailed activity response: %v", err)
		writeInternalServerError(writer, "Failed to encode detailed activity response")
	}
}

func toActivityComparisonDto(comparison *activitiesApp.ActivityComparison) *dto.ActivityComparisonDto {
	if comparison == nil {
		return nil
	}
	similarActivities := make([]dto.ActivityComparisonActivityDto, 0, len(comparison.SimilarActivities))
	for _, activity := range comparison.SimilarActivities {
		similarActivities = append(similarActivities, dto.ActivityComparisonActivityDto{
			ID:               activity.ID,
			Name:             activity.Name,
			Date:             activity.Date,
			Distance:         activity.Distance,
			ElevationGain:    activity.ElevationGain,
			MovingTime:       activity.MovingTime,
			AverageSpeed:     activity.AverageSpeed,
			AverageHeartrate: activity.AverageHeartrate,
			AverageWatts:     activity.AverageWatts,
			AverageCadence:   activity.AverageCadence,
			SimilarityScore:  activity.SimilarityScore,
		})
	}

	commonSegments := make([]dto.ActivityComparisonSegmentDto, 0, len(comparison.CommonSegments))
	for _, segment := range comparison.CommonSegments {
		commonSegments = append(commonSegments, dto.ActivityComparisonSegmentDto{
			ID:            segment.ID,
			Name:          segment.Name,
			MatchCount:    segment.MatchCount,
			ActivityIDs:   append([]int64(nil), segment.ActivityIDs...),
			ActivityNames: append([]string(nil), segment.ActivityNames...),
		})
	}

	return &dto.ActivityComparisonDto{
		Status: comparison.Status,
		Label:  comparison.Label,
		Criteria: dto.ActivityComparisonCriteriaDto{
			ActivityType: comparison.Criteria.ActivityType,
			Year:         comparison.Criteria.Year,
			SampleSize:   comparison.Criteria.SampleSize,
		},
		Baseline: dto.ActivityComparisonBaselineDto{
			Distance:         comparison.Baseline.Distance,
			ElevationGain:    comparison.Baseline.ElevationGain,
			MovingTime:       comparison.Baseline.MovingTime,
			AverageSpeed:     comparison.Baseline.AverageSpeed,
			AverageHeartrate: comparison.Baseline.AverageHeartrate,
			AverageWatts:     comparison.Baseline.AverageWatts,
			AverageCadence:   comparison.Baseline.AverageCadence,
		},
		Deltas: dto.ActivityComparisonDeltasDto{
			Distance:         comparison.Deltas.Distance,
			ElevationGain:    comparison.Deltas.ElevationGain,
			MovingTime:       comparison.Deltas.MovingTime,
			AverageSpeed:     comparison.Deltas.AverageSpeed,
			AverageSpeedPct:  comparison.Deltas.AverageSpeedPct,
			AverageHeartrate: comparison.Deltas.AverageHeartrate,
			AverageWatts:     comparison.Deltas.AverageWatts,
			AverageCadence:   comparison.Deltas.AverageCadence,
		},
		SimilarActivities: similarActivities,
		CommonSegments:    commonSegments,
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
	year, activityTypes, err := parseActivityRequestParams(request)
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
	year, activityTypes, err := parseActivityRequestParams(request)
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
	year, activityTypes, err := parseActivityRequestParams(request)
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
