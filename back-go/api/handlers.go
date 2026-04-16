package api

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"mystravastats/api/dto"
	"mystravastats/domain/business"
	activitiesDomain "mystravastats/internal/activities/domain"
	routesDomain "mystravastats/internal/routes/domain"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
)

const (
	defaultRoutesVariantCount = 4
	maxGeneratedVariantCount  = 24
	generatedRouteCacheTTL    = 6 * time.Hour
)

type routeStartPointPayload struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

type generateTargetRoutesPayload struct {
	StartPoint      *routeStartPointPayload `json:"startPoint"`
	RouteType       string                  `json:"routeType"`
	StartDirection  string                  `json:"startDirection"`
	DistanceTarget  float64                 `json:"distanceTargetKm"`
	ElevationTarget *float64                `json:"elevationTargetM,omitempty"`
	VariantCount    *int                    `json:"variantCount,omitempty"`
}

type generateShapeRoutesPayload struct {
	ShapeInputType  string                  `json:"shapeInputType"`
	ShapeData       string                  `json:"shapeData"`
	StartPoint      *routeStartPointPayload `json:"startPoint,omitempty"`
	DistanceTarget  *float64                `json:"distanceTargetKm,omitempty"`
	ElevationTarget *float64                `json:"elevationTargetM,omitempty"`
	RouteType       string                  `json:"routeType"`
	VariantCount    *int                    `json:"variantCount,omitempty"`
}

type generatedRouteCacheEntry struct {
	Name      string
	Points    [][]float64
	ExpiresAt time.Time
}

var generatedRouteCache = struct {
	mu    sync.RWMutex
	items map[string]generatedRouteCacheEntry
}{
	items: map[string]generatedRouteCacheEntry{},
}

// getHealthDetails godoc
// @Summary Get cache health details
// @Description Returns cache diagnostics including manifest/warmup/best-effort status
// @Tags health
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {string} string "Internal server error"
// @Router /api/health/details [get]
func getHealthDetails(w http.ResponseWriter, _ *http.Request) {
	details := getContainer().getCacheHealthDetailsUseCase.Execute()
	if err := writeJSON(w, http.StatusOK, details); err != nil {
		log.Printf("failed to write cache health response: %v", err)
		writeInternalServerError(w, "Failed to encode cache health response")
	}
}

// getAthlete godoc
// @Summary Get athlete information
// @Description Returns the current athlete information
// @Tags athlete
// @Produce json
// @Success 200 {object} dto.AthleteDto
// @Failure 500 {string} string "Internal server error"
// @Router /api/athletes/me [get]
func getAthlete(w http.ResponseWriter, _ *http.Request) {
	athlete := getContainer().getAthleteUseCase.Execute()
	athleteDto := dto.ToAthleteDto(athlete)

	if err := writeJSON(w, http.StatusOK, athleteDto); err != nil {
		log.Printf("failed to write athlete response: %v", err)
		writeInternalServerError(w, "Failed to encode athlete response")
	}
}

func getAthleteHeartRateZones(w http.ResponseWriter, _ *http.Request) {
	settings := getContainer().getHeartRateZoneSettingsUseCase.Execute()
	settingsDto := dto.ToHeartRateZoneSettingsDto(settings)

	if err := writeJSON(w, http.StatusOK, settingsDto); err != nil {
		log.Printf("failed to write heart rate settings response: %v", err)
		writeInternalServerError(w, "Failed to encode heart rate settings response")
	}
}

func putAthleteHeartRateZones(w http.ResponseWriter, r *http.Request) {
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(r.Body)

	var settingsDto dto.HeartRateZoneSettingsDto
	if err := json.NewDecoder(r.Body).Decode(&settingsDto); err != nil {
		writeBadRequest(w, "Invalid request body", "heart rate zone settings payload is invalid")
		return
	}

	settings := dto.ToHeartRateZoneSettings(settingsDto)
	updatedSettings := getContainer().updateHeartRateZoneSettingsUseCase.Execute(settings)
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

	activitiesByActivityTypeAndYear := getContainer().listActivitiesUseCase.Execute(year, activityTypes)
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

	statisticsByActivityTypeAndYear := getContainer().listStatisticsUseCase.Execute(year, activityTypes)
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

	timeline := getContainer().listPersonalRecordsTimelineUseCase.Execute(year, metric, activityTypes)
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

	analysis := getContainer().getHeartRateZoneAnalysisUseCase.Execute(year, activityTypes)
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

	progression := getContainer().getSegmentClimbProgressionUseCase.Execute(year, metric, targetType, targetId, activityTypes)
	progressionDto := dto.ToSegmentClimbProgressionDto(progression)

	if err := writeJSON(w, http.StatusOK, progressionDto); err != nil {
		log.Printf("failed to write segment/climb progression response: %v", err)
		writeInternalServerError(w, "Failed to encode segment/climb progression response")
	}
}

