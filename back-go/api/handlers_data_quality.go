package api

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"mystravastats/internal/shared/domain/business"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

func getDataQualityIssues(writer http.ResponseWriter, _ *http.Request) {
	report := getContainer().getDataQualityReportUseCase.Execute()
	writer.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(writer).Encode(report); err != nil {
		log.Printf("failed to write data quality response: %v", err)
		writeInternalServerError(writer, "Failed to encode data quality response")
	}
}

func putDataQualityStatsExclusion(writer http.ResponseWriter, request *http.Request) {
	activityID, err := strconv.ParseInt(mux.Vars(request)["activityId"], 10, 64)
	if err != nil || activityID <= 0 {
		writeBadRequest(writer, "Invalid request parameters", "invalid activityId")
		return
	}

	payload := business.DataQualityExclusionRequest{}
	if request.Body != nil {
		defer request.Body.Close()
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil && !errors.Is(err, io.EOF) {
			writeBadRequest(writer, "Invalid request body", err.Error())
			return
		}
	}

	report, err := getContainer().excludeActivityFromStatsUseCase.Execute(activityID, payload.Reason)
	if err != nil {
		writeBadRequest(writer, "Invalid data quality exclusion", err.Error())
		return
	}
	if err := writeJSON(writer, http.StatusOK, report); err != nil {
		log.Printf("failed to write data quality exclusion response: %v", err)
		writeInternalServerError(writer, "Failed to encode data quality response")
	}
}

func deleteDataQualityStatsExclusion(writer http.ResponseWriter, request *http.Request) {
	activityID, err := strconv.ParseInt(mux.Vars(request)["activityId"], 10, 64)
	if err != nil || activityID <= 0 {
		writeBadRequest(writer, "Invalid request parameters", "invalid activityId")
		return
	}

	report, err := getContainer().includeActivityInStatsUseCase.Execute(activityID)
	if err != nil {
		writeBadRequest(writer, "Invalid data quality exclusion", err.Error())
		return
	}
	if err := writeJSON(writer, http.StatusOK, report); err != nil {
		log.Printf("failed to write data quality inclusion response: %v", err)
		writeInternalServerError(writer, "Failed to encode data quality response")
	}
}

func getDataQualityCorrectionPreview(writer http.ResponseWriter, request *http.Request) {
	issueID := mux.Vars(request)["issueId"]
	preview, err := getContainer().previewDataQualityCorrectionUseCase.Execute(issueID)
	if err != nil {
		writeBadRequest(writer, "Invalid data quality correction preview", err.Error())
		return
	}
	if err := writeJSON(writer, http.StatusOK, preview); err != nil {
		log.Printf("failed to write data quality correction preview response: %v", err)
		writeInternalServerError(writer, "Failed to encode data quality correction preview response")
	}
}

func getDataQualitySafeCorrectionPreview(writer http.ResponseWriter, _ *http.Request) {
	preview := getContainer().previewSafeDataQualityCorrectionsUseCase.Execute()
	if err := writeJSON(writer, http.StatusOK, preview); err != nil {
		log.Printf("failed to write data quality safe correction preview response: %v", err)
		writeInternalServerError(writer, "Failed to encode data quality correction preview response")
	}
}

func postDataQualityCorrection(writer http.ResponseWriter, request *http.Request) {
	issueID := mux.Vars(request)["issueId"]
	report, err := getContainer().applyDataQualityCorrectionUseCase.Execute(issueID)
	if err != nil {
		writeBadRequest(writer, "Invalid data quality correction", err.Error())
		return
	}
	if err := writeJSON(writer, http.StatusOK, report); err != nil {
		log.Printf("failed to write data quality correction response: %v", err)
		writeInternalServerError(writer, "Failed to encode data quality response")
	}
}

func postDataQualitySafeCorrections(writer http.ResponseWriter, _ *http.Request) {
	report, err := getContainer().applySafeDataQualityCorrectionsUseCase.Execute()
	if err != nil {
		writeBadRequest(writer, "Invalid data quality safe corrections", err.Error())
		return
	}
	if err := writeJSON(writer, http.StatusOK, report); err != nil {
		log.Printf("failed to write data quality safe corrections response: %v", err)
		writeInternalServerError(writer, "Failed to encode data quality response")
	}
}

func deleteDataQualityCorrection(writer http.ResponseWriter, request *http.Request) {
	correctionID := mux.Vars(request)["correctionId"]
	report, err := getContainer().revertDataQualityCorrectionUseCase.Execute(correctionID)
	if err != nil {
		writeBadRequest(writer, "Invalid data quality correction revert", err.Error())
		return
	}
	if err := writeJSON(writer, http.StatusOK, report); err != nil {
		log.Printf("failed to write data quality correction revert response: %v", err)
		writeInternalServerError(writer, "Failed to encode data quality response")
	}
}
