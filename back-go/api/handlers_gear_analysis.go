package api

import (
	"encoding/json"
	"log"
	"mystravastats/api/dto"
	"net/http"

	"github.com/gorilla/mux"
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

func postGearMaintenanceRecord(writer http.ResponseWriter, request *http.Request) {
	payload := dto.GearMaintenanceRecordRequestDto{}
	if request.Body != nil {
		defer request.Body.Close()
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			writeBadRequest(writer, "Invalid request body", err.Error())
			return
		}
	}

	record, err := getContainer().saveGearMaintenanceRecordUseCase.Execute(dto.ToGearMaintenanceRecordRequest(payload))
	if err != nil {
		writeBadRequest(writer, "Invalid gear maintenance record", err.Error())
		return
	}
	if err := writeJSON(writer, http.StatusCreated, dto.ToGearMaintenanceRecordDto(record)); err != nil {
		log.Printf("failed to write gear maintenance response: %v", err)
		writeInternalServerError(writer, "Failed to encode gear maintenance response")
	}
}

func deleteGearMaintenanceRecord(writer http.ResponseWriter, request *http.Request) {
	recordID := mux.Vars(request)["recordId"]
	if err := getContainer().deleteGearMaintenanceRecordUseCase.Execute(recordID); err != nil {
		writeBadRequest(writer, "Invalid gear maintenance record", err.Error())
		return
	}
	writer.WriteHeader(http.StatusNoContent)
}