func getSegmentsByActivityType(w http.ResponseWriter, r *http.Request) {
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
	query := getQueryParam(r)
	from, err := getFromDateParam(r)
	if err != nil {
		writeBadRequest(w, "Invalid request parameters", err.Error())
		return
	}
	to, err := getToDateParam(r)
	if err != nil {
		writeBadRequest(w, "Invalid request parameters", err.Error())
		return
	}

	segments := getContainer().listSegmentsUseCase.Execute(year, metric, query, from, to, activityTypes)
	segmentsDto := make([]dto.SegmentClimbTargetSummaryDto, len(segments))
	for i, segment := range segments {
		segmentsDto[i] = dto.ToSegmentClimbTargetSummaryDto(segment)
	}

	if err := writeJSON(w, http.StatusOK, segmentsDto); err != nil {
		log.Printf("failed to write segments response: %v", err)
		writeInternalServerError(w, "Failed to encode segments response")
	}
}

func getSegmentEffortsByActivityType(w http.ResponseWriter, r *http.Request) {
	segmentID, err := getSegmentIDPathParam(r)
	if err != nil {
		writeBadRequest(w, "Invalid request parameters", err.Error())
		return
	}
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
	from, err := getFromDateParam(r)
	if err != nil {
		writeBadRequest(w, "Invalid request parameters", err.Error())
		return
	}
	to, err := getToDateParam(r)
	if err != nil {
		writeBadRequest(w, "Invalid request parameters", err.Error())
		return
	}

	efforts := getContainer().listSegmentEffortsUseCase.Execute(year, metric, segmentID, from, to, activityTypes)
	effortsDto := make([]dto.SegmentClimbAttemptDto, len(efforts))
	for i, effort := range efforts {
		effortsDto[i] = dto.ToSegmentClimbAttemptDto(effort)
	}

	if err := writeJSON(w, http.StatusOK, effortsDto); err != nil {
		log.Printf("failed to write segment efforts response: %v", err)
		writeInternalServerError(w, "Failed to encode segment efforts response")
	}
}

func getSegmentSummaryByActivityType(w http.ResponseWriter, r *http.Request) {
	segmentID, err := getSegmentIDPathParam(r)
	if err != nil {
		writeBadRequest(w, "Invalid request parameters", err.Error())
		return
	}
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
	from, err := getFromDateParam(r)
	if err != nil {
		writeBadRequest(w, "Invalid request parameters", err.Error())
		return
	}
	to, err := getToDateParam(r)
	if err != nil {
		writeBadRequest(w, "Invalid request parameters", err.Error())
		return
	}

	summary := getContainer().getSegmentSummaryUseCase.Execute(year, metric, segmentID, from, to, activityTypes)
	if summary == nil {
		writeNotFound(w, "Segment not found", "No attempts found for this segment with current filters")
		return
	}

	var prDto *dto.SegmentClimbAttemptDto
	if summary.PersonalRecord != nil {
		pr := dto.ToSegmentClimbAttemptDto(*summary.PersonalRecord)
		prDto = &pr
	}
	topEffortsDto := make([]dto.SegmentClimbAttemptDto, len(summary.TopEfforts))
	for i, effort := range summary.TopEfforts {
		topEffortsDto[i] = dto.ToSegmentClimbAttemptDto(effort)
	}

	response := dto.SegmentSummaryDto{
		Metric:         summary.Metric,
		Segment:        dto.ToSegmentClimbTargetSummaryDto(summary.Segment),
		PersonalRecord: prDto,
		TopEfforts:     topEffortsDto,
	}

	if err := writeJSON(w, http.StatusOK, response); err != nil {
		log.Printf("failed to write segment summary response: %v", err)
		writeInternalServerError(w, "Failed to encode segment summary response")
	}
}

func getRouteRecommendationsByActivityType(w http.ResponseWriter, r *http.Request) {
	year, activityTypes, request, err := parseRouteExplorerRequestParams(r)
	if err != nil {
		writeBadRequest(w, "Invalid request parameters", err.Error())
		return
	}

	explorer := getContainer().getRouteExplorerUseCase.Execute(year, request, activityTypes)
	explorerDto := dto.ToRouteExplorerResultDto(explorer)
	if err := writeJSON(w, http.StatusOK, explorerDto); err != nil {
		log.Printf("failed to write routes explorer response: %v", err)
		writeInternalServerError(w, "Failed to encode routes explorer response")
	}
}

func getRouteRecommendationGPXByActivityType(w http.ResponseWriter, r *http.Request) {
	year, activityTypes, request, err := parseRouteExplorerRequestParams(r)
	if err != nil {
		writeBadRequest(w, "Invalid request parameters", err.Error())
		return
	}

	routeID := strings.TrimSpace(r.URL.Query().Get("routeId"))
	if routeID == "" {
		writeBadRequest(w, "Invalid request parameters", "routeId is required")
		return
	}

	explorer := getContainer().getRouteExplorerUseCase.Execute(year, request, activityTypes)
	name, points, found := findRouteForGPXExport(explorer, routeID)
	if !found {
		writeNotFound(w, "Route not found", fmt.Sprintf("No route found for routeId=%s with current filters", routeID))
		return
	}

	gpxPayload, err := buildRouteGPX(name, points)
	if err != nil {
		writeBadRequest(w, "Invalid route geometry", err.Error())
		return
	}

	fileName := sanitizeRouteFileName(routeID)
	if fileName == "" {
		fileName = "route"
	}

	w.Header().Set("Content-Type", "application/gpx+xml; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.gpx\"", fileName))
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte(gpxPayload)); err != nil {
		log.Printf("failed to write route gpx response: %v", err)
	}
}

