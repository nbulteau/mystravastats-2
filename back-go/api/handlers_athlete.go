package api

import (
	"encoding/json"
	"io"
	"log"
	"mystravastats/api/dto"
	"net/http"
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
