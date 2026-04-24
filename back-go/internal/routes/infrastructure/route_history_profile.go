package infrastructure

import (
	"fmt"
	"math"
	"mystravastats/internal/helpers"
	"mystravastats/internal/platform/runtimeconfig"
	"mystravastats/internal/routes/application"
	"mystravastats/internal/shared/domain/business"
	"mystravastats/internal/shared/domain/strava"
	"strings"
	"time"
)

const (
	defaultRoutingHistoryHalfLifeDays = 75
	historyAxisNodePrecision          = 4
	historyZonePrecision              = 2
	minHistorySegmentLengthM          = 25.0
)

func routingHistoryBiasEnabled() bool {
	return runtimeconfig.BoolValue("OSM_ROUTING_HISTORY_BIAS_ENABLED", false)
}

func routingHistoryHalfLifeDays() float64 {
	return float64(runtimeconfig.RoutingHistoryHalfLifeDays())
}

func buildRoutingHistoryProfileFromActivities(
	activities []*strava.Activity,
	routeType string,
	now time.Time,
	halfLifeDays float64,
) *application.RoutingHistoryProfile {
	if len(activities) == 0 {
		return nil
	}
	normalizedRouteType := normalizeHistoryRouteType(routeType)
	if halfLifeDays <= 0 {
		halfLifeDays = float64(defaultRoutingHistoryHalfLifeDays)
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}

	axisScores := make(map[string]float64)
	zoneScores := make(map[string]float64)
	activityCount := 0
	segmentCount := 0
	latestActivityEpochMs := int64(0)

	for _, activity := range activities {
		if activity == nil || !historyRouteTypeMatchesActivity(normalizedRouteType, activity) {
			continue
		}
		points := extractHistoryTrackPoints(activity)
		if len(points) < 2 {
			continue
		}

		activityWeight := historyRecencyWeight(activity, now, halfLifeDays)
		if activityWeight <= 0 {
			continue
		}

		activityContributed := false
		for index := 1; index < len(points); index++ {
			from := points[index-1]
			to := points[index]
			segmentLengthM := haversineDistanceMeters(from[0], from[1], to[0], to[1])
			if !isFinitePositive(segmentLengthM) || segmentLengthM < minHistorySegmentLengthM {
				continue
			}

			axisID := historyAxisKey(from[0], from[1], to[0], to[1])
			zoneID := historyZoneKey((from[0]+to[0])/2.0, (from[1]+to[1])/2.0)
			contribution := segmentLengthM * activityWeight

			axisScores[axisID] += contribution
			zoneScores[zoneID] += contribution
			segmentCount++
			activityContributed = true
		}

		if !activityContributed {
			continue
		}
		activityCount++
		if activityTime, ok := historyActivityTimestamp(activity); ok {
			epochMs := activityTime.UnixMilli()
			if epochMs > latestActivityEpochMs {
				latestActivityEpochMs = epochMs
			}
		}
	}

	if activityCount == 0 || segmentCount == 0 || len(axisScores) == 0 {
		return nil
	}

	return &application.RoutingHistoryProfile{
		RouteType:             normalizedRouteType,
		HalfLifeDays:          int(math.Round(halfLifeDays)),
		ActivityCount:         activityCount,
		SegmentCount:          segmentCount,
		AxisScores:            axisScores,
		ZoneScores:            zoneScores,
		LatestActivityEpochMs: latestActivityEpochMs,
	}
}

func normalizeHistoryRouteType(routeType string) string {
	normalized := strings.ToUpper(strings.TrimSpace(routeType))
	switch normalized {
	case "RIDE", "MTB", "GRAVEL", "RUN", "TRAIL", "HIKE":
		return normalized
	default:
		return "RIDE"
	}
}

func historyRouteTypeMatchesActivity(routeType string, activity *strava.Activity) bool {
	activityType, ok := resolveHistoryActivityType(activity)
	if !ok {
		return false
	}
	switch routeType {
	case "GRAVEL":
		return activityType == business.GravelRide
	case "MTB":
		return activityType == business.MountainBikeRide
	case "RUN":
		return activityType == business.Run
	case "TRAIL":
		return activityType == business.TrailRun
	case "HIKE":
		return activityType == business.Hike || activityType == business.Walk
	case "RIDE":
		return activityType == business.Ride || activityType == business.Commute || activityType == business.VirtualRide
	default:
		return activityType == business.Ride
	}
}

func resolveHistoryActivityType(activity *strava.Activity) (business.ActivityType, bool) {
	if activity == nil {
		return 0, false
	}
	rawType := strings.TrimSpace(activity.SportType)
	if rawType == "" {
		rawType = strings.TrimSpace(activity.Type)
	}
	activityType, ok := business.ActivityTypes[rawType]
	if !ok {
		return 0, false
	}
	return activityType, true
}

func extractHistoryTrackPoints(activity *strava.Activity) [][]float64 {
	if activity == nil || activity.Stream == nil || activity.Stream.LatLng == nil {
		return [][]float64{}
	}
	rawPoints := activity.Stream.LatLng.Data
	if len(rawPoints) < 2 {
		return [][]float64{}
	}
	points := make([][]float64, 0, len(rawPoints))
	for _, point := range rawPoints {
		if len(point) < 2 {
			continue
		}
		lat := point[0]
		lng := point[1]
		if !isFiniteCoordinate(lat, lng) {
			continue
		}
		points = append(points, []float64{lat, lng})
	}
	if len(points) < 2 {
		return [][]float64{}
	}
	return points
}

func historyActivityTimestamp(activity *strava.Activity) (time.Time, bool) {
	if activity == nil {
		return time.Time{}, false
	}
	dateRaw := helpers.FirstNonEmpty(activity.StartDateLocal, activity.StartDate)
	if dateRaw == "" {
		return time.Time{}, false
	}
	activityTime, ok := helpers.ParseActivityDate(dateRaw)
	if !ok {
		return time.Time{}, false
	}
	return activityTime.UTC(), true
}

func historyRecencyWeight(activity *strava.Activity, now time.Time, halfLifeDays float64) float64 {
	if halfLifeDays <= 0 {
		halfLifeDays = float64(defaultRoutingHistoryHalfLifeDays)
	}
	activityTime, ok := historyActivityTimestamp(activity)
	if !ok {
		return 1.0
	}
	if activityTime.After(now) {
		return 1.0
	}
	ageDays := now.Sub(activityTime).Hours() / 24.0
	decayExponent := -math.Ln2 * ageDays / halfLifeDays
	return math.Exp(decayExponent)
}

func historyAxisKey(lat1 float64, lng1 float64, lat2 float64, lng2 float64) string {
	from := historyNodeKey(lat1, lng1, historyAxisNodePrecision)
	to := historyNodeKey(lat2, lng2, historyAxisNodePrecision)
	return canonicalEdgeKey(from, to)
}

func historyZoneKey(lat float64, lng float64) string {
	return historyNodeKey(lat, lng, historyZonePrecision)
}

func historyNodeKey(lat float64, lng float64, precision int) string {
	return fmt.Sprintf("%.*f:%.*f", precision, lat, precision, lng)
}

func isFiniteCoordinate(lat float64, lng float64) bool {
	if math.IsNaN(lat) || math.IsInf(lat, 0) || math.IsNaN(lng) || math.IsInf(lng, 0) {
		return false
	}
	return lat >= -90.0 && lat <= 90.0 && lng >= -180.0 && lng <= 180.0
}

func isFinitePositive(value float64) bool {
	return !(math.IsNaN(value) || math.IsInf(value, 0)) && value > 0.0
}
