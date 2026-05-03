package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPostStravaOAuthStartWritesCredentialsAndReturnsAuthorizeURL(t *testing.T) {
	// GIVEN
	root := t.TempDir()
	body := strings.NewReader(`{"path":` + quoteJSON(root) + `,"clientId":"12345","clientSecret":"secret","useCache":false}`)
	request := httptest.NewRequest(http.MethodPost, "/api/source-modes/strava/oauth/start", body)
	recorder := httptest.NewRecorder()

	// WHEN
	postStravaOAuthStart(recorder, request)

	// THEN
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	var result struct {
		Status       string `json:"status"`
		AuthorizeURL string `json:"authorizeUrl"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON response: %v", err)
	}
	if result.Status != "oauth_started" || !strings.Contains(result.AuthorizeURL, "state=") {
		t.Fatalf("expected OAuth start result, got %#v", result)
	}
	content, err := os.ReadFile(filepath.Join(root, ".strava"))
	if err != nil {
		t.Fatalf("expected .strava to be written: %v", err)
	}
	if !strings.Contains(string(content), "clientId=12345") || !strings.Contains(string(content), "useCache=false") {
		t.Fatalf("unexpected .strava content: %s", string(content))
	}
}

func TestPostStravaOAuthStartSupportsCacheOnly(t *testing.T) {
	// GIVEN
	root := t.TempDir()
	body := strings.NewReader(`{"path":` + quoteJSON(root) + `,"clientId":"12345","useCache":true}`)
	request := httptest.NewRequest(http.MethodPost, "/api/source-modes/strava/oauth/start", body)
	recorder := httptest.NewRecorder()

	// WHEN
	postStravaOAuthStart(recorder, request)

	// THEN
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	var result struct {
		Status       string `json:"status"`
		AuthorizeURL string `json:"authorizeUrl"`
		CacheOnly    bool   `json:"cacheOnly"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON response: %v", err)
	}
	if result.Status != "cache_only" || result.AuthorizeURL != "" || !result.CacheOnly {
		t.Fatalf("expected cache-only result, got %#v", result)
	}
}

func quoteJSON(value string) string {
	data, _ := json.Marshal(value)
	return string(data)
}
