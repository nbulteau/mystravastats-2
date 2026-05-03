package infrastructure

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"mystravastats/internal/helpers"
	"mystravastats/internal/platform/runtimeconfig"
	"mystravastats/internal/shared/domain/business"
	"mystravastats/internal/shared/domain/strava"
	fitprovider "mystravastats/internal/shared/infrastructure/fit"
	gpxprovider "mystravastats/internal/shared/infrastructure/gpx"
	"mystravastats/internal/shared/infrastructure/localrepository"
)

const maxPreviewErrors = 8
const stravaSettingsURL = "https://www.strava.com/settings/api"

var requiredStravaScopes = []string{"read_all", "activity:read_all", "profile:read_all"}

type SourceModeServiceAdapter struct{}

func NewSourceModeServiceAdapter() *SourceModeServiceAdapter {
	return &SourceModeServiceAdapter{}
}

func (adapter *SourceModeServiceAdapter) PreviewSourceMode(request business.SourceModePreviewRequest) (preview business.SourceModePreview) {
	defer func() {
		preview = normalizeSourceModePreview(preview)
	}()

	mode := normalizeSourceMode(request.Mode)
	path := strings.TrimSpace(request.Path)

	switch mode {
	case business.SourceModeFIT:
		if path == "" {
			path, _ = runtimeconfig.OptionalValue("FIT_FILES_PATH")
		}
		return previewLocalSourceMode(mode, "FIT_FILES_PATH", ".fit", path, runtimeconfig.OptionalValue, decodeFITPreviewActivity)
	case business.SourceModeGPX:
		if path == "" {
			path, _ = runtimeconfig.OptionalValue("GPX_FILES_PATH")
		}
		return previewLocalSourceMode(mode, "GPX_FILES_PATH", ".gpx", path, runtimeconfig.OptionalValue, decodeGPXPreviewActivity)
	case business.SourceModeStrava:
		if path == "" {
			path = helpers.StravaCachePath
		}
		return previewStravaSourceMode(path)
	default:
		return business.SourceModePreview{
			Mode:      business.SourceMode(request.Mode),
			Path:      path,
			Supported: false,
			Errors: []business.SourceModePreviewError{{
				Message: fmt.Sprintf("unsupported source mode %q", request.Mode),
			}},
			Recommendations: []string{"Choose STRAVA, FIT or GPX."},
		}
	}
}

func normalizeSourceModePreview(preview business.SourceModePreview) business.SourceModePreview {
	if preview.ActiveMode == "" {
		preview.ActiveMode = activeSourceMode()
	}
	if preview.Supported && preview.ConfigKey != "" {
		preview.Active = preview.ActiveMode == preview.Mode && !preview.RestartNeeded
		preview.ActivationCommand = sourceModeActivationCommand(preview.Mode, preview.ConfigKey, preview.Path)
		preview.Environment = sourceModeEnvironment(preview.Mode, preview.ConfigKey, preview.Path)
	}
	if preview.Years == nil {
		preview.Years = []business.SourceModeYearPreview{}
	}
	if preview.MissingFields == nil {
		preview.MissingFields = []string{}
	}
	if preview.Environment == nil {
		preview.Environment = []business.SourceModeEnvironmentVariable{}
	}
	if preview.Errors == nil {
		preview.Errors = []business.SourceModePreviewError{}
	}
	if preview.Recommendations == nil {
		preview.Recommendations = []string{}
	}
	return preview
}

func normalizeSourceMode(raw string) business.SourceMode {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case "FIT":
		return business.SourceModeFIT
	case "GPX":
		return business.SourceModeGPX
	case "STRAVA", "":
		return business.SourceModeStrava
	default:
		return business.SourceMode(strings.ToUpper(strings.TrimSpace(raw)))
	}
}

type sourceModeDecoder func(filePath string, athleteID int64, year int) (*strava.Activity, error)
type optionalConfigReader func(key string) (string, bool)

