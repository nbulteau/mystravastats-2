package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mystravastats/internal/shared/domain/business"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

const stravaOAuthSettingsURL = "https://www.strava.com/settings/api"
const stravaOAuthTokenURL = "https://www.strava.com/oauth/token"
const stravaOAuthAthleteURL = "https://www.strava.com/api/v3/athlete"
const stravaOAuthScope = "read_all,activity:read_all,profile:read_all"
const stravaOAuthSessionTTL = 10 * time.Minute

var stravaClientIDPattern = regexp.MustCompile(`^\d+$`)
var stravaOAuthSessions sync.Map

func postSourceModePreview(writer http.ResponseWriter, request *http.Request) {
	var previewRequest business.SourceModePreviewRequest
	if err := json.NewDecoder(request.Body).Decode(&previewRequest); err != nil {
		writeBadRequest(writer, "Invalid request body", err.Error())
		return
	}

	preview := getContainer().previewSourceModeUseCase.Execute(previewRequest)
	if err := writeJSON(writer, http.StatusOK, preview); err != nil {
		log.Printf("failed to write source mode preview response: %v", err)
		writeInternalServerError(writer, "Failed to encode source mode preview response")
	}
}

func postStravaOAuthStart(writer http.ResponseWriter, request *http.Request) {
	var payload business.StravaOAuthStartRequest
	if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
		writeBadRequest(writer, "Invalid request body", err.Error())
		return
	}

	result, err := startStravaOAuthEnrollment(payload, stravaOAuthCallbackURLFromRequest(request))
	if err != nil {
		writeBadRequest(writer, "Invalid Strava OAuth enrollment", err.Error())
		return
	}
	if err := writeJSON(writer, http.StatusOK, result); err != nil {
		log.Printf("failed to write Strava OAuth start response: %v", err)
		writeInternalServerError(writer, "Failed to encode Strava OAuth start response")
	}
}

func getStravaOAuthCallback(writer http.ResponseWriter, request *http.Request) {
	html, status := completeStravaOAuthEnrollment(request.URL.Query())
	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	writer.WriteHeader(status)
	_, _ = writer.Write([]byte(html))
}

type stravaOAuthSession struct {
	Path         string
	ClientID     string
	ClientSecret string
	TokenFile    string
	CreatedAt    time.Time
}

func startStravaOAuthEnrollment(request business.StravaOAuthStartRequest, callbackURL string) (business.StravaOAuthStartResult, error) {
	cleanupStravaOAuthSessions()
	path := strings.TrimSpace(request.Path)
	if path == "" {
		path = "strava-cache"
	}
	clientID := strings.TrimSpace(request.ClientID)
	clientSecret := strings.TrimSpace(request.ClientSecret)
	useCache := request.UseCache
	existingID, existingSecret, _ := readStravaCredentials(path)
	if clientID == "" {
		clientID = existingID
	}
	if clientSecret == "" {
		clientSecret = existingSecret
	}
	if clientID == "" {
		return business.StravaOAuthStartResult{}, fmt.Errorf("clientId is required")
	}
	if !stravaClientIDPattern.MatchString(clientID) {
		return business.StravaOAuthStartResult{}, fmt.Errorf("clientId must be numeric")
	}
	if !useCache && clientSecret == "" {
		return business.StravaOAuthStartResult{}, fmt.Errorf("clientSecret is required for live OAuth")
	}
	if err := writeStravaCredentials(path, clientID, clientSecret, useCache); err != nil {
		return business.StravaOAuthStartResult{}, err
	}

	result := business.StravaOAuthStartResult{
		Status:           "credentials_saved",
		Message:          "Strava credentials saved.",
		SettingsURL:      stravaOAuthSettingsURL,
		CallbackDomain:   "127.0.0.1",
		OAuthCallbackURL: callbackURL,
		CredentialsFile:  filepath.Join(path, ".strava"),
		TokenFile:        filepath.Join(path, ".strava-token.json"),
		CacheOnly:        useCache,
	}
	if useCache {
		result.Status = "cache_only"
		result.Message = "Strava cache-only mode saved."
		return result, nil
	}

	state, err := newStravaOAuthState()
	if err != nil {
		return business.StravaOAuthStartResult{}, err
	}
	stravaOAuthSessions.Store(state, stravaOAuthSession{
		Path:         path,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		TokenFile:    filepath.Join(path, ".strava-token.json"),
		CreatedAt:    time.Now(),
	})
	result.Status = "oauth_started"
	result.Message = "Open Strava authorization to finish OAuth."
	result.AuthorizeURL = stravaAuthorizeURL(clientID, callbackURL, state)
	return result, nil
}

