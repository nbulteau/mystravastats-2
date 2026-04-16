package stravaapi

import (
	"errors"
	"fmt"
	"mystravastats/internal/shared/domain/business"
	"mystravastats/internal/shared/domain/strava"
	"mystravastats/internal/shared/infrastructure/localrepository"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"sync/atomic"
	"testing"
	"time"
)

func TestFilterActivitiesByType(t *testing.T) {
	// GIVEN
	activities := []*strava.Activity{
		{Type: "Ride", Commute: false},
		{Type: "Run", Commute: false},
		{Type: "Ride", Commute: true},
	}

	// WHEN
	rides := FilterActivitiesByType(activities, business.Ride)
	commutes := FilterActivitiesByType(activities, business.Commute)

	// THEN
	if len(rides) != 1 {
		t.Errorf("Expected 1 ride, got %d", len(rides))
	}

	if len(commutes) != 1 {
		t.Errorf("Expected 1 commute, got %d", len(commutes))
	}
}

func TestShouldBootstrapFromStravaAPIWhenCurrentYearCacheIsFresh(t *testing.T) {
	// GIVEN
	cacheDir := t.TempDir()
	repo := localrepository.NewStravaRepository(cacheDir)
	clientID := "123"
	currentYear := time.Now().Year()
	repo.InitLocalStorageForClientId(clientID)
	repo.SaveAthleteToCache(clientID, strava.Athlete{Id: 123})
	repo.SaveActivitiesToCache(clientID, currentYear, []strava.Activity{{Id: 1, StartDate: fmt.Sprintf("%d-01-01T10:00:00Z", currentYear)}})

	provider := &StravaActivityProvider{
		localStorageProvider: repo,
	}

	// WHEN
	shouldBootstrap := provider.shouldBootstrapFromStravaAPI(clientID)

	// THEN
	if shouldBootstrap {
		t.Fatal("expected fresh local cache to skip Strava bootstrap")
	}
}

func TestShouldBootstrapFromStravaAPIWhenCurrentYearCacheIsMissing(t *testing.T) {
	// GIVEN
	cacheDir := t.TempDir()
	repo := localrepository.NewStravaRepository(cacheDir)
	clientID := "123"
	repo.InitLocalStorageForClientId(clientID)
	repo.SaveAthleteToCache(clientID, strava.Athlete{Id: 123})

	provider := &StravaActivityProvider{
		localStorageProvider: repo,
	}

	// WHEN
	shouldBootstrap := provider.shouldBootstrapFromStravaAPI(clientID)

	// THEN
	if !shouldBootstrap {
		t.Fatal("expected missing current-year cache to require Strava bootstrap")
	}
}

func TestShouldBootstrapFromStravaAPIWhenCacheIsTooOld(t *testing.T) {
	// GIVEN
	cacheDir := t.TempDir()
	repo := localrepository.NewStravaRepository(cacheDir)
	clientID := "123"
	currentYear := time.Now().Year()
	repo.InitLocalStorageForClientId(clientID)
	repo.SaveAthleteToCache(clientID, strava.Athlete{Id: 123})
	repo.SaveActivitiesToCache(clientID, currentYear, []strava.Activity{{Id: 1, StartDate: fmt.Sprintf("%d-01-01T10:00:00Z", currentYear)}})

	activitiesFile := filepath.Join(cacheDir, fmt.Sprintf("strava-%s", clientID), fmt.Sprintf("strava-%s-%d", clientID, currentYear), fmt.Sprintf("activities-%s-%d.json", clientID, currentYear))
	oldTime := time.Date(2025, time.August, 1, 0, 0, 0, 0, time.UTC)
	if err := os.Chtimes(activitiesFile, oldTime, oldTime); err != nil {
		t.Fatalf("failed to age cache file: %v", err)
	}

	provider := &StravaActivityProvider{
		localStorageProvider: repo,
	}

	// WHEN
	shouldBootstrap := provider.shouldBootstrapFromStravaAPI(clientID)

	// THEN
	if !shouldBootstrap {
		t.Fatal("expected stale current-year cache to require Strava bootstrap")
	}
}

func BenchmarkGetActivityIndexed(b *testing.B) {
	provider := benchmarkProvider(50000)
	targetID := int64(49999)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if provider.GetActivity(targetID) == nil {
			b.Fatal("expected activity to be found")
		}
	}
}