func previewLocalSourceMode(
	mode business.SourceMode,
	configKey string,
	extension string,
	path string,
	configReader optionalConfigReader,
	decoder sourceModeDecoder,
) business.SourceModePreview {
	configuredPath, configured := configReader(configKey)
	path = strings.TrimSpace(path)
	preview := business.SourceModePreview{
		Mode:          mode,
		Path:          path,
		ConfigKey:     configKey,
		Supported:     true,
		Configured:    configured,
		RestartNeeded: activeSourceMode() != mode || strings.TrimSpace(configuredPath) != path,
	}
	if path == "" {
		preview.Errors = append(preview.Errors, business.SourceModePreviewError{Message: "path is required"})
		preview.MissingFields = []string{"activities"}
		preview.Recommendations = append(preview.Recommendations, fmt.Sprintf("Set %s to a local %s directory.", configKey, mode))
		return preview
	}

	info, err := os.Stat(path)
	if err != nil {
		preview.Errors = append(preview.Errors, business.SourceModePreviewError{Path: path, Message: err.Error()})
		preview.MissingFields = []string{"activities"}
		preview.Recommendations = append(preview.Recommendations, "Choose a directory readable by the backend process.")
		return preview
	}
	if !info.IsDir() {
		preview.Errors = append(preview.Errors, business.SourceModePreviewError{Path: path, Message: "path is not a directory"})
		preview.MissingFields = []string{"activities"}
		preview.Recommendations = append(preview.Recommendations, "Choose the parent directory containing year folders such as 2025/ and 2026/.")
		return preview
	}
	preview.Readable = true

	yearEntries, err := os.ReadDir(path)
	if err != nil {
		preview.Errors = append(preview.Errors, business.SourceModePreviewError{Path: path, Message: err.Error()})
		return preview
	}

	years := make([]business.SourceModeYearPreview, 0)
	fieldStats := sourceModeFieldStats{}
	for _, yearEntry := range yearEntries {
		if !yearEntry.IsDir() || !isYearDirectory(yearEntry.Name()) {
			continue
		}
		year, _ := strconv.Atoi(yearEntry.Name())
		yearPath := filepath.Join(path, yearEntry.Name())
		fileEntries, err := os.ReadDir(yearPath)
		if err != nil {
			appendPreviewError(&preview, yearPath, err.Error())
			continue
		}

		yearPreview := business.SourceModeYearPreview{Year: yearEntry.Name()}
		for _, fileEntry := range fileEntries {
			if fileEntry.IsDir() || !strings.EqualFold(filepath.Ext(fileEntry.Name()), extension) {
				continue
			}
			preview.FileCount++
			yearPreview.FileCount++
			filePath := filepath.Join(yearPath, fileEntry.Name())
			activity, err := decoder(filePath, 0, year)
			if err != nil {
				preview.InvalidFileCount++
				appendPreviewError(&preview, filePath, err.Error())
				continue
			}
			preview.ValidFileCount++
			preview.ActivityCount++
			yearPreview.ValidFileCount++
			yearPreview.ActivityCount++
			fieldStats.add(activity)
		}
		if yearPreview.FileCount > 0 {
			years = append(years, yearPreview)
		}
	}

	sort.Slice(years, func(i, j int) bool { return years[i].Year > years[j].Year })
	preview.Years = years
	preview.ValidStructure = len(years) > 0
	preview.MissingFields = fieldStats.missingFields(preview.ActivityCount)
	preview.Recommendations = localSourceRecommendations(preview)
	return preview
}

func previewStravaSourceMode(path string) business.SourceModePreview {
	configuredPath, configured := runtimeconfig.OptionalValue("STRAVA_CACHE_PATH")
	if configuredPath == "" {
		configuredPath = helpers.StravaCachePath
	}
	preview := business.SourceModePreview{
		Mode:          business.SourceModeStrava,
		Path:          path,
		ConfigKey:     "STRAVA_CACHE_PATH",
		Supported:     true,
		Configured:    configured,
		RestartNeeded: activeSourceMode() != business.SourceModeStrava || strings.TrimSpace(configuredPath) != path,
	}
	preview.StravaOAuth = inspectStravaOAuth(path, "", "", false)

	info, err := os.Stat(path)
	if err != nil {
		preview.Errors = append(preview.Errors, business.SourceModePreviewError{Path: path, Message: err.Error()})
		preview.Recommendations = []string{"Choose the Strava cache directory containing the .strava file."}
		return preview
	}
	if !info.IsDir() {
		preview.Errors = append(preview.Errors, business.SourceModePreviewError{Path: path, Message: "path is not a directory"})
		return preview
	}
	preview.Readable = true

	repository := localrepository.NewStravaRepository(path)
	clientID, clientSecret, useCache := repository.ReadStravaAuthentication(path)
	preview.StravaOAuth = inspectStravaOAuth(path, clientID, clientSecret, useCache)
	if clientID == "" {
		preview.Errors = append(preview.Errors, business.SourceModePreviewError{
			Path:    filepath.Join(path, ".strava"),
			Message: ".strava file is missing or does not contain clientId",
		})
		preview.Recommendations = []string{"Configure Strava credentials or switch to FIT/GPX local mode."}
		return preview
	}

	preview.ValidStructure = true
	athleteDirectory := filepath.Join(path, fmt.Sprintf("strava-%s", clientID))
	preview.Years, preview.FileCount, preview.ActivityCount = scanStravaCacheYears(repository, athleteDirectory, clientID)
	preview.ValidFileCount = preview.FileCount
	if useCache {
		preview.Recommendations = append(preview.Recommendations, "Strava cache-only mode is enabled.")
	} else if _, err := os.Stat(filepath.Join(path, ".strava-token.json")); err == nil {
		preview.Recommendations = append(preview.Recommendations, "Strava OAuth token is available for refresh.")
	} else {
		preview.Recommendations = append(preview.Recommendations, "Run node scripts/setup-strava-oauth.mjs to create .strava-token.json before live Strava refresh.")
	}
	if preview.RestartNeeded {
		preview.Recommendations = append(preview.Recommendations, "Restart the backend after changing STRAVA_CACHE_PATH or switching source mode.")
	}
	return preview
}

