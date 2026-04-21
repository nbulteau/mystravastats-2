package infrastructure

import (
	"encoding/json"
	"fmt"
	routesDomain "mystravastats/internal/routes/domain"
	"mystravastats/internal/shared/domain/strava"
	"os"
	"path/filepath"
	"testing"
)

type routeExplorerParityFixture struct {
	Cases []routeExplorerParityCase `json:"cases"`
}

type routeExplorerParityCase struct {
	Name       string                      `json:"name"`
	Request    routeExplorerParityRequest  `json:"request"`
	Activities []routeExplorerParityAction `json:"activities"`
	Expect     routeExplorerParityExpect   `json:"expect"`
}

type routeExplorerParityRequest struct {
	DistanceTargetKm  *float64                  `json:"distanceTargetKm"`
	ElevationTargetM  *float64                  `json:"elevationTargetM"`
	DurationTargetMin *int                      `json:"durationTargetMin"`
	StartDirection    *string                   `json:"startDirection"`
	StartPoint        *routeExplorerParityPoint `json:"startPoint"`
	RouteType         *string                   `json:"routeType"`
	Limit             int                       `json:"limit"`
}

type routeExplorerParityAction struct {
	ID          int64       `json:"id"`
	Name        string      `json:"name"`
	StartDate   string      `json:"startDate"`
	DistanceKm  float64     `json:"distanceKm"`
	ElevationM  float64     `json:"elevationM"`
	DurationSec int         `json:"durationSec"`
	Start       []float64   `json:"start"`
	Track       [][]float64 `json:"track"`
	Type        string      `json:"type"`
	SportType   string      `json:"sportType"`
}

type routeExplorerParityExpect struct {
	TopClosestLoopName string `json:"topClosestLoopName"`
}

type routeExplorerParityPoint struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

func TestRouteExplorerParityFixture_ClosestLoopTopResult(t *testing.T) {
	fixture := loadRouteExplorerParityFixture(t)
	if len(fixture.Cases) == 0 {
		t.Fatal("expected at least one parity fixture case")
	}

	for _, testCase := range fixture.Cases {
		testCase := testCase
		t.Run(testCase.Name, func(t *testing.T) {
			request := routesDomain.RouteExplorerRequest{
				DistanceTargetKm:  testCase.Request.DistanceTargetKm,
				ElevationTargetM:  testCase.Request.ElevationTargetM,
				DurationTargetMin: testCase.Request.DurationTargetMin,
				StartDirection:    testCase.Request.StartDirection,
				RouteType:         testCase.Request.RouteType,
				Limit:             testCase.Request.Limit,
			}
			if testCase.Request.StartPoint != nil {
				request.StartPoint = &routesDomain.Coordinates{
					Lat: testCase.Request.StartPoint.Lat,
					Lng: testCase.Request.StartPoint.Lng,
				}
			}

			activities := make([]*strava.Activity, 0, len(testCase.Activities))
			for _, activity := range testCase.Activities {
				activities = append(activities, toParityStravaActivity(activity))
			}

			result := computeRouteExplorerFromActivities(activities, request)
			if len(result.ClosestLoops) == 0 {
				t.Fatalf("expected at least one closest loop recommendation for case %q", testCase.Name)
			}
			got := result.ClosestLoops[0].Activity.Name
			if got != testCase.Expect.TopClosestLoopName {
				t.Fatalf("top closest loop mismatch for case %q: got %q want %q", testCase.Name, got, testCase.Expect.TopClosestLoopName)
			}
		})
	}
}

func toParityStravaActivity(activity routeExplorerParityAction) *strava.Activity {
	activityType := activity.Type
	if activityType == "" {
		activityType = "Ride"
	}
	sportType := activity.SportType
	if sportType == "" {
		sportType = activityType
	}
	return &strava.Activity{
		Id:                 activity.ID,
		Name:               activity.Name,
		Type:               activityType,
		SportType:          sportType,
		StartDate:          activity.StartDate,
		StartDateLocal:     activity.StartDate,
		Distance:           activity.DistanceKm * 1000.0,
		TotalElevationGain: activity.ElevationM,
		MovingTime:         activity.DurationSec,
		ElapsedTime:        activity.DurationSec,
		StartLatlng:        activity.Start,
		Stream: &strava.Stream{
			LatLng: &strava.LatLngStream{
				Data: activity.Track,
			},
		},
	}
}

func loadRouteExplorerParityFixture(t *testing.T) routeExplorerParityFixture {
	t.Helper()
	fixturePath, err := findFixtureFile("test-fixtures/routes/route-explorer-parity.json")
	if err != nil {
		t.Fatalf("failed to locate parity fixture file: %v", err)
	}
	payload, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("failed to read parity fixture file %q: %v", fixturePath, err)
	}
	var fixture routeExplorerParityFixture
	if err := json.Unmarshal(payload, &fixture); err != nil {
		t.Fatalf("failed to decode parity fixture file %q: %v", fixturePath, err)
	}
	return fixture
}

func findFixtureFile(relativePath string) (string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	visited := map[string]struct{}{}
	for {
		if _, seen := visited[currentDir]; seen {
			break
		}
		visited[currentDir] = struct{}{}

		candidate := filepath.Join(currentDir, filepath.FromSlash(relativePath))
		if info, statErr := os.Stat(candidate); statErr == nil && !info.IsDir() {
			return candidate, nil
		}

		parent := filepath.Dir(currentDir)
		if parent == currentDir {
			break
		}
		currentDir = parent
	}

	return "", fmt.Errorf("unable to find %q from current directory tree", relativePath)
}