func generateTargetRoutesByActivityType(w http.ResponseWriter, r *http.Request) {
	year, activityTypes, err := parseRouteGenerationFilters(r)
	if err != nil {
		writeBadRequest(w, "Invalid request parameters", err.Error())
		return
	}

	payload, err := parseGenerateTargetRoutesPayload(r)
	if err != nil {
		writeBadRequest(w, "Invalid request body", err.Error())
		return
	}
	if err := validateGenerateTargetRoutesPayload(payload); err != nil {
		writeBadRequest(w, "Invalid request body", err.Error())
		return
	}

	routeType := normalizeGenerateRouteType(payload.RouteType)
	startDirection := normalizeGenerateStartDirection(payload.StartDirection)
	variantCount := normalizeGenerateVariantCount(payload.VariantCount)
	preferredStart := &routesDomain.Coordinates{
		Lat: payload.StartPoint.Lat,
		Lng: payload.StartPoint.Lng,
	}
	request := routesDomain.RouteExplorerRequest{
		DistanceTargetKm: &payload.DistanceTarget,
		ElevationTargetM: payload.ElevationTarget,
		StartPoint:       preferredStart,
		StartDirection:   optionalNonEmptyString(startDirection),
		RouteType:        optionalNonEmptyString(routeType),
		Limit:            variantCount,
	}

	result := getContainer().getRouteExplorerUseCase.Execute(year, request, activityTypes)
	response := buildTargetGeneratedRoutesResponse(result, payload.DistanceTarget, payload.ElevationTarget, routeType, startDirection, variantCount)
	cacheGeneratedRoutes(response.Routes)

	if err := writeJSON(w, http.StatusOK, response); err != nil {
		log.Printf("failed to write generated target routes response: %v", err)
		writeInternalServerError(w, "Failed to encode generated routes response")
	}
}

func generateShapeRoutesByActivityType(w http.ResponseWriter, r *http.Request) {
	year, activityTypes, err := parseRouteGenerationFilters(r)
	if err != nil {
		writeBadRequest(w, "Invalid request parameters", err.Error())
		return
	}

	payload, err := parseGenerateShapeRoutesPayload(r)
	if err != nil {
		writeBadRequest(w, "Invalid request body", err.Error())
		return
	}
	if err := validateGenerateShapeRoutesPayload(payload); err != nil {
		writeBadRequest(w, "Invalid request body", err.Error())
		return
	}

	routeType := normalizeGenerateRouteType(payload.RouteType)
	variantCount := normalizeGenerateVariantCount(payload.VariantCount)
	shapeFilter := inferShapeFilter(payload.ShapeInputType, payload.ShapeData)
	var preferredStart *routesDomain.Coordinates
	if payload.StartPoint != nil {
		preferredStart = &routesDomain.Coordinates{
			Lat: payload.StartPoint.Lat,
			Lng: payload.StartPoint.Lng,
		}
	}

	request := routesDomain.RouteExplorerRequest{
		DistanceTargetKm: payload.DistanceTarget,
		ElevationTargetM: payload.ElevationTarget,
		StartPoint:       preferredStart,
		RouteType:        optionalNonEmptyString(routeType),
		Limit:            variantCount,
		Shape:            optionalNonEmptyString(shapeFilter),
		ShapePolyline:    optionalNonEmptyString(strings.TrimSpace(payload.ShapeData)),
		IncludeRemix:     true,
	}
	result := getContainer().getRouteExplorerUseCase.Execute(year, request, activityTypes)
	response := buildShapeGeneratedRoutesResponse(
		result,
		payload.DistanceTarget,
		payload.ElevationTarget,
		routeType,
		variantCount,
	)
	cacheGeneratedRoutes(response.Routes)

	if err := writeJSON(w, http.StatusOK, response); err != nil {
		log.Printf("failed to write generated shape routes response: %v", err)
		writeInternalServerError(w, "Failed to encode generated routes response")
	}
}

func getGeneratedRouteGPXByID(w http.ResponseWriter, r *http.Request) {
	routeID := strings.TrimSpace(mux.Vars(r)["routeId"])
	if routeID == "" {
		writeBadRequest(w, "Invalid request parameters", "routeId is required")
		return
	}

	entry, found := getGeneratedRouteFromCache(routeID)
	if !found {
		writeNotFound(w, "Route not found", fmt.Sprintf("No generated route found for routeId=%s", routeID))
		return
	}

	gpxPayload, err := buildRouteGPX(entry.Name, entry.Points)
	if err != nil {
		writeBadRequest(w, "Invalid route geometry", err.Error())
		return
	}

	fileName := sanitizeRouteFileName(routeID)
	if fileName == "" {
		fileName = "route"
	}

	w.Header().Set("Content-Type", "application/gpx+xml; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.gpx\"", fileName))
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte(gpxPayload)); err != nil {
		log.Printf("failed to write generated route gpx response: %v", err)
	}
}

func parseGenerateTargetRoutesPayload(r *http.Request) (generateTargetRoutesPayload, error) {
	defer func() {
		_ = r.Body.Close()
	}()

	var payload generateTargetRoutesPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		return generateTargetRoutesPayload{}, fmt.Errorf("target payload is invalid")
	}
	return payload, nil
}

