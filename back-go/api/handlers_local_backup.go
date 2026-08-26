package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"mystravastats/internal/localbackup"
	"mystravastats/internal/platform/activityprovider"
)

const maximumLocalBackupBytes = 5 << 20

func getLocalDataBackup(writer http.ResponseWriter, _ *http.Request) {
	provider := activityprovider.Get()
	bundle, err := localbackup.Export(provider.CacheRootPath(), provider.ClientID())
	if err != nil {
		writeInternalServerError(writer, err.Error())
		return
	}
	writer.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="mystravastats-local-data-%s.json"`, provider.ClientID()))
	if err := writeJSON(writer, http.StatusOK, bundle); err != nil {
		writeInternalServerError(writer, "Failed to encode local data backup")
	}
}

func postLocalDataRestore(writer http.ResponseWriter, request *http.Request) {
	defer request.Body.Close()
	decoder := json.NewDecoder(http.MaxBytesReader(writer, request.Body, maximumLocalBackupBytes))
	decoder.DisallowUnknownFields()
	bundle := localbackup.Bundle{}
	if err := decoder.Decode(&bundle); err != nil {
		writeBadRequest(writer, "Invalid local data backup", err.Error())
		return
	}
	provider := activityprovider.Get()
	result, err := localbackup.Restore(provider.CacheRootPath(), provider.ClientID(), bundle)
	if err != nil {
		writeBadRequest(writer, "Local data restore rejected", err.Error())
		return
	}
	if err := writeJSON(writer, http.StatusOK, result); err != nil {
		writeInternalServerError(writer, "Failed to encode local data restore result")
	}
}
