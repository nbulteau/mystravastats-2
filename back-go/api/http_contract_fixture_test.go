package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	activitiesApp "mystravastats/internal/activities/application"
	"mystravastats/internal/sourcesync"
)

func TestSharedHTTPContract(t *testing.T) {
	payload, err := os.ReadFile("../../test-fixtures/api/http-contract.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		Name, Method, Path, Body, Scenario, BodyType, BodyStatus string
		Status                                                   int
		Error, EmptyBody, ReplaceRequestId                       bool
		RequestId                                                *string
		Allow                                                    []string
	}
	if err := json.Unmarshal(payload, &cases); err != nil {
		t.Fatal(err)
	}
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			reader := &contractDetailedActivityReaderStub{}
			if tc.Scenario == "read-failure" {
				reader.err = errors.New("private storage failure")
			}
			setTestContainer(t, &container{
				listActivitiesUseCase:      activitiesApp.NewListActivitiesUseCase(&contractActivitiesReaderStub{}),
				getDetailedActivityUseCase: activitiesApp.NewGetDetailedActivityUseCase(reader),
			})
			router := NewRouter()
			router.Get("PostSourceSyncSynchronize").Handler(sourceSyncHandler(func(reason string) sourcesync.SyncResult {
				if reason != "manual" {
					t.Fatalf("unexpected reason %s", reason)
				}
				return sourcesync.SyncResult{Status: tc.BodyStatus, Reason: reason}
			}))
			// Match production ordering: static fallback must never swallow API errors.
			router.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) { t.Fatal("API reached SPA fallback") })
			request := httptest.NewRequest(tc.Method, tc.Path, strings.NewReader(tc.Body))
			request.Header.Set("Content-Type", "application/json")
			requestID := "shared-http-contract"
			if tc.RequestId != nil {
				requestID = *tc.RequestId
			}
			request.Header.Set("X-Request-Id", requestID)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != tc.Status {
				t.Fatalf("status=%d expected=%d body=%s", response.Code, tc.Status, response.Body)
			}
			if !strings.HasPrefix(response.Header().Get("Content-Type"), "application/json") {
				t.Fatalf("unexpected content type %s", response.Header())
			}
			id := response.Header().Get("X-Request-Id")
			if id == "" || (requestID != "" && !tc.ReplaceRequestId && id != requestID) {
				t.Fatalf("invalid request id %q", id)
			}
			if tc.ReplaceRequestId && id == requestID {
				t.Fatal("unsafe request id echoed")
			}
			for _, method := range tc.Allow {
				if !strings.Contains(response.Header().Get("Allow"), method) {
					t.Fatalf("missing Allow %s: %s", method, response.Header())
				}
			}
			if tc.EmptyBody {
				if response.Body.Len() != 0 {
					t.Fatal("HEAD response has body")
				}
				return
			}
			if tc.BodyType == "array" {
				var body []any
				if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil || body == nil {
					t.Fatalf("expected array, got %s", response.Body)
				}
				return
			}
			var body map[string]any
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if tc.Error {
				for _, field := range []string{"message", "description", "requestId"} {
					if value, ok := body[field].(string); !ok || value == "" {
						t.Fatalf("missing string %s: %v", field, body)
					}
				}
				if body["requestId"] != id || body["code"] != float64(1) {
					t.Fatalf("invalid error envelope %v", body)
				}
			}
			if tc.BodyStatus != "" && body["status"] != tc.BodyStatus {
				t.Fatalf("unexpected result %v", body)
			}
			if strings.Contains(response.Body.String(), "private storage failure") {
				t.Fatal("internal error leaked")
			}
		})
	}
}

func TestRequestIDOnAPIRedirect(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/activities/", nil)
	request.Header.Set("X-Request-Id", "redirect-test")
	response := httptest.NewRecorder()
	NewRouter().ServeHTTP(response, request)
	if response.Code != http.StatusMovedPermanently || response.Header().Get("X-Request-Id") != "redirect-test" {
		t.Fatalf("unexpected redirect response: status=%d headers=%v", response.Code, response.Header())
	}
}