func parseGenerateShapeRoutesPayload(r *http.Request) (generateShapeRoutesPayload, error) {
	defer func() {
		_ = r.Body.Close()
	}()

	var payload generateShapeRoutesPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		return generateShapeRoutesPayload{}, fmt.Errorf("shape payload is invalid")
	}
	return payload, nil
}

func validateGenerateTargetRoutesPayload(payload generateTargetRoutesPayload) error {
	if payload.StartPoint == nil {
		return fmt.Errorf("startPoint is required")
	}
	if !isValidLatLng(payload.StartPoint.Lat, payload.StartPoint.Lng) {
		return fmt.Errorf("startPoint has invalid coordinates")
	}
	if payload.DistanceTarget <= 0 {
		return fmt.Errorf("distanceTargetKm must be greater than 0")
	}
	if payload.ElevationTarget != nil && *payload.ElevationTarget < 0 {
		return fmt.Errorf("elevationTargetM must be greater than or equal to 0")
	}
	if payload.VariantCount != nil && (*payload.VariantCount < 1 || *payload.VariantCount > maxGeneratedVariantCount) {
		return fmt.Errorf("variantCount must be between 1 and %d", maxGeneratedVariantCount)
	}
	if direction := normalizeGenerateStartDirection(payload.StartDirection); payload.StartDirection != "" && direction == "" {
		return fmt.Errorf("startDirection must be one of N/S/E/W")
	}
	return nil
}

func validateGenerateShapeRoutesPayload(payload generateShapeRoutesPayload) error {
	inputType := strings.ToLower(strings.TrimSpace(payload.ShapeInputType))
	if inputType == "" {
		return fmt.Errorf("shapeInputType is required")
	}
	switch inputType {
	case "draw", "gpx", "svg", "polyline":
	default:
		return fmt.Errorf("shapeInputType must be one of draw/gpx/svg/polyline")
	}
	if strings.TrimSpace(payload.ShapeData) == "" {
		return fmt.Errorf("shapeData is required")
	}
	if payload.DistanceTarget != nil && *payload.DistanceTarget <= 0 {
		return fmt.Errorf("distanceTargetKm must be greater than 0")
	}
	if payload.ElevationTarget != nil && *payload.ElevationTarget < 0 {
		return fmt.Errorf("elevationTargetM must be greater than or equal to 0")
	}
	if payload.StartPoint != nil && !isValidLatLng(payload.StartPoint.Lat, payload.StartPoint.Lng) {
		return fmt.Errorf("startPoint has invalid coordinates")
	}
	if payload.VariantCount != nil && (*payload.VariantCount < 1 || *payload.VariantCount > maxGeneratedVariantCount) {
		return fmt.Errorf("variantCount must be between 1 and %d", maxGeneratedVariantCount)
	}
	return nil
}

func parseRouteGenerationFilters(r *http.Request) (*int, []business.ActivityType, error) {
	year, err := getYearParam(r)
	if err != nil {
		return nil, nil, err
	}

	activityTypeRaw := strings.TrimSpace(r.URL.Query().Get("activityType"))
	if activityTypeRaw == "" {
		return year, defaultRouteGenerationActivityTypes(), nil
	}

	activityTypes, err := getActivityTypeParam(r)
	if err != nil {
		return nil, nil, err
	}
	return year, activityTypes, nil
}

func defaultRouteGenerationActivityTypes() []business.ActivityType {
	return []business.ActivityType{
		business.Ride,
		business.GravelRide,
		business.MountainBikeRide,
		business.Commute,
		business.VirtualRide,
		business.Run,
		business.TrailRun,
		business.Hike,
	}
}

func buildTargetGeneratedRoutesResponse(
	result routesDomain.RouteExplorerResult,
	distanceTarget float64,
	elevationTarget *float64,
	routeType string,
	startDirection string,
	limit int,
) dto.GenerateRoutesResponseDto {
	// Target mode must return newly generated loops only.
	recommendations := make([]routesDomain.RouteRecommendation, 0, len(result.RoadGraphLoops))
	recommendations = append(recommendations, result.RoadGraphLoops...)

	routes := toGeneratedRoutesFromRecommendations(recommendations, &distanceTarget, elevationTarget, routeType, startDirection, limit)
	return dto.GenerateRoutesResponseDto{Routes: routes}
}

func buildShapeGeneratedRoutesResponse(
	result routesDomain.RouteExplorerResult,
	distanceTarget *float64,
	elevationTarget *float64,
	routeType string,
	limit int,
) dto.GenerateRoutesResponseDto {
	routes := make([]dto.GeneratedRouteDto, 0, limit)
	seen := make(map[string]struct{}, limit)

	appendRoute := func(recommendation routesDomain.RouteRecommendation) {
		if len(routes) >= limit {
			return
		}
		if _, exists := seen[recommendation.RouteID]; exists {
			return
		}
		score := buildGeneratedRouteScore(recommendation, distanceTarget, elevationTarget, "")
		converted := dto.ToGeneratedRouteDto(recommendation, score, routeType, "")
		routes = append(routes, converted)
		seen[recommendation.RouteID] = struct{}{}
	}

	for _, recommendation := range result.ShapeMatches {
		appendRoute(recommendation)
	}
	for _, recommendation := range result.RoadGraphLoops {
		appendRoute(recommendation)
	}
	for _, recommendation := range result.ClosestLoops {
		appendRoute(recommendation)
	}
	for _, remix := range result.ShapeRemixes {
		if len(routes) >= limit {
			break
		}
		if _, exists := seen[remix.ID]; exists {
			continue
		}
		score := dto.RouteGenerationScoreDto{
			Global:      clampScore(remix.MatchScore),
			Distance:    clampScore(remix.MatchScore),
			Elevation:   clampScore(remix.MatchScore),
			Duration:    clampScore(remix.MatchScore),
			Direction:   clampScore(remix.MatchScore),
			Shape:       clampScore(remix.MatchScore),
			RoadFitness: 75.0,
		}
		routes = append(routes, dto.ToGeneratedRouteFromShapeRemixDto(remix, score, routeType))
		seen[remix.ID] = struct{}{}
	}

	return dto.GenerateRoutesResponseDto{Routes: routes}
}

