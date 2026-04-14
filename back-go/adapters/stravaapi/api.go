package stravaapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mystravastats/domain/strava"
	"mystravastats/internal/helpers"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

type StravaProperties struct {
	PageSize int
	URL      string
}

type StravaApi struct {
	clientId     string
	clientSecret string
	accessToken  string
	properties   StravaProperties
	httpClient   *http.Client
}

type Token struct {
	AccessToken string `json:"access_token"`
}

var ErrStravaRateLimitReached = errors.New("strava rate limit reached (429)")

func IsRateLimitError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrStravaRateLimitReached) {
		return true
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "429") && strings.Contains(message, "rate limit")
}

func NewStravaApi(clientId, clientSecret string) *StravaApi {
	properties := StravaProperties{
		PageSize: 200,
		URL:      "https://www.strava.com",
	}
	api := &StravaApi{
		clientId:     clientId,
		clientSecret: clientSecret,
		properties:   properties,
		httpClient:   &http.Client{},
	}

	err := api.setAccessToken(clientId, clientSecret)
	if err != nil {
		log.Printf("Failed to set access token: %v", err)
		return nil
	}

	return api
}

func (api *StravaApi) setAccessToken(clientId, clientSecret string) error {
	authURL := fmt.Sprintf("%s/api/v3/oauth/authorize?client_id=%s&response_type=code&redirect_uri=http://localhost:8090/exchange_token&approval_prompt=auto&scope=read_all,activity:read_all,profile:read_all", api.properties.URL, clientId)
	fmt.Println("To grant MyStravaStats to read your Strava activities data: copy paste this URL in a browser")
	fmt.Println(authURL)

	// Create a channel to signal when the accessToken is set
	tokenChan := make(chan struct{})
	var tokenReady sync.Once
	mux := http.NewServeMux()
	server := &http.Server{
		Addr:    ":8090",
		Handler: mux,
	}

	// Create a channel to communicate errors
	errorChan := make(chan error, 1)

	// Start a local server to handle the OAuth callback
	mux.HandleFunc("/exchange_token", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		token, err := api.getToken(clientId, clientSecret, code)
		if err != nil {
			errorChan <- err
			_, _ = fmt.Fprint(w, buildResponseHtml(clientId))
			tokenReady.Do(func() {
				close(tokenChan)
			})
			go func() {
				shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				_ = server.Shutdown(shutdownCtx)
			}()
			return
		}
		api.accessToken = token.AccessToken
		_, _ = fmt.Fprint(w, buildResponseHtml(clientId))

		tokenReady.Do(func() {
			close(tokenChan)
		})
		go func() {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = server.Shutdown(shutdownCtx)
		}()
	})

	go func() {
		err := server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Failed to start local server: %v", err)
		}
	}()

	// Open the browser
	helpers.OpenBrowser(authURL)

	// Wait for the accessToken to be set
	<-tokenChan

	// Check if there was an error
	select {
	case err := <-errorChan:
		return err
	default:
	}

	return nil
}

func (api *StravaApi) getToken(clientId, clientSecret, authorizationCode string) (Token, error) {
	tokenURL := fmt.Sprintf("%s/api/v3/oauth/token", api.properties.URL)
	payload := url.Values{}
	payload.Set("client_id", clientId)
	payload.Set("client_secret", clientSecret)
	payload.Set("code", authorizationCode)
	payload.Set("grant_type", "authorization_code")

	// Use form-encoded payload and add timeout
	client := *api.httpClient
	client.Timeout = 15 * time.Second

	var resp *http.Response
	var err error
	var lastErr error

	// Retry logic with exponential backoff
	maxAttempts := 3
	backoff := time.Second

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		resp, err = client.PostForm(tokenURL, payload)
		if err == nil {
			break
		}
		lastErr = err
		if attempt < maxAttempts {
			log.Printf("Token request failed (attempt %d/%d): %v, retrying in %v", attempt, maxAttempts, err, backoff)
			time.Sleep(backoff)
			backoff *= 2
		}
	}

	if err != nil {
		return Token{}, fmt.Errorf("failed to get token after %d attempts: %w", maxAttempts, lastErr)
	}

	defer func(Body io.ReadCloser) {
		closeErr := Body.Close()
		if closeErr != nil {
			log.Printf("warning: failed to close response body: %v", closeErr)
		}
	}(resp.Body)

	var token Token
	decodeErr := json.NewDecoder(resp.Body).Decode(&token)
	if decodeErr != nil {
		return Token{}, fmt.Errorf("failed to decode token response: %w", decodeErr)
	}

	return token, nil
}

func (api *StravaApi) RetrieveLoggedInAthlete() (*strava.Athlete, error) {
	baseAthleteUrl := fmt.Sprintf("%s/api/v3/athlete", api.properties.URL)
	return api.retrieveAthlete(baseAthleteUrl)
}