type stravaOAuthTokenFile struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    int64  `json:"expires_at"`
	Scope        string `json:"scope"`
	Athlete      struct {
		ID        int64  `json:"id"`
		Username  string `json:"username"`
		FirstName string `json:"firstname"`
		LastName  string `json:"lastname"`
	} `json:"athlete"`
}

func inspectStravaOAuth(path string, clientID string, clientSecret string, useCache bool) *business.StravaOAuthStatus {
	trimmedPath := strings.TrimSpace(path)
	credentialsFile := filepath.Join(trimmedPath, ".strava")
	tokenFile := filepath.Join(trimmedPath, ".strava-token.json")
	status := &business.StravaOAuthStatus{
		Status:              "needs_credentials",
		SettingsURL:         stravaSettingsURL,
		CallbackDomain:      "127.0.0.1",
		OAuthCallbackURL:    "http://127.0.0.1:8090/exchange_token",
		SetupCommand:        stravaSetupCommand(trimmedPath),
		CredentialsFile:     credentialsFile,
		TokenFile:           tokenFile,
		ClientIDPresent:     strings.TrimSpace(clientID) != "",
		ClientSecretPresent: strings.TrimSpace(clientSecret) != "",
		CacheOnly:           useCache,
		RequiredScopes:      append([]string(nil), requiredStravaScopes...),
		GrantedScopes:       []string{},
		MissingScopes:       []string{},
	}
	status.CredentialsPresent = status.ClientIDPresent && status.ClientSecretPresent
	if _, err := os.Stat(credentialsFile); err == nil {
		status.CredentialsFilePresent = true
	}

	if !status.CredentialsPresent || !status.ClientIDPresent || !status.ClientSecretPresent {
		status.Message = "Create a Strava app, then run the local setup assistant with Client ID and Client Secret."
		return status
	}
	if useCache {
		status.Status = "cache_only"
		status.Message = "Strava cache-only mode is enabled; OAuth token is not required until live refresh is re-enabled."
		return status
	}

	status.TokenPresent = true
	tokenPayload, err := readStravaOAuthToken(tokenFile)
	if err != nil {
		if os.IsNotExist(err) {
			status.TokenPresent = false
			status.Status = "needs_token"
			status.Message = "Credentials are present; run the OAuth assistant to create .strava-token.json."
			return status
		}
		status.Status = "token_unreadable"
		status.TokenError = err.Error()
		status.Message = "The OAuth token file exists but cannot be read."
		return status
	}

	status.TokenReadable = true
	status.AccessTokenPresent = strings.TrimSpace(tokenPayload.AccessToken) != ""
	status.RefreshTokenPresent = strings.TrimSpace(tokenPayload.RefreshToken) != ""
	status.TokenExpired = tokenPayload.ExpiresAt > 0 && time.Unix(tokenPayload.ExpiresAt, 0).Before(time.Now().Add(2*time.Minute))
	if tokenPayload.ExpiresAt > 0 {
		status.TokenExpiresAt = time.Unix(tokenPayload.ExpiresAt, 0).UTC().Format(time.RFC3339)
	}
	if tokenPayload.Athlete.ID > 0 {
		status.AthleteID = strconv.FormatInt(tokenPayload.Athlete.ID, 10)
	}
	status.AthleteName = athleteDisplayName(tokenPayload.Athlete.Username, tokenPayload.Athlete.FirstName, tokenPayload.Athlete.LastName)
	status.GrantedScopes = splitStravaScopes(tokenPayload.Scope)
	status.ScopesVerified = len(status.GrantedScopes) > 0
	if status.ScopesVerified {
		status.MissingScopes = missingStravaScopes(status.GrantedScopes, requiredStravaScopes)
	}

	status.Status, status.Message = stravaOAuthStatusMessage(status)
	return status
}