func toGeneratedRoutesFromRecommendations(
	recommendations []routesDomain.RouteRecommendation,
	distanceTarget *float64,
	elevationTarget *float64,
	routeType string,
	startDirection string,
	limit int,
) []dto.GeneratedRouteDto {
	routes := make([]dto.GeneratedRouteDto, 0, limit)
	seen := make(map[string]struct{}, limit)
	for _, recommendation := range recommendations {
		if len(routes) >= limit {
			break
		}
		if recommendation.RouteID == "" {
			continue
		}
		if _, exists := seen[recommendation.RouteID]; exists {
			continue
		}
		score := buildGeneratedRouteScore(recommendation, distanceTarget, elevationTarget, startDirection)
		routes = append(routes, dto.ToGeneratedRouteDto(recommendation, score, routeType, startDirection))
		seen[recommendation.RouteID] = struct{}{}
	}
	return routes
}

func buildGeneratedRouteScore(
	recommendation routesDomain.RouteRecommendation,
	distanceTarget *float64,
	elevationTarget *float64,
	startDirection string,
) dto.RouteGenerationScoreDto {
	global := clampScore(recommendation.MatchScore)
	distance := global
	elevation := global
	duration := global
	direction := global

	if distanceTarget != nil && *distanceTarget > 0 {
		distance = proximityScore(recommendation.DistanceKm, *distanceTarget)
	}
	if elevationTarget != nil && *elevationTarget >= 0 {
		elevation = proximityScore(recommendation.ElevationGainM, *elevationTarget)
	}
	if startDirection != "" && recommendation.Start != nil && recommendation.End != nil {
		direction = directionScore(*recommendation.Start, *recommendation.End, startDirection)
	}

	shape := 50.0
	if recommendation.ShapeScore != nil {
		shape = clampScore(*recommendation.ShapeScore * 100.0)
	}

	roadFitness := 70.0
	if recommendation.VariantType == routesDomain.RouteVariantRoadGraph {
		roadFitness = 100.0
	} else if recommendation.IsLoop {
		roadFitness = 82.0
	}

	return dto.RouteGenerationScoreDto{
		Global:      global,
		Distance:    distance,
		Elevation:   elevation,
		Duration:    duration,
		Direction:   direction,
		Shape:       shape,
		RoadFitness: roadFitness,
	}
}

func proximityScore(actual float64, target float64) float64 {
	if target <= 0 {
		return 50.0
	}
	deltaRatio := mathAbs(actual-target) / target
	return clampScore(100.0 - (deltaRatio * 100.0))
}

func directionScore(start routesDomain.Coordinates, end routesDomain.Coordinates, expected string) float64 {
	actual := normalizedDirectionFromCoordinates(start, end)
	if actual == "" {
		return 50.0
	}
	if actual == expected {
		return 100.0
	}
	return 40.0
}

func normalizedDirectionFromCoordinates(start routesDomain.Coordinates, end routesDomain.Coordinates) string {
	dLat := end.Lat - start.Lat
	dLng := end.Lng - start.Lng
	if mathAbs(dLat) < 0.0001 && mathAbs(dLng) < 0.0001 {
		return ""
	}
	if mathAbs(dLat) >= mathAbs(dLng) {
		if dLat >= 0 {
			return "N"
		}
		return "S"
	}
	if dLng >= 0 {
		return "E"
	}
	return "W"
}

func clampScore(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return math.Round(value*10.0) / 10.0
}

func cacheGeneratedRoutes(routes []dto.GeneratedRouteDto) {
	now := time.Now()
	generatedRouteCache.mu.Lock()
	defer generatedRouteCache.mu.Unlock()

	for routeID, entry := range generatedRouteCache.items {
		if entry.ExpiresAt.Before(now) {
			delete(generatedRouteCache.items, routeID)
		}
	}

	for _, route := range routes {
		if strings.TrimSpace(route.RouteID) == "" || len(route.PreviewLatLng) < 2 {
			continue
		}
		generatedRouteCache.items[route.RouteID] = generatedRouteCacheEntry{
			Name:      route.Title,
			Points:    route.PreviewLatLng,
			ExpiresAt: now.Add(generatedRouteCacheTTL),
		}
	}
}