func (api *StravaApi) retrieveAthlete(url string) (*strava.Athlete, error) {
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+api.accessToken)
	resp, err := api.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to Strava API: %v", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Fatalf("Failed to close response body: %v", err)
		}
	}(resp.Body)

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, ErrStravaRateLimitReached
	}

	var athlete strava.Athlete
	if err := json.NewDecoder(resp.Body).Decode(&athlete); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return &athlete, nil
}

func (api *StravaApi) GetActivities(year int) ([]strava.Activity, error) {
	return api.getActivities(year, false)
}

func (api *StravaApi) GetActivitiesFailFastOnRateLimit(year int) ([]strava.Activity, error) {
	return api.getActivities(year, true)
}

func (api *StravaApi) getActivities(year int, failFastOnRateLimit bool) ([]strava.Activity, error) {
	// Use UTC boundaries for predictable server-side filtering
	after := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC).Unix()
	before := time.Date(year+1, 1, 1, 0, 0, 0, 0, time.UTC).Unix()

	baseURL := fmt.Sprintf("%s/api/v3/athlete/activities", api.properties.URL)
	activitiesURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	// Clamp perPage to Strava's documented max
	perPage := api.properties.PageSize
	if perPage <= 0 || perPage > 200 {
		perPage = 200
	}

	q := activitiesURL.Query()
	q.Set("before", strconv.FormatInt(before, 10))
	q.Set("after", strconv.FormatInt(after, 10))
	q.Set("per_page", strconv.Itoa(perPage))

	var activities []strava.Activity
	page := 1
	backoff := time.Second // base backoff for 429
	maxBackoff := 30 * time.Second

	for {
		// Build page URL
		q.Set("page", strconv.Itoa(page))
		activitiesURL.RawQuery = q.Encode()

		// Build request
		req, err := http.NewRequest(http.MethodGet, activitiesURL.String(), nil)
		if err != nil {
			return nil, fmt.Errorf("request build error: %w", err)
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Authorization", "Bearer "+api.accessToken)

		// Per-call timeout without changing the shared client
		client := *api.httpClient
		client.Timeout = 15 * time.Second

		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("http call error: %w", err)
		}

		// Handle HTTP status codes first
		if resp.StatusCode == http.StatusUnauthorized {
			_ = resp.Body.Close()
			return nil, fmt.Errorf("invalid token (401 Unauthorized)")
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			if failFastOnRateLimit {
				_ = resp.Body.Close()
				return nil, ErrStravaRateLimitReached
			}

			// Respect Retry-After (seconds or HTTP-date)
			wait := backoff
			if ra := resp.Header.Get("Retry-After"); ra != "" {
				if secs, aerr := strconv.Atoi(ra); aerr == nil && secs > 0 {
					wait = time.Duration(secs) * time.Second
				} else if t, perr := http.ParseTime(ra); perr == nil {
					now := time.Now()
					if t.After(now) {
						wait = t.Sub(now)
					}
				}
			}
			_ = resp.Body.Close()
			time.Sleep(wait)
			// Exponential backoff with cap
			if backoff < maxBackoff {
				backoff *= 2
				if backoff > maxBackoff {
					backoff = maxBackoff
				}
			}
			// Retry the same page (do not increment)
			continue
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			limited := io.LimitReader(resp.Body, 4096)
			body, _ := io.ReadAll(limited)
			_ = resp.Body.Close()
			return nil, fmt.Errorf("strava call failed: %d - %s", resp.StatusCode, string(body))
		}

		// Decode page
		var pageItems []strava.Activity
		if derr := json.NewDecoder(resp.Body).Decode(&pageItems); derr != nil {
			_ = resp.Body.Close()
			return nil, fmt.Errorf("json decoding error: %w", derr)
		}
		if cerr := resp.Body.Close(); cerr != nil {
			log.Printf("warning: closing response body failed: %v", cerr)
		}

		// Append to aggregate
		activities = append(activities, pageItems...)

		// Reset backoff after a successful page
		backoff = time.Second

		// Stop when last page reached
		if len(pageItems) == 0 || len(pageItems) < perPage {
			break
		}

		// Next page
		page++
	}

	return activities, nil
}