func BenchmarkGetActivitiesByYearAndActivityTypesCached(b *testing.B) {
	provider := benchmarkProvider(50000)
	year := 2024

	if got := provider.GetActivitiesByYearAndActivityTypes(&year, business.Ride); len(got) == 0 {
		b.Fatal("expected non-empty filtered activities")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = provider.GetActivitiesByYearAndActivityTypes(&year, business.Ride)
	}
}

func TestGetActivitiesByYearAndActivityTypesReturnsDefensiveCopy(t *testing.T) {
	// GIVEN
	provider := benchmarkProvider(100)
	year := 2024

	firstCall := provider.GetActivitiesByYearAndActivityTypes(&year, business.Ride)
	if len(firstCall) < 2 {
		t.Fatalf("expected at least 2 activities, got %d", len(firstCall))
	}
	expectedFirstID := firstCall[0].Id

	// WHEN: Mutate the returned ordering; cached data should not be impacted.
	sort.Slice(firstCall, func(i, j int) bool {
		return firstCall[i].Id > firstCall[j].Id
	})

	secondCall := provider.GetActivitiesByYearAndActivityTypes(&year, business.Ride)

	// THEN
	if len(secondCall) == 0 {
		t.Fatal("expected non-empty activities on second call")
	}
	if secondCall[0].Id != expectedFirstID {
		t.Fatalf(
			"expected defensive copy to preserve cached ordering, got first id %d (expected %d)",
			secondCall[0].Id,
			expectedFirstID,
		)
	}
}

func TestRateLimitCircuitBreaker(t *testing.T) {
	// GIVEN
	provider := &StravaActivityProvider{}

	// WHEN & THEN: Initial state
	if provider.isStravaRateLimitedNow() {
		t.Fatal("expected rate limit breaker to be inactive initially")
	}

	// WHEN: Mark with non-rate-limit error
	provider.markStravaRateLimited(errors.New("network timeout"), "non-rate-limit")

	// THEN: Should not activate
	if provider.isStravaRateLimitedNow() {
		t.Fatal("expected non-rate-limit errors to not activate breaker")
	}

	// WHEN: Mark with actual rate limit error
	provider.markStravaRateLimited(ErrStravaRateLimitReached, "unit-test")

	// THEN: Should be active
	if !provider.isStravaRateLimitedNow() {
		t.Fatal("expected rate limit breaker to be active after 429")
	}

	// WHEN: Set expiration in the past
	provider.rateLimitUntilUnix.Store(time.Now().Add(-time.Second).Unix())

	// THEN: Should expire
	if provider.isStravaRateLimitedNow() {
		t.Fatal("expected breaker to expire after cooldown deadline")
	}
}

func TestGetDetailedActivity_SkipsStravaCallWhenRateLimitAlreadyActive(t *testing.T) {
	// GIVEN
	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v3/activities/42" {
			atomic.AddInt32(&calls, 1)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":42}`))
	}))
	defer server.Close()

	cacheDir := t.TempDir()
	repo := localrepository.NewStravaRepository(cacheDir)
	clientID := "123"
	repo.InitLocalStorageForClientId(clientID)

	provider := &StravaActivityProvider{
		clientId:             clientID,
		localStorageProvider: repo,
		StravaApi: &StravaApi{
			accessToken: "test-token",
			properties: StravaProperties{
				URL: server.URL,
			},
			httpClient: server.Client(),
		},
		activities: []*strava.Activity{
			{
				Id:             42,
				StartDateLocal: "2020-01-01T00:00:00Z",
				StartDate:      "2020-01-01T00:00:00Z",
				UploadId:       1,
			},
		},
	}
	provider.indexActivities()
	provider.rateLimitUntilUnix.Store(time.Now().Add(5 * time.Minute).Unix())

	// WHEN
	detailed := provider.GetDetailedActivity(42)
	apiCalls := atomic.LoadInt32(&calls)

	// THEN
	if detailed == nil {
		t.Fatal("expected fallback detailed activity while rate limit is active")
	}
	if apiCalls != 0 {
		t.Fatalf("expected no Strava API call during active rate limit, got %d", apiCalls)
	}
}

