package stravaapi

import (
	"fmt"
	"mystravastats/domain/strava"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestRetrieveLoggedInAthlete_FailFastOnTooManyRequests(t *testing.T) {
	var calls int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/athlete" {
			http.NotFound(w, r)
			return
		}

		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	api := &StravaApi{
		accessToken: "test-token",
		properties: StravaProperties{
			URL: server.URL,
		},
		httpClient: server.Client(),
	}

	athlete, err := api.RetrieveLoggedInAthlete()
	if !IsRateLimitError(err) {
		t.Fatalf("expected rate limit error, got %v", err)
	}
	if athlete != nil {
		t.Fatalf("expected nil athlete on fail-fast 429, got %+v", athlete)
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("expected a single call before fail-fast stop, got %d", got)
	}
}

func TestGetActivitiesFailFastOnRateLimit(t *testing.T) {
	var calls int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/v3/athlete/activities") {
			http.NotFound(w, r)
			return
		}

		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	api := &StravaApi{
		accessToken: "test-token",
		properties: StravaProperties{
			URL:      server.URL,
			PageSize: 200,
		},
		httpClient: server.Client(),
	}

	_, err := api.GetActivitiesFailFastOnRateLimit(2026)
	if !IsRateLimitError(err) {
		t.Fatalf("expected rate limit error, got %v", err)
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("expected a single call before fail-fast stop, got %d", got)
	}
}

func TestGetActivitiesKeepsRetryBehaviorOnRateLimit(t *testing.T) {
	var calls int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/v3/athlete/activities") {
			http.NotFound(w, r)
			return
		}

		current := atomic.AddInt32(&calls, 1)
		if current == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `[]`)
	}))
	defer server.Close()

	api := &StravaApi{
		accessToken: "test-token",
		properties: StravaProperties{
			URL:      server.URL,
			PageSize: 200,
		},
		httpClient: server.Client(),
	}

	activities, err := api.GetActivities(2026)
	if err != nil {
		t.Fatalf("expected retry path to succeed, got error: %v", err)
	}
	if len(activities) != 0 {
		t.Fatalf("expected empty activities list, got %d", len(activities))
	}
	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Fatalf("expected 2 calls (429 then success), got %d", got)
	}
}

func TestGetDetailedActivity_FailFastOnRateLimit(t *testing.T) {
	var calls int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/v3/activities/") {
			http.NotFound(w, r)
			return
		}

		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	api := &StravaApi{
		accessToken: "test-token",
		properties: StravaProperties{
			URL:      server.URL,
			PageSize: 200,
		},
		httpClient: server.Client(),
	}

	_, err := api.GetDetailedActivity(42)
	if !IsRateLimitError(err) {
		t.Fatalf("expected rate limit error, got %v", err)
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("expected a single call for fail-fast detailed activity, got %d", got)
	}
}

func TestGetActivityStream_FailFastOnRateLimit(t *testing.T) {
	var calls int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/v3/activities/") {
			http.NotFound(w, r)
			return
		}

		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	api := &StravaApi{
		accessToken: "test-token",
		properties: StravaProperties{
			URL:      server.URL,
			PageSize: 200,
		},
		httpClient: server.Client(),
	}

	_, err := api.GetActivityStream(strava.Activity{
		Id:       42,
		UploadId: 1,
	})
	if !IsRateLimitError(err) {
		t.Fatalf("expected rate limit error, got %v", err)
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("expected a single call for fail-fast stream request, got %d", got)
	}
}