func (api *StravaApi) GetDetailedActivity(activityId int64) (*strava.DetailedActivity, error) {
	baseActivitiesUel := fmt.Sprintf("%s/api/v3/activities/%d?include_all_efforts=true", api.properties.URL, activityId)
	resp, err := api.doGetWithRateLimitRetry(baseActivitiesUel, 20*time.Second, 6, true)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		if closeErr := Body.Close(); closeErr != nil {
			log.Printf("warning: failed to close response body: %v", closeErr)
		}
	}(resp.Body)

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("invalid token (401 Unauthorized)")
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("activity %d not found (404)", activityId)
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("%w while loading detailed activity %d", ErrStravaRateLimitReached, activityId)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		limited := io.LimitReader(resp.Body, 4096)
		body, _ := io.ReadAll(limited)
		return nil, fmt.Errorf("strava detailed activity call failed: %d - %s", resp.StatusCode, string(body))
	}

	var activity strava.DetailedActivity
	if err := json.NewDecoder(resp.Body).Decode(&activity); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}
	if activity.Id != activityId {
		return nil, fmt.Errorf("invalid detailed activity payload: expected id=%d got id=%d", activityId, activity.Id)
	}

	return &activity, nil
}

func (api *StravaApi) GetActivityStream(stravaActivity strava.Activity) (*strava.Stream, error) {
	if stravaActivity.UploadId == 0 {
		return nil, nil
	}

	baseStreamsUrl := fmt.Sprintf("%s/api/v3/activities/%d/streams?keys=time,distance,latlng,altitude,velocity_smooth,heartrate,cadence,watts,moving,grade_smooth&key_by_type=true", api.properties.URL, stravaActivity.Id)
	resp, err := api.doGetWithRateLimitRetry(baseStreamsUrl, 10*time.Second, 4, true)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		if closeErr := Body.Close(); closeErr != nil {
			log.Printf("warning: failed to close response body: %v", closeErr)
		}
	}(resp.Body)

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("%w while loading streams for activity %d", ErrStravaRateLimitReached, stravaActivity.Id)
	}
	if resp.StatusCode != http.StatusOK {
		limited := io.LimitReader(resp.Body, 4096)
		body, _ := io.ReadAll(limited)
		return nil, fmt.Errorf("unable to load streams for activity %d: %d - %s", stravaActivity.Id, resp.StatusCode, string(body))
	}

	var stream strava.Stream
	if err := json.NewDecoder(resp.Body).Decode(&stream); err != nil {
		return nil, err
	}

	return &stream, nil
}

func (api *StravaApi) doGetWithRateLimitRetry(url string, timeout time.Duration, maxAttempts int, failFastOnRateLimit bool) (*http.Response, error) {
	if maxAttempts <= 0 {
		maxAttempts = 1
	}

	backoff := time.Second
	maxBackoff := 30 * time.Second

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			return nil, fmt.Errorf("request build error: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+api.accessToken)
		req.Header.Set("Accept", "application/json")

		client := *api.httpClient
		client.Timeout = timeout

		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("unable to connect to Strava API: %w", err)
		}

		if resp.StatusCode != http.StatusTooManyRequests {
			return resp, nil
		}

		if failFastOnRateLimit {
			_ = resp.Body.Close()
			return nil, ErrStravaRateLimitReached
		}

		waitDuration := retryAfterDuration(resp.Header.Get("Retry-After"), backoff)
		_ = resp.Body.Close()

		if attempt == maxAttempts {
			return nil, fmt.Errorf("%w after %d attempts", ErrStravaRateLimitReached, maxAttempts)
		}

		log.Printf("Strava rate limit reached (429) for %s, retrying in %s (%d/%d)", url, waitDuration, attempt, maxAttempts)
		time.Sleep(waitDuration)

		backoff *= 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}

	return nil, fmt.Errorf("strava request failed after %d attempts", maxAttempts)
}

func retryAfterDuration(retryAfterHeader string, fallback time.Duration) time.Duration {
	if retryAfterHeader == "" {
		return fallback
	}

	if seconds, err := strconv.Atoi(retryAfterHeader); err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}

	if retryAt, err := http.ParseTime(retryAfterHeader); err == nil {
		wait := time.Until(retryAt)
		if wait > 0 {
			return wait
		}
	}

	return fallback
}

func buildResponseHtml(clientId string) string {
	return fmt.Sprintf(`
  <!DOCTYPE html>
  <html lang="en">
  <head>
   <meta charset="UTF-8">
   <meta name="viewport" content="width=device-width, initial-scale=1.0">
   <title>Access Granted</title>
   <style>
    body {
     font-family: Arial, sans-serif;
     background-color: #f4f4f4;
     display: flex;
     justify-content: center;
     align-items: center;
     height: 100vh;
     margin: 0;
    }
    .container {
     background-color: #fff;
     padding: 20px;
     border-radius: 8px;
     box-shadow: 0 0 10px rgba(0, 0, 0, 0.1);
     text-align: center;
    }
    .custom-class {
     color: #007bff;
     font-weight: bold;
    }
    h1 {
     color: #333;
    }
    p {
     color: #666;
    }
   </style>
  </head>
  <body>
   <div class="container">
    <h1>Access Granted</h1>
    <p class="custom-class">Access granted to read activities of clientId: %s.</p>
    <p>You can now close this window.</p>
   </div>
  </body>
  </html>
 `, clientId)
}
