package stravaapi

import (
	"errors"
	"fmt"
	"io"
	"mystravastats/internal/shared/domain/strava"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

type closeErrorBody struct {
	io.Reader
	closed *atomic.Bool
}

func (body *closeErrorBody) Close() error {
	body.closed.Store(true)
	return errors.New("simulated close failure")
}

func TestRetrieveLoggedInAthlete_FailFastOnTooManyRequests(t *testing.T) {
	// GIVEN
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

	// WHEN
	athlete, err := api.RetrieveLoggedInAthlete()
	apiCalls := atomic.LoadInt32(&calls)

	// THEN
	if !IsRateLimitError(err) {
		t.Fatalf("expected rate limit error, got %v", err)
	}
	if athlete != nil {
		t.Fatalf("expected nil athlete on fail-fast 429, got %+v", athlete)
	}
	if apiCalls != 1 {
		t.Fatalf("expected a single call before fail-fast stop, got %d", apiCalls)
	}
}

func TestRetrieveLoggedInAthlete_UsesConfiguredAPIBaseURLAsV3Root(t *testing.T) {
	var requestedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedPath = r.URL.Path
		if r.URL.Path != "/athlete" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"id":42,"username":"test-athlete"}`)
	}))
	defer server.Close()

	api := &StravaApi{
		accessToken: "test-token",
		properties: StravaProperties{
			APIBaseURL: server.URL,
		},
		httpClient: server.Client(),
	}

	athlete, err := api.RetrieveLoggedInAthlete()

	if err != nil {
		t.Fatalf("expected athlete request to succeed, got %v", err)
	}
	if athlete == nil || athlete.Id != 42 {
		t.Fatalf("expected athlete id 42, got %#v", athlete)
	}
	if requestedPath != "/athlete" {
		t.Fatalf("expected configured API base URL to be treated as V3 root, got path %q", requestedPath)
	}
}

func TestRetrieveLoggedInAthlete_RejectsNonSuccessStatusWithBoundedBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = fmt.Fprint(w, "upstream unavailable")
	}))
	defer server.Close()

	api := &StravaApi{
		accessToken: "test-token",
		properties:  StravaProperties{APIBaseURL: server.URL},
		httpClient:  server.Client(),
	}

	athlete, err := api.RetrieveLoggedInAthlete()

	if athlete != nil {
		t.Fatalf("expected no athlete for HTTP 502, got %#v", athlete)
	}
	if err == nil || !strings.Contains(err.Error(), "HTTP 502") || !strings.Contains(err.Error(), "upstream unavailable") {
		t.Fatalf("expected explicit HTTP status error, got %v", err)
	}
}

func TestRetrieveLoggedInAthlete_TimesOut(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(100 * time.Millisecond)
		_, _ = fmt.Fprint(w, `{"id":42}`)
	}))
	defer server.Close()

	api := &StravaApi{
		accessToken:    "test-token",
		properties:     StravaProperties{APIBaseURL: server.URL},
		httpClient:     server.Client(),
		requestTimeout: 10 * time.Millisecond,
	}

	startedAt := time.Now()
	athlete, err := api.RetrieveLoggedInAthlete()

	if athlete != nil {
		t.Fatalf("expected no athlete after timeout, got %#v", athlete)
	}
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "timeout") && !strings.Contains(strings.ToLower(err.Error()), "deadline") {
		t.Fatalf("expected timeout error, got %v", err)
	}
	if elapsed := time.Since(startedAt); elapsed > 80*time.Millisecond {
		t.Fatalf("expected request timeout before server response, took %s", elapsed)
	}
}

func TestRetrieveLoggedInAthlete_CloseFailureDoesNotFailRequest(t *testing.T) {
	var closed atomic.Bool
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body: &closeErrorBody{
				Reader: strings.NewReader(`{"id":42,"username":"test-athlete"}`),
				closed: &closed,
			},
		}, nil
	})}
	api := &StravaApi{
		accessToken: "test-token",
		properties:  StravaProperties{APIBaseURL: "https://strava.test/api/v3"},
		httpClient:  client,
	}

	athlete, err := api.RetrieveLoggedInAthlete()

	if err != nil {
		t.Fatalf("expected close failure to remain non-fatal, got %v", err)
	}
	if athlete == nil || athlete.Id != 42 {
		t.Fatalf("expected decoded athlete, got %#v", athlete)
	}
	if !closed.Load() {
		t.Fatalf("expected response body to be closed")
	}
}

func TestRetrieveLoggedInAthlete_ReportsRequestBuildError(t *testing.T) {
	api := &StravaApi{
		accessToken: "test-token",
		properties:  StravaProperties{APIBaseURL: "://invalid"},
		httpClient:  http.DefaultClient,
	}

	athlete, err := api.RetrieveLoggedInAthlete()

	if athlete != nil {
		t.Fatalf("expected no athlete for malformed URL, got %#v", athlete)
	}
	if err == nil || !strings.Contains(err.Error(), "request build error") {
		t.Fatalf("expected request build error, got %v", err)
	}
}

func TestGetActivitiesFailFastOnRateLimit(t *testing.T) {
	// GIVEN
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

	// WHEN
	_, err := api.GetActivitiesFailFastOnRateLimit(2026)
	apiCalls := atomic.LoadInt32(&calls)

	// THEN
	if !IsRateLimitError(err) {
		t.Fatalf("expected rate limit error, got %v", err)
	}
	if apiCalls != 1 {
		t.Fatalf("expected a single call before fail-fast stop, got %d", apiCalls)
	}
}

func TestGetActivitiesKeepsRetryBehaviorOnRateLimit(t *testing.T) {
	// GIVEN
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

	// WHEN
	activities, err := api.GetActivities(2026)
	apiCalls := atomic.LoadInt32(&calls)

	// THEN
	if err != nil {
		t.Fatalf("expected retry path to succeed, got error: %v", err)
	}
	if len(activities) != 0 {
		t.Fatalf("expected empty activities list, got %d", len(activities))
	}
	if apiCalls != 2 {
		t.Fatalf("expected 2 calls (429 then success), got %d", apiCalls)
	}
}

func TestGetDetailedActivity_FailFastOnRateLimit(t *testing.T) {
	// GIVEN
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

	// WHEN
	_, err := api.GetDetailedActivity(42)
	apiCalls := atomic.LoadInt32(&calls)

	// THEN
	if !IsRateLimitError(err) {
		t.Fatalf("expected rate limit error, got %v", err)
	}
	if apiCalls != 1 {
		t.Fatalf("expected a single call for fail-fast detailed activity, got %d", apiCalls)
	}
}

func TestGetActivityStream_FailFastOnRateLimit(t *testing.T) {
	// GIVEN
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

	// WHEN
	_, err := api.GetActivityStream(strava.Activity{
		Id:       42,
		UploadId: 1,
	})
	apiCalls := atomic.LoadInt32(&calls)

	// THEN
	if !IsRateLimitError(err) {
		t.Fatalf("expected rate limit error, got %v", err)
	}
	if apiCalls != 1 {
		t.Fatalf("expected a single call for fail-fast stream request, got %d", apiCalls)
	}
}

func TestAuthorizationURLIncludesStateAndScopes(t *testing.T) {
	api := &StravaApi{
		properties: StravaProperties{
			URL: "https://www.strava.com",
		},
	}

	authURL := api.authorizationURL("12345", "http://localhost:8090/exchange_token", "state-token")

	if !strings.HasPrefix(authURL, "https://www.strava.com/oauth/authorize?") {
		t.Fatalf("unexpected authorize URL: %s", authURL)
	}
	for _, expected := range []string{
		"client_id=12345",
		"response_type=code",
		"approval_prompt=auto",
		"state=state-token",
		"scope=read_all%2Cactivity%3Aread_all%2Cprofile%3Aread_all",
	} {
		if !strings.Contains(authURL, expected) {
			t.Fatalf("expected authorize URL to contain %q, got %s", expected, authURL)
		}
	}
}

func TestUsePersistedTokenWhenStillValid(t *testing.T) {
	tokenPath := filepath.Join(t.TempDir(), ".strava-token.json")
	writeTokenFixture(t, tokenPath, fmt.Sprintf(`{
		"access_token":"persisted-token",
		"refresh_token":"refresh-token",
		"expires_at":%d
	}`, time.Now().Add(2*time.Hour).Unix()))
	api := &StravaApi{
		tokenStore:  tokenPath,
		httpClient:  http.DefaultClient,
		properties:  StravaProperties{URL: "https://www.strava.com"},
		accessToken: "",
	}

	used, err := api.usePersistedTokenIfAvailable("12345", "secret")

	if err != nil {
		t.Fatalf("expected persisted token to be usable, got error: %v", err)
	}
	if !used {
		t.Fatalf("expected persisted token to be used")
	}
	if api.accessToken != "persisted-token" {
		t.Fatalf("expected persisted access token, got %q", api.accessToken)
	}
}

func TestUsePersistedTokenRefreshesExpiredToken(t *testing.T) {
	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/oauth/token" {
			http.NotFound(w, r)
			return
		}
		atomic.AddInt32(&calls, 1)
		if err := r.ParseForm(); err != nil {
			t.Fatalf("unable to parse form: %v", err)
		}
		if got := r.Form.Get("grant_type"); got != "refresh_token" {
			t.Fatalf("expected refresh_token grant, got %q", got)
		}
		if got := r.Form.Get("refresh_token"); got != "old-refresh-token" {
			t.Fatalf("expected old refresh token, got %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{
			"token_type":"Bearer",
			"access_token":"refreshed-token",
			"refresh_token":"new-refresh-token",
			"expires_at":%d,
			"expires_in":21600
		}`, time.Now().Add(6*time.Hour).Unix())
	}))
	defer server.Close()

	tokenPath := filepath.Join(t.TempDir(), ".strava-token.json")
	writeTokenFixture(t, tokenPath, fmt.Sprintf(`{
		"access_token":"expired-token",
		"refresh_token":"old-refresh-token",
		"expires_at":%d
	}`, time.Now().Add(-time.Hour).Unix()))
	api := &StravaApi{
		tokenStore: tokenPath,
		httpClient: server.Client(),
		properties: StravaProperties{URL: server.URL},
	}

	used, err := api.usePersistedTokenIfAvailable("12345", "secret")

	if err != nil {
		t.Fatalf("expected expired token to refresh, got error: %v", err)
	}
	if !used {
		t.Fatalf("expected refreshed token to be used")
	}
	if api.accessToken != "refreshed-token" {
		t.Fatalf("expected refreshed access token, got %q", api.accessToken)
	}
	if atomic.LoadInt32(&calls) != 1 {
		t.Fatalf("expected one refresh call, got %d", calls)
	}

	data, err := os.ReadFile(tokenPath)
	if err != nil {
		t.Fatalf("expected refreshed token file: %v", err)
	}
	if !strings.Contains(string(data), `"refresh_token": "new-refresh-token"`) {
		t.Fatalf("expected refreshed token file to contain new refresh token, got %s", string(data))
	}
}

func writeTokenFixture(t *testing.T, path string, payload string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(payload), 0o600); err != nil {
		t.Fatalf("unable to write token fixture: %v", err)
	}
}