func getGeneratedRouteFromCache(routeID string) (generatedRouteCacheEntry, bool) {
	now := time.Now()
	generatedRouteCache.mu.RLock()
	entry, found := generatedRouteCache.items[routeID]
	generatedRouteCache.mu.RUnlock()
	if !found {
		return generatedRouteCacheEntry{}, false
	}

	if entry.ExpiresAt.Before(now) {
		generatedRouteCache.mu.Lock()
		delete(generatedRouteCache.items, routeID)
		generatedRouteCache.mu.Unlock()
		return generatedRouteCacheEntry{}, false
	}
	return entry, true
}

func normalizeGenerateRouteType(value string) string {
	normalized := strings.ToUpper(strings.TrimSpace(value))
	switch normalized {
	case "RIDE", "MTB", "GRAVEL", "RUN", "TRAIL", "HIKE":
		return normalized
	default:
		return "RIDE"
	}
}

func normalizeGenerateStartDirection(value string) string {
	normalized := strings.ToUpper(strings.TrimSpace(value))
	switch normalized {
	case "N", "S", "E", "W":
		return normalized
	default:
		return ""
	}
}

func normalizeGenerateVariantCount(value *int) int {
	if value == nil {
		return defaultRoutesVariantCount
	}
	if *value < 1 {
		return 1
	}
	if *value > maxGeneratedVariantCount {
		return maxGeneratedVariantCount
	}
	return *value
}

func optionalNonEmptyString(value string) *string {
	normalized := strings.TrimSpace(value)
	if normalized == "" {
		return nil
	}
	return &normalized
}

func inferShapeFilter(shapeInputType string, shapeData string) string {
	switch strings.ToLower(strings.TrimSpace(shapeInputType)) {
	case "draw", "polyline":
		points, err := parseShapePolylineCoordinates(shapeData)
		if err != nil || len(points) < 2 {
			return ""
		}
		return inferShapeFromCoordinates(points)
	default:
		return ""
	}
}

func parseShapePolylineCoordinates(raw string) ([][]float64, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, errors.New("shape data is empty")
	}

	var points [][]float64
	if err := json.Unmarshal([]byte(trimmed), &points); err == nil {
		return sanitizePolylineCoordinates(points), nil
	}

	var wrapped struct {
		Points      [][]float64 `json:"points"`
		Coordinates [][]float64 `json:"coordinates"`
		LatLng      [][]float64 `json:"latLng"`
	}
	if err := json.Unmarshal([]byte(trimmed), &wrapped); err != nil {
		return nil, errors.New("shapeData must be a JSON array of [lat,lng] coordinates")
	}
	switch {
	case len(wrapped.Points) > 0:
		return sanitizePolylineCoordinates(wrapped.Points), nil
	case len(wrapped.Coordinates) > 0:
		return sanitizePolylineCoordinates(wrapped.Coordinates), nil
	case len(wrapped.LatLng) > 0:
		return sanitizePolylineCoordinates(wrapped.LatLng), nil
	default:
		return nil, errors.New("shapeData does not contain coordinates")
	}
}

func sanitizePolylineCoordinates(points [][]float64) [][]float64 {
	result := make([][]float64, 0, len(points))
	for _, point := range points {
		if len(point) < 2 {
			continue
		}
		if !isValidLatLng(point[0], point[1]) {
			continue
		}
		result = append(result, []float64{point[0], point[1]})
	}
	return result
}

func inferShapeFromCoordinates(points [][]float64) string {
	if len(points) < 2 {
		return ""
	}
	start := routesDomain.Coordinates{Lat: points[0][0], Lng: points[0][1]}
	end := routesDomain.Coordinates{Lat: points[len(points)-1][0], Lng: points[len(points)-1][1]}
	startEndDistance := haversineDistanceMeters(start, end)
	pathDistance := 0.0
	maxFromStart := 0.0
	for index := 1; index < len(points); index++ {
		prev := routesDomain.Coordinates{Lat: points[index-1][0], Lng: points[index-1][1]}
		next := routesDomain.Coordinates{Lat: points[index][0], Lng: points[index][1]}
		segment := haversineDistanceMeters(prev, next)
		pathDistance += segment
		startDistance := haversineDistanceMeters(start, next)
		if startDistance > maxFromStart {
			maxFromStart = startDistance
		}
	}

	loopThreshold := maxFloat(350.0, pathDistance*0.08)
	if startEndDistance <= loopThreshold {
		return "LOOP"
	}
	if maxFromStart > 0 && startEndDistance <= maxFloat(220.0, maxFromStart*0.18) {
		return "OUT_AND_BACK"
	}
	return "POINT_TO_POINT"
}

func haversineDistanceMeters(left routesDomain.Coordinates, right routesDomain.Coordinates) float64 {
	const earthRadiusM = 6371000.0
	lat1 := left.Lat * (math.Pi / 180.0)
	lat2 := right.Lat * (math.Pi / 180.0)
	dLat := (right.Lat - left.Lat) * (math.Pi / 180.0)
	dLng := (right.Lng - left.Lng) * (math.Pi / 180.0)

	a := math.Sin(dLat/2.0)*math.Sin(dLat/2.0) +
		math.Cos(lat1)*math.Cos(lat2)*math.Sin(dLng/2.0)*math.Sin(dLng/2.0)
	c := 2.0 * math.Atan2(math.Sqrt(a), math.Sqrt(1.0-a))
	return earthRadiusM * c
}

func isValidLatLng(lat float64, lng float64) bool {
	return lat >= -90.0 && lat <= 90.0 && lng >= -180.0 && lng <= 180.0
}