func TestGetDetailedActivity_ReturnsCachedDetailedWithoutBaseActivity(t *testing.T) {
	// GIVEN
	cacheDir := t.TempDir()
	repo := localrepository.NewStravaRepository(cacheDir)
	clientID := "123"
	repo.InitLocalStorageForClientId(clientID)
	repo.SaveActivitiesToCache(clientID, 2024, []strava.Activity{})

	expectedID := int64(4242)
	repo.SaveDetailedActivityToCache(clientID, 2024, strava.DetailedActivity{
		Id:   expectedID,
		Name: "cached detailed activity",
	})

	provider := &StravaActivityProvider{
		clientId:             clientID,
		localStorageProvider: repo,
	}

	// WHEN
	detailed := provider.GetDetailedActivity(expectedID)

	// THEN
	if detailed == nil {
		t.Fatal("expected cached detailed activity even without base activity metadata")
	}
	if detailed.Id != expectedID {
		t.Fatalf("expected detailed activity id %d, got %d", expectedID, detailed.Id)
	}
}

func TestGetDetailedActivity_PersistsFallbackDetailedToDiskCache(t *testing.T) {
	// GIVEN
	cacheDir := t.TempDir()
	repo := localrepository.NewStravaRepository(cacheDir)
	clientID := "123"
	repo.InitLocalStorageForClientId(clientID)

	activityID := int64(42)
	activityYear := 2021
	repo.SaveActivitiesToCache(clientID, activityYear, []strava.Activity{})
	provider := &StravaActivityProvider{
		clientId:             clientID,
		localStorageProvider: repo,
		activities: []*strava.Activity{
			{
				Id:             activityID,
				Name:           "fallback detailed",
				StartDateLocal: "2021-07-01T08:00:00Z",
				StartDate:      "2021-07-01T08:00:00Z",
				Distance:       1234,
				ElapsedTime:    300,
				MovingTime:     295,
				Type:           "Ride",
				SportType:      "Ride",
				UploadId:       1,
			},
		},
	}
	provider.indexActivities()

	// WHEN
	detailed := provider.GetDetailedActivity(activityID)

	// THEN
	if detailed == nil {
		t.Fatal("expected fallback detailed activity")
	}
	cachedDetailed := repo.LoadDetailedActivityFromCache(clientID, activityYear, activityID)
	if cachedDetailed == nil {
		t.Fatalf("expected detailed activity %d to be persisted in local cache for year %d", activityID, activityYear)
	}
}

func TestGetDetailedActivity_PersistsDetailedFetchedWithoutBaseActivity(t *testing.T) {
	// GIVEN
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":77,"start_date":"2019-02-03T00:00:00Z","start_date_local":"2019-02-03T00:00:00Z"}`))
	}))
	defer server.Close()

	cacheDir := t.TempDir()
	repo := localrepository.NewStravaRepository(cacheDir)
	clientID := "123"
	repo.InitLocalStorageForClientId(clientID)
	repo.SaveActivitiesToCache(clientID, 2019, []strava.Activity{})

	provider := &StravaActivityProvider{
		clientId:             clientID,
		localStorageProvider: repo,
		StravaApi: &StravaApi{
			accessToken: "test-token",
			properties: StravaProperties{
				URL: server.URL,
			},
			httpClient: server.Client(),
		},
	}

	// WHEN
	detailed := provider.GetDetailedActivity(77)

	// THEN
	if detailed == nil {
		t.Fatal("expected detailed activity fetched from Strava API")
	}
	cachedDetailed := repo.LoadDetailedActivityFromCache(clientID, 2019, 77)
	if cachedDetailed == nil {
		t.Fatal("expected fetched detailed activity to be persisted in local cache")
	}
}

func benchmarkProvider(size int) *StravaActivityProvider {
	activities := make([]*strava.Activity, 0, size)
	for i := 0; i < size; i++ {
		year := 2023
		activityType := "Run"
		if i%2 == 0 {
			year = 2024
			activityType = "Ride"
		}
		activities = append(activities, &strava.Activity{
			Id:             int64(i),
			Type:           activityType,
			SportType:      activityType,
			StartDateLocal: fmt.Sprintf("%d-01-01T10:00:00Z", year),
			Distance:       1000,
		})
	}

	provider := &StravaActivityProvider{
		activities: activities,
	}
	provider.indexActivities()
	return provider
}
