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
)

var alpes = loadBadgeSet("alpes", "famous-climb/alpes.json")
var pyrenees = loadBadgeSet("pyrenees", "famous-climb/pyrenees.json")

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

func loadBadgeSet(name string, climbsJsonFilePath string) badges.BadgeSet {
	var famousClimbBadgeList []badges.Badge

	filePath, err := filepath.Abs(climbsJsonFilePath)
	if err != nil {
		log.Fatalf("Error getting absolute path: %v", err)
	}

	file, err := os.Open(filePath)
	if err != nil {
		log.Fatalf("Error opening file: %v", err)
	}
	defer file.Close()

	byteValue, err := io.ReadAll(file)
	if err != nil {
		log.Fatalf("Error reading file: %v", err)
	}

	var famousClimbs []badges.FamousClimb
	err = json.Unmarshal(byteValue, &famousClimbs)
	if err != nil {
		log.Fatalf("Error unmarshalling JSON: %v", err)
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
