package api

import (
	"log"
	"net/http"
)

// getHealthDetails godoc
// @Summary Get cache health details
// @Description Returns cache diagnostics including manifest/warmup/best-effort status
// @Tags health
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {string} string "Internal server error"
// @Router /api/health/details [get]
func getHealthDetails(writer http.ResponseWriter, _ *http.Request) {
	details := getContainer().getCacheHealthDetailsUseCase.Execute()
	if err := writeJSON(writer, http.StatusOK, details); err != nil {
		log.Printf("failed to write cache health response: %v", err)
		writeInternalServerError(writer, "Failed to encode cache health response")
	}
}
