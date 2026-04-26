package api

import (
	"encoding/json"
	"log"
	"mystravastats/internal/shared/domain/business"
	"net/http"
)

func postSourceModePreview(writer http.ResponseWriter, request *http.Request) {
	var previewRequest business.SourceModePreviewRequest
	if err := json.NewDecoder(request.Body).Decode(&previewRequest); err != nil {
		writeBadRequest(writer, "Invalid request body", err.Error())
		return
	}

	preview := getContainer().previewSourceModeUseCase.Execute(previewRequest)
	if err := writeJSON(writer, http.StatusOK, preview); err != nil {
		log.Printf("failed to write source mode preview response: %v", err)
		writeInternalServerError(writer, "Failed to encode source mode preview response")
	}
}
