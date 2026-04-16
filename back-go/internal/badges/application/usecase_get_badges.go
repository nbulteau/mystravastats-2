package application

import "mystravastats/internal/shared/domain/business"

type GetBadgesUseCase struct {
	reader BadgesReader
}

func NewGetBadgesUseCase(reader BadgesReader) *GetBadgesUseCase {
	return &GetBadgesUseCase{
		reader: reader,
	}
}

func (uc *GetBadgesUseCase) Execute(year *int, badgeSet *business.BadgeSetEnum, activityTypes []business.ActivityType) []business.BadgeCheckResult {
	switch {
	case badgeSet != nil && *badgeSet == business.GENERAL:
		return uc.reader.FindGeneralBadges(year, activityTypes...)
	case badgeSet != nil && *badgeSet == business.FAMOUS:
		return uc.reader.FindFamousBadges(year, activityTypes...)
	default:
		generalBadges := uc.reader.FindGeneralBadges(year, activityTypes...)
		famousBadges := uc.reader.FindFamousBadges(year, activityTypes...)
		return append(generalBadges, famousBadges...)
	}
}
