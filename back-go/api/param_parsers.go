package api

import (
	"fmt"
	routesDomain "mystravastats/internal/routes/domain"
	"mystravastats/internal/shared/domain/business"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

func getActivityTypeParam(request *http.Request) ([]business.ActivityType, error) {
	activityTypeStr := request.URL.Query().Get("activityType")
	if activityTypeStr == "" {
		return nil, fmt.Errorf("activity type must not be empty")
	}
	parts := strings.Split(activityTypeStr, "_")
	activityTypes := make(map[business.ActivityType]struct{}, len(parts))
	for _, p := range parts {
		if p == "" {
			return nil, fmt.Errorf("activity type must not be empty")
		}
		t, ok := business.ActivityTypes[p]
		if !ok {
			return nil, fmt.Errorf("unknown activity type: %s", p)
		}
		activityTypes[t] = struct{}{}
	}
	types := make([]business.ActivityType, 0, len(activityTypes))
	for t := range activityTypes {
		types = append(types, t)
	}
	sort.Slice(types, func(i, j int) bool { return types[i] < types[j] })
	return types, nil
}

func getYearParam(request *http.Request) (*int, error) {
	yearStr := request.URL.Query().Get("year")
	if yearStr == "" {
		return nil, nil
	}
	y, err := strconv.Atoi(yearStr)
	if err != nil {
		return nil, fmt.Errorf("invalid year: %q", yearStr)
	}
	return &y, nil
}

func parseActivityRequestParams(request *http.Request) (*int, []business.ActivityType, error) {
	year, err := getYearParam(request)
	if err != nil {
		return nil, nil, err
	}
	activityTypes, err := getActivityTypeParam(request)
	if err != nil {
		return nil, nil, err
	}
	return year, activityTypes, nil
}

func getPeriodParam(request *http.Request) (business.Period, error) {
	periodParam := request.URL.Query().Get("period")
	if periodParam == "" {
		return "", fmt.Errorf("period is required")
	}
	period := business.Period(periodParam)
	switch period {
	case business.PeriodDays, business.PeriodWeeks, business.PeriodMonths:
		return period, nil
	default:
		return "", fmt.Errorf("invalid period: %q", periodParam)
	}
}

func getBadgeSetParam(request *http.Request) (*business.BadgeSetEnum, error) {
	value := strings.TrimSpace(request.URL.Query().Get("badgeSet"))
	if value == "" {
		return nil, nil
	}
	badgeSet := business.BadgeSetEnum(value)
	switch badgeSet {
	case business.GENERAL, business.FAMOUS:
		return &badgeSet, nil
	default:
		return nil, fmt.Errorf("invalid badgeSet: %q", value)
	}
}

func getMetricParam(request *http.Request) *string {
	metric := strings.TrimSpace(request.URL.Query().Get("metric"))
	if metric == "" {
		return nil
	}
	return &metric
}

func getQueryParam(request *http.Request) *string {
	query := strings.TrimSpace(request.URL.Query().Get("query"))
	if query == "" {
		return nil
	}
	return &query
}

func getFromDateParam(request *http.Request) (*string, error) {
	return getDateParam(request, "from")
}

func getToDateParam(request *http.Request) (*string, error) {
	return getDateParam(request, "to")
}

func getDateParam(request *http.Request, key string) (*string, error) {
	value := strings.TrimSpace(request.URL.Query().Get(key))
	if value == "" {
		return nil, nil
	}
	if _, err := time.Parse("2006-01-02", value); err != nil {
		return nil, fmt.Errorf("invalid %s date: %q (expected YYYY-MM-DD)", key, value)
	}
	return &value, nil
}

func getFloatParam(request *http.Request, key string) (*float64, error) {
	value := strings.TrimSpace(request.URL.Query().Get(key))
	if value == "" {
		return nil, nil
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid %s: %q", key, value)
	}
	return &parsed, nil
}

func getIntParam(request *http.Request, key string) (*int, error) {
	value := strings.TrimSpace(request.URL.Query().Get(key))
	if value == "" {
		return nil, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return nil, fmt.Errorf("invalid %s: %q", key, value)
	}
	return &parsed, nil
}

func getBoolParam(request *http.Request, key string) (*bool, error) {
	value := strings.TrimSpace(request.URL.Query().Get(key))
	if value == "" {
		return nil, nil
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return nil, fmt.Errorf("invalid %s: %q", key, value)
	}
	return &parsed, nil
}

func getOptionalStringParam(request *http.Request, key string) *string {
	value := strings.TrimSpace(request.URL.Query().Get(key))
	if value == "" {
		return nil
	}
	return &value
}

func getTargetTypeParam(request *http.Request) *string {
	targetType := strings.TrimSpace(request.URL.Query().Get("targetType"))
	if targetType == "" {
		return nil
	}
	return &targetType
}

func getTargetIDParam(request *http.Request) (*int64, error) {
	targetID := strings.TrimSpace(request.URL.Query().Get("targetId"))
	if targetID == "" {
		return nil, nil
	}
	id, err := strconv.ParseInt(targetID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid targetId: %q", targetID)
	}
	return &id, nil
}

func getSegmentIDPathParam(request *http.Request) (int64, error) {
	segmentIDValue := strings.TrimSpace(mux.Vars(request)["segmentId"])
	if segmentIDValue == "" {
		return 0, fmt.Errorf("segmentId path parameter is required")
	}
	segmentID, err := strconv.ParseInt(segmentIDValue, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid segmentId: %q", segmentIDValue)
	}
	return segmentID, nil
}

func toOptionalStartPoint(lat *float64, lng *float64) (*routesDomain.Coordinates, error) {
	if lat == nil && lng == nil {
		return nil, nil
	}
	if lat == nil || lng == nil {
		return nil, fmt.Errorf("startLat and startLng must be provided together")
	}
	if !isValidLatLng(*lat, *lng) {
		return nil, fmt.Errorf("invalid startLat/startLng coordinates")
	}
	return &routesDomain.Coordinates{Lat: *lat, Lng: *lng}, nil
}

func parseRouteExplorerRequestParams(request *http.Request) (*int, []business.ActivityType, routesDomain.RouteExplorerRequest, error) {
	year, err := getYearParam(request)
	if err != nil {
		return nil, nil, routesDomain.RouteExplorerRequest{}, err
	}
	activityTypes, err := getActivityTypeParam(request)
	if err != nil {
		return nil, nil, routesDomain.RouteExplorerRequest{}, err
	}
	distanceTargetKm, err := getFloatParam(request, "distanceTargetKm")
	if err != nil {
		return nil, nil, routesDomain.RouteExplorerRequest{}, err
	}
	elevationTargetM, err := getFloatParam(request, "elevationTargetM")
	if err != nil {
		return nil, nil, routesDomain.RouteExplorerRequest{}, err
	}
	durationTargetMin, err := getIntParam(request, "durationTargetMin")
	if err != nil {
		return nil, nil, routesDomain.RouteExplorerRequest{}, err
	}
	startLat, err := getFloatParam(request, "startLat")
	if err != nil {
		return nil, nil, routesDomain.RouteExplorerRequest{}, err
	}
	startLng, err := getFloatParam(request, "startLng")
	if err != nil {
		return nil, nil, routesDomain.RouteExplorerRequest{}, err
	}
	startPoint, err := toOptionalStartPoint(startLat, startLng)
	if err != nil {
		return nil, nil, routesDomain.RouteExplorerRequest{}, err
	}
	limit, err := getIntParam(request, "limit")
	if err != nil {
		return nil, nil, routesDomain.RouteExplorerRequest{}, err
	}
	includeRemix, err := getBoolParam(request, "includeRemix")
	if err != nil {
		return nil, nil, routesDomain.RouteExplorerRequest{}, err
	}

	req := routesDomain.RouteExplorerRequest{
		DistanceTargetKm:  distanceTargetKm,
		ElevationTargetM:  elevationTargetM,
		DurationTargetMin: durationTargetMin,
		StartPoint:        startPoint,
		StartDirection:    getOptionalStringParam(request, "startDirection"),
		RouteType:         getOptionalStringParam(request, "routeType"),
		Season:            getOptionalStringParam(request, "season"),
		Limit:             optionalIntValue(limit),
		Shape:             getOptionalStringParam(request, "shape"),
		ShapePolyline:     getOptionalStringParam(request, "shapePolyline"),
		IncludeRemix:      includeRemix != nil && *includeRemix,
	}
	return year, activityTypes, req, nil
}

func optionalIntValue(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}
