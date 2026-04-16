package infrastructure

import (
	"mystravastats/internal/platform/activityprovider"
	"mystravastats/internal/shared/domain/business"
)

// HeartRateServiceAdapter computes heart-rate zone analysis from provider data.
type HeartRateServiceAdapter struct{}

func NewHeartRateServiceAdapter() *HeartRateServiceAdapter {
	return &HeartRateServiceAdapter{}
}

func (adapter *HeartRateServiceAdapter) FindHeartRateZoneSettings() business.HeartRateZoneSettings {
	return activityprovider.Get().GetHeartRateZoneSettings()
}

func (adapter *HeartRateServiceAdapter) SaveHeartRateZoneSettings(settings business.HeartRateZoneSettings) business.HeartRateZoneSettings {
	return activityprovider.Get().SaveHeartRateZoneSettings(settings)
}

func (adapter *HeartRateServiceAdapter) FindHeartRateZoneAnalysisByYearAndTypes(year *int, activityTypes ...business.ActivityType) business.HeartRateZoneAnalysis {
	return computeHeartRateZoneAnalysisByYearAndTypes(year, activityTypes...)
}
