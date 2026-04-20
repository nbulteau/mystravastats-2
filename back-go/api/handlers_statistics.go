package api

import (
	"log"
	"mystravastats/api/dto"
	"net/http"
)

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
func getStatisticsByActivityType(writer http.ResponseWriter, request *http.Request) {
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

	statistics := getContainer().listStatisticsUseCase.Execute(year, activityTypes)
	statisticsDto := make([]dto.StatisticDto, len(statistics))
	for i, statistic := range statistics {
		statisticsDto[i] = dto.ToStatisticDto(statistic)
	}

	if err := writeJSON(writer, http.StatusOK, statisticsDto); err != nil {
		log.Printf("failed to write statistics response: %v", err)
		writeInternalServerError(writer, "Failed to encode statistics response")
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
func getPersonalRecordsTimelineByActivityType(writer http.ResponseWriter, request *http.Request) {
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
	metric := getMetricParam(request)

	timeline := getContainer().listPersonalRecordsTimelineUseCase.Execute(year, metric, activityTypes)
	timelineDto := make([]dto.PersonalRecordTimelineDto, len(timeline))
	for i, entry := range timeline {
		timelineDto[i] = dto.ToPersonalRecordTimelineDto(entry)
	}

	if err := writeJSON(writer, http.StatusOK, timelineDto); err != nil {
		log.Printf("failed to write personal records timeline response: %v", err)
		writeInternalServerError(writer, "Failed to encode personal records timeline response")
	}
}

func getHeartRateZoneAnalysisByActivityType(writer http.ResponseWriter, request *http.Request) {
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

	analysis := getContainer().getHeartRateZoneAnalysisUseCase.Execute(year, activityTypes)
	analysisDto := dto.ToHeartRateZoneAnalysisDto(analysis)

	if err := writeJSON(writer, http.StatusOK, analysisDto); err != nil {
		log.Printf("failed to write heart rate zone analysis response: %v", err)
		writeInternalServerError(writer, "Failed to encode heart rate zone analysis response")
	}
}

// getSegmentClimbProgressionByActivityType godoc
// @Summary Get segment and climb progression
// @Description Returns progression for favorite segments and climbs
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
func getSegmentClimbProgressionByActivityType(writer http.ResponseWriter, request *http.Request) {
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
	metric := getMetricParam(request)
	targetType := getTargetTypeParam(request)
	targetId, err := getTargetIDParam(request)
	if err != nil {
		writeBadRequest(writer, "Invalid request parameters", err.Error())
		return
	}

	progression := getContainer().getSegmentClimbProgressionUseCase.Execute(year, metric, targetType, targetId, activityTypes)
	progressionDto := dto.ToSegmentClimbProgressionDto(progression)

	if err := writeJSON(writer, http.StatusOK, progressionDto); err != nil {
		log.Printf("failed to write segment/climb progression response: %v", err)
		writeInternalServerError(writer, "Failed to encode segment/climb progression response")
	}
}

func getSegmentsByActivityType(writer http.ResponseWriter, request *http.Request) {
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
	metric := getMetricParam(request)
	query := getQueryParam(request)
	from, err := getFromDateParam(request)
	if err != nil {
		writeBadRequest(writer, "Invalid request parameters", err.Error())
		return
	}
	to, err := getToDateParam(request)
	if err != nil {
		writeBadRequest(writer, "Invalid request parameters", err.Error())
		return
	}

	segments := getContainer().listSegmentsUseCase.Execute(year, metric, query, from, to, activityTypes)
	segmentsDto := make([]dto.SegmentClimbTargetSummaryDto, len(segments))
	for i, segment := range segments {
		segmentsDto[i] = dto.ToSegmentClimbTargetSummaryDto(segment)
	}

	if err := writeJSON(writer, http.StatusOK, segmentsDto); err != nil {
		log.Printf("failed to write segments response: %v", err)
		writeInternalServerError(writer, "Failed to encode segments response")
	}
}

func getSegmentEffortsByActivityType(writer http.ResponseWriter, request *http.Request) {
	segmentID, err := getSegmentIDPathParam(request)
	if err != nil {
		writeBadRequest(writer, "Invalid request parameters", err.Error())
		return
	}
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
	metric := getMetricParam(request)
	from, err := getFromDateParam(request)
	if err != nil {
		writeBadRequest(writer, "Invalid request parameters", err.Error())
		return
	}
	to, err := getToDateParam(request)
	if err != nil {
		writeBadRequest(writer, "Invalid request parameters", err.Error())
		return
	}

	efforts := getContainer().listSegmentEffortsUseCase.Execute(year, metric, segmentID, from, to, activityTypes)
	effortsDto := make([]dto.SegmentClimbAttemptDto, len(efforts))
	for i, effort := range efforts {
		effortsDto[i] = dto.ToSegmentClimbAttemptDto(effort)
	}

	if err := writeJSON(writer, http.StatusOK, effortsDto); err != nil {
		log.Printf("failed to write segment efforts response: %v", err)
		writeInternalServerError(writer, "Failed to encode segment efforts response")
	}
}

func getSegmentSummaryByActivityType(writer http.ResponseWriter, request *http.Request) {
	segmentID, err := getSegmentIDPathParam(request)
	if err != nil {
		writeBadRequest(writer, "Invalid request parameters", err.Error())
		return
	}
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
	metric := getMetricParam(request)
	from, err := getFromDateParam(request)
	if err != nil {
		writeBadRequest(writer, "Invalid request parameters", err.Error())
		return
	}
	to, err := getToDateParam(request)
	if err != nil {
		writeBadRequest(writer, "Invalid request parameters", err.Error())
		return
	}

	summary := getContainer().getSegmentSummaryUseCase.Execute(year, metric, segmentID, from, to, activityTypes)
	if summary == nil {
		writeNotFound(writer, "Segment not found", "No attempts found for this segment with current filters")
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

	if err := writeJSON(writer, http.StatusOK, response); err != nil {
		log.Printf("failed to write segment summary response: %v", err)
		writeInternalServerError(writer, "Failed to encode segment summary response")
	}
}
