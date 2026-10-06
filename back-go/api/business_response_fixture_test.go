package api

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"mystravastats/api/dto"
	statisticsDomain "mystravastats/domain/statistics"
	activitiesApp "mystravastats/internal/activities/application"
	athleteApp "mystravastats/internal/athlete/application"
	heartRateApp "mystravastats/internal/heartrate/application"
	"mystravastats/internal/shared/domain/business"
	"mystravastats/internal/shared/domain/strava"
	statisticsApp "mystravastats/internal/statistics/application"
)

type responseFixtureStatistic struct{ dto.StatisticDto }

func (s responseFixtureStatistic) Label() string { return s.StatisticDto.Label }
func (s responseFixtureStatistic) Value() string { return s.StatisticDto.Value }
func (s responseFixtureStatistic) Activity() *business.ActivityShort {
	a := s.StatisticDto.Activity
	if a == nil {
		return nil
	}
	return &business.ActivityShort{Id: a.ID, Name: a.Name, Type: business.ActivityTypes[a.Type]}
}

func TestSharedBusinessResponses(t *testing.T) {
	data, err := os.ReadFile("../../test-fixtures/api/business-responses.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		Name, OperationID, Method, Path string
		Status                          int
		Activity                        *strava.DetailedActivity
		Activities                      []*strava.Activity
		Settings                        dto.AthletePerformanceSettingsDto
		Statistics                      []dto.StatisticDto
		Timeline                        []dto.PersonalRecordTimelineDto
		HeartRate                       business.HeartRateZoneAnalysis
		RequestBody                     json.RawMessage
		Expected                        any
	}
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	outputs := make([]map[string]any, 0, len(cases))
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			athlete := &contractAthleteReaderStub{activities: tc.Activities, performanceSettings: dto.ToAthletePerformanceSettings(tc.Settings)}
			statistics := make([]statisticsDomain.Statistic, len(tc.Statistics))
			for i, stat := range tc.Statistics {
				statistics[i] = responseFixtureStatistic{stat}
			}
			timeline := make([]business.PersonalRecordTimelineEntry, len(tc.Timeline))
			for i, row := range tc.Timeline {
				timeline[i] = business.PersonalRecordTimelineEntry{MetricKey: row.MetricKey, MetricLabel: row.MetricLabel, ActivityDate: row.ActivityDate, Value: row.Value, PreviousValue: row.PreviousValue, Improvement: row.Improvement, Activity: business.ActivityShort{Id: row.Activity.ID, Name: row.Activity.Name, Type: business.ActivityTypes[row.Activity.Type]}}
			}
			setTestContainer(t, &container{
				listActivitiesUseCase:              activitiesApp.NewListActivitiesUseCase(&contractActivitiesReaderStub{activities: tc.Activities}),
				getDetailedActivityUseCase:         activitiesApp.NewGetDetailedActivityUseCase(&contractDetailedActivityReaderStub{activity: tc.Activity}),
				getPerformanceSettingsUseCase:      athleteApp.NewGetPerformanceSettingsUseCase(athlete),
				updatePerformanceSettingsUseCase:   athleteApp.NewUpdatePerformanceSettingsUseCase(athlete),
				getFtpEstimateUseCase:              athleteApp.NewGetFtpEstimateUseCase(athlete),
				listStatisticsUseCase:              statisticsApp.NewListStatisticsUseCase(&contractStatisticsReaderStub{statistics: statistics}),
				listPersonalRecordsTimelineUseCase: statisticsApp.NewListPersonalRecordsTimelineUseCase(&contractPersonalRecordsTimelineReaderStub{timeline: timeline}),
				getHeartRateZoneAnalysisUseCase:    heartRateApp.NewGetHeartRateZoneAnalysisUseCase(&contractHeartRateReaderStub{analysis: tc.HeartRate}),
			})
			request := httptest.NewRequest(tc.Method, tc.Path, strings.NewReader(string(tc.RequestBody)))
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			NewRouter().ServeHTTP(response, request)
			if response.Code != tc.Status {
				t.Fatalf("status=%d body=%s", response.Code, response.Body)
			}
			var actual any
			if err := json.Unmarshal(response.Body.Bytes(), &actual); err != nil {
				t.Fatal(err)
			}
			compareBusinessJSON(t, tc.Expected, actual, "$")
			outputs = append(outputs, map[string]any{"name": tc.Name, "operationId": tc.OperationID, "status": response.Code, "body": actual})
			if tc.Method == "PUT" {
				stored, _ := json.Marshal(dto.ToAthletePerformanceSettingsDto(athlete.performanceSettings))
				var persisted any
				if err := json.Unmarshal(stored, &persisted); err != nil {
					t.Fatal(err)
				}
				compareBusinessJSON(t, tc.Expected, persisted, "stored")
			}
		})
	}
	if t.Failed() {
		return
	}
	output := "../../test-results/api-go-responses.json"
	if err := os.MkdirAll(filepath.Dir(output), 0755); err != nil {
		t.Fatal(err)
	}
	encoded, err := json.MarshalIndent(outputs, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(output, encoded, 0644); err != nil {
		t.Fatal(err)
	}
}

// Missing and null optional properties are equivalent on the legacy APIs.
// The separate OpenAPI validation step still enforces required nullable fields.
func compareBusinessJSON(t *testing.T, expected, actual any, path string) {
	t.Helper()
	switch e := expected.(type) {
	case map[string]any:
		a, ok := actual.(map[string]any)
		if !ok {
			t.Fatalf("%s: expected object, got %v", path, actual)
		}
		for key, value := range e {
			compareBusinessJSON(t, value, a[key], path+"."+key)
		}
		for key, value := range a {
			if _, found := e[key]; !found && value != nil {
				t.Errorf("%s: unexpected field %s=%v", path, key, value)
			}
		}
	case []any:
		a, ok := actual.([]any)
		if !ok || len(a) != len(e) {
			t.Fatalf("%s: expected %v, got %v", path, expected, actual)
		}
		for i := range e {
			compareBusinessJSON(t, e[i], a[i], fmt.Sprintf("%s[%d]", path, i))
		}
	case float64:
		a, ok := actual.(float64)
		if !ok || math.Abs(e-a) > 1e-6 {
			t.Errorf("%s: expected %v, got %v", path, expected, actual)
		}
	default:
		if !reflect.DeepEqual(expected, actual) {
			t.Errorf("%s: expected %v, got %v", path, expected, actual)
		}
	}
}
