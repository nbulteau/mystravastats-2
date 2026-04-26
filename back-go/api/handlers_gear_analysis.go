package api

import (
	"log"
	"mystravastats/api/dto"
	"net/http"
)

func getGearAnalysisByActivityType(writer http.ResponseWriter, request *http.Request) {
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

	analysis := getContainer().getGearAnalysisUseCase.Execute(year, activityTypes)
	analysisDto := dto.ToGearAnalysisDto(analysis)

	if err := writeJSON(writer, http.StatusOK, analysisDto); err != nil {
		log.Printf("failed to write gear analysis response: %v", err)
		writeInternalServerError(writer, "Failed to encode gear analysis response")
	}
}
