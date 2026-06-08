package composite

import (
	"testing"

	"mystravastats/internal/shared/domain/business"
	"mystravastats/internal/shared/domain/strava"
)

func TestCompositeKeepsStravaIDAndEnrichesWithLocalStream(t *testing.T) {
	stravaActivity := testActivity(123, "strava ride", "Ride", "2026-05-01T08:00:00Z", 10000, 3600, nil)
	gpxStream := testStream(120)
	gpxActivity := testActivity(9001, "local ride", "Ride", "2026-05-01T08:04:00Z", 10100, 3620, gpxStream)

	provider := NewCompositeActivityProvider([]Source{
		{Name: "strava", Provider: testProvider{name: "strava", activities: []*strava.Activity{stravaActivity}}},
		{Name: "gpx", Provider: testProvider{name: "gpx", activities: []*strava.Activity{gpxActivity}}},
	})

	activities := provider.GetActivitiesByYearAndActivityTypes(nil, business.Ride)
	if len(activities) != 1 {
		t.Fatalf("expected one merged activity, got %d", len(activities))
	}
	if activities[0].Id != 123 {
		t.Fatalf("expected Strava id to be preserved, got %d", activities[0].Id)
	}
	if activities[0].Stream == nil || activities[0].Stream.LatLng == nil || len(activities[0].Stream.LatLng.Data) != 120 {
		t.Fatalf("expected GPX stream enrichment, got %#v", activities[0].Stream)
	}
	detailed := provider.GetDetailedActivity(123)
	if detailed == nil || detailed.Source == nil {
		t.Fatalf("expected detailed activity provenance, got %#v", detailed)
	}
	if detailed.Source.PrimaryProvider != "strava" || detailed.Source.StreamProvider != "gpx" {
		t.Fatalf("expected Strava primary and GPX stream provenance, got %#v", detailed.Source)
	}
	if len(detailed.Source.Sources) != 2 {
		t.Fatalf("expected two source references, got %#v", detailed.Source.Sources)
	}

	details := provider.CacheDiagnostics()
	compositeDetails := details["composite"].(map[string]any)
	if compositeDetails["matchedActivities"] != 1 {
		t.Fatalf("expected one matched activity, got %#v", compositeDetails["matchedActivities"])
	}
}

func TestCompositeMatchesOneHourTimezoneOffset(t *testing.T) {
	stravaActivity := testActivity(8395020437, "strava ride", "Ride", "2023-01-15T10:52:42Z", 40000, 7200, nil)
	gpxActivity := testActivity(853484847, "local ride", "Ride", "2023-01-15T11:52:42Z", 40150, 7210, testStream(180))

	provider := NewCompositeActivityProvider([]Source{
		{Name: "strava", Provider: testProvider{name: "strava", activities: []*strava.Activity{stravaActivity}}},
		{Name: "gpx", Provider: testProvider{name: "gpx", activities: []*strava.Activity{gpxActivity}}},
	})

	activities := provider.GetActivitiesByYearAndActivityTypes(nil, business.Ride)
	if len(activities) != 1 {
		t.Fatalf("expected one merged activity, got %d", len(activities))
	}
	if activities[0].Id != 8395020437 {
		t.Fatalf("expected Strava id to be preserved, got %d", activities[0].Id)
	}

	details := provider.CacheDiagnostics()
	compositeDetails := details["composite"].(map[string]any)
	if compositeDetails["matchedActivities"] != 1 {
		t.Fatalf("expected one matched activity, got %#v", compositeDetails["matchedActivities"])
	}
}

func TestCompositeMatchesSummerTimezoneOffset(t *testing.T) {
	stravaActivity := testActivity(8813720582, "strava ride", "Ride", "2023-04-01T13:45:00Z", 40000, 7200, nil)
	gpxActivity := testActivity(3004085239, "local ride", "Ride", "2023-04-01T15:45:00Z", 40150, 7210, testStream(180))

	provider := NewCompositeActivityProvider([]Source{
		{Name: "strava", Provider: testProvider{name: "strava", activities: []*strava.Activity{stravaActivity}}},
		{Name: "gpx", Provider: testProvider{name: "gpx", activities: []*strava.Activity{gpxActivity}}},
	})

	activities := provider.GetActivitiesByYearAndActivityTypes(nil, business.Ride)
	if len(activities) != 1 {
		t.Fatalf("expected one merged activity, got %d", len(activities))
	}
	if activities[0].Id != 8813720582 {
		t.Fatalf("expected Strava id to be preserved, got %d", activities[0].Id)
	}

	details := provider.CacheDiagnostics()
	compositeDetails := details["composite"].(map[string]any)
	if compositeDetails["matchedActivities"] != 1 {
		t.Fatalf("expected one matched activity, got %#v", compositeDetails["matchedActivities"])
	}
}

