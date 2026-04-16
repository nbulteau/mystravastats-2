package application

import "mystravastats/internal/shared/domain/business"

// BadgesReader is an outbound port used by badges use cases.
// Infrastructure adapters implement this interface.
type BadgesReader interface {
	FindGeneralBadges(year *int, activityTypes ...business.ActivityType) []business.BadgeCheckResult
	FindFamousBadges(year *int, activityTypes ...business.ActivityType) []business.BadgeCheckResult
}
