package application

import (
	"mystravastats/internal/shared/domain/business"
	"mystravastats/internal/shared/domain/strava"
	"testing"
	"time"
)

type athleteReaderStub struct {
	athlete             strava.Athlete
	activities          []*strava.Activity
	performanceSettings business.AthletePerformanceSettings
}

func (stub *athleteReaderStub) FindAthlete() strava.Athlete {
	return stub.athlete
}

func (stub *athleteReaderStub) FindActivitiesByYearAndTypes(_ *int, _ ...business.ActivityType) []*strava.Activity {
	return stub.activities
}

func (stub *athleteReaderStub) FindPerformanceSettings() business.AthletePerformanceSettings {
	return stub.performanceSettings
}

func (stub *athleteReaderStub) SavePerformanceSettings(settings business.AthletePerformanceSettings) business.AthletePerformanceSettings {
	stub.performanceSettings = settings
	return settings
}

func TestGetAthleteUseCase_Execute_ReturnsAthlete(t *testing.T) {
	// GIVEN
	expected := strava.Athlete{Id: 42}
	reader := &athleteReaderStub{athlete: expected}
	useCase := NewGetAthleteUseCase(reader)

	// WHEN
	result := useCase.Execute()

	// THEN
	if result.Id != expected.Id {
		t.Fatalf("expected athlete id %d, got %d", expected.Id, result.Id)
	}
}

func TestUpdatePerformanceSettingsUseCase_Execute_NormalizesAndSavesSettings(t *testing.T) {
	// GIVEN
	reader := &athleteReaderStub{}
	useCase := NewUpdatePerformanceSettingsUseCase(reader)
	weight := 72.5

	// WHEN
	result := useCase.Execute(business.AthletePerformanceSettings{
		WeightKg: &weight,
		FtpHistory: []business.AthleteFtpSetting{
			{EffectiveFrom: "2026-02-01", Ftp: 170},
			{EffectiveFrom: "bad-date", Ftp: 999},
			{EffectiveFrom: "2026-01-01", Ftp: 160},
			{EffectiveFrom: "2026-02-01", Ftp: 175},
			{EffectiveFrom: "2026-03-01", Ftp: -1},
		},
	})

	// THEN
	if result.WeightKg == nil || *result.WeightKg != weight {
		t.Fatalf("expected weightKg=%f, got %+v", weight, result.WeightKg)
	}
	if len(result.FtpHistory) != 2 {
		t.Fatalf("expected 2 FTP entries, got %+v", result.FtpHistory)
	}
	if result.FtpHistory[0].EffectiveFrom != "2026-01-01" || result.FtpHistory[0].Ftp != 160 {
		t.Fatalf("unexpected first FTP entry: %+v", result.FtpHistory[0])
	}
	if result.FtpHistory[1].EffectiveFrom != "2026-02-01" || result.FtpHistory[1].Ftp != 175 {
		t.Fatalf("unexpected second FTP entry: %+v", result.FtpHistory[1])
	}
}

func TestGetFtpEstimateUseCase_UsesRecentDeviceBest60MinutePower(t *testing.T) {
	// GIVEN
	reader := &athleteReaderStub{
		activities: []*strava.Activity{
			syntheticPowerActivity(101, "Estimated power", "2026-06-18T09:00:00Z", false, 300, 3600, 600),
			syntheticPowerActivity(102, "Current power meter", "2026-06-20T09:00:00Z", true, 210, 3600, 600),
			syntheticPowerActivity(103, "Old power meter", "2025-01-01T09:00:00Z", true, 260, 3600, 600),
		},
	}
	useCase := NewGetFtpEstimateUseCase(reader)

	// WHEN
	result := useCase.Execute([]business.ActivityType{business.Ride}, 180)

	// THEN
	if !result.Available {
		t.Fatalf("expected FTP estimate to be available")
	}
	if result.Ftp != 210 {
		t.Fatalf("expected FTP 210 from recent device power, got %d", result.Ftp)
	}
	if result.ActivityID != 102 {
		t.Fatalf("expected activity 102, got %d", result.ActivityID)
	}
	if result.Method != "best-60min" || result.Confidence != "high" {
		t.Fatalf("unexpected method/confidence: %s/%s", result.Method, result.Confidence)
	}
}

