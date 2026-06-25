package api

import (
	"encoding/json"
	"io"
	"log"
	"mystravastats/api/dto"
	athleteApp "mystravastats/internal/athlete/application"
	"mystravastats/internal/shared/domain/business"
	"net/http"
	"strconv"
	"strings"
)

// getAthlete godoc
// @Summary Get athlete information
// @Description Returns the current athlete information
// @Tags athlete
// @Produce json
// @Success 200 {object} dto.AthleteDto
// @Failure 500 {string} string "Internal server error"
// @Router /api/athletes/me [get]
func getAthlete(writer http.ResponseWriter, _ *http.Request) {
	athlete := getContainer().getAthleteUseCase.Execute()
	athleteDto := dto.ToAthleteDto(athlete)
	if err := writeJSON(writer, http.StatusOK, athleteDto); err != nil {
		log.Printf("failed to write athlete response: %v", err)
		writeInternalServerError(writer, "Failed to encode athlete response")
	}
}

func getAthletePerformanceSettings(writer http.ResponseWriter, _ *http.Request) {
	settings := getContainer().getPerformanceSettingsUseCase.Execute()
	settingsDto := dto.ToAthletePerformanceSettingsDto(settings)
	if err := writeJSON(writer, http.StatusOK, settingsDto); err != nil {
		log.Printf("failed to write performance settings response: %v", err)
		writeInternalServerError(writer, "Failed to encode performance settings response")
	}
}

func getAthleteFtpEstimate(writer http.ResponseWriter, request *http.Request) {
	activityTypes, err := getOptionalFtpEstimateActivityTypes(request)
	if err != nil {
		writeBadRequest(writer, "Invalid activity type", err.Error())
		return
	}

	windowDays, err := getOptionalFtpEstimateWindowDays(request)
	if err != nil {
		writeBadRequest(writer, "Invalid days", err.Error())
		return
	}

	estimate := getContainer().getFtpEstimateUseCase.Execute(activityTypes, windowDays)
	estimateDto := dto.ToFtpEstimateDto(estimate)
	if err := writeJSON(writer, http.StatusOK, estimateDto); err != nil {
		log.Printf("failed to write FTP estimate response: %v", err)
		writeInternalServerError(writer, "Failed to encode FTP estimate response")
	}
}

func putAthletePerformanceSettings(writer http.ResponseWriter, request *http.Request) {
	defer func(Body io.ReadCloser) {
		if err := Body.Close(); err != nil {
			log.Printf("failed to close request body: %v", err)
		}
	}(request.Body)

	var settingsDto dto.AthletePerformanceSettingsDto
	if err := json.NewDecoder(request.Body).Decode(&settingsDto); err != nil {
		writeBadRequest(writer, "Invalid request body", "performance settings payload is invalid")
		return
	}

	settings := dto.ToAthletePerformanceSettings(settingsDto)
	updatedSettings := getContainer().updatePerformanceSettingsUseCase.Execute(settings)
	updatedSettingsDto := dto.ToAthletePerformanceSettingsDto(updatedSettings)

	if err := writeJSON(writer, http.StatusOK, updatedSettingsDto); err != nil {
		log.Printf("failed to write updated performance settings response: %v", err)
		writeInternalServerError(writer, "Failed to encode updated performance settings response")
	}
}

func getOptionalFtpEstimateActivityTypes(request *http.Request) ([]business.ActivityType, error) {
	if strings.TrimSpace(request.URL.Query().Get("activityType")) == "" {
		return athleteApp.DefaultFtpEstimateActivityTypes(), nil
	}
	return getActivityTypeParam(request)
}

func getOptionalFtpEstimateWindowDays(request *http.Request) (int, error) {
	value := strings.TrimSpace(request.URL.Query().Get("days"))
	if value == "" {
		return 0, nil
	}
	days, err := strconv.Atoi(value)
	if err != nil {
		return 0, err
	}
	return days, nil
}

func getAthleteHeartRateZones(writer http.ResponseWriter, _ *http.Request) {
	settings := getContainer().getHeartRateZoneSettingsUseCase.Execute()
	settingsDto := dto.ToHeartRateZoneSettingsDto(settings)
	if err := writeJSON(writer, http.StatusOK, settingsDto); err != nil {
		log.Printf("failed to write heart rate settings response: %v", err)
		writeInternalServerError(writer, "Failed to encode heart rate settings response")
	}
}

func putAthleteHeartRateZones(writer http.ResponseWriter, request *http.Request) {
	defer func(Body io.ReadCloser) {
		if err := Body.Close(); err != nil {
			log.Printf("failed to close request body: %v", err)
		}
	}(request.Body)

	var settingsDto dto.HeartRateZoneSettingsDto
	if err := json.NewDecoder(request.Body).Decode(&settingsDto); err != nil {
		writeBadRequest(writer, "Invalid request body", "heart rate zone settings payload is invalid")
		return
	}

	settings := dto.ToHeartRateZoneSettings(settingsDto)
	updatedSettings := getContainer().updateHeartRateZoneSettingsUseCase.Execute(settings)
	updatedSettingsDto := dto.ToHeartRateZoneSettingsDto(updatedSettings)

	if err := writeJSON(writer, http.StatusOK, updatedSettingsDto); err != nil {
		log.Printf("failed to write updated heart rate settings response: %v", err)
		writeInternalServerError(writer, "Failed to encode updated heart rate settings response")
	}
}
