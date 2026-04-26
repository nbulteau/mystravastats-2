package application

import "mystravastats/internal/shared/domain/business"

type GearAnalysisReader interface {
	FindGearAnalysis(year *int, activityTypes ...business.ActivityType) business.GearAnalysis
}