func TestGetFtpEstimateUseCase_FallsBackToTwentyMinutePower(t *testing.T) {
	// GIVEN
	reader := &athleteReaderStub{
		activities: []*strava.Activity{
			syntheticPowerActivity(104, "Twenty minute test", "2026-06-20T09:00:00Z", true, 200, 1200, 600),
		},
	}
	useCase := NewGetFtpEstimateUseCase(reader)

	// WHEN
	result := useCase.Execute([]business.ActivityType{business.Ride}, 180)

	// THEN
	if !result.Available {
		t.Fatalf("expected FTP estimate to be available")
	}
	if result.Ftp != 190 {
		t.Fatalf("expected FTP 190 from 95%% of 200 W, got %d", result.Ftp)
	}
	if result.Method != "95-percent-20min" {
		t.Fatalf("expected 20-minute method, got %s", result.Method)
	}
	if result.BestPower != 200 || result.BasedOnSeconds != 1200 {
		t.Fatalf("unexpected effort details: %+v", result)
	}
}

func TestGetFtpEstimateUseCase_SelectsHighestAveragePowerNotLongestDistance(t *testing.T) {
	// GIVEN
	reader := &athleteReaderStub{
		activities: []*strava.Activity{
			syntheticPowerActivityWithDistance(105, "Long steady ride", "2026-06-19T09:00:00Z", true, 180, 3600, 6000),
			syntheticPowerActivityWithDistance(106, "Short hard ride", "2026-06-20T09:00:00Z", true, 230, 3600, 1500),
		},
	}
	useCase := NewGetFtpEstimateUseCase(reader)

	// WHEN
	result := useCase.Execute([]business.ActivityType{business.Ride}, 180)

	// THEN
	if result.ActivityID != 106 {
		t.Fatalf("expected the highest-power activity, got %d", result.ActivityID)
	}
	if result.Ftp != 230 {
		t.Fatalf("expected FTP 230, got %d", result.Ftp)
	}
}

func syntheticPowerActivity(id int64, name string, startDateLocal string, deviceWatts bool, watts int, durationSeconds int, stepSeconds int) *strava.Activity {
	return syntheticPowerActivityWithDistance(id, name, startDateLocal, deviceWatts, watts, durationSeconds, float64(durationSeconds/stepSeconds)*500)
}

func syntheticPowerActivityWithDistance(id int64, name string, startDateLocal string, deviceWatts bool, watts int, durationSeconds int, totalDistance float64) *strava.Activity {
	points := durationSeconds/600 + 1
	if points < 2 {
		points = 2
	}
	distances := make([]float64, points)
	times := make([]int, points)
	altitudes := make([]float64, points)
	powers := make([]float64, points)
	for i := 0; i < points; i++ {
		distances[i] = totalDistance * float64(i) / float64(points-1)
		times[i] = i * 600
		altitudes[i] = 100 + float64(i)
		powers[i] = float64(watts)
	}
	return &strava.Activity{
		Id:             id,
		Name:           name,
		Type:           "Ride",
		SportType:      "Ride",
		DeviceWatts:    deviceWatts,
		StartDate:      startDateLocal,
		StartDateLocal: startDateLocal,
		Stream: &strava.Stream{
			Distance: strava.DistanceStream{Data: distances, OriginalSize: len(distances), Resolution: "high", SeriesType: "distance"},
			Time:     strava.TimeStream{Data: times, OriginalSize: len(times), Resolution: "high", SeriesType: "time"},
			Altitude: &strava.AltitudeStream{Data: altitudes, OriginalSize: len(altitudes), Resolution: "high", SeriesType: "distance"},
			Watts:    &strava.PowerStream{Data: powers, OriginalSize: len(powers), Resolution: "high", SeriesType: "time"},
		},
	}
}

func TestGetFtpEstimateUseCase_UsesClockWhenDatesAreMissing(t *testing.T) {
	// GIVEN
	reader := &athleteReaderStub{
		activities: []*strava.Activity{
			syntheticPowerActivity(107, "No date ride", "", true, 180, 3600, 600),
		},
	}
	useCase := NewGetFtpEstimateUseCase(reader)
	useCase.now = func() time.Time {
		return time.Date(2026, 6, 25, 0, 0, 0, 0, time.UTC)
	}

	// WHEN
	result := useCase.Execute([]business.ActivityType{business.Ride}, 180)

	// THEN
	if result.Ftp != 180 {
		t.Fatalf("expected all-time fallback FTP 180, got %d", result.Ftp)
	}
	if result.Source != "Power meter, all time" {
		t.Fatalf("expected all-time source, got %q", result.Source)
	}
}
