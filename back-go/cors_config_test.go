package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCORSPreflight_AllowsConfiguredOriginWithCredentials(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://app.example")

	response := performPreflight("https://app.example", "GET", "authorization,x-request-id")

	if response.Code != http.StatusNoContent {
		t.Fatalf("expected preflight status 204, got %d", response.Code)
	}
	if got := response.Header().Get("Access-Control-Allow-Origin"); got != "https://app.example" {
		t.Fatalf("expected configured origin, got %q", got)
	}
	if got := response.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("expected credentials=true, got %q", got)
	}
	allowMethods := response.Header().Get("Access-Control-Allow-Methods")
	if !headerContains(allowMethods, "GET") {
		t.Fatalf("expected GET in allowed methods, got %q", allowMethods)
	}
	allowHeaders := response.Header().Get("Access-Control-Allow-Headers")
	if !headerContains(allowHeaders, "Authorization") || !headerContains(allowHeaders, "X-Request-Id") {
		t.Fatalf("expected Authorization and X-Request-Id in allowed headers, got %q", allowHeaders)
	}
}

func TestCORSPreflight_RejectsUnconfiguredOrigin(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://app.example")

	response := performPreflight("https://evil.example", "GET", "Authorization")

	if got := response.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("expected no allowed origin for rejected origin, got %q", got)
	}
	if got := response.Header().Get("Access-Control-Allow-Credentials"); got != "" {
		t.Fatalf("expected no credentials header for rejected origin, got %q", got)
	}
}

func performPreflight(origin string, requestMethod string, requestHeaders string) *httptest.ResponseRecorder {
	handler := newCORSHandler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	request := httptest.NewRequest(http.MethodOptions, "http://api.local/api/health/details", nil)
	request.Header.Set("Origin", origin)
	request.Header.Set("Access-Control-Request-Method", requestMethod)
	request.Header.Set("Access-Control-Request-Headers", requestHeaders)

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func headerContains(header string, expected string) bool {
	for _, part := range strings.Split(header, ",") {
		if strings.EqualFold(strings.TrimSpace(part), expected) {
			return true
		}
	}
	return false
}
