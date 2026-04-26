package runtimeconfig

import (
	"os"
	"strconv"
	"strings"
)

const (
	defaultStravaCachePath          = "strava-cache"
	defaultServerHost               = "localhost"
	defaultServerPort               = "8080"
	defaultOSMRoutingBaseURL        = "http://localhost:5000"
	defaultOSMRoutingTimeoutMs      = 3000
	defaultOSMRoutingV3Enabled      = true
	defaultOSMRoutingProfileFile    = "./osm/region.osrm.profile"
	defaultRoutingHistoryHalfLife   = 75
	defaultCORSAllowedOriginsSource = "default"
)

var defaultCORSAllowedOrigins = []string{"http://localhost", "http://localhost:5173"}
var defaultCORSAllowedMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
var defaultCORSAllowedHeaders = []string{"Content-Type", "Authorization", "X-Request-Id"}

func Details() map[string]any {
	fitFilesPath, fitConfigured := optionalEnv("FIT_FILES_PATH")
	gpxFilesPath, gpxConfigured := optionalEnv("GPX_FILES_PATH")
	dataProvider := "strava"
	if fitConfigured {
		dataProvider = "fit"
	} else if gpxConfigured {
		dataProvider = "gpx"
	}

	corsOrigins, corsSource := corsAllowedOriginsWithSource()

	return map[string]any{
		"backend": "go",
		"data": map[string]any{
			"provider":               dataProvider,
			"stravaCachePath":        readStringEnv("STRAVA_CACHE_PATH", defaultStravaCachePath),
			"stravaCacheConfigured":  isConfigured("STRAVA_CACHE_PATH"),
			"fitFilesPath":           fitFilesPath,
			"fitFilesConfigured":     fitConfigured,
			"gpxFilesPath":           gpxFilesPath,
			"gpxFilesConfigured":     gpxConfigured,
			"gpxFilesSupported":      true,
			"providerSelectionOrder": []string{"FIT_FILES_PATH", "GPX_FILES_PATH", "STRAVA_CACHE_PATH"},
		},
		"server": map[string]any{
			"host":              readFirstStringEnv(defaultServerHost, "SERVER_HOST", "HOST"),
			"port":              readStringEnv("PORT", defaultServerPort),
			"openBrowser":       readBoolEnv("OPEN_BROWSER", true),
			"openBrowserSource": sourceFor("OPEN_BROWSER"),
		},
		"cors": map[string]any{
			"allowedOrigins":   corsOrigins,
			"allowedMethods":   CORSAllowedMethods(),
			"allowedHeaders":   CORSAllowedHeaders(),
			"allowCredentials": true,
			"source":           corsSource,
		},
		"routing": map[string]any{
			"enabled":             readBoolEnv("OSM_ROUTING_ENABLED", true),
			"v3Enabled":           readBoolEnv("OSM_ROUTING_V3_ENABLED", defaultOSMRoutingV3Enabled),
			"debug":               readBoolEnv("OSM_ROUTING_DEBUG", false),
			"baseUrl":             strings.TrimRight(readStringEnv("OSM_ROUTING_BASE_URL", defaultOSMRoutingBaseURL), "/"),
			"timeoutMs":           normalizedTimeoutMs(),
			"profile":             readStringEnv("OSM_ROUTING_PROFILE", ""),
			"extractProfile":      readStringEnv("OSM_ROUTING_EXTRACT_PROFILE", ""),
			"extractProfileFile":  readStringEnv("OSM_ROUTING_EXTRACT_PROFILE_FILE", defaultOSMRoutingProfileFile),
			"historyBiasEnabled":  readBoolEnv("OSM_ROUTING_HISTORY_BIAS_ENABLED", false),
			"historyHalfLifeDays": RoutingHistoryHalfLifeDays(),
		},
	}
}

func CORSAllowedOrigins() []string {
	origins, _ := corsAllowedOriginsWithSource()
	return origins
}

func CORSAllowedMethods() []string {
	return append([]string{}, defaultCORSAllowedMethods...)
}

func CORSAllowedHeaders() []string {
	return append([]string{}, defaultCORSAllowedHeaders...)
}

func OptionalValue(key string) (string, bool) {
	return optionalEnv(key)
}

func StringValue(key string, fallback string) string {
	return readStringEnv(key, fallback)
}

func FirstStringValue(fallback string, keys ...string) string {
	return readFirstStringEnv(fallback, keys...)
}

func BoolValue(key string, fallback bool) bool {
	return readBoolEnv(key, fallback)
}

func IntValue(key string, fallback int) int {
	return readIntEnv(key, fallback)
}

func OSMRoutingTimeoutMs() int {
	return normalizedTimeoutMs()
}

func RoutingHistoryHalfLifeDays() int {
	days := readIntEnv("OSM_ROUTING_HISTORY_HALF_LIFE_DAYS", defaultRoutingHistoryHalfLife)
	if days < 1 {
		return defaultRoutingHistoryHalfLife
	}
	return days
}

func corsAllowedOriginsWithSource() ([]string, string) {
	raw := strings.TrimSpace(os.Getenv("CORS_ALLOWED_ORIGINS"))
	if raw == "" {
		return append([]string{}, defaultCORSAllowedOrigins...), defaultCORSAllowedOriginsSource
	}

	parts := strings.Split(raw, ",")
	origins := make([]string, 0, len(parts))
	for _, part := range parts {
		if origin := strings.TrimSpace(part); origin != "" {
			origins = append(origins, origin)
		}
	}
	if len(origins) == 0 {
		return append([]string{}, defaultCORSAllowedOrigins...), defaultCORSAllowedOriginsSource
	}
	return origins, "CORS_ALLOWED_ORIGINS"
}

func optionalEnv(key string) (string, bool) {
	value := strings.TrimSpace(os.Getenv(key))
	return value, value != ""
}

func isConfigured(key string) bool {
	return strings.TrimSpace(os.Getenv(key)) != ""
}

func sourceFor(key string) string {
	if isConfigured(key) {
		return key
	}
	return "default"
}

func readFirstStringEnv(fallback string, keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value
		}
	}
	return fallback
}

func readStringEnv(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func readBoolEnv(key string, fallback bool) bool {
	raw := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if raw == "" {
		return fallback
	}
	switch raw {
	case "1", "true", "yes", "y", "on":
		return true
	case "0", "false", "no", "n", "off":
		return false
	default:
		return fallback
	}
}

func readIntEnv(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}

func normalizedTimeoutMs() int {
	timeoutMs := readIntEnv("OSM_ROUTING_TIMEOUT_MS", defaultOSMRoutingTimeoutMs)
	if timeoutMs < 200 {
		return defaultOSMRoutingTimeoutMs
	}
	return timeoutMs
}
