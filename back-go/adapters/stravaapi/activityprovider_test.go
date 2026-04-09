package stravaapi

import (
	"fmt"
	"mystravastats/adapters/localrepository"
	"mystravastats/domain/business"
	"mystravastats/domain/strava"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFilterActivitiesByType(t *testing.T) {
	activities := []*strava.Activity{
		{Type: "Ride", Commute: false},
		{Type: "Run", Commute: false},
		{Type: "Ride", Commute: true},
	}

	rides := FilterActivitiesByType(activities, business.Ride)
	if len(rides) != 1 {
		t.Errorf("Expected 1 ride, got %d", len(rides))
	}

	commutes := FilterActivitiesByType(activities, business.Commute)
	if len(commutes) != 1 {
		t.Errorf("Expected 1 commute, got %d", len(commutes))
	}
}

func TestShouldBootstrapFromStravaAPIWhenCurrentYearCacheIsFresh(t *testing.T) {
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

	if provider.shouldBootstrapFromStravaAPI(clientID) {
		t.Fatal("expected fresh local cache to skip Strava bootstrap")
	}
}

func TestShouldBootstrapFromStravaAPIWhenCurrentYearCacheIsMissing(t *testing.T) {
	cacheDir := t.TempDir()
	repo := localrepository.NewStravaRepository(cacheDir)
	clientID := "123"
	repo.InitLocalStorageForClientId(clientID)
	repo.SaveAthleteToCache(clientID, strava.Athlete{Id: 123})

	provider := &StravaActivityProvider{
		localStorageProvider: repo,
	}

	if !provider.shouldBootstrapFromStravaAPI(clientID) {
		t.Fatal("expected missing current-year cache to require Strava bootstrap")
	}
}

func TestShouldBootstrapFromStravaAPIWhenCacheIsTooOld(t *testing.T) {
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

	if !provider.shouldBootstrapFromStravaAPI(clientID) {
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