func readStravaOAuthToken(path string) (stravaOAuthTokenFile, error) {
	var token stravaOAuthTokenFile
	data, err := os.ReadFile(path)
	if err != nil {
		return token, err
	}
	if err := json.Unmarshal(data, &token); err != nil {
		return token, err
	}
	return token, nil
}

func stravaSetupCommand(path string) string {
	if strings.TrimSpace(path) == "" {
		return "node scripts/setup-strava-oauth.mjs"
	}
	return fmt.Sprintf("node scripts/setup-strava-oauth.mjs --cache %s", shellQuote(path))
}

func splitStravaScopes(scope string) []string {
	fields := strings.FieldsFunc(scope, func(r rune) bool { return r == ',' || r == ' ' || r == '\t' || r == '\n' })
	scopes := make([]string, 0, len(fields))
	seen := map[string]bool{}
	for _, field := range fields {
		normalized := strings.TrimSpace(field)
		if normalized == "" || seen[normalized] {
			continue
		}
		seen[normalized] = true
		scopes = append(scopes, normalized)
	}
	sort.Strings(scopes)
	return scopes
}

func missingStravaScopes(granted []string, required []string) []string {
	grantedSet := map[string]bool{}
	for _, scope := range granted {
		grantedSet[scope] = true
	}
	missing := make([]string, 0)
	for _, scope := range required {
		if !grantedSet[scope] {
			missing = append(missing, scope)
		}
	}
	return missing
}

func athleteDisplayName(username string, firstName string, lastName string) string {
	parts := []string{strings.TrimSpace(firstName), strings.TrimSpace(lastName)}
	fullName := strings.TrimSpace(strings.Join(parts, " "))
	if fullName != "" {
		return fullName
	}
	return strings.TrimSpace(username)
}

func stravaOAuthStatusMessage(status *business.StravaOAuthStatus) (string, string) {
	if !status.AccessTokenPresent || !status.RefreshTokenPresent {
		return "token_incomplete", "The token file is incomplete; run the OAuth assistant again."
	}
	if len(status.MissingScopes) > 0 {
		return "scope_incomplete", "The token is missing Strava scopes; run the OAuth assistant and accept every requested permission."
	}
	if status.TokenExpired {
		if status.RefreshTokenPresent {
			return "refreshable", "The access token is expired, but the backend can refresh it on next live Strava call."
		}
		return "token_expired", "The access token is expired and no refresh token is available."
	}
	if !status.ScopesVerified {
		return "ready_unverified_scopes", "OAuth token is available; scopes are not recorded in the token file."
	}
	return "ready", "Strava credentials and OAuth token are ready."
}

func decodeFITPreviewActivity(filePath string, athleteID int64, _ int) (*strava.Activity, error) {
	return fitprovider.DecodeFITActivity(filePath, athleteID)
}

func decodeGPXPreviewActivity(filePath string, athleteID int64, year int) (*strava.Activity, error) {
	return gpxprovider.DecodeGPXActivity(filePath, athleteID, year)
}

func activeSourceMode() business.SourceMode {
	if _, configured := runtimeconfig.OptionalValue("FIT_FILES_PATH"); configured {
		return business.SourceModeFIT
	}
	if _, configured := runtimeconfig.OptionalValue("GPX_FILES_PATH"); configured {
		return business.SourceModeGPX
	}
	return business.SourceModeStrava
}

func sourceModeActivationCommand(mode business.SourceMode, configKey string, path string) string {
	path = strings.TrimSpace(path)
	if path == "" || configKey == "" {
		return ""
	}

	parts := []string{"env"}
	for _, key := range sourceModeUnsetKeys(mode) {
		parts = append(parts, "-u", key)
	}
	parts = append(parts, fmt.Sprintf("%s=%s", configKey, shellQuote(path)))
	parts = append(parts, "./mystravastats", "-port", shellQuote(runtimeconfig.StringValue("PORT", "8080")))
	return strings.Join(parts, " ")
}

func sourceModeEnvironment(mode business.SourceMode, configKey string, path string) []business.SourceModeEnvironmentVariable {
	variables := []business.SourceModeEnvironmentVariable{{
		Key:      configKey,
		Value:    strings.TrimSpace(path),
		Required: true,
	}}
	for _, key := range sourceModeUnsetKeys(mode) {
		variables = append(variables, business.SourceModeEnvironmentVariable{
			Key:      key,
			Value:    "",
			Required: false,
		})
	}
	return variables
}

