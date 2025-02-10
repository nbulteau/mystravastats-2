package stravaapi

import (
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