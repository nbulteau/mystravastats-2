package infrastructure

import (
	"encoding/json"
	"fmt"
	routesDomain "mystravastats/internal/routes/domain"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// osrmClient owns HTTP transport and OSRM wire-protocol validation. Routing
// orchestration stays in OSMRoutingAdapter and only consumes typed results.
type osrmClient struct {
	baseURL string
	client  *http.Client
}

func newOSRMClient(baseURL string, timeout time.Duration) *osrmClient {
	return &osrmClient{baseURL: baseURL, client: &http.Client{Timeout: timeout}}
}

func (client *osrmClient) healthStatus() (int, error) {
	request, err := http.NewRequest(http.MethodGet, client.baseURL+"/", nil)
	if err != nil {
		return 0, err
	}
	response, err := client.client.Do(request)
	if err != nil {
		return 0, err
	}
	defer func() { _ = response.Body.Close() }()
	return response.StatusCode, nil
}

func (client *osrmClient) routes(profile string, waypoints []routesDomain.Coordinates, continueStraight bool) ([]osrmRoute, error) {
	if len(waypoints) < 2 {
		return nil, fmt.Errorf("at least 2 waypoints are required")
	}
	coordinates := make([]string, 0, len(waypoints))
	for _, point := range waypoints {
		coordinates = append(coordinates, fmt.Sprintf("%.6f,%.6f", point.Lng, point.Lat))
	}
	url := fmt.Sprintf("%s/route/v1/%s/%s?alternatives=true&steps=true&overview=full&geometries=geojson&continue_straight=%s", client.baseURL, profile, strings.Join(coordinates, ";"), strconv.FormatBool(continueStraight))

	var payload osrmRouteResponse
	if err := client.getJSON(url, &payload); err != nil {
		return nil, fmt.Errorf("osrm route API: %w", err)
	}
	if !strings.EqualFold(payload.Code, "ok") {
		if payload.Message == "" {
			return nil, fmt.Errorf("osrm route API returned code %s", payload.Code)
		}
		return nil, fmt.Errorf("osrm route API returned code %s: %s", payload.Code, payload.Message)
	}
	return payload.Routes, nil
}

func (client *osrmClient) match(profile string, shape []routesDomain.Coordinates) ([]osrmRoute, error) {
	if len(shape) < 2 {
		return nil, fmt.Errorf("at least 2 shape points are required for map matching")
	}
	sampled := sampleCoordinatesByDistance(shape, 48, 45.0)
	if len(sampled) < 2 {
		return nil, fmt.Errorf("at least 2 sampled shape points are required for map matching")
	}
	coordinates := make([]string, 0, len(sampled))
	for _, point := range sampled {
		coordinates = append(coordinates, fmt.Sprintf("%.6f,%.6f", point.Lng, point.Lat))
	}

	var lastErr error
	for _, radiusMeters := range []float64{35, 80, 160, 320} {
		radiuses := make([]string, len(sampled))
		for index := range radiuses {
			radiuses[index] = fmt.Sprintf("%.0f", radiusMeters)
		}
		url := fmt.Sprintf("%s/match/v1/%s/%s?steps=true&overview=full&geometries=geojson&gaps=ignore&tidy=true&radiuses=%s", client.baseURL, profile, strings.Join(coordinates, ";"), strings.Join(radiuses, ";"))
		var payload osrmMatchResponse
		if err := client.getJSON(url, &payload); err != nil {
			lastErr = fmt.Errorf("osrm match API: %w", err)
			continue
		}
		if !strings.EqualFold(payload.Code, "ok") {
			lastErr = fmt.Errorf("osrm match API returned code %s", payload.Code)
			if !strings.EqualFold(payload.Code, "nomatch") {
				return nil, lastErr
			}
			continue
		}
		routes := make([]osrmRoute, 0, len(payload.Matchings))
		for _, route := range payload.Matchings {
			if points, ok := osrmRouteToPreviewPoints(route); ok && len(points) >= 2 && route.Distance > 0 {
				routes = append(routes, route)
			}
		}
		if len(routes) > 0 {
			return routes, nil
		}
		lastErr = fmt.Errorf("osrm match API returned no valid matchings")
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("osrm match API returned no route")
}

func (client *osrmClient) nearest(profile string, point routesDomain.Coordinates) (routesDomain.Coordinates, float64, bool) {
	url := fmt.Sprintf("%s/nearest/v1/%s/%.6f,%.6f?number=1", client.baseURL, profile, point.Lng, point.Lat)
	var payload osrmNearestResponse
	if err := client.getJSON(url, &payload); err != nil || !strings.EqualFold(strings.TrimSpace(payload.Code), "ok") || len(payload.Waypoints) == 0 {
		return routesDomain.Coordinates{}, 0, false
	}
	waypoint := payload.Waypoints[0]
	if len(waypoint.Location) < 2 {
		return routesDomain.Coordinates{}, 0, false
	}
	return routesDomain.Coordinates{Lat: waypoint.Location[1], Lng: waypoint.Location[0]}, max(0, waypoint.Distance), true
}

func (client *osrmClient) getJSON(url string, target any) error {
	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	response, err := client.client.Do(request)
	if err != nil {
		return err
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("returned status %d", response.StatusCode)
	}
	return json.NewDecoder(response.Body).Decode(target)
}
