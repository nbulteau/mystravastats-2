package api

import (
	"net/http"
	"time"
)

func getTrainingLoad(w http.ResponseWriter, r *http.Request) {
	_, types, err := parseActivityRequestParams(r)
	if err != nil {
		writeBadRequest(w, "Invalid request parameters", err.Error())
		return
	}
	week, err := time.Parse("2006-01-02", r.URL.Query().Get("week"))
	if err != nil {
		writeBadRequest(w, "Invalid request parameters", "week must be YYYY-MM-DD")
		return
	}
	result := getContainer().getTrainingLoadUseCase.Execute(week, types)
	if err := writeJSON(w, http.StatusOK, result); err != nil {
		writeInternalServerError(w, "Failed to encode training load")
	}
}
