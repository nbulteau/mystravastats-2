package api

import (
	"log"
	"net/http"

	"mystravastats/internal/sourcesync"
)

func postSourceSyncSynchronize(writer http.ResponseWriter, request *http.Request) {
	sourceSyncHandler(sourcesync.Synchronize).ServeHTTP(writer, request)
}

func sourceSyncHandler(synchronize func(string) sourcesync.SyncResult) http.HandlerFunc {
	return func(writer http.ResponseWriter, _ *http.Request) {
		result := synchronize("manual")
		status := http.StatusOK
		if result.Status == "failed" {
			status = http.StatusInternalServerError
		}
		if err := writeJSON(writer, status, result); err != nil {
			log.Printf("failed to write source synchronization response: %v", err)
			writeInternalServerError(writer, "Failed to encode source synchronization response")
		}
	}
}
