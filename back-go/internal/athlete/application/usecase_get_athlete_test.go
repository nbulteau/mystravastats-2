package application

import (
	"mystravastats/internal/shared/domain/business"
	"mystravastats/internal/shared/domain/strava"
	"testing"
)

type athleteReaderStub struct {
	athlete             strava.Athlete
	performanceSettings business.AthletePerformanceSettings
}

func (stub *athleteReaderStub) FindAthlete() strava.Athlete {
	return stub.athlete
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
