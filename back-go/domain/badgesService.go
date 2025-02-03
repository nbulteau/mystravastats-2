package domain

import (
	"log"
	"mystravastats/domain/business"
	"mystravastats/domain/strava"
)

func GetGeneralBadges(activityType business.ActivityType, year *int) []strava.BadgeCheckResult {

	// TODO: Implement the logic to get the general badges
	log.Default().Print("GetGeneralBadges not implemented yet")

	return make([]strava.BadgeCheckResult, 0)
}

func GetFamousBadges(activityType business.ActivityType, year *int) []strava.BadgeCheckResult {

	// TODO: Implement the logic to get the general badges
	log.Default().Print("GetFamousBadges not implemented yet")

	return make([]strava.BadgeCheckResult, 0)
}
