package api

import (
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestWriteJSON_SanitizesNonFiniteFloatsAndNilLists(t *testing.T) {
	// GIVEN
	payload := struct {
		At     time.Time `json:"at"`
		Value  float64   `json:"value"`
		Values []float64 `json:"values"`
		Items  []string  `json:"items"`
	}{
		At:     time.Date(2026, 4, 26, 8, 0, 0, 0, time.UTC),
		Value:  math.NaN(),
		Values: []float64{1, math.Inf(1), math.Inf(-1)},
		Items:  nil,
	}
	recorder := httptest.NewRecorder()

	// WHEN
	err := writeJSON(recorder, http.StatusOK, payload)

	// THEN
	if err != nil {
		t.Fatalf("expected JSON write to succeed, got %v", err)
	}
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}

	var decoded struct {
		At     string    `json:"at"`
		Value  float64   `json:"value"`
		Values []float64 `json:"values"`
		Items  []string  `json:"items"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("expected valid JSON, got %v: %s", err, recorder.Body.String())
	}
	if decoded.Value != 0 {
		t.Fatalf("expected NaN to be sanitized to zero, got %.2f", decoded.Value)
	}
	if len(decoded.Values) != 3 || decoded.Values[1] != 0 || decoded.Values[2] != 0 {
		t.Fatalf("expected infinities to be sanitized to zero, got %#v", decoded.Values)
	}
	if decoded.Items == nil {
		t.Fatalf("expected nil slice to be encoded as an empty JSON list")
	}
	if decoded.At == "" {
		t.Fatalf("expected time marshaler to be preserved")
	}
}
