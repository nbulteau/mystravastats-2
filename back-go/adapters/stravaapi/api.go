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
	url := fmt.Sprintf("%s/api/v3/athlete", api.properties.URL)

	var athlete *strava.Athlete
	var err error
	retryCount := 3
	backoffDelay := time.Second

	for i := 0; i < retryCount; i++ {
		athlete, err = api.retrieveAthlete(url)
		if err == nil {
			return athlete, nil
		}

		if errors.Is(err, errors.New("too many requests")) {
			log.Printf("Too many requests, retrying in %v...", backoffDelay)
			time.Sleep(backoffDelay)
			backoffDelay *= 2 // Backoff exponentiel
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
	loc, _ := time.LoadLocation("Europe/Paris")

	beforeEpoch := time.Date(year, 12, 31, 23, 59, 0, 0, loc).Unix()
	afterEpoch := time.Date(year, 1, 1, 0, 0, 0, 0, loc).Unix()
	//url := fmt.Sprintf("%s/api/v3/athlete/activities?per_page=%d&before=%d&after=%d", api.properties.URL, api.properties.PageSize, before, after)

	baseActivitiesURL := fmt.Sprintf("%s/api/v3/athlete/activities", api.properties.URL)
	u, _ := url.Parse(baseActivitiesURL)
	q := u.Query()
	q.Set("per_page", strconv.Itoa(api.properties.PageSize))
	q.Set("after", strconv.FormatInt(afterEpoch, 10))
	q.Set("before", strconv.FormatInt(beforeEpoch, 10))
	u.RawQuery = q.Encode()

	var activities []strava.Activity
	page := 1

	for {
		pageURL, err := url.Parse(u.String())
		if err != nil {
			return nil, fmt.Errorf("URL de page invalide: %w", err)
		}
		pq := pageURL.Query()
		pq.Set("page", strconv.Itoa(page))
		pageURL.RawQuery = pq.Encode()

		req, err := http.NewRequest(http.MethodGet, pageURL.String(), nil)
		if err != nil {
			return nil, fmt.Errorf("erreur création requête: %w", err)
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+api.accessToken)

		resp, err := api.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("erreur appel HTTP: %w", err)
		}
		func() {
			defer func(Body io.ReadCloser) {
				err := Body.Close()
				if err != nil {
					log.Fatalf("Failed to close response body: %v", err)
				}
			}(resp.Body)
			switch resp.StatusCode {
			case http.StatusUnauthorized: // 401
				err = fmt.Errorf("token invalide (401 Unauthorized)")
				return
			case http.StatusTooManyRequests: // 429
				err = fmt.Errorf("limite de quota atteinte (429 Too Many Requests)")
				return
			}

			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				body, _ := io.ReadAll(resp.Body)
				err = fmt.Errorf("appel Strava non réussi: %d - %s", resp.StatusCode, string(body))
				return
			}

			var pageItems []strava.Activity
			dec := json.NewDecoder(resp.Body)
			if derr := dec.Decode(&pageItems); derr != nil {
				err = fmt.Errorf("erreur de décodage JSON: %w", derr)
				return
			}

			// Fin de pagination si la page est vide
			if len(pageItems) == 0 {
				err = io.EOF // signal d’arrêt de boucle
				return
			}

			activities = append(activities, pageItems...)
		}()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		page++
	}

	return activities, nil
}

func (api *StravaApi) getActivities(url string) ([]strava.Activity, error) {
	req, _ := http.NewRequest("GET", url, nil)
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

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, errors.New("too many requests")
	}

	var activities []strava.Activity
	if err := json.NewDecoder(resp.Body).Decode(&activities); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}
	return activities, nil
}

func (api *StravaApi) GetDetailedActivity(activityId int64) (*strava.DetailedActivity, error) {
	url := fmt.Sprintf("%s/api/v3/activities/%d?include_all_efforts=true", api.properties.URL, activityId)
	req, _ := http.NewRequest("GET", url, nil)
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

	url := fmt.Sprintf("https://www.strava.com/api/v3/activities/%d/streams?keys=time,distance,latlng,altitude,velocity_smooth,heartrate,cadence,watts,moving,grade_smooth&key_by_type=true", stravaActivity.Id)
	req, _ := http.NewRequest("GET", url, nil)
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
