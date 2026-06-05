package runtimeconfig

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	defaultStravaCachePath          = "strava-cache"
	defaultStravaAPIBaseURL         = "https://www.strava.com/api/v3"
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
	fitInboxPath, fitInboxConfigured, fitInboxSource := FITInboxPath()
	garminFitSyncBin, garminFitSyncConfigured, garminFitSyncSource := GarminFITSyncBin()
	gpxFilesPath, gpxConfigured := optionalEnv("GPX_FILES_PATH")
	stravaConfigured := isConfigured("STRAVA_CACHE_PATH")
	dataProvider, activeProviders := dataProviderDetails(stravaConfigured, fitConfigured, gpxConfigured)

	corsOrigins, corsSource := corsAllowedOriginsWithSource()

	return map[string]any{
		"backend": "go",
		"data": map[string]any{
			"provider":                  dataProvider,
			"stravaCachePath":           readStringEnv("STRAVA_CACHE_PATH", defaultStravaCachePath),
			"stravaCacheConfigured":     isConfigured("STRAVA_CACHE_PATH"),
			"stravaApiBaseUrl":          StravaAPIBaseURL(),
			"stravaApiBaseConfigured":   isConfigured("STRAVA_API_BASE_URL"),
			"fitFilesPath":              fitFilesPath,
			"fitFilesConfigured":        fitConfigured,
			"fitInboxPath":              fitInboxPath,
			"fitInboxConfigured":        fitInboxConfigured,
			"fitInboxSource":            fitInboxSource,
			"garminFitSourcePath":       readStringEnv("GARMIN_FIT_SOURCE_PATH", ""),
			"garminFitSourceConfigured": isConfigured("GARMIN_FIT_SOURCE_PATH"),
			"garminFitSyncBin":          garminFitSyncBin,
			"garminFitSyncConfigured":   garminFitSyncConfigured,
			"garminFitSyncSource":       garminFitSyncSource,
			"gpxFilesPath":              gpxFilesPath,
			"gpxFilesConfigured":        gpxConfigured,
			"gpxFilesSupported":         true,
			"activeProviders":           activeProviders,
			"compositeAutoEnabled":      len(activeProviders) > 1,
			"providerSelectionOrder":    []string{"STRAVA_CACHE_PATH", "FIT_FILES_PATH", "GPX_FILES_PATH"},
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
			"controlEnabled":      readBoolEnv("OSRM_CONTROL_ENABLED", true),
			"controlTimeoutMs":    readIntEnv("OSRM_CONTROL_TIMEOUT_MS", 30000),
			"controlProjectDir":   readStringEnv("OSRM_CONTROL_PROJECT_DIR", ""),
			"controlComposeFile":  readStringEnv("OSRM_CONTROL_COMPOSE_FILE", "docker-compose-routing-osrm.yml"),
			"controlDockerBin":    readStringEnv("OSRM_CONTROL_DOCKER_BIN", ""),
		},
	}
}

func dataProviderDetails(stravaConfigured, fitConfigured, gpxConfigured bool) (string, []string) {
	activeProviders := make([]string, 0, 3)
	if stravaConfigured {
		activeProviders = append(activeProviders, "strava")
	}
	if fitConfigured {
		activeProviders = append(activeProviders, "fit")
	}
	if gpxConfigured {
		activeProviders = append(activeProviders, "gpx")
	}
	if len(activeProviders) > 1 {
		return "composite", activeProviders
	}
	if len(activeProviders) == 1 {
		return activeProviders[0], activeProviders
	}
	return "strava", []string{"strava"}
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

func FITInboxPath() (string, bool, string) {
	if value, configured := optionalEnv("FIT_INBOX_PATH"); configured {
		return value, true, "FIT_INBOX_PATH"
	}
	if fitFilesPath, configured := optionalEnv("FIT_FILES_PATH"); configured {
		return filepath.Join(fitFilesPath, "_inbox"), true, "derived"
	}
	return "", false, "not_configured"
}

func GarminFITSyncBin() (string, bool, string) {
	if value, configured := optionalEnv("GARMIN_FIT_SYNC_BIN"); configured {
		return value, true, "GARMIN_FIT_SYNC_BIN"
	}
	for _, candidate := range garminFITSyncCandidates() {
		if isExecutableFile(candidate) {
			return candidate, true, "auto"
		}
	}
	return "", false, "not_configured"
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

func StravaAPIBaseURL() string {
	return strings.TrimRight(readStringEnv("STRAVA_API_BASE_URL", defaultStravaAPIBaseURL), "/")
}

func StravaAPIURL(path string) string {
	return StravaAPIBaseURL() + "/" + strings.TrimLeft(path, "/")
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

func garminFITSyncCandidates() []string {
	name := "garmin-fit-sync"
	if filepath.Separator == '\\' {
		name += ".exe"
	}

	candidates := []string{
		filepath.Join("tools", "garmin-fit-sync", "target", "release", name),
		filepath.Join("..", "tools", "garmin-fit-sync", "target", "release", name),
	}
	if executablePath, err := os.Executable(); err == nil {
		executableDir := filepath.Dir(executablePath)
		candidates = append(candidates,
			filepath.Join(executableDir, "tools", "garmin-fit-sync", "target", "release", name),
			filepath.Join(executableDir, "..", "tools", "garmin-fit-sync", "target", "release", name),
		)
	}
	return candidates
}

func isExecutableFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	if filepath.Separator == '\\' {
		return true
	}
	return info.Mode()&0o111 != 0
}

func normalizedTimeoutMs() int {
	timeoutMs := readIntEnv("OSM_ROUTING_TIMEOUT_MS", defaultOSMRoutingTimeoutMs)
	if timeoutMs < 200 {
		return defaultOSMRoutingTimeoutMs
	}
	return timeoutMs
}
