package infrastructure

import (
	"mystravastats/domain/business"
	"mystravastats/internal/platform/activityprovider"
	"mystravastats/internal/services"
)

// HeartRateServiceAdapter bridges the current internal/services layer
// to the hexagonal outbound ports used by heart-rate use cases.
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
	return services.FetchHeartRateZoneAnalysisByActivityTypeAndYear(year, activityTypes...)
}
