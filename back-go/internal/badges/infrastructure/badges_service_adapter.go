package infrastructure

import (
	"mystravastats/domain/business"
	"mystravastats/internal/services"
)

// BadgesServiceAdapter bridges the current internal/services layer
// to the hexagonal outbound ports used by badges use cases.
type BadgesServiceAdapter struct{}

func NewBadgesServiceAdapter() *BadgesServiceAdapter {
	return &BadgesServiceAdapter{}
}

func (adapter *BadgesServiceAdapter) FindGeneralBadges(year *int, activityTypes ...business.ActivityType) []business.BadgeCheckResult {
	return services.GetGeneralBadges(year, activityTypes...)
}

func (adapter *BadgesServiceAdapter) FindFamousBadges(year *int, activityTypes ...business.ActivityType) []business.BadgeCheckResult {
	return services.GetFamousBadges(year, activityTypes...)
}
