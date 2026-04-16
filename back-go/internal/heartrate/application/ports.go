package application

import "mystravastats/internal/shared/domain/business"

// HeartRateReader is an outbound port used by heart-rate use cases.
// Infrastructure adapters implement this interface.
type HeartRateReader interface {
	FindHeartRateZoneSettings() business.HeartRateZoneSettings
	SaveHeartRateZoneSettings(settings business.HeartRateZoneSettings) business.HeartRateZoneSettings
	FindHeartRateZoneAnalysisByYearAndTypes(year *int, activityTypes ...business.ActivityType) business.HeartRateZoneAnalysis
}
