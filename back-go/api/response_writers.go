package api

import (
	"bytes"
	"encoding/json"
	"log"
	"mystravastats/api/dto"
	"net/http"
)

func writeJSON(writer http.ResponseWriter, status int, v any) error {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(true)

	if err := enc.Encode(v); err != nil {
		return err
	}

	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.WriteHeader(status)
	_, _ = writer.Write(buf.Bytes())
	return nil
}

func writeBadRequest(writer http.ResponseWriter, message string, description string) {
	writeAPIError(writer, http.StatusBadRequest, message, description)
}

func writeNotFound(writer http.ResponseWriter, message string, description string) {
	writeAPIError(writer, http.StatusNotFound, message, description)
}

func writeInternalServerError(writer http.ResponseWriter, description string) {
	writeAPIError(writer, http.StatusInternalServerError, "Internal server error", description)
}

func writeAPIError(writer http.ResponseWriter, status int, message string, description string) {
	if err := writeJSON(writer, status, dto.ErrorResponseMessageDto{
		Message:     message,
		Description: description,
		Code:        1,
	}); err != nil {
		log.Printf("failed to write API error response: %v", err)
	}
}
