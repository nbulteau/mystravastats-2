package application

import (
	"mystravastats/domain/business"
	"testing"
)

type heartRateReaderStub struct {
	settings      business.HeartRateZoneSettings
	analysis      business.HeartRateZoneAnalysis
	receivedYear  *int
	receivedTypes []business.ActivityType
}

func (stub *heartRateReaderStub) FindHeartRateZoneSettings() business.HeartRateZoneSettings {
	return stub.settings
}

func (stub *heartRateReaderStub) SaveHeartRateZoneSettings(settings business.HeartRateZoneSettings) business.HeartRateZoneSettings {
	stub.settings = settings
	return settings
}

func (stub *heartRateReaderStub) FindHeartRateZoneAnalysisByYearAndTypes(year *int, activityTypes ...business.ActivityType) business.HeartRateZoneAnalysis {
	stub.receivedYear = year
	stub.receivedTypes = append([]business.ActivityType(nil), activityTypes...)
	return stub.analysis
}

func TestGetHeartRateZoneSettingsUseCase_Execute_ReturnsSettings(t *testing.T) {
	// GIVEN
	maxHR := 190
	reader := &heartRateReaderStub{
		settings: business.HeartRateZoneSettings{MaxHr: &maxHR},
	}
	useCase := NewGetHeartRateZoneSettingsUseCase(reader)

	// WHEN
	result := useCase.Execute()

	// THEN
	if result.MaxHr == nil || *result.MaxHr != maxHR {
		t.Fatalf("expected maxHr=%d, got %+v", maxHR, result.MaxHr)
	}
}

func TestUpdateHeartRateZoneSettingsUseCase_Execute_SavesAndReturnsSettings(t *testing.T) {
	// GIVEN
	reader := &heartRateReaderStub{}
	useCase := NewUpdateHeartRateZoneSettingsUseCase(reader)
	threshold := 170

	// WHEN
	result := useCase.Execute(business.HeartRateZoneSettings{ThresholdHr: &threshold})

	// THEN
	if result.ThresholdHr == nil || *result.ThresholdHr != threshold {
		t.Fatalf("expected thresholdHr=%d, got %+v", threshold, result.ThresholdHr)
	}
}

func TestGetHeartRateZoneAnalysisUseCase_Execute_ForwardsInputs(t *testing.T) {
	// GIVEN
	year := 2026
	reader := &heartRateReaderStub{
		analysis: business.HeartRateZoneAnalysis{HasHeartRateData: true},
	}
	useCase := NewGetHeartRateZoneAnalysisUseCase(reader)
	types := []business.ActivityType{business.Ride}

	// WHEN
	result := useCase.Execute(&year, types)

	// THEN
	if !result.HasHeartRateData {
		t.Fatal("expected analysis to be returned from reader")
	}
	if reader.receivedYear == nil || *reader.receivedYear != year {
		t.Fatalf("expected year=%d, got %+v", year, reader.receivedYear)
	}
	if len(reader.receivedTypes) != len(types) {
		t.Fatalf("expected %d activity types, got %d", len(types), len(reader.receivedTypes))
	}
}
