package badges

import (
	"mystravastats/domain/business"
	"mystravastats/domain/strava"
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
	return results
}

func (b BadgeSet) Plus(anotherBadgeSet BadgeSet) BadgeSet {
	return BadgeSet{
		Name:   b.Name,
		Badges: append(b.Badges, anotherBadgeSet.Badges...),
	}
}