func maxFloat(left float64, right float64) float64 {
	if left > right {
		return left
	}
	return right
}

func mathAbs(value float64) float64 {
	if value < 0 {
		return -value
	}
	return value
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

	gpx := getContainer().getMapsGPXUseCase.Execute(year, activityTypes)

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

	distanceByPeriod := getContainer().getDistanceByPeriodUseCase.Execute(year, period, activityTypes)

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

	dashboardData := getContainer().getDashboardDataUseCase.Execute(activityTypes)
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

	cumulativeData := getContainer().getCumulativeDataPerYearUseCase.Execute(activityTypes)
	cumulativeDataDto := dto.CumulativeDataPerYearDto{
		Distance:  cumulativeData.Distance,
		Elevation: cumulativeData.Elevation,
	}

	if err := writeJSON(w, http.StatusOK, cumulativeDataDto); err != nil {
		log.Printf("failed to write cumulative dashboard response: %v", err)
		writeInternalServerError(w, "Failed to encode cumulative dashboard response")
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
func getDashboardActivityHeatmap(w http.ResponseWriter, r *http.Request) {
	activityTypes, err := getActivityTypeParam(r)
	if err != nil {
		writeBadRequest(w, "Invalid request parameters", err.Error())
		return
	}

	heatmap := getContainer().getActivityHeatmapUseCase.Execute(activityTypes)

	if err := writeJSON(w, http.StatusOK, heatmap); err != nil {
		log.Printf("failed to write activity heatmap response: %v", err)
		writeInternalServerError(w, "Failed to encode activity heatmap response")
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

	edNum := getContainer().getEddingtonNumberUseCase.Execute(activityTypes)
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

	badges := getContainer().getBadgesUseCase.Execute(year, badgeSet, activityTypes)

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

func getQueryParam(r *http.Request) *string {
	query := strings.TrimSpace(r.URL.Query().Get("query"))
	if query == "" {
		return nil
	}
	return &query
}

func getFromDateParam(r *http.Request) (*string, error) {
	return getDateParam(r, "from")
}

func getToDateParam(r *http.Request) (*string, error) {
	return getDateParam(r, "to")
}

func getDateParam(r *http.Request, key string) (*string, error) {
	value := strings.TrimSpace(r.URL.Query().Get(key))
	if value == "" {
		return nil, nil
	}
	if _, err := time.Parse("2006-01-02", value); err != nil {
		return nil, fmt.Errorf("invalid %s date: %q (expected YYYY-MM-DD)", key, value)
	}
	return &value, nil
}

func getFloatParam(r *http.Request, key string) (*float64, error) {
	value := strings.TrimSpace(r.URL.Query().Get(key))
	if value == "" {
		return nil, nil
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid %s: %q", key, value)
	}
	return &parsed, nil
}

func getIntParam(r *http.Request, key string) (*int, error) {
	value := strings.TrimSpace(r.URL.Query().Get(key))
	if value == "" {
		return nil, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return nil, fmt.Errorf("invalid %s: %q", key, value)
	}
	return &parsed, nil
}

func getBoolParam(r *http.Request, key string) (*bool, error) {
	value := strings.TrimSpace(r.URL.Query().Get(key))
	if value == "" {
		return nil, nil
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return nil, fmt.Errorf("invalid %s: %q", key, value)
	}
	return &parsed, nil
}

func getOptionalStringParam(r *http.Request, key string) *string {
	value := strings.TrimSpace(r.URL.Query().Get(key))
	if value == "" {
		return nil
	}
	return &value
}

func toOptionalStartPoint(lat *float64, lng *float64) (*routesDomain.Coordinates, error) {
	if lat == nil && lng == nil {
		return nil, nil
	}
	if lat == nil || lng == nil {
		return nil, fmt.Errorf("startLat and startLng must be provided together")
	}
	if !isValidLatLng(*lat, *lng) {
		return nil, fmt.Errorf("invalid startLat/startLng coordinates")
	}
	return &routesDomain.Coordinates{
		Lat: *lat,
		Lng: *lng,
	}, nil
}

func parseRouteExplorerRequestParams(r *http.Request) (*int, []business.ActivityType, routesDomain.RouteExplorerRequest, error) {
	year, err := getYearParam(r)
	if err != nil {
		return nil, nil, routesDomain.RouteExplorerRequest{}, err
	}
	activityTypes, err := getActivityTypeParam(r)
	if err != nil {
		return nil, nil, routesDomain.RouteExplorerRequest{}, err
	}

	distanceTargetKm, err := getFloatParam(r, "distanceTargetKm")
	if err != nil {
		return nil, nil, routesDomain.RouteExplorerRequest{}, err
	}
	elevationTargetM, err := getFloatParam(r, "elevationTargetM")
	if err != nil {
		return nil, nil, routesDomain.RouteExplorerRequest{}, err
	}
	durationTargetMin, err := getIntParam(r, "durationTargetMin")
	if err != nil {
		return nil, nil, routesDomain.RouteExplorerRequest{}, err
	}
	startLat, err := getFloatParam(r, "startLat")
	if err != nil {
		return nil, nil, routesDomain.RouteExplorerRequest{}, err
	}
	startLng, err := getFloatParam(r, "startLng")
	if err != nil {
		return nil, nil, routesDomain.RouteExplorerRequest{}, err
	}
	startPoint, err := toOptionalStartPoint(startLat, startLng)
	if err != nil {
		return nil, nil, routesDomain.RouteExplorerRequest{}, err
	}
	startDirection := getOptionalStringParam(r, "startDirection")
	routeType := getOptionalStringParam(r, "routeType")
	season := getOptionalStringParam(r, "season")
	shape := getOptionalStringParam(r, "shape")
	shapePolyline := getOptionalStringParam(r, "shapePolyline")
	limit, err := getIntParam(r, "limit")
	if err != nil {
		return nil, nil, routesDomain.RouteExplorerRequest{}, err
	}
	includeRemix, err := getBoolParam(r, "includeRemix")
	if err != nil {
		return nil, nil, routesDomain.RouteExplorerRequest{}, err
	}

	request := routesDomain.RouteExplorerRequest{
		DistanceTargetKm:  distanceTargetKm,
		ElevationTargetM:  elevationTargetM,
		DurationTargetMin: durationTargetMin,
		StartPoint:        startPoint,
		StartDirection:    startDirection,
		RouteType:         routeType,
		Season:            season,
		Limit:             optionalIntValue(limit),
		Shape:             shape,
		ShapePolyline:     shapePolyline,
		IncludeRemix:      includeRemix != nil && *includeRemix,
	}
	return year, activityTypes, request, nil
}

func findRouteForGPXExport(
	result routesDomain.RouteExplorerResult,
	routeID string,
) (string, [][]float64, bool) {
	recommendations := make([]routesDomain.RouteRecommendation, 0, len(result.ClosestLoops)+len(result.Variants)+len(result.Seasonal)+len(result.RoadGraphLoops)+len(result.ShapeMatches))
	recommendations = append(recommendations, result.ClosestLoops...)
	recommendations = append(recommendations, result.Variants...)
	recommendations = append(recommendations, result.Seasonal...)
	recommendations = append(recommendations, result.RoadGraphLoops...)
	recommendations = append(recommendations, result.ShapeMatches...)

	for _, recommendation := range recommendations {
		if recommendation.RouteID == routeID {
			name := recommendation.Activity.Name
			if name == "" {
				name = recommendation.RouteID
			}
			return name, recommendation.PreviewLatLng, true
		}
	}

	for _, remix := range result.ShapeRemixes {
		if remix.ID == routeID {
			name := remix.ID
			if len(remix.Components) > 0 && remix.Components[0].Name != "" {
				name = fmt.Sprintf("Remix - %s", remix.Components[0].Name)
			}
			return name, remix.PreviewLatLng, true
		}
	}

	return "", nil, false
}

func buildRouteGPX(name string, latLng [][]float64) (string, error) {
	validPoints := make([][]float64, 0, len(latLng))
	for _, point := range latLng {
		if len(point) < 2 {
			continue
		}
		lat := point[0]
		lng := point[1]
		if lat < -90 || lat > 90 || lng < -180 || lng > 180 {
			continue
		}
		validPoints = append(validPoints, []float64{lat, lng})
	}

	if len(validPoints) < 2 {
		return "", errors.New("at least 2 valid points are required to export GPX")
	}

	safeName := strings.TrimSpace(name)
	if safeName == "" {
		safeName = "MyStravaStats route"
	}
	var escapedNameBuffer bytes.Buffer
	if err := xml.EscapeText(&escapedNameBuffer, []byte(safeName)); err != nil {
		return "", err
	}
	escapedName := escapedNameBuffer.String()

	var builder strings.Builder
	builder.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	builder.WriteString(`<gpx version="1.1" creator="MyStravaStats" xmlns="http://www.topografix.com/GPX/1/1">` + "\n")
	builder.WriteString("  <trk>\n")
	builder.WriteString("    <name>" + escapedName + "</name>\n")
	builder.WriteString("    <trkseg>\n")
	for _, point := range validPoints {
		builder.WriteString(fmt.Sprintf("      <trkpt lat=\"%.7f\" lon=\"%.7f\"></trkpt>\n", point[0], point[1]))
	}
	builder.WriteString("    </trkseg>\n")
	builder.WriteString("  </trk>\n")
	builder.WriteString("</gpx>\n")
	return builder.String(), nil
}

func sanitizeRouteFileName(input string) string {
	value := strings.ToLower(strings.TrimSpace(input))
	if value == "" {
		return ""
	}

	replacer := strings.NewReplacer(
		" ", "-",
		"/", "-",
		"\\", "-",
		":", "-",
		";", "-",
		",", "-",
		"\"", "",
		"'", "",
		"(", "",
		")", "",
		"[", "",
		"]", "",
	)
	value = replacer.Replace(value)
	value = strings.Trim(value, "-._")
	if value == "" {
		return ""
	}
	return value
}

func optionalIntValue(value *int) int {
	if value == nil {
		return 0
	}
	return *value
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

func getSegmentIDPathParam(r *http.Request) (int64, error) {
	segmentIDValue := strings.TrimSpace(mux.Vars(r)["segmentId"])
	if segmentIDValue == "" {
		return 0, fmt.Errorf("segmentId path parameter is required")
	}

	segmentID, err := strconv.ParseInt(segmentIDValue, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid segmentId: %q", segmentIDValue)
	}

	return segmentID, nil
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
