package infrastructure

import (
	"fmt"
	application "mystravastats/internal/activities/application"
	dataqualityInfra "mystravastats/internal/dataquality/infrastructure"
	"mystravastats/internal/helpers"
	"mystravastats/internal/platform/activityprovider"
	"mystravastats/internal/shared/domain/business"
	"mystravastats/internal/shared/domain/strava"
	"strings"
)

// DetailedActivityServiceAdapter computes activity read models from provider data.
type DetailedActivityServiceAdapter struct{}

func NewDetailedActivityServiceAdapter() *DetailedActivityServiceAdapter {
	return &DetailedActivityServiceAdapter{}
}

func (adapter *DetailedActivityServiceAdapter) FindDetailedActivityByID(activityID int64) (*strava.DetailedActivity, error) {
	detailedActivity := activityprovider.Get().GetDetailedActivity(activityID)
	if detailedActivity == nil {
		return nil, fmt.Errorf("activity %d not found", activityID)
	}
	return detailedActivity, nil
}

func (adapter *DetailedActivityServiceAdapter) FindActivitiesByYearAndTypes(year *int, activityTypes ...business.ActivityType) []*strava.Activity {
	return activityprovider.Get().GetActivitiesByYearAndActivityTypes(year, activityTypes...)
}

func (adapter *DetailedActivityServiceAdapter) ExportCSVByYearAndTypes(year *int, activityTypes ...business.ActivityType) string {
	return computeExportCSVByYearAndTypes(year, activityTypes...)
}

func (adapter *DetailedActivityServiceAdapter) FindGPXByYearAndTypes(year *int, activityTypes ...business.ActivityType) []application.MapTrack {
	activities := activityprovider.Get().GetActivitiesByYearAndActivityTypes(year, activityTypes...)

	step := 100
	if year != nil {
		step = 10
	}

	var result []application.MapTrack
	for _, activity := range activities {
		if activity.Stream == nil || activity.Stream.LatLng == nil {
			continue
		}
		var coordinates [][]float64
		for i, pair := range activity.Stream.LatLng.Data {
			if i%step == 0 {
				coordinates = append(coordinates, []float64{pair[0], pair[1]})
			}
		}
		if len(activity.Stream.LatLng.Data) > 0 {
			lastPair := activity.Stream.LatLng.Data[len(activity.Stream.LatLng.Data)-1]
			if len(coordinates) == 0 || coordinates[len(coordinates)-1][0] != lastPair[0] || coordinates[len(coordinates)-1][1] != lastPair[1] {
				coordinates = append(coordinates, []float64{lastPair[0], lastPair[1]})
			}
		}
		if len(coordinates) < 2 {
			continue
		}
		result = append(result, application.MapTrack{
			ActivityID:     activity.Id,
			ActivityName:   activity.Name,
			ActivityDate:   activity.StartDateLocal,
			ActivityType:   resolveMapTrackActivityType(activity),
			DistanceKm:     activity.Distance / 1000.0,
			ElevationGainM: activity.TotalElevationGain,
			Coordinates:    coordinates,
		})
	}

	return result
}

func (adapter *DetailedActivityServiceAdapter) FindPassagesByYearAndTypes(year *int, activityTypes ...business.ActivityType) application.MapPassagesResponse {
	activities := activityprovider.Get().GetActivitiesByYearAndActivityTypes(year, activityTypes...)
	return computeMapPassagesWithOptions(activities, dataqualityInfra.CurrentProviderExclusions(), mapPassageOptionsForYear(year))
}

func resolveMapTrackActivityType(activity *strava.Activity) string {
	if activity == nil {
		return ""
	}

	if activity.Commute {
		return business.Commute.String()
	}

	sportType := strings.TrimSpace(helpers.FirstNonEmpty(activity.SportType, activity.Type))
	if sportType != "" {
		return sportType
	}
	return business.Ride.String()
}
