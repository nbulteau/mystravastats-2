package stravaapi

import (
    "fmt"
    "mystravastats/domain/business"
    "mystravastats/domain/strava"
    "testing"
)

func TestFilterActivitiesByType(t *testing.T) {
    activities := []*strava.Activity{
        {Type: "Ride", Commute: false},
        {Type: "Run", Commute: false},
        {Type: "Ride", Commute: true},
    }

    // Test filtering rides
    rides := FilterActivitiesByType(activities, business.Ride)
    if len(rides) != 1 {
        t.Errorf("Expected 1 ride, got %d", len(rides))
    }

    // Test filtering commutes
    commutes := FilterActivitiesByType(activities, business.Commute)
    if len(commutes) != 1 {
        t.Errorf("Expected 1 commute, got %d", len(commutes))
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