func completeStravaOAuthEnrollment(query url.Values) (string, int) {
	state := query.Get("state")
	sessionValue, ok := stravaOAuthSessions.Load(state)
	if !ok {
		return stravaOAuthHTML("Authorization failed", "OAuth session is missing or expired."), http.StatusBadRequest
	}
	session := sessionValue.(stravaOAuthSession)
	if time.Since(session.CreatedAt) > stravaOAuthSessionTTL {
		stravaOAuthSessions.Delete(state)
		return stravaOAuthHTML("Authorization failed", "OAuth session expired. Restart Strava enrollment from MyStravaStats."), http.StatusBadRequest
	}
	if oauthError := query.Get("error"); oauthError != "" {
		stravaOAuthSessions.Delete(state)
		return stravaOAuthHTML("Authorization failed", "Strava OAuth failed: "+oauthError), http.StatusBadRequest
	}
	code := query.Get("code")
	if code == "" {
		return stravaOAuthHTML("Authorization failed", "Strava did not return an authorization code."), http.StatusBadRequest
	}
	scope := query.Get("scope")
	if missing := missingStravaOAuthScopes(scope); len(missing) > 0 {
		return stravaOAuthHTML("Authorization failed", "Missing required scope(s): "+strings.Join(missing, ", ")), http.StatusBadRequest
	}

	token, err := exchangeStravaOAuthCode(session.ClientID, session.ClientSecret, code)
	if err != nil {
		return stravaOAuthHTML("Authorization failed", err.Error()), http.StatusBadGateway
	}
	if scope == "" {
		scope = stravaOAuthScope
	}
	token["scope"] = scope
	athlete, err := fetchStravaAthlete(fmt.Sprint(token["access_token"]))
	if err != nil {
		return stravaOAuthHTML("Authorization failed", err.Error()), http.StatusBadGateway
	}
	token["athlete"] = athlete
	token["created_at"] = time.Now().UTC().Format(time.RFC3339)
	if err := writePrivateJSON(session.TokenFile, token); err != nil {
		return stravaOAuthHTML("Authorization failed", err.Error()), http.StatusInternalServerError
	}
	stravaOAuthSessions.Delete(state)
	return stravaOAuthHTML("Access granted", "Strava OAuth token saved. You can close this window."), http.StatusOK
}

func readStravaCredentials(path string) (clientID string, clientSecret string, useCache bool) {
	data, err := os.ReadFile(filepath.Join(path, ".strava"))
	if err != nil {
		return "", "", false
	}
	for _, line := range strings.Split(string(data), "\n") {
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		switch strings.TrimSpace(key) {
		case "clientId":
			clientID = strings.TrimSpace(value)
		case "clientSecret":
			clientSecret = strings.TrimSpace(value)
		case "useCache":
			useCache = strings.EqualFold(strings.TrimSpace(value), "true")
		}
	}
	return clientID, clientSecret, useCache
}

func writeStravaCredentials(path string, clientID string, clientSecret string, useCache bool) error {
	if err := os.MkdirAll(path, 0o700); err != nil {
		return err
	}
	content := fmt.Sprintf("clientId=%s\nclientSecret=%s\nuseCache=%t\n", clientID, clientSecret, useCache)
	file := filepath.Join(path, ".strava")
	if err := os.WriteFile(file, []byte(content), 0o600); err != nil {
		return err
	}
	return os.Chmod(file, 0o600)
}