func TestCompositeRejectsSameStartWhenDistancesDisagree(t *testing.T) {
	stravaActivity := testActivity(9101, "strava ride", "Ride", "2023-07-08T08:00:00Z", 72519, 10248, nil)
	fitActivity := testActivity(9102, "fit ride", "Ride", "2023-07-08T08:03:00Z", 51585, 11312, testStream(180))

	provider := NewCompositeActivityProvider([]Source{
		{Name: "strava", Provider: testProvider{name: "strava", activities: []*strava.Activity{stravaActivity}}},
		{Name: "fit", Provider: testProvider{name: "fit", activities: []*strava.Activity{fitActivity}}},
	})

	activities := provider.GetActivitiesByYearAndActivityTypes(nil, business.Ride)
	if len(activities) != 2 {
		t.Fatalf("expected the conflicting-distance activities to stay separate, got %d", len(activities))
	}

	details := provider.CacheDiagnostics()
	compositeDetails := details["composite"].(map[string]any)
	if compositeDetails["matchedActivities"] != 0 {
		t.Fatalf("expected no matched activity, got %#v", compositeDetails["matchedActivities"])
	}
}

func TestCompositeMatchesSameDistanceWhenMovingTimeDiffers(t *testing.T) {
	stravaActivity := testActivity(9201, "strava ride", "Ride", "2023-07-08T08:00:00Z", 50000, 10757, nil)
	fitActivity := testActivity(9202, "fit ride", "Ride", "2023-07-08T08:03:00Z", 50100, 13197, testStream(180))

	provider := NewCompositeActivityProvider([]Source{
		{Name: "strava", Provider: testProvider{name: "strava", activities: []*strava.Activity{stravaActivity}}},
		{Name: "fit", Provider: testProvider{name: "fit", activities: []*strava.Activity{fitActivity}}},
	})

	activities := provider.GetActivitiesByYearAndActivityTypes(nil, business.Ride)
	if len(activities) != 1 {
		t.Fatalf("expected one merged activity when distance and start match, got %d", len(activities))
	}
	if activities[0].Id != 9201 {
		t.Fatalf("expected Strava id to be preserved, got %d", activities[0].Id)
	}

	details := provider.CacheDiagnostics()
	compositeDetails := details["composite"].(map[string]any)
	if compositeDetails["matchedActivities"] != 1 {
		t.Fatalf("expected one matched activity, got %#v", compositeDetails["matchedActivities"])
	}
}

func TestCompositeKeepsUnmatchedLocalActivitiesInUnion(t *testing.T) {
	fitActivity := testActivity(7001, "fit run", "Run", "2026-05-01T08:00:00Z", 5000, 1800, testStream(20))
	gpxActivity := testActivity(8001, "gpx run", "Run", "2026-05-02T08:00:00Z", 6000, 2100, testStream(25))

	provider := NewCompositeActivityProvider([]Source{
		{Name: "fit", Provider: testProvider{name: "fit", activities: []*strava.Activity{fitActivity}}},
		{Name: "gpx", Provider: testProvider{name: "gpx", activities: []*strava.Activity{gpxActivity}}},
	})

	activities := provider.GetActivitiesByYearAndActivityTypes(nil, business.Run)
	if len(activities) != 2 {
		t.Fatalf("expected two union activities, got %d", len(activities))
	}

	ids := map[int64]bool{}
	for _, activity := range activities {
		ids[activity.Id] = true
	}
	if !ids[7001] || !ids[8001] {
		t.Fatalf("expected local activity ids to be preserved, got %#v", ids)
	}
}

func TestCompositeRebuildsWhenSourceActivityCountChanges(t *testing.T) {
	firstActivity := testActivity(9301, "morning ride", "Ride", "2026-06-08T07:30:00Z", 20000, 3600, nil)
	nextActivity := testActivity(9302, "lunch ride", "Ride", "2026-06-08T12:00:00Z", 15000, 2700, nil)
	source := &testProvider{name: "strava", activities: []*strava.Activity{firstActivity}}
	provider := NewCompositeActivityProvider([]Source{
		{Name: "strava", Provider: source},
	})

	if activities := provider.GetActivitiesByYearAndActivityTypes(nil, business.Ride); len(activities) != 1 {
		t.Fatalf("expected initial cached activity, got %d", len(activities))
	}

	source.activities = append(source.activities, nextActivity)

	activities := provider.GetActivitiesByYearAndActivityTypes(nil, business.Ride)
	if len(activities) != 2 {
		t.Fatalf("expected composite provider to rebuild after source count changed, got %d", len(activities))
	}
	diagnostics := provider.CacheDiagnostics()
	if diagnostics["activities"] != 2 {
		t.Fatalf("expected diagnostics to report rebuilt composite activity count, got %#v", diagnostics["activities"])
	}
}

