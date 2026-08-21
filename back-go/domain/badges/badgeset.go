package badges

import (
	"fmt"
	"mystravastats/internal/shared/domain/business"
	"mystravastats/internal/shared/domain/strava"
	"strings"
)

type BadgeSet struct {
	Name   string
	Badges []Badge
}

func NewBadgeSet(name string, badges []Badge) BadgeSet {
	return BadgeSet{
		Name:   name,
		Badges: badges,
	}
}

func (b BadgeSet) Check(activities []*strava.Activity) []business.BadgeCheckResult {
	var results []business.BadgeCheckResult
	for _, badge := range b.Badges {
		checkedActivities, isCompleted := badge.Check(activities)
		results = append(results, business.BadgeCheckResult{
			Badge:       badge,
			Activities:  checkedActivities,
			IsCompleted: isCompleted,
		})
	}
	return deduplicateFamousClimbActivities(results)
}

type famousClimbActivityWinner struct {
	resultIndex int
	quality     float64
}

func deduplicateFamousClimbActivities(results []business.BadgeCheckResult) []business.BadgeCheckResult {
	winners := make(map[string]map[*strava.Activity]famousClimbActivityWinner)
	for resultIndex, result := range results {
		badge, ok := result.Badge.(FamousClimbBadge)
		if !ok {
			continue
		}
		groupKey := famousClimbGroupKey(badge)
		if winners[groupKey] == nil {
			winners[groupKey] = make(map[*strava.Activity]famousClimbActivityWinner)
		}
		for _, activity := range result.Activities {
			quality, matched := badge.matchQuality(activity)
			if !matched {
				continue
			}
			winner, alreadyAssigned := winners[groupKey][activity]
			if !alreadyAssigned || quality < winner.quality {
				winners[groupKey][activity] = famousClimbActivityWinner{resultIndex: resultIndex, quality: quality}
			}
		}
	}

	for resultIndex := range results {
		badge, ok := results[resultIndex].Badge.(FamousClimbBadge)
		if !ok {
			continue
		}
		groupKey := famousClimbGroupKey(badge)
		filteredActivities := make([]*strava.Activity, 0, len(results[resultIndex].Activities))
		for _, activity := range results[resultIndex].Activities {
			if winner, assigned := winners[groupKey][activity]; assigned && winner.resultIndex == resultIndex {
				filteredActivities = append(filteredActivities, activity)
			}
		}
		results[resultIndex].Activities = filteredActivities
		results[resultIndex].IsCompleted = len(filteredActivities) > 0
	}

	return results
}

func famousClimbGroupKey(badge FamousClimbBadge) string {
	return fmt.Sprintf(
		"%s|%.5f|%.5f",
		strings.ToLower(strings.TrimSpace(badge.Name)),
		badge.End.Latitude,
		badge.End.Longitude,
	)
}

func (b BadgeSet) Plus(anotherBadgeSet BadgeSet) BadgeSet {
	return BadgeSet{
		Name:   b.Name,
		Badges: append(b.Badges, anotherBadgeSet.Badges...),
	}
}