func sourceModeUnsetKeys(mode business.SourceMode) []string {
	switch mode {
	case business.SourceModeStrava:
		return []string{"FIT_FILES_PATH", "GPX_FILES_PATH"}
	case business.SourceModeGPX:
		return []string{"FIT_FILES_PATH"}
	case business.SourceModeFIT:
		return []string{"GPX_FILES_PATH"}
	default:
		return []string{}
	}
}

func shellQuote(value string) string {
	if value == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func isYearDirectory(name string) bool {
	if len(name) != 4 {
		return false
	}
	year, err := strconv.Atoi(name)
	return err == nil && year >= 1900 && year <= 2200
}

func appendPreviewError(preview *business.SourceModePreview, path string, message string) {
	if len(preview.Errors) >= maxPreviewErrors {
		return
	}
	preview.Errors = append(preview.Errors, business.SourceModePreviewError{
		Path:    path,
		Message: message,
	})
}

type sourceModeFieldStats struct {
	heartRate int
	power     int
	cadence   int
	elevation int
	trace     int
}

func (stats *sourceModeFieldStats) add(activity *strava.Activity) {
	if activity == nil || activity.Stream == nil {
		return
	}
	stream := activity.Stream
	if stream.HeartRate != nil && len(stream.HeartRate.Data) > 0 {
		stats.heartRate++
	}
	if stream.Watts != nil && len(stream.Watts.Data) > 0 {
		stats.power++
	}
	if stream.Cadence != nil && len(stream.Cadence.Data) > 0 {
		stats.cadence++
	}
	if stream.Altitude != nil && len(stream.Altitude.Data) > 0 {
		stats.elevation++
	}
	if stream.LatLng != nil && len(stream.LatLng.Data) > 0 {
		stats.trace++
	}
}

func (stats sourceModeFieldStats) missingFields(activityCount int) []string {
	if activityCount <= 0 {
		return []string{"activities"}
	}
	missing := make([]string, 0)
	if stats.trace == 0 {
		missing = append(missing, "trace")
	}
	if stats.elevation == 0 {
		missing = append(missing, "elevation")
	}
	if stats.heartRate == 0 {
		missing = append(missing, "heartRate")
	}
	if stats.power == 0 {
		missing = append(missing, "power")
	}
	if stats.cadence == 0 {
		missing = append(missing, "cadence")
	}
	return missing
}

func localSourceRecommendations(preview business.SourceModePreview) []string {
	recommendations := make([]string, 0)
	if !preview.ValidStructure {
		recommendations = append(recommendations, "Use year folders such as 2025/ and 2026/ under the selected directory.")
	}
	if preview.ActivityCount == 0 && preview.FileCount > 0 {
		recommendations = append(recommendations, "Check the invalid files list before using this source mode.")
	}
	if preview.ActivityCount > 0 {
		recommendations = append(recommendations, fmt.Sprintf("Set %s=%s to use this source.", preview.ConfigKey, preview.Path))
	}
	if preview.RestartNeeded {
		recommendations = append(recommendations, "Restart the backend after changing the source mode.")
	}
	return recommendations
}

func scanStravaCacheYears(repository *localrepository.StravaRepository, athleteDirectory string, clientID string) ([]business.SourceModeYearPreview, int, int) {
	entries, err := os.ReadDir(athleteDirectory)
	if err != nil {
		return nil, 0, 0
	}

	years := make([]business.SourceModeYearPreview, 0)
	totalFiles := 0
	totalActivities := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		prefix := fmt.Sprintf("strava-%s-", clientID)
		if !strings.HasPrefix(entry.Name(), prefix) {
			continue
		}
		year := strings.TrimPrefix(entry.Name(), prefix)
		if !isYearDirectory(year) {
			continue
		}
		yearValue, _ := strconv.Atoi(year)
		activitiesFile := filepath.Join(athleteDirectory, entry.Name(), fmt.Sprintf("activities-%s-%s.json", clientID, year))
		if _, err := os.Stat(activitiesFile); err != nil {
			continue
		}
		activityCount := len(repository.LoadActivitiesFromCache(clientID, yearValue))
		totalFiles++
		totalActivities += activityCount
		years = append(years, business.SourceModeYearPreview{
			Year:           year,
			FileCount:      1,
			ValidFileCount: 1,
			ActivityCount:  activityCount,
		})
	}

	sort.Slice(years, func(i, j int) bool { return years[i].Year > years[j].Year })
	return years, totalFiles, totalActivities
}
