package infrastructure

import (
	"mystravastats/internal/shared/domain/business"
	"mystravastats/internal/shared/domain/strava"
	"testing"
)

func TestBuildGearAnalysis_AggregatesGearAndUnassignedActivities(t *testing.T) {
	// GIVEN
	nickname := "Fast bike"
	retired := false
	activities := []*strava.Activity{
		gearAnalysisActivity(1, "Morning ride", "Ride", "2026-01-03T08:00:00Z", "b123", 10000, 1800, 100),
		gearAnalysisActivity(2, "Long ride", "Ride", "2026-02-05T08:00:00Z", "b123", 20000, 3000, 300),
		gearAnalysisActivity(3, "Run shoes", "Run", "2026-03-07T08:00:00Z", "g456", 5000, 1500, 20),
		gearAnalysisActivity(4, "No gear", "Hike", "2026-03-08T08:00:00Z", "", 7000, 2400, 200),
	}
	athlete := strava.Athlete{
		Bikes: []strava.Bike{
			{Id: "b123", Name: "Road Bike", Nickname: &nickname, Primary: true, Retired: &retired},
		},
		Shoes: []strava.Shoe{
			{Id: "g456", Name: "Trail Shoes", Primary: true},
		},
	}

	// WHEN
	result := buildGearAnalysis(activities, athlete, nil)

	// THEN
	if result.Coverage.TotalActivities != 4 || result.Coverage.AssignedActivities != 3 || result.Coverage.UnassignedActivities != 1 {
		t.Fatalf("unexpected coverage: %#v", result.Coverage)
	}
	if len(result.Items) != 2 {
		t.Fatalf("expected 2 gear items, got %d", len(result.Items))
	}

	bike := result.Items[0]
	if bike.ID != "b123" || bike.Name != "Fast bike" || bike.Kind != business.GearKindBike || !bike.Primary || bike.Retired {
		t.Fatalf("unexpected bike metadata: %#v", bike)
	}
	if bike.Distance != 30000 || bike.MovingTime != 4800 || bike.ElevationGain != 400 || bike.Activities != 2 {
		t.Fatalf("unexpected bike totals: %#v", bike)
	}
	if bike.AverageSpeed != 6.3 {
		t.Fatalf("expected weighted average speed 6.3m/s, got %.1f", bike.AverageSpeed)
	}
	if bike.FirstUsed != "2026-01-03T08:00:00Z" || bike.LastUsed != "2026-02-05T08:00:00Z" {
		t.Fatalf("unexpected first/last use: %s %s", bike.FirstUsed, bike.LastUsed)
	}
	if bike.LongestActivity == nil || bike.LongestActivity.Id != 2 {
		t.Fatalf("expected longest activity 2, got %#v", bike.LongestActivity)
	}
	if bike.FastestActivity == nil || bike.FastestActivity.Id != 2 {
		t.Fatalf("expected fastest activity 2, got %#v", bike.FastestActivity)
	}
	if len(bike.MonthlyDistance) != 2 || bike.MonthlyDistance[0].PeriodKey != "2026-01" || bike.MonthlyDistance[0].Value != 10000 {
		t.Fatalf("unexpected monthly distance: %#v", bike.MonthlyDistance)
	}

	if result.Unassigned.Activities != 1 || result.Unassigned.Distance != 7000 || result.Unassigned.ElevationGain != 200 {
		t.Fatalf("unexpected unassigned summary: %#v", result.Unassigned)
	}
}

func TestBuildGearAnalysis_AddsBikeMaintenanceTasksAndHistory(t *testing.T) {
	// GIVEN
	activities := []*strava.Activity{
		gearAnalysisActivity(1, "Morning ride", "Ride", "2026-01-03T08:00:00Z", "b123", 2000000, 1800, 100),
	}
	athlete := strava.Athlete{
		Bikes: []strava.Bike{
			{Id: "b123", Name: "Road Bike", Primary: true},
		},
	}
	records := []business.GearMaintenanceRecord{
		{
			ID:             "gm-1",
			GearID:         "b123",
			GearName:       "Road Bike",
			Component:      "CHAIN",
			ComponentLabel: "Chain",
			Operation:      "Chain changed",
			Date:           "2026-01-01",
			Distance:       100000,
			CreatedAt:      "2026-01-01T00:00:00Z",
			UpdatedAt:      "2026-01-01T00:00:00Z",
		},
	}

	// WHEN
	result := buildGearAnalysis(activities, athlete, records)

	// THEN
	bike := result.Items[0]
	if len(bike.MaintenanceHistory) != 1 {
		t.Fatalf("expected maintenance history, got %#v", bike.MaintenanceHistory)
	}
	chain := gearMaintenanceTaskByComponent(bike.MaintenanceTasks, "CHAIN")
	if chain == nil {
		t.Fatalf("expected chain maintenance task, got %#v", bike.MaintenanceTasks)
	}
	if chain.Status != "OVERDUE" {
		t.Fatalf("expected overdue chain task, got %#v", chain)
	}
	if chain.DistanceSince != 1900000 {
		t.Fatalf("expected 1900km since service, got %.1f", chain.DistanceSince)
	}
}

func gearAnalysisActivity(id int64, name string, activityType string, date string, gearID string, distance float64, movingTime int, elevationGain float64) *strava.Activity {
	var gearIDPtr *string
	if gearID != "" {
		gearIDPtr = &gearID
	}
	return &strava.Activity{
		Id:                 id,
		Name:               name,
		Type:               activityType,
		SportType:          activityType,
		StartDateLocal:     date,
		Distance:           distance,
		MovingTime:         movingTime,
		ElapsedTime:        movingTime,
		TotalElevationGain: elevationGain,
		GearId:             gearIDPtr,
	}
}

func gearMaintenanceTaskByComponent(tasks []business.GearMaintenanceTask, component string) *business.GearMaintenanceTask {
	for index := range tasks {
		if tasks[index].Component == component {
			return &tasks[index]
		}
	}
	return nil
}
