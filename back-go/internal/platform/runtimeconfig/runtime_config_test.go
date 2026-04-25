package runtimeconfig

import (
	"reflect"
	"testing"
)

func TestDetails_DefaultsToStravaRuntimeConfig(t *testing.T) {
	t.Setenv("STRAVA_CACHE_PATH", "")
	t.Setenv("FIT_FILES_PATH", "")
	t.Setenv("GPX_FILES_PATH", "")
	t.Setenv("CORS_ALLOWED_ORIGINS", "")
	t.Setenv("OSM_ROUTING_BASE_URL", "")

	details := Details()
	data := details["data"].(map[string]any)
	cors := details["cors"].(map[string]any)
	routing := details["routing"].(map[string]any)

	if data["provider"] != "strava" {
		t.Fatalf("expected provider=strava, got %#v", data["provider"])
	}
	if data["stravaCachePath"] != defaultStravaCachePath {
		t.Fatalf("expected default Strava cache path, got %#v", data["stravaCachePath"])
	}
	if !reflect.DeepEqual(cors["allowedOrigins"], defaultCORSAllowedOrigins) {
		t.Fatalf("expected default CORS origins, got %#v", cors["allowedOrigins"])
	}
	if !reflect.DeepEqual(cors["allowedHeaders"], defaultCORSAllowedHeaders) {
		t.Fatalf("expected default CORS headers, got %#v", cors["allowedHeaders"])
	}
	if cors["allowCredentials"] != true {
		t.Fatalf("expected CORS credentials enabled, got %#v", cors["allowCredentials"])
	}
	if routing["baseUrl"] != defaultOSMRoutingBaseURL {
		t.Fatalf("expected default OSRM base URL, got %#v", routing["baseUrl"])
	}
}

func TestDetails_ExposesConfiguredRuntimeValues(t *testing.T) {
	t.Setenv("FIT_FILES_PATH", "/data/fit")
	t.Setenv("GPX_FILES_PATH", "/data/gpx")
	t.Setenv("STRAVA_CACHE_PATH", "/data/strava")
	t.Setenv("CORS_ALLOWED_ORIGINS", "http://localhost:5173, https://app.example")
	t.Setenv("OSM_ROUTING_ENABLED", "false")
	t.Setenv("OSM_ROUTING_HISTORY_HALF_LIFE_DAYS", "90")

	details := Details()
	data := details["data"].(map[string]any)
	cors := details["cors"].(map[string]any)
	routing := details["routing"].(map[string]any)

	if data["provider"] != "fit" {
		t.Fatalf("expected FIT provider, got %#v", data["provider"])
	}
	if data["fitFilesPath"] != "/data/fit" || data["fitFilesConfigured"] != true {
		t.Fatalf("expected configured FIT path, got %#v", data)
	}
	if data["gpxFilesSupported"] != false {
		t.Fatalf("expected Go GPX support to be false, got %#v", data["gpxFilesSupported"])
	}
	expectedOrigins := []string{"http://localhost:5173", "https://app.example"}
	if !reflect.DeepEqual(cors["allowedOrigins"], expectedOrigins) {
		t.Fatalf("expected configured CORS origins, got %#v", cors["allowedOrigins"])
	}
	if routing["enabled"] != false {
		t.Fatalf("expected routing disabled, got %#v", routing["enabled"])
	}
	if routing["historyHalfLifeDays"] != 90 {
		t.Fatalf("expected history half-life 90, got %#v", routing["historyHalfLifeDays"])
	}
}
