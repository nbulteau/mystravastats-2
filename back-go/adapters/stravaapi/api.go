package stravaapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mystravastats/domain/helpers"
	"mystravastats/domain/strava"
	"net/http"
	"net/url"
	"strconv"
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

func NewStravaApi(clientId, clientSecret string) *StravaApi {
	properties := StravaProperties{
		PageSize: 150,
		URL:      "https://www.strava.com",
	}
	api := &StravaApi{
		clientId:     clientId,
		clientSecret: clientSecret,
		properties:   properties,
		httpClient:   &http.Client{},
	}
	api.setAccessToken(clientId, clientSecret)

	return api
}

func (api *StravaApi) setAccessToken(clientId, clientSecret string) {
	authURL := fmt.Sprintf("%s/api/v3/oauth/authorize?client_id=%s&response_type=code&redirect_uri=http://localhost:8090/exchange_token&approval_prompt=auto&scope=read_all,activity:read_all,profile:read_all", api.properties.URL, clientId)
	fmt.Println("To grant MyStravaStats to read your Strava activities data: copy paste this URL in a browser")
	fmt.Println(authURL)

	// Create a channel to signal when the accessToken is set
	tokenChan := make(chan struct{})

	// Start a local server to handle the OAuth callback
	http.HandleFunc("/exchange_token", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		token := api.getToken(clientId, clientSecret, code)
		api.accessToken = token.AccessToken
		_, _ = fmt.Fprint(w, buildResponseHtml(clientId))

		// Signal that the token is set
		close(tokenChan)

		// Remove port binding
		_ = http.ListenAndServe(":8090", nil)
	})

	go func() {
		err := http.ListenAndServe(":8090", nil)
		if err != nil {
			log.Fatalf("Failed to start local server: %v", err)
		}
	}()

	// Open the browser
	helpers.OpenBrowser(authURL)

	// Wait for the accessToken to be set
	<-tokenChan
}

func (api *StravaApi) getToken(clientId, clientSecret, authorizationCode string) Token {
	tokenURL := fmt.Sprintf("%s/api/v3/oauth/token", api.properties.URL)
	payload := map[string]string{
		"client_id":     clientId,
		"client_secret": clientSecret,
		"code":          authorizationCode,
		"grant_type":    "authorization_code",
	}
	body, _ := json.Marshal(payload)
	resp, err := api.httpClient.Post(tokenURL, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Fatalf("Failed to get token: %v", err)
	}
	defer func(Body interface{}) {
		err := resp.Body.Close()
		if err != nil {
			log.Fatalf("Failed to close response body: %v", err)
		}
	}(resp.Body)

	var token Token
	err = json.NewDecoder(resp.Body).Decode(&token)
	if err != nil {
		return Token{}
	}

	return token
}

func (api *StravaApi) RetrieveLoggedInAthlete() (*strava.Athlete, error) {
	baseAthleteUrl := fmt.Sprintf("%s/api/v3/athlete", api.properties.URL)

	var athlete *strava.Athlete
	var err error
	retryCount := 3
	backoffDelay := time.Second

	for i := 0; i < retryCount; i++ {
		athlete, err = api.retrieveAthlete(baseAthleteUrl)
		if err == nil {
			return athlete, nil
		}

		if errors.Is(err, errors.New("too many requests")) {
			log.Printf("Too many requests, retrying in %v...", backoffDelay)
			time.Sleep(backoffDelay)
			backoffDelay *= 2 // Backoff exponential
			continue
		}

		return nil, err // If the error is not "too many requests", return it immediately
	}

	return nil, fmt.Errorf("failed to retrieve athlete after %d retries: %v", retryCount, err)
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
		return nil, errors.New("too many requests")
	}

	var athlete strava.Athlete
	if err := json.NewDecoder(resp.Body).Decode(&athlete); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return &athlete, nil
}

func (api *StravaApi) GetActivities(year int) ([]strava.Activity, error) {
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
	req, _ := http.NewRequest("GET", baseActivitiesUel, nil)
	req.Header.Set("Authorization", "Bearer "+api.accessToken)
	resp, err := api.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to Strava API: %v", err)
	}
	defer func(Body interface{}) {
		err := resp.Body.Close()
		if err != nil {
			log.Fatalf("Failed to close response body: %v", err)
		}
	}(resp.Body)

	var activity strava.DetailedActivity
	if err := json.NewDecoder(resp.Body).Decode(&activity); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return &activity, nil
}

func (api *StravaApi) GetActivityStream(stravaActivity strava.Activity) (*strava.Stream, error) {
	if stravaActivity.UploadId == 0 {
		return nil, nil
	}

	baseStreamsUrl := fmt.Sprintf("https://www.strava.com/api/v3/activities/%d/streams?keys=time,distance,latlng,altitude,velocity_smooth,heartrate,cadence,watts,moving,grade_smooth&key_by_type=true", stravaActivity.Id)
	req, _ := http.NewRequest("GET", baseStreamsUrl, nil)
	req.Header.Set("Authorization", "Bearer "+api.accessToken)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func(Body interface{}) {
		_ = resp.Body.Close()
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("unable to load streams for activity")
	}

	var stream strava.Stream
	if err := json.NewDecoder(resp.Body).Decode(&stream); err != nil {
		return nil, err
	}

	return &stream, nil
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
