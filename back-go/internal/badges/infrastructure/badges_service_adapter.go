package infrastructure

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mystravastats/domain/badges"
	"mystravastats/domain/business"
	"mystravastats/internal/platform/activityprovider"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var (
	famousBadgeSetsOnce sync.Once
	alpesBadgeSet       badges.BadgeSet
	pyreneesBadgeSet    badges.BadgeSet
)

// BadgesServiceAdapter computes badges directly from provider activities.
type BadgesServiceAdapter struct{}

func NewBadgesServiceAdapter() *BadgesServiceAdapter {
	return &BadgesServiceAdapter{}
}

func (adapter *BadgesServiceAdapter) FindGeneralBadges(year *int, activityTypes ...business.ActivityType) []business.BadgeCheckResult {
	log.Printf("Checking general badges for %s in %v", activityTypes, year)
	if len(activityTypes) == 0 {
		return []business.BadgeCheckResult{}
	}

	activities := activityprovider.Get().GetActivitiesByYearAndActivityTypes(year, activityTypes...)
	activityType := activityTypes[0]

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

func (adapter *BadgesServiceAdapter) FindFamousBadges(year *int, activityTypes ...business.ActivityType) []business.BadgeCheckResult {
	log.Printf("Checking famous badges for %s in %v", activityTypes, year)
	if len(activityTypes) == 0 {
		return []business.BadgeCheckResult{}
	}

	famousBadgeSetsOnce.Do(func() {
		alpesBadgeSet = loadBadgeSet("alpes", "strava-cache/famous-climb/alpes.json")
		pyreneesBadgeSet = loadBadgeSet("pyrenees", "strava-cache/famous-climb/pyrenees.json")
	})

	activities := activityprovider.Get().GetActivitiesByYearAndActivityTypes(year, activityTypes...)
	activityType := activityTypes[0]
	if activityType == business.Ride {
		return append(alpesBadgeSet.Check(activities), pyreneesBadgeSet.Check(activities)...)
	}

	return []business.BadgeCheckResult{}
}

func loadBadgeSet(name string, climbsJSONFilePath string) badges.BadgeSet {
	var famousClimbBadgeList []badges.Badge

	filePath, err := filepath.Abs(climbsJSONFilePath)
	if err != nil {
		log.Printf("Error getting absolute path: %v", err)
		return badges.BadgeSet{Name: name, Badges: famousClimbBadgeList}
	}

	file, err := os.Open(filePath)
	if err != nil {
		log.Printf("Error opening file: %v", err)
		return badges.BadgeSet{Name: name, Badges: famousClimbBadgeList}
	}
	defer func(file *os.File) {
		if closeErr := file.Close(); closeErr != nil {
			log.Printf("Error closing file: %v", closeErr)
		}
	}(file)

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
				Start:           alternative.GeoCoordinate,
				End:             famousClimb.GeoCoordinate,
				Difficulty:      alternative.Difficulty,
				Category:        normalizeClimbCategory(alternative.Category, alternative.Difficulty),
				Length:          alternative.Length,
				TotalAscent:     alternative.TotalAscent,
				AverageGradient: alternative.AverageGradient,
			})
		}
	}

	return badges.BadgeSet{Name: name, Badges: famousClimbBadgeList}
}

func normalizeClimbCategory(category string, difficulty int) string {
	if category != "" {
		normalized := strings.TrimSpace(strings.ToUpper(category))
		switch normalized {
		case "HC", "1", "2", "3", "4":
			return normalized
		}
	}

	switch {
	case difficulty >= 1000:
		return "HC"
	case difficulty >= 600:
		return "1"
	case difficulty >= 300:
		return "2"
	case difficulty >= 150:
		return "3"
	default:
		return "4"
	}
}