func TestCompositeDiagnosticsAggregatesSourceRefresh(t *testing.T) {
	source := testProvider{
		name:       "strava",
		activities: []*strava.Activity{testActivity(9401, "morning ride", "Ride", "2026-06-08T07:30:00Z", 20000, 3600, nil)},
		refresh: map[string]any{
			"backgroundInProgress": true,
			"warmupInProgress":     false,
		},
	}
	provider := NewCompositeActivityProvider([]Source{
		{Name: "strava", Provider: source},
	})

	refresh := provider.CacheDiagnostics()["refresh"].(map[string]any)
	if refresh["backgroundInProgress"] != true {
		t.Fatalf("expected composite refresh to include source background refresh, got %#v", refresh)
	}
}

type testProvider struct {
	name       string
	activities []*strava.Activity
	refresh    map[string]any
}

func (provider testProvider) GetDetailedActivity(activityId int64) *strava.DetailedActivity {
	for _, activity := range provider.activities {
		if activity.Id == activityId {
			return activity.ToStravaDetailedActivity()
		}
	}
	return nil
}

func (provider testProvider) GetCachedDetailedActivity(activityId int64) *strava.DetailedActivity {
	return provider.GetDetailedActivity(activityId)
}

func (provider testProvider) GetActivitiesByYearAndActivityTypes(year *int, activityTypes ...business.ActivityType) []*strava.Activity {
	return filterActivitiesByType(filterActivitiesByYear(provider.activities, year), activityTypes...)
}

func (provider testProvider) GetActivitiesByActivityTypeGroupByYear(activityTypes ...business.ActivityType) map[string][]*strava.Activity {
	return groupActivitiesByYear(filterActivitiesByType(provider.activities, activityTypes...))
}

func (provider testProvider) GetActivitiesByActivityTypeGroupByActiveDays(activityTypes ...business.ActivityType) map[string]int {
	return map[string]int{}
}

func (provider testProvider) GetAthlete() strava.Athlete {
	return strava.Athlete{Id: 42}
}

func (provider testProvider) GetHeartRateZoneSettings() business.HeartRateZoneSettings {
	return business.HeartRateZoneSettings{}
}

func (provider testProvider) SaveHeartRateZoneSettings(settings business.HeartRateZoneSettings) business.HeartRateZoneSettings {
	return settings
}

func (provider testProvider) GetPerformanceSettings() business.AthletePerformanceSettings {
	return business.AthletePerformanceSettings{}
}

func (provider testProvider) SavePerformanceSettings(settings business.AthletePerformanceSettings) business.AthletePerformanceSettings {
	return settings
}

func (provider testProvider) CacheDiagnostics() map[string]any {
	diagnostics := map[string]any{
		"provider":          provider.name,
		"activities":        len(provider.activities),
		"availableYearBins": availableYearBins(provider.activities),
	}
	if provider.refresh != nil {
		diagnostics["refresh"] = provider.refresh
	}
	return diagnostics
}

func (provider testProvider) ClientID() string {
	return provider.name + "-athlete"
}

func (provider testProvider) CacheRootPath() string {
	return provider.name + "-cache"
}

func testActivity(id int64, name string, sport string, start string, distance float64, movingTime int, stream *strava.Stream) *strava.Activity {
	return &strava.Activity{
		Id:                 id,
		Name:               name,
		Type:               sport,
		SportType:          sport,
		StartDate:          start,
		StartDateLocal:     start,
		StartLatlng:        []float64{48.8566, 2.3522},
		Distance:           distance,
		ElapsedTime:        movingTime,
		MovingTime:         movingTime,
		AverageSpeed:       distance / float64(movingTime),
		TotalElevationGain: 100,
		Stream:             stream,
	}
}

func testStream(points int) *strava.Stream {
	latlng := make([][]float64, points)
	distance := make([]float64, points)
	times := make([]int, points)
	for index := 0; index < points; index++ {
		latlng[index] = []float64{48.8566 + float64(index)*0.0001, 2.3522}
		distance[index] = float64(index * 10)
		times[index] = index
	}
	return &strava.Stream{
		Distance: strava.DistanceStream{Data: distance, OriginalSize: points},
		Time:     strava.TimeStream{Data: times, OriginalSize: points},
		LatLng:   &strava.LatLngStream{Data: latlng, OriginalSize: points},
	}
}