func stravaOAuthCallbackURLFromRequest(request *http.Request) string {
	port := "8080"
	if _, parsedPort, err := net.SplitHostPort(request.Host); err == nil && parsedPort != "" {
		port = parsedPort
	}
	return fmt.Sprintf("http://127.0.0.1:%s/api/source-modes/strava/oauth/callback", port)
}

func stravaAuthorizeURL(clientID string, callbackURL string, state string) string {
	values := url.Values{}
	values.Set("client_id", clientID)
	values.Set("response_type", "code")
	values.Set("redirect_uri", callbackURL)
	values.Set("approval_prompt", "auto")
	values.Set("scope", stravaOAuthScope)
	values.Set("state", state)
	return "https://www.strava.com/oauth/authorize?" + values.Encode()
}

func missingStravaOAuthScopes(scope string) []string {
	if strings.TrimSpace(scope) == "" {
		return nil
	}
	granted := map[string]bool{}
	for _, part := range strings.Split(scope, ",") {
		granted[strings.TrimSpace(part)] = true
	}
	missing := make([]string, 0)
	for _, required := range strings.Split(stravaOAuthScope, ",") {
		if !granted[required] {
			missing = append(missing, required)
		}
	}
	return missing
}

func exchangeStravaOAuthCode(clientID string, clientSecret string, code string) (map[string]any, error) {
	payload := url.Values{}
	payload.Set("client_id", clientID)
	payload.Set("client_secret", clientSecret)
	payload.Set("code", code)
	payload.Set("grant_type", "authorization_code")

	client := http.Client{Timeout: 15 * time.Second}
	response, err := client.PostForm(stravaOAuthTokenURL, payload)
	if err != nil {
		return nil, fmt.Errorf("unable to exchange Strava authorization code: %w", err)
	}
	defer response.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("Strava token exchange failed (%d): %s", response.StatusCode, string(body))
	}
	var token map[string]any
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, fmt.Errorf("unable to decode Strava token response: %w", err)
	}
	if strings.TrimSpace(fmt.Sprint(token["access_token"])) == "" || strings.TrimSpace(fmt.Sprint(token["refresh_token"])) == "" {
		return nil, fmt.Errorf("Strava token response is missing access_token or refresh_token")
	}
	return token, nil
}

func fetchStravaAthlete(accessToken string) (map[string]any, error) {
	request, err := http.NewRequest(http.MethodGet, stravaOAuthAthleteURL, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Authorization", "Bearer "+accessToken)
	client := http.Client{Timeout: 15 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("unable to validate Strava athlete: %w", err)
	}
	defer response.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("unable to validate Strava athlete (%d): %s", response.StatusCode, string(body))
	}
	var athlete map[string]any
	if err := json.Unmarshal(body, &athlete); err != nil {
		return nil, fmt.Errorf("unable to decode Strava athlete response: %w", err)
	}
	return athlete, nil
}

func writePrivateJSON(path string, payload map[string]any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return err
	}
	return os.Chmod(path, 0o600)
}

func newStravaOAuthState() (string, error) {
	bytes := make([]byte, 24)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("unable to generate OAuth state: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

func cleanupStravaOAuthSessions() {
	stravaOAuthSessions.Range(func(key, value any) bool {
		session, ok := value.(stravaOAuthSession)
		if ok && time.Since(session.CreatedAt) > stravaOAuthSessionTTL {
			stravaOAuthSessions.Delete(key)
		}
		return true
	})
}

func stravaOAuthHTML(title string, message string) string {
	return fmt.Sprintf(`<!doctype html>
<html lang="en">
<head><meta charset="utf-8"><title>%s</title></head>
<body style="font-family: system-ui, sans-serif; margin: 40px; line-height: 1.4;">
<h1>%s</h1>
<p>%s</p>
</body>
</html>`, htmlEscape(title), htmlEscape(title), htmlEscape(message))
}

func htmlEscape(value string) string {
	replacer := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;", "'", "&#39;")
	return replacer.Replace(value)
}
