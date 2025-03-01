package services

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"mystravastats/domain/badges"
	"mystravastats/domain/business"
	"mystravastats/domain/helpers"
)

var alpes = loadBadgeSet("alpes", helpers.StravaCachePath+"/famous-climb/alpes.json")
var pyrenees = loadBadgeSet("pyrenees", helpers.StravaCachePath+"/famous-climb/pyrenees.json")

// GetGeneralBadges returns the general badges for the given activity type and year
// The general badges are the ones that are not specific to a famous climb
func GetGeneralBadges(activityType business.ActivityType, year *int) []business.BadgeCheckResult {
	log.Printf("Checking general badges for %s in %v", activityType, year)

	activities := activityProvider.GetActivitiesByActivityTypeAndYear(activityType, year)

	switch activityType {
	case business.Ride:
		return append(append(badges.DistanceRideBadgeSet.Check(activities),
			badges.ElevationRideBadgeSet.Check(activities)...),
			badges.MovingTimeBadgesSet.Check(activities)...)
	case business.Hike:
		return append(append(badges.DistanceHikeBadgeSet.Check(activities),
			badges.ElevationHikeBadgeSet.Check(activities)...),
			badges.MovingTimeBadgesSet.Check(activities)...)
	case business.Run:
		return append(append(badges.DistanceRunBadgeSet.Check(activities),
			badges.ElevationRunBadgeSet.Check(activities)...),
			badges.MovingTimeBadgesSet.Check(activities)...)
	default:
		return []business.BadgeCheckResult{}
	}
}

// GetFamousBadges returns the famous badges for the given activity type and year
// The famous badges are the ones that are specific to a famous climb
func GetFamousBadges(activityType business.ActivityType, year *int) []business.BadgeCheckResult {
	log.Printf("Checking famous badges for %s in %v", activityType, year)

	activities := activityProvider.GetActivitiesByActivityTypeAndYear(activityType, year)

	switch activityType {
	case business.Ride:
		return append(alpes.Check(activities), pyrenees.Check(activities)...)
	default:
		return []business.BadgeCheckResult{}
	}
}

// loadBadgeSet loads the badge set from the given JSON file
func loadBadgeSet(name string, climbsJsonFilePath string) badges.BadgeSet {
	var famousClimbBadgeList []badges.Badge

	filePath, err := filepath.Abs(climbsJsonFilePath)
	if err != nil {
		log.Printf("Error getting absolute path: %v", err)
		return badges.BadgeSet{Name: name, Badges: famousClimbBadgeList}
	}

	file, err := os.Open(filePath)
	if err != nil {
		log.Printf("Error opening file: %v", err)
		return badges.BadgeSet{Name: name, Badges: famousClimbBadgeList}
	}
	defer file.Close()

	byteValue, err := io.ReadAll(file)
	if err != nil {
		log.Printf("Error reading file: %v", err)
		return badges.BadgeSet{Name: name, Badges: famousClimbBadgeList}
	}

	var famousClimbs []badges.FamousClimb
	if err := json.Unmarshal(byteValue, &famousClimbs); err != nil {
		log.Printf("Error unmarshalling JSON: %v", err)
		return badges.BadgeSet{Name: name, Badges: famousClimbBadgeList}
	}

	for _, famousClimb := range famousClimbs {
		for _, alternative := range famousClimb.Alternatives {
			famousClimbBadgeList = append(famousClimbBadgeList, badges.FamousClimbBadge{
				Name:            famousClimb.Name,
				Label:           fmt.Sprintf("%s from %s", famousClimb.Name, alternative.Name),
				TopOfTheAscent:  famousClimb.TopOfTheAscent,
				Start:           famousClimb.GeoCoordinate,
				End:             alternative.GeoCoordinate,
				Difficulty:      alternative.Difficulty,
				Length:          alternative.Length,
				TotalAscent:     alternative.TotalAscent,
				AverageGradient: alternative.AverageGradient,
			})
		}
	}

	return badges.BadgeSet{Name: name, Badges: famousClimbBadgeList}
}
