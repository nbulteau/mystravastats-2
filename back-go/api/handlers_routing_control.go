package api

import (
	"errors"
	"log"
	"net/http"

	routingControlInfra "mystravastats/internal/routingcontrol/infrastructure"
)

func postOSRMStart(writer http.ResponseWriter, request *http.Request) {
	result, err := getContainer().osrmControl.StartOSRM(request.Context())
	if err != nil {
		var controlErr routingControlInfra.OSRMControlError
		if errors.As(err, &controlErr) {
			writeAPIError(writer, controlErr.StatusCode, "OSRM control unavailable", controlErr.Description)
			return
		}
		writeInternalServerError(writer, err.Error())
		return
	}
	if err := writeJSON(writer, http.StatusOK, result); err != nil {
		log.Printf("failed to write OSRM start response: %v", err)
		writeInternalServerError(writer, "Failed to encode OSRM start response")
	}
}
