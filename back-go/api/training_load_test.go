package api

import (
	"encoding/json"
	"fmt"
	"math"
	"mystravastats/internal/shared/domain/business"
	"mystravastats/internal/shared/domain/strava"
	training "mystravastats/internal/training/application"
	"net/http/httptest"
	"os"
	"testing"
)

func TestTrainingLoadSharedHTTPFixtures(t *testing.T) {
	data, err := os.ReadFile("../../test-fixtures/api/training-load.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixtures []struct {
		Name, Week         string
		Settings           business.AthletePerformanceSettings
		Activities         []*strava.Activity
		ExpectedSummary    map[string]any
		ExpectedActivities map[string]map[string]any
		UndatedActivities  int
	}
	if err = json.Unmarshal(data, &fixtures); err != nil {
		t.Fatal(err)
	}
	outputs := []map[string]any{}
	for _, tc := range fixtures {
		t.Run(tc.Name, func(t *testing.T) {
			reader := &contractAthleteReaderStub{activities: tc.Activities, performanceSettings: tc.Settings}
			setTestContainer(t, &container{getTrainingLoadUseCase: training.NewUseCase(reader)})
			request := httptest.NewRequest("GET", "/api/statistics/training-load?activityType=Ride_Run&week="+tc.Week, nil)
			request.Header.Set("X-Request-Id", "training-fixture")
			response := httptest.NewRecorder()
			NewRouter().ServeHTTP(response, request)
			if response.Code != 200 {
				t.Fatalf("%d %s", response.Code, response.Body)
			}
			if response.Header().Get("X-Request-Id") != "training-fixture" {
				t.Fatal("request id not propagated")
			}
			var body map[string]any
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			weeks := body["weeks"].([]any)
			if len(weeks) != 5 {
				t.Fatal("need five weeks")
			}
			summary := weeks[0].(map[string]any)["summary"].(map[string]any)
			assertTrainingSubset(t, tc.ExpectedSummary, summary)
			for _, week := range weeks {
				if len(week.(map[string]any)["days"].([]any)) != 7 {
					t.Fatal("need seven explicit days")
				}
			}
			if body["undatedActivities"] != float64(tc.UndatedActivities) {
				t.Fatal("undated count mismatch")
			}
			for id, expected := range tc.ExpectedActivities {
				found := false
				for _, raw := range body["activities"].([]any) {
					row := raw.(map[string]any)
					if fmt.Sprint(row["activityId"]) == id {
						found = true
						assertTrainingSubset(t, expected, row)
					}
				}
				if !found {
					t.Fatalf("activity %s missing", id)
				}
			}
			outputs = append(outputs, map[string]any{"name": tc.Name, "operationId": "getTrainingLoad", "status": 200, "body": body})
		})
	}
	encoded, err := json.MarshalIndent(outputs, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.MkdirAll("../../test-results", 0755); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile("../../test-results/training-go-responses.json", encoded, 0644); err != nil {
		t.Fatal(err)
	}
}
func assertTrainingSubset(t *testing.T, want, got map[string]any) {
	t.Helper()
	for key, expected := range want {
		actual, exists := got[key]
		if !exists {
			t.Fatalf("missing field %s", key)
		}
		if number, ok := expected.(float64); ok {
			value, ok := actual.(float64)
			if !ok || math.Abs(number-value) > 1e-6 {
				t.Fatalf("%s got %v want %v", key, actual, expected)
			}
		} else if expected != actual {
			t.Fatalf("%s got %v want %v", key, actual, expected)
		}
	}
}
func TestTrainingLoadRejectsInvalidParameters(t *testing.T) {
	for _, query := range []string{"activityType=Ride", "activityType=Ride&week=2026-02-30", "activityType=Unknown&week=2026-10-05", "week=2026-10-05"} {
		request := httptest.NewRequest("GET", "/api/statistics/training-load?"+query, nil)
		response := httptest.NewRecorder()
		NewRouter().ServeHTTP(response, request)
		if response.Code != 400 {
			t.Fatalf("query %s returned %d", query, response.Code)
		}
	}
}
