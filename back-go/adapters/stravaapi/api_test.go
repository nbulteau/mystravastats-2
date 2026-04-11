package stravaapi

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestRetrieveLoggedInAthlete_RetriesOnTooManyRequests(t *testing.T) {
	var calls int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/athlete" {
			http.NotFound(w, r)
			return
		}

		current := atomic.AddInt32(&calls, 1)
		if current == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"id":7,"username":"nico"}`)
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
	if err != nil {
		t.Fatalf("expected retry to succeed, got error: %v", err)
	}
	if athlete == nil || athlete.Id != 7 {
		t.Fatalf("expected athlete id 7, got %+v", athlete)
	}
	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Fatalf("expected 2 calls (429 then success), got %d", got)
	}
}
